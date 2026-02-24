package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mapping"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type server struct {
	db *pgxpool.Pool
}

// mockS3UploadFromReader simulates uploading a file to S3 by generating metadata
func mockS3UploadFromReader(
	fileName string,
	fileData io.Reader,
) (s3Key string, fileSize int64, checksum string, mimeType string, err error) {
	// Read file to calculate checksum and size
	hasher := sha256.New()
	fileSize, err = io.Copy(hasher, fileData)
	if err != nil {
		return "", 0, "", "", err
	}

	checksum = fmt.Sprintf("%x", hasher.Sum(nil))
	s3Key = checksum

	// Simple MIME type detection from file extension
	mimeType = "application/octet-stream"
	if len(fileName) > 4 {
		ext := fileName[len(fileName)-4:]
		switch ext {
		case ".txt":
			mimeType = "text/plain"
		case ".pdf":
			mimeType = "application/pdf"
		case ".jpg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		}
	}

	return s3Key, fileSize, checksum, mimeType, nil
}

// DeleteApiV1DateiId implements [StrictServerInterface].
func (s *server) DeleteApiV1DateiId(
	ctx context.Context,
	request DeleteApiV1DateiIdRequestObject,
) (DeleteApiV1DateiIdResponseObject, error) {
	queries := db.New(s.db)
	dateiID := request.Id

	// Get Datei to verify it exists
	_, err := queries.GetDateiByID(ctx, dateiID)
	if err != nil {
		return DeleteApiV1DateiId404Response{}, nil
	}

	// Soft delete by setting trashed_at
	_, err = queries.SetDateiTrashedAt(ctx, dateiID)
	if err != nil {
		return DeleteApiV1DateiId409Response{}, nil
	}

	// Return 204 No Content
	return DeleteApiV1DateiId204Response{}, nil
}

// GetApiV1Datei implements [StrictServerInterface].
func (s *server) GetApiV1Datei(
	ctx context.Context,
	request GetApiV1DateiRequestObject,
) (GetApiV1DateiResponseObject, error) {
	queries := db.New(s.db)

	// Get all Datei records
	allDatei, err := queries.ListDatei(ctx)
	if err != nil {
		return GetApiV1Datei400Response{}, err
	}

	// Get pagination parameters
	limit := 100
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}

	total := len(allDatei)

	// Apply pagination
	start := min(offset, len(allDatei))
	end := min(offset+limit, len(allDatei))

	paginatedDatei := allDatei[start:end]

	// Fetch latest versions and names for each Datei
	versions := make(map[uuid.UUID]*db.DateiVersion)
	names := make(map[uuid.UUID]*string)

	for _, d := range paginatedDatei {
		if d.LatestVersionID != nil {
			v, err := queries.GetDateiVersionByID(ctx, *d.LatestVersionID)
			if err == nil {
				versions[*d.LatestVersionID] = &v
			}
		}
		if d.LatestNameID != nil {
			n, err := queries.GetDateiNameByID(ctx, *d.LatestNameID)
			if err == nil {
				names[*d.LatestNameID] = &n.Name
			}
		}
	}

	// Map to API response
	items := mapping.MapDBDateiSliceToAPI(paginatedDatei, versions, names)

	response := api.ListDateiResponse{
		Items: items,
		Total: total,
	}

	return GetApiV1Datei200JSONResponse(response), nil
}

// PatchApiV1DateiId implements [StrictServerInterface].
func (s *server) PatchApiV1DateiId(
	ctx context.Context,
	request PatchApiV1DateiIdRequestObject,
) (PatchApiV1DateiIdResponseObject, error) {
	queries := db.New(s.db)
	dateiID := request.Id

	// Get existing Datei
	datei, err := queries.GetDateiByID(ctx, dateiID)
	if err != nil {
		return PatchApiV1DateiId404Response{}, nil
	}

	// The request.Body is already a multipart reader
	reader := request.Body
	var name *string
	var fileData io.Reader
	var fileName string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}

		switch part.FormName() {
		case "name":
			buf := make([]byte, 256)
			n, _ := part.Read(buf)
			if n > 0 {
				nameStr := string(buf[:n])
				name = &nameStr
			}
		case "file":
			fileName = part.FileName()
			fileData = part
		}
	}

	// Update name if provided
	if name != nil {
		nameRecord, err := queries.CreateDateiName(ctx, db.CreateDateiNameParams{
			DateiID: datei.ID,
			Name:    *name,
		})
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}

		datei, err = queries.UpdateDateiLatestNameID(ctx, db.UpdateDateiLatestNameIDParams{
			ID:           datei.ID,
			LatestNameID: &nameRecord.ID,
		})
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}
	}

	// Update file if provided
	var latestVersion *db.DateiVersion
	if fileData != nil && fileName != "" {
		s3Key, fileSize, checksum, mimeType, err := mockS3UploadFromReader(fileName, fileData)
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    s3Key,
			FileSize: fileSize,
			Checksum: checksum,
			MimeType: mimeType,
		})
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}

		datei, err = queries.UpdateDateiLatestVersionID(ctx, db.UpdateDateiLatestVersionIDParams{
			ID:              datei.ID,
			LatestVersionID: &versionRecord.ID,
		})
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}

		latestVersion = &versionRecord
	}

	// Get latest version if not just updated
	if latestVersion == nil && datei.LatestVersionID != nil {
		v, err := queries.GetDateiVersionByID(ctx, *datei.LatestVersionID)
		if err == nil {
			latestVersion = &v
		}
	}

	// Get name if not just updated
	var currentName *string
	if datei.LatestNameID != nil {
		n, err := queries.GetDateiNameByID(ctx, *datei.LatestNameID)
		if err == nil {
			currentName = &n.Name
		}
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&datei, latestVersion, currentName)
	return PatchApiV1DateiId200JSONResponse(*response), nil
}

// PostApiV1Datei implements [StrictServerInterface].
func (s *server) PostApiV1Datei(
	ctx context.Context,
	request PostApiV1DateiRequestObject,
) (PostApiV1DateiResponseObject, error) {
	// The request.Body is already a multipart reader
	reader := request.Body
	var name string
	var fileData io.Reader
	var fileName string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return PostApiV1Datei400Response{}, nil
		}

		switch part.FormName() {
		case "name":
			buf := make([]byte, 256)
			n, _ := part.Read(buf)
			name = string(buf[:n])
		case "file":
			fileName = part.FileName()
			fileData = part
		}
	}

	if name == "" {
		return PostApiV1Datei400Response{}, nil
	}

	queries := db.New(s.db)

	// Create Datei record
	isDirectory := fileData == nil
	datei, err := queries.CreateDatei(ctx, isDirectory)
	if err != nil {
		return PostApiV1Datei400Response{}, nil
	}

	// Create DateiName
	nameRecord, err := queries.CreateDateiName(ctx, db.CreateDateiNameParams{
		DateiID: datei.ID,
		Name:    name,
	})
	if err != nil {
		return PostApiV1Datei400Response{}, nil
	}

	// Update Datei with latest name ID
	datei, err = queries.UpdateDateiLatestNameID(ctx, db.UpdateDateiLatestNameIDParams{
		ID:           datei.ID,
		LatestNameID: &nameRecord.ID,
	})
	if err != nil {
		return PostApiV1Datei400Response{}, nil
	}

	// Handle file upload if provided
	var latestVersion *db.DateiVersion
	if fileData != nil && fileName != "" {
		s3Key, fileSize, checksum, mimeType, err := mockS3UploadFromReader(fileName, fileData)
		if err != nil {
			return PostApiV1Datei400Response{}, nil
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    s3Key,
			FileSize: fileSize,
			Checksum: checksum,
			MimeType: mimeType,
		})
		if err != nil {
			return PostApiV1Datei400Response{}, nil
		}

		// Update Datei with latest version ID
		datei, err = queries.UpdateDateiLatestVersionID(ctx, db.UpdateDateiLatestVersionIDParams{
			ID:              datei.ID,
			LatestVersionID: &versionRecord.ID,
		})
		if err != nil {
			return PostApiV1Datei400Response{}, nil
		}

		latestVersion = &versionRecord
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&datei, latestVersion, &name)
	return PostApiV1Datei201JSONResponse(*response), nil
}

func NewServer(db *pgxpool.Pool) *server {
	return &server{db: db}
}

var _ StrictServerInterface = (*server)(nil)

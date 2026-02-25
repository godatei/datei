package server

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mapping"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/pkg/api"
	"github.com/jackc/pgx/v5/pgxpool"
)

const fileFormField = "file"

type server struct {
	db    *pgxpool.Pool
	store storage.Store
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

	// Get all Datei records with details in a single query
	allDateiWithDetails, err := queries.ListDateiWithDetails(ctx)
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

	total := len(allDateiWithDetails)

	// Apply pagination
	start := min(offset, len(allDateiWithDetails))
	end := min(offset+limit, len(allDateiWithDetails))

	paginatedDatei := allDateiWithDetails[start:end]

	// Map to API response
	items := mapping.MapDBDateiDetailsSliceToAPI(paginatedDatei)

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
	contentType := "application/octet-stream"

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
		case fileFormField:
			fileName = part.FileName()
			if fileDataBytes, err := io.ReadAll(part); err != nil {
				return PatchApiV1DateiId400Response{}, nil
			} else {
				fileData = bytes.NewReader(fileDataBytes)
			}
			contentType = part.Header.Get("Content-Type")
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

	if fileData != nil && fileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, fileData, contentType)
		if err != nil {
			return PatchApiV1DateiId400Response{}, nil
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    hash,
			FileSize: fileSize,
			Checksum: hash,
			MimeType: contentType,
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
	}

	details, err := queries.GetDateiByIDWithDetails(ctx, datei.ID)
	if err != nil {
		return PatchApiV1DateiId400Response{}, nil
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&details.Datei, &details.DateiVersion, &details.DateiName.Name)
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
	var contentType string

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
		case fileFormField:
			fileName = part.FileName()
			if fileDataBytes, err := io.ReadAll(part); err != nil {
				return PostApiV1Datei400Response{}, nil
			} else {
				fileData = bytes.NewReader(fileDataBytes)
			}
			contentType = part.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
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
		hash, fileSize, err := s.store.PutObject(ctx, fileData, contentType)
		if err != nil {
			return PostApiV1Datei400Response{}, nil
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    hash,
			FileSize: fileSize,
			Checksum: hash,
			MimeType: contentType,
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

// GetApiV1DateiIdDownload implements [StrictServerInterface].
func (s *server) GetApiV1DateiIdDownload(
	ctx context.Context,
	request GetApiV1DateiIdDownloadRequestObject,
) (GetApiV1DateiIdDownloadResponseObject, error) {
	queries := db.New(s.db)
	dateiID := request.Id

	// Get Datei with details to check if it exists and has a version
	details, err := queries.GetDateiByIDWithDetails(ctx, dateiID)
	if err != nil {
		return GetApiV1DateiIdDownload404Response{}, nil
	}

	// Check if it's a directory
	if details.Datei.IsDirectory {
		return GetApiV1DateiIdDownload409Response{}, nil
	}

	// Get the file from storage
	reader, err := s.store.GetObject(ctx, details.DateiVersion.S3Key)
	if err != nil {
		return GetApiV1DateiIdDownload404Response{}, nil
	}

	// Determine the filename
	filename := details.DateiName.Name

	// Return the file with appropriate headers
	return GetApiV1DateiIdDownload200ApplicationoctetStreamResponse{
		Body: reader,
		Headers: GetApiV1DateiIdDownload200ResponseHeaders{
			ContentDisposition: fmt.Sprintf(`attachment; filename="%v"`, filename),
			ContentType:        details.DateiVersion.MimeType,
		},
		ContentLength: details.DateiVersion.FileSize,
	}, nil
}

func NewServer(db *pgxpool.Pool, store storage.Store) *server {
	return &server{db: db, store: store}
}

var _ StrictServerInterface = (*server)(nil)

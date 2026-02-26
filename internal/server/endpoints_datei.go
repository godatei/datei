package server

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mapping"
	"github.com/godatei/datei/pkg/api"
)

// ListDatei implements [StrictServerInterface].
func (s *server) ListDatei(
	ctx context.Context,
	request ListDateiRequestObject,
) (ListDateiResponseObject, error) {
	queries := db.New(s.db)

	// Get all Datei records with details in a single query
	allDateiWithDetails, err := queries.ListDateiWithDetails(ctx)
	if err != nil {
		return ListDatei400Response{}, err
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

	return ListDatei200JSONResponse(response), nil
}

// CreateDatei implements [StrictServerInterface].
func (s *server) CreateDatei(
	ctx context.Context,
	request CreateDateiRequestObject,
) (CreateDateiResponseObject, error) {
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
			return CreateDatei400Response{}, nil
		}

		switch part.FormName() {
		case "name":
			buf := make([]byte, 256)
			n, _ := part.Read(buf)
			name = string(buf[:n])
		case fileFormField:
			fileName = part.FileName()
			if fileDataBytes, err := io.ReadAll(part); err != nil {
				return CreateDatei400Response{}, nil
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
		return CreateDatei400Response{}, nil
	}

	queries := db.New(s.db)

	// Create Datei record
	isDirectory := fileData == nil
	datei, err := queries.CreateDatei(ctx, isDirectory)
	if err != nil {
		return CreateDatei400Response{}, nil
	}

	// Create DateiName
	nameRecord, err := queries.CreateDateiName(ctx, db.CreateDateiNameParams{
		DateiID: datei.ID,
		Name:    name,
	})
	if err != nil {
		return CreateDatei400Response{}, nil
	}

	// Update Datei with latest name ID
	datei, err = queries.UpdateDateiLatestNameID(ctx, db.UpdateDateiLatestNameIDParams{
		ID:           datei.ID,
		LatestNameID: &nameRecord.ID,
	})
	if err != nil {
		return CreateDatei400Response{}, nil
	}

	// Handle file upload if provided
	var latestVersion *db.DateiVersion
	if fileData != nil && fileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, fileData, contentType)
		if err != nil {
			return CreateDatei400Response{}, nil
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    hash,
			FileSize: fileSize,
			Checksum: hash,
			MimeType: contentType,
		})
		if err != nil {
			return CreateDatei400Response{}, nil
		}

		// Update Datei with latest version ID
		datei, err = queries.UpdateDateiLatestVersionID(ctx, db.UpdateDateiLatestVersionIDParams{
			ID:              datei.ID,
			LatestVersionID: &versionRecord.ID,
		})
		if err != nil {
			return CreateDatei400Response{}, nil
		}

		latestVersion = &versionRecord
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&datei, latestVersion, &name)
	return CreateDatei201JSONResponse(*response), nil
}

// DownloadDatei implements [StrictServerInterface].
func (s *server) DownloadDatei(
	ctx context.Context,
	request DownloadDateiRequestObject,
) (DownloadDateiResponseObject, error) {
	queries := db.New(s.db)
	dateiID := request.Id

	// Get Datei with details to check if it exists and has a version
	details, err := queries.GetDateiByIDWithDetails(ctx, dateiID)
	if err != nil {
		return DownloadDatei404Response{}, nil
	}

	// Check if it's a directory
	if details.Datei.IsDirectory {
		return DownloadDatei409Response{}, nil
	}

	// Get the file from storage
	reader, err := s.store.GetObject(ctx, details.DateiVersion.S3Key)
	if err != nil {
		return DownloadDatei404Response{}, nil
	}

	// Determine the filename
	filename := details.DateiName.Name

	// Return the file with appropriate headers
	return DownloadDatei200ApplicationoctetStreamResponse{
		Body: reader,
		Headers: DownloadDatei200ResponseHeaders{
			ContentDisposition: fmt.Sprintf(`attachment; filename="%v"`, filename),
			ContentType:        details.DateiVersion.MimeType,
		},
		ContentLength: details.DateiVersion.FileSize,
	}, nil
}

// UpdateDatei implements [StrictServerInterface].
func (s *server) UpdateDatei(
	ctx context.Context,
	request UpdateDateiRequestObject,
) (UpdateDateiResponseObject, error) {
	queries := db.New(s.db)
	dateiID := request.Id

	// Get existing Datei
	datei, err := queries.GetDateiByID(ctx, dateiID)
	if err != nil {
		return UpdateDatei404Response{}, nil
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
			return UpdateDatei400Response{}, nil
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
				return UpdateDatei400Response{}, nil
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
			return UpdateDatei400Response{}, nil
		}

		datei, err = queries.UpdateDateiLatestNameID(ctx, db.UpdateDateiLatestNameIDParams{
			ID:           datei.ID,
			LatestNameID: &nameRecord.ID,
		})
		if err != nil {
			return UpdateDatei400Response{}, nil
		}
	}

	if fileData != nil && fileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, fileData, contentType)
		if err != nil {
			return UpdateDatei400Response{}, nil
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    hash,
			FileSize: fileSize,
			Checksum: hash,
			MimeType: contentType,
		})
		if err != nil {
			return UpdateDatei400Response{}, nil
		}

		datei, err = queries.UpdateDateiLatestVersionID(ctx, db.UpdateDateiLatestVersionIDParams{
			ID:              datei.ID,
			LatestVersionID: &versionRecord.ID,
		})
		if err != nil {
			return UpdateDatei400Response{}, nil
		}
	}

	details, err := queries.GetDateiByIDWithDetails(ctx, datei.ID)
	if err != nil {
		return UpdateDatei400Response{}, nil
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&details.Datei, &details.DateiVersion, &details.DateiName.Name)
	return UpdateDatei200JSONResponse(*response), nil
}

// DeleteDatei implements [StrictServerInterface].
func (s *server) DeleteDatei(
	ctx context.Context,
	request DeleteDateiRequestObject,
) (DeleteDateiResponseObject, error) {
	queries := db.New(s.db)
	dateiID := request.Id

	// Get Datei to verify it exists
	_, err := queries.GetDateiByID(ctx, dateiID)
	if err != nil {
		return DeleteDatei404Response{}, nil
	}

	// Soft delete by setting trashed_at
	_, err = queries.SetDateiTrashedAt(ctx, dateiID)
	if err != nil {
		return DeleteDatei409Response{}, nil
	}

	// Return 204 No Content
	return DeleteDatei204Response{}, nil
}

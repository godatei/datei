package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

// ListTrash implements [StrictServerInterface].
func (s *server) ListTrash(
	ctx context.Context,
	request ListTrashRequestObject,
) (ListTrashResponseObject, error) {
	limit := 0
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}

	result, err := s.dateiService.ListTrash(ctx, datei.ListTrashInput{
		ParentID: request.Params.ParentId,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	return ListTrash200JSONResponse(api.ListTrashResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// ListDatei implements [StrictServerInterface].
func (s *server) ListDatei(
	ctx context.Context,
	request ListDateiRequestObject,
) (ListDateiResponseObject, error) {
	limit := 0
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}

	result, err := s.dateiService.ListDatei(ctx, datei.ListDateiInput{
		ParentID: request.Params.ParentId,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return ListDatei400Response{}, err
	}

	response := api.ListDateiResponse{
		Items: result.Items,
		Total: result.Total,
	}

	return ListDatei200JSONResponse(response), nil
}

// CreateDatei implements [StrictServerInterface].
func (s *server) CreateDatei(
	ctx context.Context,
	request CreateDateiRequestObject,
) (CreateDateiResponseObject, error) {
	// Parse multipart request
	reader := request.Body
	var parentID *uuid.UUID
	var fileData io.Reader
	var fileName string
	var contentType string

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return CreateDatei400JSONResponse{Message: err.Error()}, nil
		}

		switch part.FormName() {
		case "parentId":
			raw, err := io.ReadAll(io.LimitReader(part, 65))
			if err != nil {
				return CreateDatei400JSONResponse{Message: err.Error()}, nil
			}
			if len(raw) > 64 {
				return CreateDatei400JSONResponse{Message: "invalid parentId"}, nil
			}
			if s := strings.TrimSpace(string(raw)); s != "" {
				parsed, err := uuid.Parse(s)
				if err != nil {
					return CreateDatei400JSONResponse{Message: "invalid parentId"}, nil
				}
				parentID = &parsed
			}
		case nameFormField:
			fileNameData, err := io.ReadAll(io.LimitReader(part, 1024))
			if err != nil {
				return CreateDatei400JSONResponse{Message: err.Error()}, nil
			}
			fileName = strings.TrimSpace(string(fileNameData))
		case fileFormField:
			if fileName == "" {
				fileName = part.FileName()
			}
			if fileDataBytes, err := io.ReadAll(part); err != nil {
				return CreateDatei400JSONResponse{Message: err.Error()}, nil
			} else {
				fileData = bytes.NewReader(fileDataBytes)
			}
			contentType = part.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		}
	}

	if fileName == "" {
		return CreateDatei400JSONResponse{Message: "filename is required"}, nil
	}

	result, err := s.dateiService.CreateDatei(ctx, datei.CreateDateiInput{
		ParentID:    parentID,
		Reader:      fileData,
		FileName:    fileName,
		ContentType: contentType,
	})
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrParentNotFound):
			return CreateDatei400JSONResponse{Message: "parent directory not found"}, nil
		case errors.Is(err, dateierrors.ErrParentNotDirectory):
			return CreateDatei400JSONResponse{Message: "parent is not a directory"}, nil
		case errors.Is(err, dateierrors.ErrParentTrashed):
			return CreateDatei400JSONResponse{Message: "parent directory is trashed"}, nil
		default:
			slog.Error("endpoint error", "error", err)
			return CreateDatei400JSONResponse{Message: err.Error()}, nil
		}
	}

	return CreateDatei201JSONResponse(*result), nil
}

// GetDateiPath implements [StrictServerInterface].
func (s *server) GetDateiPath(
	ctx context.Context,
	request GetDateiPathRequestObject,
) (GetDateiPathResponseObject, error) {
	path, err := s.dateiService.GetDateiPath(ctx, request.Id)
	if err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return GetDateiPath404Response{}, nil
		}
		return nil, err
	}
	return GetDateiPath200JSONResponse(path), nil
}

// DownloadDatei implements [StrictServerInterface].
func (s *server) DownloadDatei(
	ctx context.Context,
	request DownloadDateiRequestObject,
) (DownloadDateiResponseObject, error) {
	result, err := s.dateiService.DownloadDatei(ctx, request.Id)
	if err != nil {
		if err == dateierrors.ErrIsDirectory {
			return DownloadDatei409Response{}, nil
		}
		slog.Error("download error", "error", err)
		return DownloadDatei404Response{}, nil
	}

	return DownloadDatei200ApplicationoctetStreamResponse{
		Body: result.Reader,
		Headers: DownloadDatei200ResponseHeaders{
			ContentDisposition: fmt.Sprintf(`attachment; filename="%v"`, result.ContentFileName),
			ContentType:        result.ContentType,
		},
		ContentLength: result.ContentLength,
	}, nil
}

// UpdateDatei implements [StrictServerInterface].
func (s *server) UpdateDatei(
	ctx context.Context,
	request UpdateDateiRequestObject,
) (UpdateDateiResponseObject, error) {
	var name *string
	var fileData io.Reader
	var fileName string
	contentType := "application/octet-stream"

	if reader := request.MultipartBody; reader != nil {
		// Parse multipart request
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
					nameStr := strings.TrimSpace(string(buf[:n]))
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
	} else if reader := request.FormdataBody; reader != nil {
		name = reader.Name
	}

	result, err := s.dateiService.UpdateDatei(ctx, datei.UpdateDateiInput{
		ID:          request.Id,
		Name:        name,
		Reader:      fileData,
		FileName:    fileName,
		ContentType: contentType,
	})
	if err != nil {
		return UpdateDatei404Response{}, nil
	}

	return UpdateDatei200JSONResponse(*result), nil
}

// GetDateiThumbnail implements [StrictServerInterface].
func (s *server) GetDateiThumbnail(
	ctx context.Context,
	request GetDateiThumbnailRequestObject,
) (GetDateiThumbnailResponseObject, error) {
	ifNoneMatch := ""
	if request.Params.IfNoneMatch != nil {
		ifNoneMatch = *request.Params.IfNoneMatch
	}

	result, err := s.dateiService.GetThumbnail(ctx, request.Id, ifNoneMatch)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrNotModified):
			return GetDateiThumbnail304Response{}, nil
		case errors.Is(err, dateierrors.ErrIsDirectory):
			return GetDateiThumbnail409Response{}, nil
		case errors.Is(err, dateierrors.ErrUnsupportedMediaType):
			return GetDateiThumbnail415Response{}, nil
		default:
			return GetDateiThumbnail404Response{}, nil
		}
	}

	return GetDateiThumbnail200ImagejpegResponse{
		Body:          result.Body,
		ContentLength: result.ContentLength,
		Headers: GetDateiThumbnail200ResponseHeaders{
			CacheControl: "private, no-cache",
			ETag:         result.ETag,
		},
	}, nil
}

// DeleteDatei implements [StrictServerInterface].
func (s *server) DeleteDatei(
	ctx context.Context,
	request DeleteDateiRequestObject,
) (DeleteDateiResponseObject, error) {
	err := s.dateiService.DeleteDatei(ctx, request.Id)
	if err != nil {
		return DeleteDatei404Response{}, nil
	}

	// Return 204 No Content
	return DeleteDatei204Response{}, nil
}

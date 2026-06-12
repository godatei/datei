package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/file"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
)

type fileServer struct {
	svc *file.Service
}

// ListFiles implements [StrictServerInterface].
func (s *fileServer) ListFiles(
	ctx context.Context,
	request ListFilesRequestObject,
) (ListFilesResponseObject, error) {
	limit := 0
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}

	result, err := s.svc.ListFiles(ctx, file.ListFilesInput{
		ParentID: request.Params.ParentId,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, err
	}

	response := api.ListFilesResponse{
		Items: result.Items,
		Total: result.Total,
	}

	return ListFiles200JSONResponse(response), nil
}

// CreateFile implements [StrictServerInterface].
func (s *fileServer) CreateFile(
	ctx context.Context,
	request CreateFileRequestObject,
) (CreateFileResponseObject, error) {
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
			return CreateFile400JSONResponse{Message: err.Error()}, nil
		}

		switch part.FormName() {
		case parentIdFormField:
			raw, err := readLimited(part, 64)
			if err != nil {
				return CreateFile400JSONResponse{Message: "invalid parentId"}, nil
			}
			if s := strings.TrimSpace(string(raw)); s != "" {
				parsed, err := uuid.Parse(s)
				if err != nil {
					return CreateFile400JSONResponse{Message: "invalid parentId"}, nil
				}
				parentID = &parsed
			}
		case nameFormField:
			fileNameData, err := readLimited(part, 1024)
			if err != nil {
				return CreateFile400JSONResponse{Message: err.Error()}, nil
			}
			fileName = strings.TrimSpace(string(fileNameData))
		case fileFormField:
			if fileName == "" {
				fileName = part.FileName()
			}
			if fileDataBytes, err := io.ReadAll(part); err != nil {
				return CreateFile400JSONResponse{Message: err.Error()}, nil
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
		return CreateFile400JSONResponse{Message: "filename is required"}, nil
	}

	result, err := s.svc.CreateFile(ctx, file.CreateFileInput{
		ParentID:    parentID,
		Reader:      fileData,
		FileName:    fileName,
		ContentType: contentType,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrParentNotFound):
			return CreateFile400JSONResponse{Message: "parent directory not found"}, nil
		case errors.Is(err, apperrors.ErrParentNotDirectory):
			return CreateFile400JSONResponse{Message: "parent is not a directory"}, nil
		case errors.Is(err, apperrors.ErrParentTrashed):
			return CreateFile400JSONResponse{Message: "parent directory is trashed"}, nil
		default:
			return nil, err
		}
	}

	return CreateFile201JSONResponse(*result), nil
}

// GetFilePath implements [StrictServerInterface].
func (s *fileServer) GetFilePath(
	ctx context.Context,
	request GetFilePathRequestObject,
) (GetFilePathResponseObject, error) {
	path, err := s.svc.GetFilePath(ctx, request.Id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return GetFilePath404Response{}, nil
		}
		return nil, err
	}
	return GetFilePath200JSONResponse(path), nil
}

// DownloadFile implements [StrictServerInterface].
func (s *fileServer) DownloadFile(
	ctx context.Context,
	request DownloadFileRequestObject,
) (DownloadFileResponseObject, error) {
	result, err := s.svc.DownloadFile(ctx, request.Id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrIsDirectory):
			return DownloadFile409Response{}, nil
		case errors.Is(err, apperrors.ErrNotFound), errors.Is(err, apperrors.ErrNoContent):
			return DownloadFile404Response{}, nil
		default:
			return nil, err
		}
	}

	return DownloadFile200ApplicationoctetStreamResponse{
		Body: result.Reader,
		Headers: DownloadFile200ResponseHeaders{
			ContentDisposition: attachmentDisposition(result.ContentFileName),
			ContentType:        result.ContentType,
		},
		ContentLength: result.ContentLength,
	}, nil
}

// UpdateFile implements [StrictServerInterface].
func (s *fileServer) UpdateFile(
	ctx context.Context,
	request UpdateFileRequestObject,
) (UpdateFileResponseObject, error) {
	var name *string
	var moveRequested bool
	var newParentID *uuid.UUID
	var fileData io.Reader
	var fileName string
	contentType := "application/octet-stream"

	if reader := request.MultipartBody; reader != nil {
		var rawName, rawUpdateParentId, rawParentId *string
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return UpdateFile400Response{}, nil
			}

			switch part.FormName() {
			case nameFormField:
				buf, err := readLimited(part, 1024)
				if err != nil {
					return UpdateFile400Response{}, nil
				}
				s := strings.TrimSpace(string(buf))
				rawName = &s
			case updateParentIdFormField:
				buf, err := readLimited(part, 8)
				if err != nil {
					return UpdateFile400Response{}, nil
				}
				s := strings.TrimSpace(string(buf))
				rawUpdateParentId = &s
			case parentIdFormField:
				raw, err := readLimited(part, 64)
				if err != nil {
					return UpdateFile400Response{}, nil
				}
				s := strings.TrimSpace(string(raw))
				rawParentId = &s
			case fileFormField:
				fileName = part.FileName()
				if fileDataBytes, err := io.ReadAll(part); err != nil {
					return UpdateFile400Response{}, nil
				} else {
					fileData = bytes.NewReader(fileDataBytes)
				}
				if partContentType := strings.TrimSpace(part.Header.Get("Content-Type")); partContentType != "" {
					contentType = partContentType
				}
			}
		}
		if rawName != nil && *rawName != "" {
			name = rawName
		}
		if rawUpdateParentId != nil && *rawUpdateParentId == "true" {
			moveRequested = true
			if rawParentId != nil && *rawParentId != "" {
				parsed, err := uuid.Parse(*rawParentId)
				if err != nil {
					return UpdateFile400Response{}, nil
				}
				newParentID = &parsed
			}
		}
	} else if body := request.FormdataBody; body != nil {
		name = body.Name
		if body.UpdateParentId != nil && *body.UpdateParentId {
			moveRequested = true
			newParentID = body.ParentId
		}
	}

	result, err := s.svc.UpdateFile(ctx, file.UpdateFileInput{
		ID:            request.Id,
		Name:          name,
		MoveRequested: moveRequested,
		NewParentID:   newParentID,
		Reader:        fileData,
		FileName:      fileName,
		ContentType:   contentType,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return UpdateFile404Response{}, nil
		case errors.Is(err, apperrors.ErrParentNotFound),
			errors.Is(err, apperrors.ErrParentNotDirectory),
			errors.Is(err, apperrors.ErrParentTrashed),
			errors.Is(err, apperrors.ErrCycleDetected):
			return UpdateFile400Response{}, nil
		default:
			return nil, err
		}
	}

	return UpdateFile200JSONResponse(*result), nil
}

// GetFileThumbnail implements [StrictServerInterface].
func (s *fileServer) GetFileThumbnail(
	ctx context.Context,
	request GetFileThumbnailRequestObject,
) (GetFileThumbnailResponseObject, error) {
	ifNoneMatch := ""
	if request.Params.IfNoneMatch != nil {
		ifNoneMatch = *request.Params.IfNoneMatch
	}

	result, err := s.svc.GetThumbnail(ctx, request.Id, ifNoneMatch)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotModified):
			return GetFileThumbnail304Response{}, nil
		case errors.Is(err, apperrors.ErrIsDirectory):
			return GetFileThumbnail409Response{}, nil
		case errors.Is(err, apperrors.ErrUnsupportedMediaType):
			return GetFileThumbnail415Response{}, nil
		default:
			return GetFileThumbnail404Response{}, nil
		}
	}

	return GetFileThumbnail200ImagejpegResponse{
		Body:          result.Body,
		ContentLength: result.ContentLength,
		Headers: GetFileThumbnail200ResponseHeaders{
			CacheControl: "private, no-cache",
			ETag:         result.ETag,
		},
	}, nil
}

// DeleteFile implements [StrictServerInterface].
func (s *fileServer) DeleteFile(
	ctx context.Context,
	request DeleteFileRequestObject,
) (DeleteFileResponseObject, error) {
	err := s.svc.DeleteFile(ctx, request.Id)
	if err != nil {
		return DeleteFile404Response{}, nil
	}

	// Return 204 No Content
	return DeleteFile204Response{}, nil
}

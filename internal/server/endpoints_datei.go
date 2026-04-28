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
)

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
		Limit:  limit,
		Offset: offset,
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

	if fileData == nil {
		return CreateDatei400JSONResponse{Message: "file is required (directory creation is not implemented)"}, nil
	}

	result, err := s.dateiService.CreateDatei(ctx, datei.CreateDateiInput{
		Reader:      fileData,
		FileName:    fileName,
		ContentType: contentType,
	})
	if err != nil {
		slog.Error("endpoint error", "error", err)
		return CreateDatei400JSONResponse{Message: err.Error()}, nil
	}

	return CreateDatei201JSONResponse(*result), nil
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
	result, err := s.dateiService.GetThumbnail(ctx, request.Id)
	if err != nil {
		switch {
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
			CacheControl: "private, max-age=31536000, immutable",
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

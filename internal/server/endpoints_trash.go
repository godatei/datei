package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/file"
	"github.com/godatei/datei/pkg/api"
)

type trashServer struct {
	svc *file.Service
}

// ListTrash implements [StrictServerInterface].
func (s *trashServer) ListTrash(
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

	result, err := s.svc.ListTrash(ctx, file.ListTrashInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	return ListTrash200JSONResponse(api.ListTrashResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// ListTrashChildren implements [StrictServerInterface].
func (s *trashServer) ListTrashChildren(
	ctx context.Context,
	request ListTrashChildrenRequestObject,
) (ListTrashChildrenResponseObject, error) {
	limit := 0
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}

	result, err := s.svc.ListTrashChildren(ctx, file.ListTrashChildrenInput{
		ParentID: request.FileId,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrParentNotFound),
			errors.Is(err, apperrors.ErrParentNotTrashed),
			errors.Is(err, apperrors.ErrParentNotDirectory):
			return ListTrashChildren404Response{}, nil
		default:
			return nil, err
		}
	}

	return ListTrashChildren200JSONResponse(api.ListFilesResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// RestoreTrash implements [StrictServerInterface].
func (s *trashServer) RestoreTrash(
	ctx context.Context,
	request RestoreTrashRequestObject,
) (RestoreTrashResponseObject, error) {
	err := s.svc.RestoreFile(ctx, file.RestoreFileInput{
		ID:       request.FileId,
		ParentID: request.Body.ParentId,
	})
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound), errors.Is(err, apperrors.ErrNotInTrash),
			errors.Is(err, apperrors.ErrParentNotFound):
			return RestoreTrash404Response{}, nil
		case errors.Is(err, apperrors.ErrInvalidInput), errors.Is(err, apperrors.ErrParentNotDirectory),
			errors.Is(err, apperrors.ErrParentTrashed), errors.Is(err, apperrors.ErrCycleDetected):
			return RestoreTrash400Response{}, nil
		default:
			return nil, err
		}
	}

	return RestoreTrash204Response{}, nil
}

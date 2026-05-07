package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/pkg/api"
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
func (s *server) ListTrashChildren(
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

	result, err := s.dateiService.ListTrashChildren(ctx, datei.ListTrashChildrenInput{
		ParentID: request.DateiId,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrParentNotFound),
			errors.Is(err, dateierrors.ErrParentNotTrashed),
			errors.Is(err, dateierrors.ErrParentNotDirectory):
			return ListTrashChildren404Response{}, nil
		default:
			return nil, err
		}
	}

	return ListTrashChildren200JSONResponse(api.ListDateiResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// RestoreTrash implements [StrictServerInterface].
func (s *server) RestoreTrash(
	ctx context.Context,
	request RestoreTrashRequestObject,
) (RestoreTrashResponseObject, error) {
	err := s.dateiService.RestoreDatei(ctx, datei.RestoreDateiInput{
		ID:       request.DateiId,
		ParentID: request.Body.ParentId,
	})
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrNotFound), errors.Is(err, dateierrors.ErrNotInTrash),
			errors.Is(err, dateierrors.ErrParentNotFound):
			return RestoreTrash404Response{}, nil
		case errors.Is(err, dateierrors.ErrInvalidInput), errors.Is(err, dateierrors.ErrParentNotDirectory),
			errors.Is(err, dateierrors.ErrParentTrashed), errors.Is(err, dateierrors.ErrCycleDetected):
			return RestoreTrash400Response{}, nil
		default:
			slog.Error("endpoint error", "error", err)
			return nil, err
		}
	}

	return RestoreTrash204Response{}, nil
}

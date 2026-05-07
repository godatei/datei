package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/link"
	"github.com/godatei/datei/pkg/api"
)

// ListLinks implements [StrictServerInterface].
func (s *server) ListLinks(
	ctx context.Context,
	_ ListLinksRequestObject,
) (ListLinksResponseObject, error) {
	result, err := s.linkService.ListLinks(ctx)
	if err != nil {
		return nil, err
	}
	return ListLinks200JSONResponse(api.ListLinksResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// CreateLink implements [StrictServerInterface].
func (s *server) CreateLink(
	ctx context.Context,
	request CreateLinkRequestObject,
) (CreateLinkResponseObject, error) {
	if request.Body == nil {
		return CreateLink400JSONResponse{Message: "missing request body"}, nil
	}

	result, err := s.linkService.CreateLink(ctx, link.CreateLinkInput{
		Name:      request.Body.Name,
		ExpiresAt: request.Body.ExpiresAt,
		Code:      request.Body.Code,
		DateiIDs:  request.Body.DateiIds,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return CreateLink400JSONResponse{Message: "invalid input"}, nil
		}
		slog.Error("create link error", "error", err)
		return nil, err
	}
	return CreateLink201JSONResponse(*result), nil
}

// UpdateLink implements [StrictServerInterface].
func (s *server) UpdateLink(
	ctx context.Context,
	request UpdateLinkRequestObject,
) (UpdateLinkResponseObject, error) {
	if request.Body == nil {
		return UpdateLink400Response{}, nil
	}

	input := link.UpdateLinkInput{
		ID:        request.Id,
		Name:      request.Body.Name,
		ExpiresAt: request.Body.ExpiresAt,
		Code:      request.Body.Code,
	}
	if request.Body.ClearCode != nil {
		input.ClearCode = *request.Body.ClearCode
	}
	if request.Body.ClearExpiration != nil {
		input.ClearExpiration = *request.Body.ClearExpiration
	}

	result, err := s.linkService.UpdateLink(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkNotFound):
			return UpdateLink404Response{}, nil
		case errors.Is(err, dateierrors.ErrInvalidInput):
			return UpdateLink400Response{}, nil
		default:
			slog.Error("update link error", "error", err)
			return nil, err
		}
	}
	return UpdateLink200JSONResponse(*result), nil
}

// RevokeLink implements [StrictServerInterface].
func (s *server) RevokeLink(
	ctx context.Context,
	request RevokeLinkRequestObject,
) (RevokeLinkResponseObject, error) {
	err := s.linkService.RevokeLink(ctx, request.Id)
	if err != nil {
		if errors.Is(err, dateierrors.ErrLinkNotFound) {
			return RevokeLink404Response{}, nil
		}
		return nil, err
	}
	return RevokeLink204Response{}, nil
}

// RotateLinkAccessToken implements [StrictServerInterface].
func (s *server) RotateLinkAccessToken(
	ctx context.Context,
	request RotateLinkAccessTokenRequestObject,
) (RotateLinkAccessTokenResponseObject, error) {
	result, err := s.linkService.RotateAccessToken(ctx, request.Id)
	if err != nil {
		if errors.Is(err, dateierrors.ErrLinkNotFound) {
			return RotateLinkAccessToken404Response{}, nil
		}
		return nil, err
	}
	return RotateLinkAccessToken200JSONResponse(*result), nil
}

// AddDateiToLink implements [StrictServerInterface].
func (s *server) AddDateiToLink(
	ctx context.Context,
	request AddDateiToLinkRequestObject,
) (AddDateiToLinkResponseObject, error) {
	if request.Body == nil {
		return AddDateiToLink400Response{}, nil
	}

	result, err := s.linkService.AddDateiToLink(ctx, request.Id, request.Body.DateiId)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkNotFound):
			return AddDateiToLink404Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkDateiAlreadyAdded):
			return AddDateiToLink409Response{}, nil
		case errors.Is(err, dateierrors.ErrInvalidInput):
			return AddDateiToLink400Response{}, nil
		default:
			slog.Error("add datei to link error", "error", err)
			return nil, err
		}
	}
	return AddDateiToLink200JSONResponse(*result), nil
}

// RemoveDateiFromLink implements [StrictServerInterface].
func (s *server) RemoveDateiFromLink(
	ctx context.Context,
	request RemoveDateiFromLinkRequestObject,
) (RemoveDateiFromLinkResponseObject, error) {
	err := s.linkService.RemoveDateiFromLink(ctx, request.Id, request.DateiId)
	if err != nil {
		switch {
		case errors.Is(err, dateierrors.ErrLinkNotFound):
			return RemoveDateiFromLink404Response{}, nil
		case errors.Is(err, dateierrors.ErrLinkDateiNotShared):
			return RemoveDateiFromLink400Response{}, nil
		default:
			slog.Error("remove datei from link error", "error", err)
			return nil, err
		}
	}
	return RemoveDateiFromLink204Response{}, nil
}

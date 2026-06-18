package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/link"
	"github.com/godatei/datei/pkg/api"
)

type linkServer struct {
	svc *link.Service
}

// ListLinks implements [StrictServerInterface].
func (s *linkServer) ListLinks(
	ctx context.Context,
	request ListLinksRequestObject,
) (ListLinksResponseObject, error) {
	limit := 0
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}
	status := ""
	if request.Params.Status != nil {
		status = string(*request.Params.Status)
	}

	result, err := s.svc.ListLinks(ctx, link.ListLinksInput{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return ListLinks200JSONResponse(api.ListLinksResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// CreateLink implements [StrictServerInterface].
func (s *linkServer) CreateLink(
	ctx context.Context,
	request CreateLinkRequestObject,
) (CreateLinkResponseObject, error) {
	if request.Body == nil {
		return CreateLink400Response{}, nil
	}

	result, err := s.svc.CreateLink(ctx, link.CreateLinkInput{
		Name:      request.Body.Name,
		ExpiresAt: request.Body.ExpiresAt,
		Code:      request.Body.Code,
		FileIDs:   request.Body.FileIds,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return CreateLink400Response{}, nil
		}
		return nil, err
	}
	return CreateLink201JSONResponse(*result), nil
}

// GetLink implements [StrictServerInterface].
func (s *linkServer) GetLink(
	ctx context.Context,
	request GetLinkRequestObject,
) (GetLinkResponseObject, error) {
	result, err := s.svc.GetLink(ctx, request.Id)
	if err != nil {
		if errors.Is(err, apperrors.ErrLinkNotFound) {
			return GetLink404Response{}, nil
		}
		return nil, err
	}
	return GetLink200JSONResponse(*result), nil
}

// UpdateLink implements [StrictServerInterface].
func (s *linkServer) UpdateLink(
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

	result, err := s.svc.UpdateLink(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkNotFound):
			return UpdateLink404Response{}, nil
		case errors.Is(err, apperrors.ErrLinkRevoked):
			return UpdateLink403Response{}, nil
		case errors.Is(err, apperrors.ErrInvalidInput):
			return UpdateLink400Response{}, nil
		default:
			return nil, err
		}
	}
	return UpdateLink200JSONResponse(*result), nil
}

// RevokeLink implements [StrictServerInterface].
func (s *linkServer) RevokeLink(
	ctx context.Context,
	request RevokeLinkRequestObject,
) (RevokeLinkResponseObject, error) {
	err := s.svc.RevokeLink(ctx, request.Id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkNotFound):
			return RevokeLink404Response{}, nil
		case errors.Is(err, apperrors.ErrLinkRevoked):
			return RevokeLink403Response{}, nil
		default:
			return nil, err
		}
	}
	return RevokeLink204Response{}, nil
}

// RotateLinkKey implements [StrictServerInterface].
func (s *linkServer) RotateLinkKey(
	ctx context.Context,
	request RotateLinkKeyRequestObject,
) (RotateLinkKeyResponseObject, error) {
	result, err := s.svc.RotateKey(ctx, request.Id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkNotFound):
			return RotateLinkKey404Response{}, nil
		case errors.Is(err, apperrors.ErrLinkRevoked):
			return RotateLinkKey403Response{}, nil
		default:
			return nil, err
		}
	}
	return RotateLinkKey200JSONResponse(*result), nil
}

// AddFileToLink implements [StrictServerInterface].
func (s *linkServer) AddFileToLink(
	ctx context.Context,
	request AddFileToLinkRequestObject,
) (AddFileToLinkResponseObject, error) {
	if request.Body == nil {
		return AddFileToLink400Response{}, nil
	}

	result, err := s.svc.AddFileToLink(ctx, request.Id, request.Body.FileId)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkNotFound):
			return AddFileToLink404Response{}, nil
		case errors.Is(err, apperrors.ErrLinkRevoked):
			return AddFileToLink403Response{}, nil
		case errors.Is(err, apperrors.ErrLinkFileAlreadyAdded):
			return AddFileToLink409Response{}, nil
		case errors.Is(err, apperrors.ErrInvalidInput):
			return AddFileToLink400Response{}, nil
		default:
			return nil, err
		}
	}
	return AddFileToLink200JSONResponse(*result), nil
}

// RemoveFileFromLink implements [StrictServerInterface].
func (s *linkServer) RemoveFileFromLink(
	ctx context.Context,
	request RemoveFileFromLinkRequestObject,
) (RemoveFileFromLinkResponseObject, error) {
	err := s.svc.RemoveFileFromLink(ctx, request.Id, request.FileId)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrLinkNotFound):
			return RemoveFileFromLink404Response{}, nil
		case errors.Is(err, apperrors.ErrLinkRevoked):
			return RemoveFileFromLink403Response{}, nil
		case errors.Is(err, apperrors.ErrLinkFileNotShared):
			return RemoveFileFromLink400Response{}, nil
		default:
			return nil, err
		}
	}
	return RemoveFileFromLink204Response{}, nil
}

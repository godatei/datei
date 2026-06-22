package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

type tokensServer struct {
	svc *users.UserService
}

// ListPersonalAccessTokens implements [StrictServerInterface].
func (s *tokensServer) ListPersonalAccessTokens(
	ctx context.Context, _ ListPersonalAccessTokensRequestObject,
) (ListPersonalAccessTokensResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	tokens, err := s.svc.ListAccessTokens(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return ListPersonalAccessTokens200JSONResponse(api.ListPersonalAccessTokensResponse{Tokens: tokens}), nil
}

// CreatePersonalAccessToken implements [StrictServerInterface].
func (s *tokensServer) CreatePersonalAccessToken(
	ctx context.Context, request CreatePersonalAccessTokenRequestObject,
) (CreatePersonalAccessTokenResponseObject, error) {
	if request.Body == nil {
		return CreatePersonalAccessToken400Response{}, nil
	}

	user := authn.RequireCurrentUser(ctx)
	label := ""
	if request.Body.Label != nil {
		label = *request.Body.Label
	}

	result, err := s.svc.CreateAccessToken(ctx, users.CreateAccessTokenInput{
		UserID:    user.ID,
		Label:     label,
		ExpiresAt: request.Body.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return CreatePersonalAccessToken400Response{}, nil
		}
		return nil, err
	}

	return CreatePersonalAccessToken201JSONResponse(*result), nil
}

// RevokePersonalAccessToken implements [StrictServerInterface].
func (s *tokensServer) RevokePersonalAccessToken(
	ctx context.Context, request RevokePersonalAccessTokenRequestObject,
) (RevokePersonalAccessTokenResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.RevokeAccessToken(ctx, user.ID, request.Id)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			return RevokePersonalAccessToken404Response{}, nil
		case errors.Is(err, apperrors.ErrConcurrentUpdate):
			return RevokePersonalAccessToken409Response{}, nil
		}
		return nil, err
	}

	return RevokePersonalAccessToken204Response{}, nil
}

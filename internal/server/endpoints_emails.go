package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

type emailsServer struct {
	svc *users.UserService
}

// ListEmails implements [StrictServerInterface].
func (s *emailsServer) ListEmails(
	ctx context.Context, _ ListEmailsRequestObject,
) (ListEmailsResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	emails, err := s.svc.ListEmails(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return ListEmails200JSONResponse(api.ListEmailsResponse{Emails: emails}), nil
}

// AddEmail implements [StrictServerInterface].
func (s *emailsServer) AddEmail(
	ctx context.Context, request AddEmailRequestObject,
) (AddEmailResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.AddEmail(ctx, users.AddEmailInput{
		UserID: user.ID,
		Email:  string(request.Body.Email),
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrEmailAlreadyInUse) || errors.Is(err, apperrors.ErrInvalidInput) {
			return AddEmail400Response{}, nil
		}
		return nil, err
	}

	return AddEmail204Response{}, nil
}

// RemoveEmail implements [StrictServerInterface].
func (s *emailsServer) RemoveEmail(
	ctx context.Context, request RemoveEmailRequestObject,
) (RemoveEmailResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.RemoveEmail(ctx, user.ID, request.EmailId)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return RemoveEmail404Response{}, nil
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return RemoveEmail400Response{}, nil
		}
		return nil, err
	}

	return RemoveEmail204Response{}, nil
}

// SetPrimaryEmail implements [StrictServerInterface].
func (s *emailsServer) SetPrimaryEmail(
	ctx context.Context, request SetPrimaryEmailRequestObject,
) (SetPrimaryEmailResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.SetPrimaryEmail(ctx, user.ID, request.EmailId)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return SetPrimaryEmail404Response{}, nil
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return SetPrimaryEmail400Response{}, nil
		}
		return nil, err
	}

	return SetPrimaryEmail204Response{}, nil
}

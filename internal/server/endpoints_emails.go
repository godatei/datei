package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

// ListEmails implements [StrictServerInterface].
func (s *server) ListEmails(
	ctx context.Context, _ ListEmailsRequestObject,
) (ListEmailsResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	emails, err := s.userService.ListEmails(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("list emails error", "error", err)
		return nil, err
	}

	return ListEmails200JSONResponse(api.ListEmailsResponse{Emails: emails}), nil
}

// AddEmail implements [StrictServerInterface].
func (s *server) AddEmail(
	ctx context.Context, request AddEmailRequestObject,
) (AddEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil || request.Body.Email == "" {
		return AddEmail400Response{}, nil
	}

	err := s.userService.AddEmail(ctx, users.AddEmailInput{
		UserID: authInfo.UserID,
		Email:  string(request.Body.Email),
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrEmailAlreadyInUse) || errors.Is(err, dateierrors.ErrInvalidInput) {
			return AddEmail400Response{}, nil
		}
		slog.Error("add email error", "error", err)
		return nil, err
	}

	return AddEmail204Response{}, nil
}

// RemoveEmail implements [StrictServerInterface].
func (s *server) RemoveEmail(
	ctx context.Context, request RemoveEmailRequestObject,
) (RemoveEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.RemoveEmail(ctx, authInfo.UserID, request.EmailId)
	if err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return RemoveEmail404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return RemoveEmail400Response{}, nil
		}
		slog.Error("remove email error", "error", err)
		return nil, err
	}

	return RemoveEmail204Response{}, nil
}

// SetPrimaryEmail implements [StrictServerInterface].
func (s *server) SetPrimaryEmail(
	ctx context.Context, request SetPrimaryEmailRequestObject,
) (SetPrimaryEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.SetPrimaryEmail(ctx, authInfo.UserID, request.EmailId)
	if err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return SetPrimaryEmail404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return SetPrimaryEmail400Response{}, nil
		}
		slog.Error("set primary email error", "error", err)
		return nil, err
	}

	return SetPrimaryEmail204Response{}, nil
}

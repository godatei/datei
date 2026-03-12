package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// ListEmails implements [StrictServerInterface].
func (s *server) ListEmails(
	ctx context.Context, _ ListEmailsRequestObject,
) (ListEmailsResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	q := db.New(s.pool)
	rows, err := q.GetEmailsForUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to list emails", "error", err)
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	emails := make([]api.UserEmail, len(rows))
	for i, row := range rows {
		emails[i] = api.UserEmail{
			Id:        row.ID,
			Email:     openapi_types.Email(row.Email),
			IsPrimary: row.IsPrimary,
			Verified:  row.VerifiedAt != nil,
			CreatedAt: row.CreatedAt,
		}
	}

	return ListEmails200JSONResponse(api.ListEmailsResponse{
		Emails: emails,
	}), nil
}

// AddEmail implements [StrictServerInterface].
func (s *server) AddEmail(
	ctx context.Context, request AddEmailRequestObject,
) (AddEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil || request.Body.Email == "" {
		return AddEmail400Response{}, nil
	}

	// Check if email is already in use
	q := db.New(s.pool)
	_, err := q.GetUserAccountByEmail(ctx, string(request.Body.Email))
	if err == nil {
		return AddEmail400Response{}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("failed to check existing email", "error", err)
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}

	emailID := uuid.New()
	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.AddEmail(emailID, string(request.Body.Email), time.Now()); err != nil {
		return AddEmail400Response{}, nil
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		_, token, err := authjwt.GenerateVerificationToken(
			authInfo.UserID, string(request.Body.Email),
		)
		if err != nil {
			slog.Error("failed to generate verification token", "error", err)
		} else {
			verifyURL := fmt.Sprintf(
				"%s/verify?jwt=%s", config.ServerHost(), token,
			)
			if err := s.mailer.Send(
				ctx,
				mailer.EmailVerificationEmail(string(request.Body.Email), verifyURL),
			); err != nil {
				slog.Warn("failed to send verification email", "error", err)
			}
		}
	}

	return AddEmail204Response{}, nil
}

// RemoveEmail implements [StrictServerInterface].
func (s *server) RemoveEmail(
	ctx context.Context, request RemoveEmailRequestObject,
) (RemoveEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	q := db.New(s.pool)
	email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{
		ID:            request.EmailId,
		UserAccountID: authInfo.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RemoveEmail404Response{}, nil
		}
		slog.Error("failed to get email", "error", err)
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	if email.IsPrimary {
		return RemoveEmail400Response{}, nil
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.RemoveEmail(request.EmailId, time.Now()); err != nil {
		return RemoveEmail400Response{}, nil
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return RemoveEmail204Response{}, nil
}

// SetPrimaryEmail implements [StrictServerInterface].
func (s *server) SetPrimaryEmail(
	ctx context.Context, request SetPrimaryEmailRequestObject,
) (SetPrimaryEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	q := db.New(s.pool)
	email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{
		ID:            request.EmailId,
		UserAccountID: authInfo.UserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SetPrimaryEmail404Response{}, nil
		}
		slog.Error("failed to get email", "error", err)
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	if email.VerifiedAt == nil {
		return SetPrimaryEmail400Response{}, nil
	}

	if email.IsPrimary {
		return SetPrimaryEmail204Response{}, nil
	}

	primary, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get primary email", "error", err)
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.SetPrimaryEmail(primary.ID, request.EmailId, time.Now()); err != nil {
		return SetPrimaryEmail400Response{}, nil
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return SetPrimaryEmail204Response{}, nil
}

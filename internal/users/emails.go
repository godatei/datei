package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mapping"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *UserService) ListEmails(ctx context.Context, userID uuid.UUID) ([]api.UserEmail, error) {
	q := s.queries()
	rows, err := q.GetEmailsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	return mapping.MapEmailProjectionSliceToAPI(rows), nil
}

func (s *UserService) RemoveEmail(ctx context.Context, userID uuid.UUID, emailID uuid.UUID) error {
	q := s.queries()
	email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{
		ID:            emailID,
		UserAccountID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dateierrors.ErrNotFound
		}
		return fmt.Errorf("failed to get email: %w", err)
	}

	if email.IsPrimary {
		return dateierrors.ErrInvalidInput
	}

	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.RemoveEmail(emailID, time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

func (s *UserService) SetPrimaryEmail(ctx context.Context, userID uuid.UUID, emailID uuid.UUID) error {
	q := s.queries()
	email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{
		ID:            emailID,
		UserAccountID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dateierrors.ErrNotFound
		}
		return fmt.Errorf("failed to get email: %w", err)
	}

	if email.VerifiedAt == nil {
		return dateierrors.ErrInvalidInput
	}

	if email.IsPrimary {
		return nil // Already primary, no-op
	}

	primary, err := q.GetPrimaryEmailForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get primary email: %w", err)
	}

	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.SetPrimaryEmail(primary.ID, emailID, time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UpdateUserInput struct {
	UserID          uuid.UUID
	Name            *string
	Password        *string
	CurrentPassword *string
	IsPasswordReset bool
}

func (s *UserService) UpdateUser(ctx context.Context, input UpdateUserInput) error {
	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	now := time.Now()

	if input.Name != nil {
		if input.IsPasswordReset {
			return dateierrors.ErrPasswordResetOnly
		}
		if err := agg.ChangeName(*input.Name, now); err != nil {
			return dateierrors.ErrInvalidInput
		}
	}

	if input.Password != nil {
		if len(*input.Password) < 8 {
			return dateierrors.ErrInvalidInput
		}

		if !input.IsPasswordReset {
			if input.CurrentPassword == nil || *input.CurrentPassword == "" {
				return dateierrors.ErrCurrentPasswordRequired
			}
			q := s.queries()
			user, err := q.GetUserAccountByID(ctx, input.UserID)
			if err != nil {
				return fmt.Errorf("failed to get user: %w", err)
			}
			if err := security.VerifyPassword(*input.CurrentPassword, user.PasswordHash, user.PasswordSalt); err != nil {
				return dateierrors.ErrInvalidCredentials
			}
		}

		hash, salt, err := security.HashPassword(*input.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		if err := agg.ChangePassword(hash, salt, now); err != nil {
			return dateierrors.ErrInvalidInput
		}
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

type UpdateUserEmailInput struct {
	UserID   uuid.UUID
	NewEmail string
}

func (s *UserService) UpdateUserEmail(ctx context.Context, input UpdateUserEmailInput) error {
	q := s.queries()

	_, err := q.GetUserAccountByEmail(ctx, input.NewEmail)
	if err == nil {
		return dateierrors.ErrEmailAlreadyInUse
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing email: %w", err)
	}

	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to get primary email: %w", err)
	}

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.ChangeEmail(primaryEmail.Email, input.NewEmail, time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		s.sendVerificationEmail(ctx, input.UserID, input.NewEmail)
	}

	return nil
}

func (s *UserService) RequestEmailVerification(ctx context.Context, userID uuid.UUID) error {
	q := s.queries()
	email, err := q.GetPrimaryEmailForUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	s.sendVerificationEmail(ctx, userID, email.Email)
	return nil
}

type ConfirmEmailVerificationInput struct {
	UserID        uuid.UUID
	EmailVerified bool
	PasswordReset bool
	TokenEmail    string
}

func (s *UserService) ConfirmEmailVerification(ctx context.Context, input ConfirmEmailVerificationInput) error {
	if !input.EmailVerified || input.PasswordReset {
		return dateierrors.ErrInvalidToken
	}

	q := s.queries()
	email, err := q.GetPrimaryEmailForUser(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to get primary email: %w", err)
	}

	if input.TokenEmail != email.Email {
		return dateierrors.ErrEmailMismatch
	}

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.VerifyEmail(time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

type AddEmailInput struct {
	UserID uuid.UUID
	Email  string
}

func (s *UserService) AddEmail(ctx context.Context, input AddEmailInput) error {
	q := s.queries()
	_, err := q.GetUserAccountByEmail(ctx, input.Email)
	if err == nil {
		return dateierrors.ErrEmailAlreadyInUse
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing email: %w", err)
	}

	emailID := uuid.New()
	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.AddEmail(emailID, input.Email, time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		s.sendVerificationEmail(ctx, input.UserID, input.Email)
	}

	return nil
}

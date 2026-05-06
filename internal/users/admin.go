package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *UserService) ListUsers(ctx context.Context) ([]db.ListUserAccountProjectionsRow, error) {
	q := s.queries()
	rows, err := q.ListUserAccountProjections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return rows, nil
}

func (s *UserService) GetUserForAdmin(
	ctx context.Context, userID uuid.UUID,
) (db.ListUserAccountProjectionsRow, error) {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.ListUserAccountProjectionsRow{}, dateierrors.ErrNotFound
		}
		return db.ListUserAccountProjectionsRow{}, fmt.Errorf("failed to get user: %w", err)
	}
	primary, err := q.GetPrimaryEmailForUser(ctx, userID)
	row := db.ListUserAccountProjectionsRow{
		ID:             user.ID,
		Name:           user.Name,
		IsAdmin:        user.IsAdmin,
		MfaEnabled:     user.MfaEnabled,
		ArchivedAt:     user.ArchivedAt,
		CreatedAt:      user.CreatedAt,
		LastLoggedInAt: user.LastLoggedInAt,
	}
	if err == nil {
		row.PrimaryEmail = &primary.Email
		row.PrimaryEmailVerifiedAt = primary.VerifiedAt
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.ListUserAccountProjectionsRow{}, fmt.Errorf("failed to get primary email: %w", err)
	}
	return row, nil
}

type AdminCreateUserInput struct {
	Name     string
	Email    string
	Password string
	IsAdmin  bool
}

func (s *UserService) AdminCreateUser(
	ctx context.Context, input AdminCreateUserInput,
) (db.ListUserAccountProjectionsRow, error) {
	if input.Name == "" || input.Email == "" {
		return db.ListUserAccountProjectionsRow{}, dateierrors.ErrInvalidInput
	}
	if len(input.Password) < 8 {
		return db.ListUserAccountProjectionsRow{}, dateierrors.ErrInvalidInput
	}

	q := s.queries()
	if _, err := q.GetUserAccountByEmail(ctx, input.Email); err == nil {
		return db.ListUserAccountProjectionsRow{}, dateierrors.ErrEmailAlreadyInUse
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.ListUserAccountProjectionsRow{}, fmt.Errorf("failed to check existing user: %w", err)
	}

	hash, salt, err := security.HashPassword(input.Password)
	if err != nil {
		return db.ListUserAccountProjectionsRow{}, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	emailID := uuid.New()
	agg := &Aggregate{}
	if err := agg.Register(
		userID, input.Name, input.Email, emailID, hash, salt, input.IsAdmin, time.Now(),
	); err != nil {
		return db.ListUserAccountProjectionsRow{}, fmt.Errorf("failed to register: %w", err)
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return db.ListUserAccountProjectionsRow{}, fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		s.sendVerificationEmail(ctx, userID, input.Email)
	}

	return s.GetUserForAdmin(ctx, userID)
}

type AdminUpdateUserInput struct {
	UserID  uuid.UUID
	Name    *string
	IsAdmin *bool
}

func (s *UserService) AdminUpdateUser(ctx context.Context, input AdminUpdateUserInput) error {
	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	now := time.Now()
	if input.Name != nil {
		if err := agg.ChangeName(*input.Name, now); err != nil {
			return dateierrors.ErrInvalidInput
		}
	}
	if input.IsAdmin != nil {
		if err := agg.SetAdmin(*input.IsAdmin, now); err != nil {
			return dateierrors.ErrInvalidInput
		}
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

type AdminResetPasswordInput struct {
	UserID   uuid.UUID
	Password string
}

func (s *UserService) AdminResetPassword(ctx context.Context, input AdminResetPasswordInput) error {
	if len(input.Password) < 8 {
		return dateierrors.ErrInvalidInput
	}
	hash, salt, err := security.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.ChangePassword(hash, salt, time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (s *UserService) AdminListEmails(
	ctx context.Context, userID uuid.UUID,
) ([]db.UserAccountEmailProjection, error) {
	q := s.queries()
	rows, err := q.GetEmailsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	return rows, nil
}

func (s *UserService) AdminAddEmail(ctx context.Context, userID uuid.UUID, email string) error {
	q := s.queries()
	_, err := q.GetUserAccountByEmail(ctx, email)
	if err == nil {
		return dateierrors.ErrEmailAlreadyInUse
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing email: %w", err)
	}

	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.AddEmail(uuid.New(), email, time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (s *UserService) AdminRemoveEmail(ctx context.Context, userID, emailID uuid.UUID) error {
	q := s.queries()
	email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{ID: emailID, UserAccountID: userID})
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

func (s *UserService) AdminSetPrimaryEmail(ctx context.Context, userID, emailID uuid.UUID) error {
	q := s.queries()
	email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{ID: emailID, UserAccountID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dateierrors.ErrNotFound
		}
		return fmt.Errorf("failed to get email: %w", err)
	}
	if email.IsPrimary {
		return nil
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

func (s *UserService) AdminDisableMFA(ctx context.Context, userID uuid.UUID) error {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if !user.MfaEnabled {
		return dateierrors.ErrMFANotEnabled
	}

	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.DisableMFA(time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to disable MFA: %w", err)
	}
	return nil
}

func (s *UserService) AdminArchiveUser(ctx context.Context, userID uuid.UUID) error {
	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.Archive(time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (s *UserService) AdminUnarchiveUser(ctx context.Context, userID uuid.UUID) error {
	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.Unarchive(time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

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
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ListUsersInput struct {
	Limit  int
	Offset int
}

type ListUsersOutput struct {
	Items []api.AdminUserListItem
	Total int
}

func (s *UserService) ListUsers(ctx context.Context, input ListUsersInput) (*ListUsersOutput, error) {
	q := s.queries()

	limit := int32(input.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int32(max(input.Offset, 0))

	total, err := q.CountUserAccountProjections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	rows, err := q.ListUserAccountProjections(ctx, db.ListUserAccountProjectionsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	items := make([]api.AdminUserListItem, len(rows))
	for i := range rows {
		items[i] = toAdminUserListItem(rows[i])
	}
	return &ListUsersOutput{Items: items, Total: int(total)}, nil
}

func (s *UserService) GetUserForAdmin(
	ctx context.Context, userID uuid.UUID,
) (api.AdminUserListItem, error) {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.AdminUserListItem{}, dateierrors.ErrNotFound
		}
		return api.AdminUserListItem{}, fmt.Errorf("failed to get user: %w", err)
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
		return api.AdminUserListItem{}, fmt.Errorf("failed to get primary email: %w", err)
	}
	return toAdminUserListItem(row), nil
}

type AdminCreateUserInput struct {
	Name     string
	Email    string
	Password string
	IsAdmin  bool
}

func (s *UserService) AdminCreateUser(
	ctx context.Context, input AdminCreateUserInput,
) (api.AdminUserListItem, error) {
	if input.Name == "" || input.Email == "" {
		return api.AdminUserListItem{}, dateierrors.ErrInvalidInput
	}
	if len(input.Password) < 8 {
		return api.AdminUserListItem{}, dateierrors.ErrInvalidInput
	}

	q := s.queries()
	exists, err := q.UserAccountEmailExists(ctx, input.Email)
	if err != nil {
		return api.AdminUserListItem{}, fmt.Errorf("failed to check existing user: %w", err)
	}
	if exists {
		return api.AdminUserListItem{}, dateierrors.ErrEmailAlreadyInUse
	}

	hash, salt, err := security.HashPassword(input.Password)
	if err != nil {
		return api.AdminUserListItem{}, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	emailID := uuid.New()
	agg := &Aggregate{}
	if err := agg.Register(
		userID, input.Name, input.Email, emailID, hash, salt, input.IsAdmin, time.Now(),
	); err != nil {
		return api.AdminUserListItem{}, fmt.Errorf("failed to register: %w", err)
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return api.AdminUserListItem{}, fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		s.sendVerificationEmail(ctx, userID, input.Email)
	}

	return s.GetUserForAdmin(ctx, userID)
}

type AdminUpdateUserInput struct {
	UserID         uuid.UUID
	Name           *string
	IsAdmin        *bool
	Archived       *bool
	PrimaryEmailID *uuid.UUID
}

func (s *UserService) AdminUpdateUser(ctx context.Context, input AdminUpdateUserInput) error {
	q := s.queries()

	// Validate primaryEmailId belongs to this user before loading the aggregate,
	// and find the current primary so the aggregate command has both IDs.
	var currentPrimaryID uuid.UUID
	if input.PrimaryEmailID != nil {
		email, err := q.GetEmailByID(ctx, db.GetEmailByIDParams{ID: *input.PrimaryEmailID, UserAccountID: input.UserID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return dateierrors.ErrNotFound
			}
			return fmt.Errorf("failed to get email: %w", err)
		}
		if !email.IsPrimary {
			primary, err := q.GetPrimaryEmailForUser(ctx, input.UserID)
			if err != nil {
				return fmt.Errorf("failed to get primary email: %w", err)
			}
			currentPrimaryID = primary.ID
		}
	}

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
	if input.Archived != nil {
		currentlyArchived := agg.ArchivedAt != nil
		if *input.Archived && !currentlyArchived {
			if err := agg.Archive(now); err != nil {
				return dateierrors.ErrInvalidInput
			}
		} else if !*input.Archived && currentlyArchived {
			if err := agg.Unarchive(now); err != nil {
				return dateierrors.ErrInvalidInput
			}
		}
	}
	if input.PrimaryEmailID != nil && currentPrimaryID != uuid.Nil {
		if err := agg.SetPrimaryEmail(currentPrimaryID, *input.PrimaryEmailID, now); err != nil {
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
) ([]api.UserEmail, error) {
	q := s.queries()
	rows, err := q.GetEmailsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	return MapEmailProjectionSliceToAPI(rows), nil
}

func (s *UserService) AdminAddEmail(ctx context.Context, userID uuid.UUID, email string) error {
	q := s.queries()
	exists, err := q.UserAccountEmailExists(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to check existing email: %w", err)
	}
	if exists {
		return dateierrors.ErrEmailAlreadyInUse
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

func (s *UserService) AdminDisableMFA(ctx context.Context, userID uuid.UUID) error {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dateierrors.ErrNotFound
		}
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

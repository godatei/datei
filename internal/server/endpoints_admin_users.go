package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

func (s *server) requireAdmin(ctx context.Context) (authn.AuthInfo, error) {
	return authn.RequireAdmin(ctx, db.New(s.pool))
}

// ListUsersAdmin implements [StrictServerInterface].
func (s *server) ListUsersAdmin(
	ctx context.Context, _ ListUsersAdminRequestObject,
) (ListUsersAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return ListUsersAdmin403Response{}, nil
		}
		return nil, err
	}

	rows, err := s.userService.ListUsers(ctx)
	if err != nil {
		slog.Error("list users error", "error", err)
		return nil, err
	}

	items := make([]api.AdminUserListItem, len(rows))
	for i := range rows {
		items[i] = users.ToAdminUserListItem(rows[i])
	}
	return ListUsersAdmin200JSONResponse(api.ListAdminUsersResponse{Users: items}), nil
}

// CreateUserAdmin implements [StrictServerInterface].
func (s *server) CreateUserAdmin(
	ctx context.Context, request CreateUserAdminRequestObject,
) (CreateUserAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return CreateUserAdmin403Response{}, nil
		}
		return nil, err
	}

	row, err := s.userService.AdminCreateUser(ctx, users.AdminCreateUserInput{
		Name:     request.Body.Name,
		Email:    string(request.Body.Email),
		Password: request.Body.Password,
		IsAdmin:  request.Body.IsAdmin,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrEmailAlreadyInUse) || errors.Is(err, dateierrors.ErrInvalidInput) {
			return CreateUserAdmin400Response{}, nil
		}
		slog.Error("admin create user error", "error", err)
		return nil, err
	}
	return CreateUserAdmin201JSONResponse(users.ToAdminUserListItem(row)), nil
}

// GetUserAdmin implements [StrictServerInterface].
func (s *server) GetUserAdmin(
	ctx context.Context, request GetUserAdminRequestObject,
) (GetUserAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return GetUserAdmin403Response{}, nil
		}
		return nil, err
	}

	row, err := s.userService.GetUserForAdmin(ctx, request.Id)
	if err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return GetUserAdmin404Response{}, nil
		}
		slog.Error("get user admin error", "error", err)
		return nil, err
	}
	return GetUserAdmin200JSONResponse(users.ToAdminUserListItem(row)), nil
}

// UpdateUserAdmin implements [StrictServerInterface].
func (s *server) UpdateUserAdmin(
	ctx context.Context, request UpdateUserAdminRequestObject,
) (UpdateUserAdminResponseObject, error) {
	auth, err := s.requireAdmin(ctx)
	if err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return UpdateUserAdmin403Response{}, nil
		}
		return nil, err
	}

	// An admin must not be able to demote themselves — they would lose access mid-call
	// and could lock the system out if they're the last admin.
	if request.Body.IsAdmin != nil && !*request.Body.IsAdmin && auth.UserID == request.Id {
		return UpdateUserAdmin400Response{}, nil
	}

	input := users.AdminUpdateUserInput{UserID: request.Id, Name: request.Body.Name}
	if request.Body.IsAdmin != nil {
		input.IsAdmin = request.Body.IsAdmin
	}
	if err := s.userService.AdminUpdateUser(ctx, input); err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return UpdateUserAdmin404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return UpdateUserAdmin400Response{}, nil
		}
		slog.Error("admin update user error", "error", err)
		return nil, err
	}
	return UpdateUserAdmin204Response{}, nil
}

// ResetUserPasswordAdmin implements [StrictServerInterface].
func (s *server) ResetUserPasswordAdmin(
	ctx context.Context, request ResetUserPasswordAdminRequestObject,
) (ResetUserPasswordAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return ResetUserPasswordAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.userService.AdminResetPassword(ctx, users.AdminResetPasswordInput{
		UserID:   request.Id,
		Password: request.Body.Password,
	}); err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return ResetUserPasswordAdmin404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return ResetUserPasswordAdmin400Response{}, nil
		}
		slog.Error("admin reset password error", "error", err)
		return nil, err
	}
	return ResetUserPasswordAdmin204Response{}, nil
}

// ListUserEmailsAdmin implements [StrictServerInterface].
func (s *server) ListUserEmailsAdmin(
	ctx context.Context, request ListUserEmailsAdminRequestObject,
) (ListUserEmailsAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return ListUserEmailsAdmin403Response{}, nil
		}
		return nil, err
	}

	rows, err := s.userService.AdminListEmails(ctx, request.Id)
	if err != nil {
		slog.Error("admin list emails error", "error", err)
		return nil, err
	}
	return ListUserEmailsAdmin200JSONResponse(api.ListEmailsResponse{
		Emails: users.MapEmailProjectionSliceToAPI(rows),
	}), nil
}

// AddUserEmailAdmin implements [StrictServerInterface].
func (s *server) AddUserEmailAdmin(
	ctx context.Context, request AddUserEmailAdminRequestObject,
) (AddUserEmailAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return AddUserEmailAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.userService.AdminAddEmail(ctx, request.Id, string(request.Body.Email)); err != nil {
		if errors.Is(err, dateierrors.ErrEmailAlreadyInUse) || errors.Is(err, dateierrors.ErrInvalidInput) {
			return AddUserEmailAdmin400Response{}, nil
		}
		slog.Error("admin add email error", "error", err)
		return nil, err
	}
	return AddUserEmailAdmin204Response{}, nil
}

// RemoveUserEmailAdmin implements [StrictServerInterface].
func (s *server) RemoveUserEmailAdmin(
	ctx context.Context, request RemoveUserEmailAdminRequestObject,
) (RemoveUserEmailAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return RemoveUserEmailAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.userService.AdminRemoveEmail(ctx, request.Id, request.EmailId); err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return RemoveUserEmailAdmin404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return RemoveUserEmailAdmin400Response{}, nil
		}
		slog.Error("admin remove email error", "error", err)
		return nil, err
	}
	return RemoveUserEmailAdmin204Response{}, nil
}

// SetPrimaryUserEmailAdmin implements [StrictServerInterface].
func (s *server) SetPrimaryUserEmailAdmin(
	ctx context.Context, request SetPrimaryUserEmailAdminRequestObject,
) (SetPrimaryUserEmailAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return SetPrimaryUserEmailAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.userService.AdminSetPrimaryEmail(ctx, request.Id, request.EmailId); err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return SetPrimaryUserEmailAdmin404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return SetPrimaryUserEmailAdmin400Response{}, nil
		}
		slog.Error("admin set primary email error", "error", err)
		return nil, err
	}
	return SetPrimaryUserEmailAdmin204Response{}, nil
}

// DisableUserMFAAdmin implements [StrictServerInterface].
func (s *server) DisableUserMFAAdmin(
	ctx context.Context, request DisableUserMFAAdminRequestObject,
) (DisableUserMFAAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return DisableUserMFAAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.userService.AdminDisableMFA(ctx, request.Id); err != nil {
		if errors.Is(err, dateierrors.ErrMFANotEnabled) {
			return DisableUserMFAAdmin403Response{}, nil
		}
		slog.Error("admin disable mfa error", "error", err)
		return nil, err
	}
	return DisableUserMFAAdmin204Response{}, nil
}

// ArchiveUserAdmin implements [StrictServerInterface].
func (s *server) ArchiveUserAdmin(
	ctx context.Context, request ArchiveUserAdminRequestObject,
) (ArchiveUserAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return ArchiveUserAdmin403Response{}, nil
		}
		return nil, err
	}

	if info, err := authn.FromContext(ctx); err == nil && info.UserID == request.Id {
		return ArchiveUserAdmin403Response{}, nil
	}

	if err := s.userService.AdminArchiveUser(ctx, request.Id); err != nil {
		slog.Error("admin archive user error", "error", err)
		return nil, err
	}
	return ArchiveUserAdmin204Response{}, nil
}

// UnarchiveUserAdmin implements [StrictServerInterface].
func (s *server) UnarchiveUserAdmin(
	ctx context.Context, request UnarchiveUserAdminRequestObject,
) (UnarchiveUserAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return UnarchiveUserAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.userService.AdminUnarchiveUser(ctx, request.Id); err != nil {
		slog.Error("admin unarchive user error", "error", err)
		return nil, err
	}
	return UnarchiveUserAdmin204Response{}, nil
}

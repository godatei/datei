package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

func (s *server) requireAdmin(ctx context.Context) (authn.AuthInfo, error) {
	return authn.RequireAdmin(ctx)
}

// ListUsersAdmin implements [StrictServerInterface].
func (s *server) ListUsersAdmin(
	ctx context.Context, request ListUsersAdminRequestObject,
) (ListUsersAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, dateierrors.ErrForbidden) {
			return ListUsersAdmin403Response{}, nil
		}
		return nil, err
	}

	limit := 0
	offset := 0
	if request.Params.Limit != nil && *request.Params.Limit > 0 {
		limit = *request.Params.Limit
	}
	if request.Params.Offset != nil && *request.Params.Offset > 0 {
		offset = *request.Params.Offset
	}

	result, err := s.userService.ListUsers(ctx, users.ListUsersInput{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	items := make([]api.AdminUserListItem, len(result.Items))
	for i := range result.Items {
		items[i] = users.ToAdminUserListItem(result.Items[i])
	}
	return ListUsersAdmin200JSONResponse(api.ListAdminUsersResponse{
		Items: items,
		Total: result.Total,
	}), nil
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
	// and could lock the system out if they're the last admin. The same reasoning
	// applies to disabling their own account.
	if auth.UserID == request.Id {
		if request.Body.IsAdmin != nil && !*request.Body.IsAdmin {
			return UpdateUserAdmin400Response{}, nil
		}
		if request.Body.Enabled != nil && !*request.Body.Enabled {
			return UpdateUserAdmin400Response{}, nil
		}
	}

	input := users.AdminUpdateUserInput{
		UserID:         request.Id,
		Name:           request.Body.Name,
		IsAdmin:        request.Body.IsAdmin,
		Enabled:        request.Body.Enabled,
		PrimaryEmailID: request.Body.PrimaryEmailId,
	}
	if err := s.userService.AdminUpdateUser(ctx, input); err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return UpdateUserAdmin404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return UpdateUserAdmin400Response{}, nil
		}
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
		return nil, err
	}
	return RemoveUserEmailAdmin204Response{}, nil
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
		if errors.Is(err, dateierrors.ErrNotFound) {
			return DisableUserMFAAdmin404Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrMFANotEnabled) {
			return DisableUserMFAAdmin403Response{}, nil
		}
		return nil, err
	}
	return DisableUserMFAAdmin204Response{}, nil
}

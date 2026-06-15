package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

type adminUsersServer struct {
	svc *users.UserService
}

func (s *adminUsersServer) requireAdmin(ctx context.Context) (db.UserAccountProjection, error) {
	return authn.RequireAdmin(ctx)
}

// ListUsersAdmin implements [StrictServerInterface].
func (s *adminUsersServer) ListUsersAdmin(
	ctx context.Context, request ListUsersAdminRequestObject,
) (ListUsersAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
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

	result, err := s.svc.ListUsers(ctx, users.ListUsersInput{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}

	return ListUsersAdmin200JSONResponse(api.ListAdminUsersResponse{
		Items: result.Items,
		Total: result.Total,
	}), nil
}

// CreateUserAdmin implements [StrictServerInterface].
func (s *adminUsersServer) CreateUserAdmin(
	ctx context.Context, request CreateUserAdminRequestObject,
) (CreateUserAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return CreateUserAdmin403Response{}, nil
		}
		return nil, err
	}

	item, err := s.svc.AdminCreateUser(ctx, users.AdminCreateUserInput{
		Name:     request.Body.Name,
		Email:    string(request.Body.Email),
		Password: request.Body.Password,
		IsAdmin:  request.Body.IsAdmin,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrEmailAlreadyInUse) || errors.Is(err, apperrors.ErrInvalidInput) {
			return CreateUserAdmin400Response{}, nil
		}
		return nil, err
	}
	return CreateUserAdmin201JSONResponse(item), nil
}

// GetUserAdmin implements [StrictServerInterface].
func (s *adminUsersServer) GetUserAdmin(
	ctx context.Context, request GetUserAdminRequestObject,
) (GetUserAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return GetUserAdmin403Response{}, nil
		}
		return nil, err
	}

	item, err := s.svc.GetUserForAdmin(ctx, request.Id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return GetUserAdmin404Response{}, nil
		}
		return nil, err
	}
	return GetUserAdmin200JSONResponse(item), nil
}

// UpdateUserAdmin implements [StrictServerInterface].
func (s *adminUsersServer) UpdateUserAdmin(
	ctx context.Context, request UpdateUserAdminRequestObject,
) (UpdateUserAdminResponseObject, error) {
	admin, err := s.requireAdmin(ctx)
	if err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return UpdateUserAdmin403Response{}, nil
		}
		return nil, err
	}

	// An admin must not be able to demote themselves — they would lose access mid-call
	// and could lock the system out if they're the last admin. The same reasoning
	// applies to archiving their own account.
	if admin.ID == request.Id {
		if request.Body.IsAdmin != nil && !*request.Body.IsAdmin {
			return UpdateUserAdmin400Response{}, nil
		}
		if request.Body.Archived != nil && *request.Body.Archived {
			return UpdateUserAdmin400Response{}, nil
		}
	}

	input := users.AdminUpdateUserInput{
		UserID:         request.Id,
		Name:           request.Body.Name,
		IsAdmin:        request.Body.IsAdmin,
		Archived:       request.Body.Archived,
		PrimaryEmailID: request.Body.PrimaryEmailId,
	}
	if err := s.svc.AdminUpdateUser(ctx, input); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return UpdateUserAdmin404Response{}, nil
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return UpdateUserAdmin400Response{}, nil
		}
		return nil, err
	}
	return UpdateUserAdmin204Response{}, nil
}

// ResetUserPasswordAdmin implements [StrictServerInterface].
func (s *adminUsersServer) ResetUserPasswordAdmin(
	ctx context.Context, request ResetUserPasswordAdminRequestObject,
) (ResetUserPasswordAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return ResetUserPasswordAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.svc.AdminResetPassword(ctx, users.AdminResetPasswordInput{
		UserID:   request.Id,
		Password: request.Body.Password,
	}); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return ResetUserPasswordAdmin404Response{}, nil
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return ResetUserPasswordAdmin400Response{}, nil
		}
		return nil, err
	}
	return ResetUserPasswordAdmin204Response{}, nil
}

// ListUserEmailsAdmin implements [StrictServerInterface].
func (s *adminUsersServer) ListUserEmailsAdmin(
	ctx context.Context, request ListUserEmailsAdminRequestObject,
) (ListUserEmailsAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return ListUserEmailsAdmin403Response{}, nil
		}
		return nil, err
	}

	emails, err := s.svc.AdminListEmails(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	return ListUserEmailsAdmin200JSONResponse(api.ListEmailsResponse{
		Emails: emails,
	}), nil
}

// AddUserEmailAdmin implements [StrictServerInterface].
func (s *adminUsersServer) AddUserEmailAdmin(
	ctx context.Context, request AddUserEmailAdminRequestObject,
) (AddUserEmailAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return AddUserEmailAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.svc.AdminAddEmail(ctx, request.Id, string(request.Body.Email)); err != nil {
		if errors.Is(err, apperrors.ErrEmailAlreadyInUse) || errors.Is(err, apperrors.ErrInvalidInput) {
			return AddUserEmailAdmin400Response{}, nil
		}
		return nil, err
	}
	return AddUserEmailAdmin204Response{}, nil
}

// RemoveUserEmailAdmin implements [StrictServerInterface].
func (s *adminUsersServer) RemoveUserEmailAdmin(
	ctx context.Context, request RemoveUserEmailAdminRequestObject,
) (RemoveUserEmailAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return RemoveUserEmailAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.svc.AdminRemoveEmail(ctx, request.Id, request.EmailId); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return RemoveUserEmailAdmin404Response{}, nil
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return RemoveUserEmailAdmin400Response{}, nil
		}
		return nil, err
	}
	return RemoveUserEmailAdmin204Response{}, nil
}

// DisableUserMFAAdmin implements [StrictServerInterface].
func (s *adminUsersServer) DisableUserMFAAdmin(
	ctx context.Context, request DisableUserMFAAdminRequestObject,
) (DisableUserMFAAdminResponseObject, error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		if errors.Is(err, apperrors.ErrForbidden) {
			return DisableUserMFAAdmin403Response{}, nil
		}
		return nil, err
	}

	if err := s.svc.AdminDisableMFA(ctx, request.Id); err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return DisableUserMFAAdmin404Response{}, nil
		}
		if errors.Is(err, apperrors.ErrMFANotEnabled) {
			return DisableUserMFAAdmin403Response{}, nil
		}
		return nil, err
	}
	return DisableUserMFAAdmin204Response{}, nil
}

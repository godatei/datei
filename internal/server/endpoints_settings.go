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

// GetCurrentUser implements [StrictServerInterface].
func (s *server) GetCurrentUser(
	ctx context.Context, _ GetCurrentUserRequestObject,
) (GetCurrentUserResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	user, err := s.userService.GetUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("get current user error", "error", err)
		return nil, err
	}

	return GetCurrentUser200JSONResponse(api.UserResponse{Name: user.Name, MfaEnabled: user.MfaEnabled}), nil
}

// UpdateUser implements [StrictServerInterface].
func (s *server) UpdateUser(ctx context.Context, request UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.UpdateUser(ctx, users.UpdateUserInput{
		UserID:          authInfo.UserID,
		Name:            request.Body.Name,
		Password:        request.Body.Password,
		CurrentPassword: request.Body.CurrentPassword,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) ||
			errors.Is(err, dateierrors.ErrCurrentPasswordRequired) {
			return UpdateUser403Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return UpdateUser400Response{}, nil
		}
		slog.Error("update user error", "error", err)
		return nil, err
	}

	user, err := s.userService.GetUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get updated user", "error", err)
		return nil, err
	}

	return UpdateUser200JSONResponse(api.UserResponse{Name: user.Name, MfaEnabled: user.MfaEnabled}), nil
}

// UpdateUserEmail implements [StrictServerInterface].
func (s *server) UpdateUserEmail(
	ctx context.Context, request UpdateUserEmailRequestObject,
) (UpdateUserEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.UpdateUserEmail(ctx, users.UpdateUserEmailInput{
		UserID:   authInfo.UserID,
		NewEmail: string(request.Body.Email),
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return UpdateUserEmail400Response{}, nil
		}
		slog.Error("update user email error", "error", err)
		return nil, err
	}

	return UpdateUserEmail204Response{}, nil
}

// RequestEmailVerification implements [StrictServerInterface].
func (s *server) RequestEmailVerification(
	ctx context.Context, _ RequestEmailVerificationRequestObject,
) (RequestEmailVerificationResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if err := s.userService.RequestEmailVerification(ctx, authInfo.UserID); err != nil {
		slog.Error("request email verification error", "error", err)
		return nil, err
	}

	return RequestEmailVerification204Response{}, nil
}

// ConfirmEmailVerification implements [StrictServerInterface].
func (s *server) ConfirmEmailVerification(
	ctx context.Context, _ ConfirmEmailVerificationRequestObject,
) (ConfirmEmailVerificationResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.ConfirmEmailVerification(ctx, users.ConfirmEmailVerificationInput{
		UserID:     authInfo.UserID,
		TokenEmail: authInfo.Email,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrEmailMismatch) {
			return ConfirmEmailVerification403Response{}, nil
		}
		slog.Error("confirm email verification error", "error", err)
		return nil, err
	}

	return ConfirmEmailVerification204Response{}, nil
}

// SetupMFA implements [StrictServerInterface].
func (s *server) SetupMFA(ctx context.Context, _ SetupMFARequestObject) (SetupMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	result, err := s.userService.SetupMFA(ctx, authInfo.UserID)
	if err != nil {
		if errors.Is(err, dateierrors.ErrMFAAlreadyEnabled) {
			return SetupMFA403Response{}, nil
		}
		slog.Error("setup MFA error", "error", err)
		return nil, err
	}

	return SetupMFA200JSONResponse(api.SetupMFAResponse{
		Secret:    result.Secret,
		QrCodeUrl: result.QrCodeUrl,
	}), nil
}

// EnableMFA implements [StrictServerInterface].
func (s *server) EnableMFA(ctx context.Context, request EnableMFARequestObject) (EnableMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	result, err := s.userService.EnableMFA(ctx, users.EnableMFAInput{
		UserID: authInfo.UserID,
		Code:   request.Body.Code,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrMFAInvalidCode) ||
			errors.Is(err, dateierrors.ErrMFAAlreadyEnabled) ||
			errors.Is(err, dateierrors.ErrMFANotSetUp) {
			return EnableMFA403Response{}, nil
		}
		slog.Error("enable MFA error", "error", err)
		return nil, err
	}

	return EnableMFA200JSONResponse(api.EnableMFAResponse{RecoveryCodes: result.RecoveryCodes}), nil
}

// DisableMFA implements [StrictServerInterface].
func (s *server) DisableMFA(ctx context.Context, request DisableMFARequestObject) (DisableMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.DisableMFA(ctx, users.DisableMFAInput{
		UserID:   authInfo.UserID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) ||
			errors.Is(err, dateierrors.ErrMFANotEnabled) {
			return DisableMFA403Response{}, nil
		}
		slog.Error("disable MFA error", "error", err)
		return nil, err
	}

	return DisableMFA204Response{}, nil
}

// RegenerateMFARecoveryCodes implements [StrictServerInterface].
func (s *server) RegenerateMFARecoveryCodes(
	ctx context.Context, request RegenerateMFARecoveryCodesRequestObject,
) (RegenerateMFARecoveryCodesResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	codes, err := s.userService.RegenerateMFARecoveryCodes(ctx, users.RegenerateRecoveryCodesInput{
		UserID:   authInfo.UserID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) ||
			errors.Is(err, dateierrors.ErrMFANotEnabled) {
			return RegenerateMFARecoveryCodes403Response{}, nil
		}
		slog.Error("regenerate recovery codes error", "error", err)
		return nil, err
	}

	return RegenerateMFARecoveryCodes200JSONResponse(api.RegenerateMFARecoveryCodesResponse{
		RecoveryCodes: codes,
	}), nil
}

// GetMFARecoveryCodesStatus implements [StrictServerInterface].
func (s *server) GetMFARecoveryCodesStatus(
	ctx context.Context, _ GetMFARecoveryCodesStatusRequestObject,
) (GetMFARecoveryCodesStatusResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	count, err := s.userService.GetMFARecoveryCodesStatus(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("get MFA recovery codes status error", "error", err)
		return nil, err
	}

	return GetMFARecoveryCodesStatus200JSONResponse(api.MFARecoveryCodesStatusResponse{
		RemainingCodes: count,
	}), nil
}

// ConfirmResetPassword implements [StrictServerInterface].
func (s *server) ConfirmResetPassword(
	ctx context.Context, request ConfirmResetPasswordRequestObject,
) (ConfirmResetPasswordResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	err := s.userService.ConfirmResetPassword(ctx, users.ConfirmResetPasswordInput{
		UserID:   authInfo.UserID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidInput) {
			return ConfirmResetPassword400Response{}, nil
		}
		slog.Error("confirm reset password error", "error", err)
		return nil, err
	}

	return ConfirmResetPassword204Response{}, nil
}

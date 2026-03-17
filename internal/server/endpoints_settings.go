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

// UpdateUser implements [StrictServerInterface].
func (s *server) UpdateUser(ctx context.Context, request UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil {
		return UpdateUser400Response{}, nil
	}

	err := s.userService.UpdateUser(ctx, users.UpdateUserInput{
		UserID:          authInfo.UserID,
		Name:            request.Body.Name,
		Password:        request.Body.Password,
		CurrentPassword: request.Body.CurrentPassword,
		IsPasswordReset: authInfo.PasswordReset,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) {
			return UpdateUser401Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) ||
			errors.Is(err, dateierrors.ErrPasswordResetOnly) ||
			errors.Is(err, dateierrors.ErrCurrentPasswordRequired) {
			return UpdateUser400Response{}, nil
		}
		slog.Error("update user error", "error", err)
		return nil, err
	}

	return UpdateUser204Response{}, nil
}

// UpdateUserEmail implements [StrictServerInterface].
func (s *server) UpdateUserEmail(
	ctx context.Context, request UpdateUserEmailRequestObject,
) (UpdateUserEmailResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil {
		return UpdateUserEmail400Response{}, nil
	}

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
		UserID:        authInfo.UserID,
		EmailVerified: authInfo.EmailVerified,
		PasswordReset: authInfo.PasswordReset,
		TokenEmail:    authInfo.Email,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidToken) ||
			errors.Is(err, dateierrors.ErrEmailMismatch) ||
			errors.Is(err, dateierrors.ErrInvalidInput) {
			return ConfirmEmailVerification400Response{}, nil
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
		if errors.Is(err, dateierrors.ErrMFAAlreadyEnabled) || errors.Is(err, dateierrors.ErrInvalidInput) {
			return SetupMFA400Response{}, nil
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

	if request.Body == nil {
		return EnableMFA400Response{}, nil
	}

	result, err := s.userService.EnableMFA(ctx, users.EnableMFAInput{
		UserID: authInfo.UserID,
		Code:   request.Body.Code,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrMFAAlreadyEnabled) ||
			errors.Is(err, dateierrors.ErrMFANotSetUp) ||
			errors.Is(err, dateierrors.ErrMFAInvalidCode) ||
			errors.Is(err, dateierrors.ErrInvalidInput) {
			return EnableMFA400Response{}, nil
		}
		slog.Error("enable MFA error", "error", err)
		return nil, err
	}

	return EnableMFA200JSONResponse(api.EnableMFAResponse{RecoveryCodes: result.RecoveryCodes}), nil
}

// DisableMFA implements [StrictServerInterface].
func (s *server) DisableMFA(ctx context.Context, request DisableMFARequestObject) (DisableMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil {
		return DisableMFA400Response{}, nil
	}

	err := s.userService.DisableMFA(ctx, users.DisableMFAInput{
		UserID:   authInfo.UserID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) {
			return DisableMFA401Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrMFANotEnabled) || errors.Is(err, dateierrors.ErrInvalidInput) {
			return DisableMFA400Response{}, nil
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

	if request.Body == nil {
		return RegenerateMFARecoveryCodes400Response{}, nil
	}

	codes, err := s.userService.RegenerateMFARecoveryCodes(ctx, users.RegenerateRecoveryCodesInput{
		UserID:   authInfo.UserID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) {
			return RegenerateMFARecoveryCodes401Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrMFANotEnabled) || errors.Is(err, dateierrors.ErrInvalidInput) {
			return RegenerateMFARecoveryCodes400Response{}, nil
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

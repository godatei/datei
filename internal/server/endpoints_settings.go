package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

type settingsServer struct {
	svc *users.UserService
}

// GetCurrentUser implements [StrictServerInterface].
func (s *settingsServer) GetCurrentUser(
	ctx context.Context, _ GetCurrentUserRequestObject,
) (GetCurrentUserResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	return GetCurrentUser200JSONResponse(api.UserResponse{
		Name:       user.Name,
		IsAdmin:    user.IsAdmin,
		MfaEnabled: user.MfaEnabled,
	}), nil
}

// UpdateUser implements [StrictServerInterface].
func (s *settingsServer) UpdateUser(
	ctx context.Context, request UpdateUserRequestObject,
) (UpdateUserResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.UpdateUser(ctx, users.UpdateUserInput{
		UserID:          user.ID,
		Name:            request.Body.Name,
		Password:        request.Body.Password,
		CurrentPassword: request.Body.CurrentPassword,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) ||
			errors.Is(err, apperrors.ErrCurrentPasswordRequired) {
			return UpdateUser403Response{}, nil
		}
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return UpdateUser400Response{}, nil
		}
		return nil, err
	}

	user, err = s.svc.GetUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return UpdateUser200JSONResponse(api.UserResponse{
		Name:       user.Name,
		IsAdmin:    user.IsAdmin,
		MfaEnabled: user.MfaEnabled,
	}), nil
}

// UpdateUserEmail implements [StrictServerInterface].
func (s *settingsServer) UpdateUserEmail(
	ctx context.Context, request UpdateUserEmailRequestObject,
) (UpdateUserEmailResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.UpdateUserEmail(ctx, users.UpdateUserEmailInput{
		UserID:   user.ID,
		NewEmail: string(request.Body.Email),
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return UpdateUserEmail400Response{}, nil
		}
		return nil, err
	}

	return UpdateUserEmail204Response{}, nil
}

// RequestEmailVerification implements [StrictServerInterface].
func (s *settingsServer) RequestEmailVerification(
	ctx context.Context, _ RequestEmailVerificationRequestObject,
) (RequestEmailVerificationResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	if err := s.svc.RequestEmailVerification(ctx, user.ID); err != nil {
		return nil, err
	}

	return RequestEmailVerification204Response{}, nil
}

// ConfirmEmailVerification implements [StrictServerInterface].
func (s *settingsServer) ConfirmEmailVerification(
	ctx context.Context, _ ConfirmEmailVerificationRequestObject,
) (ConfirmEmailVerificationResponseObject, error) {
	identity := authn.RequireEmailIdentity(ctx)

	err := s.svc.ConfirmEmailVerification(ctx, users.ConfirmEmailVerificationInput{
		UserID:     authn.RequireCurrentUser(ctx).ID,
		TokenEmail: identity.Email,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrEmailMismatch) {
			return ConfirmEmailVerification403Response{}, nil
		}
		return nil, err
	}

	return ConfirmEmailVerification204Response{}, nil
}

// SetupMFA implements [StrictServerInterface].
func (s *settingsServer) SetupMFA(ctx context.Context, _ SetupMFARequestObject) (SetupMFAResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	result, err := s.svc.SetupMFA(ctx, user.ID)
	if err != nil {
		if errors.Is(err, apperrors.ErrMFAAlreadyEnabled) {
			return SetupMFA403Response{}, nil
		}
		return nil, err
	}

	return SetupMFA200JSONResponse(api.SetupMFAResponse{
		Secret:    result.Secret,
		QrCodeUrl: result.QrCodeUrl,
	}), nil
}

// EnableMFA implements [StrictServerInterface].
func (s *settingsServer) EnableMFA(
	ctx context.Context, request EnableMFARequestObject,
) (EnableMFAResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	result, err := s.svc.EnableMFA(ctx, users.EnableMFAInput{
		UserID: user.ID,
		Code:   request.Body.Code,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrMFAInvalidCode) ||
			errors.Is(err, apperrors.ErrMFAAlreadyEnabled) ||
			errors.Is(err, apperrors.ErrMFANotSetUp) {
			return EnableMFA403Response{}, nil
		}
		return nil, err
	}

	return EnableMFA200JSONResponse(api.EnableMFAResponse{RecoveryCodes: result.RecoveryCodes}), nil
}

// DisableMFA implements [StrictServerInterface].
func (s *settingsServer) DisableMFA(
	ctx context.Context, request DisableMFARequestObject,
) (DisableMFAResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.DisableMFA(ctx, users.DisableMFAInput{
		UserID:   user.ID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) ||
			errors.Is(err, apperrors.ErrMFANotEnabled) {
			return DisableMFA403Response{}, nil
		}
		return nil, err
	}

	return DisableMFA204Response{}, nil
}

// RegenerateMFARecoveryCodes implements [StrictServerInterface].
func (s *settingsServer) RegenerateMFARecoveryCodes(
	ctx context.Context, request RegenerateMFARecoveryCodesRequestObject,
) (RegenerateMFARecoveryCodesResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	codes, err := s.svc.RegenerateMFARecoveryCodes(ctx, users.RegenerateRecoveryCodesInput{
		UserID:   user.ID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) ||
			errors.Is(err, apperrors.ErrMFANotEnabled) {
			return RegenerateMFARecoveryCodes403Response{}, nil
		}
		return nil, err
	}

	return RegenerateMFARecoveryCodes200JSONResponse(api.RegenerateMFARecoveryCodesResponse{
		RecoveryCodes: codes,
	}), nil
}

// GetMFARecoveryCodesStatus implements [StrictServerInterface].
func (s *settingsServer) GetMFARecoveryCodesStatus(
	ctx context.Context, _ GetMFARecoveryCodesStatusRequestObject,
) (GetMFARecoveryCodesStatusResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	count, err := s.svc.GetMFARecoveryCodesStatus(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return GetMFARecoveryCodesStatus200JSONResponse(api.MFARecoveryCodesStatusResponse{
		RemainingCodes: count,
	}), nil
}

// ConfirmResetPassword implements [StrictServerInterface].
func (s *settingsServer) ConfirmResetPassword(
	ctx context.Context, request ConfirmResetPasswordRequestObject,
) (ConfirmResetPasswordResponseObject, error) {
	user := authn.RequireCurrentUser(ctx)

	err := s.svc.ConfirmResetPassword(ctx, users.ConfirmResetPasswordInput{
		UserID:   user.ID,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidInput) {
			return ConfirmResetPassword400Response{}, nil
		}
		return nil, err
	}

	return ConfirmResetPassword204Response{}, nil
}

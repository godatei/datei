package server

import (
	"context"
	"errors"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

type authServer struct {
	svc *users.UserService
}

// Login implements [StrictServerInterface].
func (s *authServer) Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error) {
	result, err := s.svc.Login(ctx, users.LoginInput{
		Email:    string(request.Body.Email),
		Password: request.Body.Password,
		MfaCode:  request.Body.MfaCode,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrInvalidCredentials) {
			return Login401Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrMFAInvalidCode) {
			return Login401Response{}, nil
		}
		return nil, err
	}

	if result.RequiresMFA {
		return Login200JSONResponse(api.LoginResponse{RequiresMfa: new(true)}), nil
	}

	return Login200JSONResponse(api.LoginResponse{Token: &result.Token}), nil
}

// GetLoginConfig implements [StrictServerInterface].
func (s *authServer) GetLoginConfig(
	_ context.Context, _ GetLoginConfigRequestObject,
) (GetLoginConfigResponseObject, error) {
	cfg := s.svc.GetLoginConfig()
	return GetLoginConfig200JSONResponse(api.LoginConfigResponse{
		RegistrationEnabled: cfg.RegistrationEnabled,
	}), nil
}

// Register implements [StrictServerInterface].
func (s *authServer) Register(ctx context.Context, request RegisterRequestObject) (RegisterResponseObject, error) {
	err := s.svc.Register(ctx, users.RegisterInput{
		Email:    string(request.Body.Email),
		Name:     request.Body.Name,
		Password: request.Body.Password,
	})
	if err != nil {
		if errors.Is(err, dateierrors.ErrRegistrationDisabled) {
			return Register403Response{}, nil
		}
		if errors.Is(err, dateierrors.ErrInvalidInput) || errors.Is(err, dateierrors.ErrEmailAlreadyInUse) {
			return Register400Response{}, nil
		}
		return nil, err
	}

	return Register204Response{}, nil
}

// ResetPassword implements [StrictServerInterface].
func (s *authServer) ResetPassword(
	ctx context.Context, request ResetPasswordRequestObject,
) (ResetPasswordResponseObject, error) {
	s.svc.ResetPassword(ctx, users.ResetPasswordInput{
		Email: string(request.Body.Email),
	})
	return ResetPassword204Response{}, nil
}

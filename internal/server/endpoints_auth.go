package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/users"
	"github.com/godatei/datei/pkg/api"
)

// Login implements [StrictServerInterface].
func (s *server) Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error) {
	result, err := s.userService.Login(ctx, users.LoginInput{
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
		slog.Error("login error", "error", err)
		return nil, err
	}

	if result.RequiresMFA {
		return Login200JSONResponse(api.LoginResponse{RequiresMfa: new(true)}), nil
	}

	return Login200JSONResponse(api.LoginResponse{Token: &result.Token}), nil
}

// GetLoginConfig implements [StrictServerInterface].
func (s *server) GetLoginConfig(
	_ context.Context, _ GetLoginConfigRequestObject,
) (GetLoginConfigResponseObject, error) {
	cfg := s.userService.GetLoginConfig()
	return GetLoginConfig200JSONResponse(api.LoginConfigResponse{
		RegistrationEnabled: cfg.RegistrationEnabled,
	}), nil
}

// Register implements [StrictServerInterface].
func (s *server) Register(ctx context.Context, request RegisterRequestObject) (RegisterResponseObject, error) {
	err := s.userService.Register(ctx, users.RegisterInput{
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
		slog.Error("register error", "error", err)
		return nil, err
	}

	return Register204Response{}, nil
}

// ResetPassword implements [StrictServerInterface].
func (s *server) ResetPassword(
	ctx context.Context, request ResetPasswordRequestObject,
) (ResetPasswordResponseObject, error) {
	s.userService.ResetPassword(ctx, users.ResetPasswordInput{
		Email: string(request.Body.Email),
	})
	return ResetPassword204Response{}, nil
}

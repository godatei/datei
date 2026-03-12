package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/security"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
)

// Login implements [StrictServerInterface].
func (s *server) Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error) {
	if request.Body == nil {
		return Login400Response{}, nil
	}

	q := db.New(s.pool)
	user, err := q.GetUserAccountByEmail(ctx, request.Body.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Login400Response{}, nil
		}
		slog.Error("failed to get user", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := security.VerifyPassword(request.Body.Password, user.PasswordHash, user.PasswordSalt); err != nil {
		return Login400Response{}, nil
	}

	if user.MfaEnabled {
		if request.Body.MfaCode == nil {
			return Login200JSONResponse(api.LoginResponse{RequiresMfa: boolPtr(true)}), nil
		}

		if user.MfaSecret == nil {
			slog.Error("MFA configuration error: secret is nil but MFA is enabled")
			return nil, fmt.Errorf("MFA configuration error")
		}

		valid := totp.Validate(*request.Body.MfaCode, *user.MfaSecret)

		if !valid {
			normalized := security.NormalizeRecoveryCode(*request.Body.MfaCode)
			codes, err := q.GetUnusedMFARecoveryCodes(ctx, user.ID)
			if err != nil {
				slog.Error("failed to get recovery codes", "error", err)
				return nil, fmt.Errorf("failed to get recovery codes: %w", err)
			}

			var matchedCodeID *uuid.UUID
			for _, code := range codes {
				if security.VerifyRecoveryCode(normalized, code.CodeHash, code.CodeSalt) {
					matchedCodeID = &code.ID
					break
				}
			}

			if matchedCodeID == nil {
				return Login401Response{}, nil
			}

			agg, err := s.userRepo.LoadByID(ctx, user.ID)
			if err != nil {
				slog.Error("failed to load user aggregate", "error", err)
				return nil, fmt.Errorf("failed to load user: %w", err)
			}
			if err := agg.UseRecoveryCode(*matchedCodeID, time.Now()); err != nil {
				slog.Error("failed to use recovery code", "error", err)
				return nil, fmt.Errorf("failed to use recovery code: %w", err)
			}
			if err := s.userRepo.Save(ctx, agg); err != nil {
				slog.Error("failed to save user aggregate", "error", err)
				return nil, fmt.Errorf("failed to save user: %w", err)
			}
		}
	}

	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get primary email", "error", err)
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}

	emailVerified := !config.AuthEmailVerificationRequired() || primaryEmail.VerifiedAt != nil

	_, tokenString, err := authjwt.GenerateDefaultToken(user.ID, user.Name, primaryEmail.Email, emailVerified)
	if err != nil {
		slog.Error("failed to generate token", "error", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	agg, err := s.userRepo.LoadByID(ctx, user.ID)
	if err != nil {
		slog.Error("failed to load user for login tracking", "error", err)
	} else {
		if err := agg.RecordLogin(time.Now()); err == nil {
			if err := s.userRepo.Save(ctx, agg); err != nil {
				slog.Error("failed to save login event", "error", err)
			}
		}
	}

	return Login200JSONResponse(api.LoginResponse{Token: &tokenString}), nil
}

// GetLoginConfig implements [StrictServerInterface].
func (s *server) GetLoginConfig(
	_ context.Context, _ GetLoginConfigRequestObject,
) (GetLoginConfigResponseObject, error) {
	return GetLoginConfig200JSONResponse(api.LoginConfigResponse{
		RegistrationEnabled: config.AuthRegistrationEnabled(),
	}), nil
}

// Register implements [StrictServerInterface].
func (s *server) Register(ctx context.Context, request RegisterRequestObject) (RegisterResponseObject, error) {
	if !config.AuthRegistrationEnabled() {
		return Register403Response{}, nil
	}

	if request.Body == nil {
		return Register400Response{}, nil
	}

	if request.Body.Email == "" || request.Body.Password == "" || request.Body.Name == "" {
		return Register400Response{}, nil
	}
	if len(request.Body.Password) < 8 {
		return Register400Response{}, nil
	}

	q := db.New(s.pool)
	_, err := q.GetUserAccountByEmail(ctx, request.Body.Email)
	if err == nil {
		return Register400Response{}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		slog.Error("failed to check existing user", "error", err)
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	passwordHash, passwordSalt, err := security.HashPassword(request.Body.Password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	emailID := uuid.New()
	agg := &aggregate.UserAggregate{}
	err = agg.Register(
		userID, request.Body.Name, request.Body.Email, emailID, passwordHash, passwordSalt, time.Now(),
	)
	if err != nil {
		slog.Error("failed to create user aggregate", "error", err)
		return nil, fmt.Errorf("failed to register: %w", err)
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		_, token, err := authjwt.GenerateVerificationToken(userID, request.Body.Email)
		if err != nil {
			slog.Error("failed to generate verification token", "error", err)
		} else {
			verifyURL := fmt.Sprintf("%s/verify?jwt=%s", config.ServerHost(), token)
			if err := s.mailer.Send(ctx, mailer.EmailVerificationEmail(request.Body.Email, verifyURL)); err != nil {
				slog.Warn("failed to send verification email", "error", err)
			}
		}
	}

	return Register204Response{}, nil
}

// ResetPassword implements [StrictServerInterface].
func (s *server) ResetPassword(
	ctx context.Context, request ResetPasswordRequestObject,
) (ResetPasswordResponseObject, error) {
	if request.Body == nil {
		return ResetPassword204Response{}, nil
	}

	q := db.New(s.pool)
	user, err := q.GetUserAccountByEmail(ctx, request.Body.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("failed to get user for password reset", "error", err)
		}
		// Don't reveal whether email exists
		return ResetPassword204Response{}, nil
	}

	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get primary email", "error", err)
		return ResetPassword204Response{}, nil
	}

	_, token, err := authjwt.GenerateResetToken(user.ID, primaryEmail.Email)
	if err != nil {
		slog.Error("failed to generate reset token", "error", err)
		return nil, fmt.Errorf("failed to generate reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset?jwt=%s", config.ServerHost(), token)
	if err := s.mailer.Send(ctx, mailer.PasswordResetEmail(primaryEmail.Email, resetURL)); err != nil {
		slog.Warn("failed to send reset email", "error", err)
		return nil, fmt.Errorf("failed to send reset email: %w", err)
	}

	return ResetPassword204Response{}, nil
}

func boolPtr(b bool) *bool { return &b }

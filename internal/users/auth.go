package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
)

type LoginInput struct {
	Email    string
	Password string
	MfaCode  *string
}

type LoginOutput struct {
	Token       string
	RequiresMFA bool
}

func (s *UserService) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	q := s.queries()
	user, err := q.GetUserAccountByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dateierrors.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := security.VerifyPassword(input.Password, user.PasswordHash, user.PasswordSalt); err != nil {
		return nil, dateierrors.ErrInvalidCredentials
	}

	if user.MfaEnabled {
		if input.MfaCode == nil {
			return &LoginOutput{RequiresMFA: true}, nil
		}

		if user.MfaSecret == nil {
			return nil, fmt.Errorf("MFA configuration error: secret is nil but MFA is enabled")
		}

		valid := totp.Validate(*input.MfaCode, *user.MfaSecret)

		if !valid {
			normalized := security.NormalizeRecoveryCode(*input.MfaCode)
			codes, err := q.GetUnusedMFARecoveryCodes(ctx, user.ID)
			if err != nil {
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
				return nil, dateierrors.ErrMFAInvalidCode
			}

			agg, err := s.repository.LoadByID(ctx, user.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to load user: %w", err)
			}
			if err := agg.UseRecoveryCode(*matchedCodeID, time.Now()); err != nil {
				return nil, fmt.Errorf("failed to use recovery code: %w", err)
			}
			if err := s.repository.Save(ctx, agg); err != nil {
				return nil, fmt.Errorf("failed to save user: %w", err)
			}
		}
	}

	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}

	emailVerified := !config.AuthEmailVerificationRequired() || primaryEmail.VerifiedAt != nil

	_, tokenString, err := authjwt.GenerateDefaultToken(user.ID, user.Name, primaryEmail.Email, emailVerified)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	agg, err := s.repository.LoadByID(ctx, user.ID)
	if err != nil {
		slog.Error("failed to load user for login tracking", "error", err)
	} else {
		if err := agg.RecordLogin(time.Now()); err == nil {
			if err := s.repository.Save(ctx, agg); err != nil {
				slog.Error("failed to save login event", "error", err)
			}
		}
	}

	return &LoginOutput{Token: tokenString}, nil
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

func (s *UserService) Register(ctx context.Context, input RegisterInput) error {
	if !config.AuthRegistrationEnabled() {
		return dateierrors.ErrRegistrationDisabled
	}

	if input.Email == "" || input.Password == "" || input.Name == "" {
		return dateierrors.ErrInvalidInput
	}
	if len(input.Password) < 8 {
		return dateierrors.ErrInvalidInput
	}

	q := s.queries()
	_, err := q.GetUserAccountByEmail(ctx, input.Email)
	if err == nil {
		return dateierrors.ErrEmailAlreadyInUse
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	passwordHash, passwordSalt, err := security.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	emailID := uuid.New()
	agg := &Aggregate{}
	if err := agg.Register(userID, input.Name, input.Email, emailID, passwordHash, passwordSalt, time.Now()); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		s.sendVerificationEmail(ctx, userID, input.Email)
	}

	return nil
}

type ResetPasswordInput struct {
	Email string
}

// ResetPassword sends a password reset email. It never returns an error to avoid
// revealing whether an email exists.
func (s *UserService) ResetPassword(ctx context.Context, input ResetPasswordInput) {
	q := s.queries()
	user, err := q.GetUserAccountByEmail(ctx, input.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("failed to get user for password reset", "error", err)
		}
		return
	}

	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, user.ID)
	if err != nil {
		slog.Error("failed to get primary email", "error", err)
		return
	}

	_, token, err := authjwt.GenerateResetToken(user.ID, primaryEmail.Email)
	if err != nil {
		slog.Error("failed to generate reset token", "error", err)
		return
	}

	resetURL := fmt.Sprintf("%s/reset?jwt=%s", config.ServerHost(), token)
	if err := s.mailer.Send(ctx, mailer.PasswordResetEmail(primaryEmail.Email, resetURL)); err != nil {
		slog.Warn("failed to send reset email", "error", err)
	}
}

func (s *UserService) GetLoginConfig() LoginConfigOutput {
	return LoginConfigOutput{
		RegistrationEnabled: config.AuthRegistrationEnabled(),
	}
}

type LoginConfigOutput struct {
	RegistrationEnabled bool
}

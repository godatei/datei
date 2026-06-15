package users

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
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
	user, err := s.verifyCredentials(ctx, input.Email, input.Password)
	if err != nil {
		return nil, err
	}

	q := s.queries()

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
				return nil, apperrors.ErrMFAInvalidCode
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

	tokenString, err := authjwt.GenerateDefaultToken(
		user.ID, user.Name, primaryEmail.Email, user.IsAdmin, emailVerified,
	)
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

type ValidateCredentialsOutput struct {
	UserID        uuid.UUID
	Name          string
	Email         string
	EmailVerified bool
	RequiresMFA   bool
}

// ValidateCredentials verifies email/password without generating a JWT or
// recording a login event. Use this for protocol-level auth (e.g. WebDAV
// Basic Auth) where a login event per request would be too noisy.
func (s *UserService) ValidateCredentials(
	ctx context.Context,
	email, password string,
) (*ValidateCredentialsOutput, error) {
	user, err := s.verifyCredentials(ctx, email, password)
	if err != nil {
		return nil, err
	}
	if user.MfaEnabled {
		return &ValidateCredentialsOutput{RequiresMFA: true}, nil
	}
	primaryEmail, err := s.queries().GetPrimaryEmailForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}
	emailVerified := !config.AuthEmailVerificationRequired() || primaryEmail.VerifiedAt != nil
	return &ValidateCredentialsOutput{
		UserID:        user.ID,
		Name:          user.Name,
		Email:         primaryEmail.Email,
		EmailVerified: emailVerified,
	}, nil
}

// verifyCredentials fetches the user by email and verifies the password hash.
func (s *UserService) verifyCredentials(ctx context.Context, email, password string) (db.UserAccountProjection, error) {
	user, err := s.queries().GetUserAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.UserAccountProjection{}, apperrors.ErrInvalidCredentials
		}
		return db.UserAccountProjection{}, fmt.Errorf("failed to get user: %w", err)
	}
	if err := security.VerifyPassword(password, user.PasswordHash, user.PasswordSalt); err != nil {
		return db.UserAccountProjection{}, apperrors.ErrInvalidCredentials
	}
	return user, nil
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

func (s *UserService) Register(ctx context.Context, input RegisterInput) error {
	if !config.AuthRegistrationEnabled() {
		return apperrors.ErrRegistrationDisabled
	}

	if input.Email == "" || input.Password == "" || input.Name == "" {
		return apperrors.ErrInvalidInput
	}
	if len(input.Password) < 8 {
		return apperrors.ErrInvalidInput
	}

	q := s.queries()
	exists, err := q.UserAccountEmailExists(ctx, input.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if exists {
		return apperrors.ErrEmailAlreadyInUse
	}

	passwordHash, passwordSalt, err := security.HashPassword(input.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	emailID := uuid.New()
	agg := &Aggregate{}
	if err := agg.Register(
		userID, input.Name, input.Email, emailID, passwordHash, passwordSalt,
		true, time.Now(),
	); err != nil {
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

	token, err := authjwt.GenerateResetToken(user.ID, primaryEmail.Email)
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

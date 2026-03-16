package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"log/slog"
	"time"

	"github.com/godatei/datei/internal/authjwt"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/security"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// UpdateUser implements [StrictServerInterface].
func (s *server) UpdateUser(ctx context.Context, request UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil {
		return UpdateUser400Response{}, nil
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	now := time.Now()
	if request.Body.Name != nil {
		// Password-reset tokens can only change the password, not the name
		if authInfo.PasswordReset {
			return UpdateUser400Response{}, nil
		}
		if err := agg.ChangeName(*request.Body.Name, now); err != nil {
			return UpdateUser400Response{}, nil
		}
	}
	if request.Body.Password != nil {
		if len(*request.Body.Password) < 8 {
			return UpdateUser400Response{}, nil
		}

		// Require current password for in-session password changes (not password-reset flow)
		if !authInfo.PasswordReset {
			if request.Body.CurrentPassword == nil || *request.Body.CurrentPassword == "" {
				return UpdateUser400Response{}, nil
			}
			q := db.New(s.pool)
			user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
			if err != nil {
				slog.Error("failed to get user for password verification", "error", err)
				return nil, fmt.Errorf("failed to get user: %w", err)
			}
			if err := security.VerifyPassword(*request.Body.CurrentPassword, user.PasswordHash, user.PasswordSalt); err != nil {
				return UpdateUser401Response{}, nil
			}
		}

		hash, salt, err := security.HashPassword(*request.Body.Password)
		if err != nil {
			slog.Error("failed to hash password", "error", err)
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		if err := agg.ChangePassword(hash, salt, now); err != nil {
			return UpdateUser400Response{}, nil
		}
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
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

	q := db.New(s.pool)
	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get primary email", "error", err)
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.ChangeEmail(primaryEmail.Email, string(request.Body.Email), time.Now()); err != nil {
		return UpdateUserEmail400Response{}, nil
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	if config.AuthEmailVerificationRequired() {
		s.sendVerificationEmail(ctx, authInfo.UserID, string(request.Body.Email))
	}

	return UpdateUserEmail204Response{}, nil
}

// RequestEmailVerification implements [StrictServerInterface].
func (s *server) RequestEmailVerification(
	ctx context.Context, _ RequestEmailVerificationRequestObject,
) (RequestEmailVerificationResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	q := db.New(s.pool)
	email, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get email", "error", err)
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	s.sendVerificationEmail(ctx, authInfo.UserID, email.Email)
	return RequestEmailVerification204Response{}, nil
}

// ConfirmEmailVerification implements [StrictServerInterface].
func (s *server) ConfirmEmailVerification(
	ctx context.Context, _ ConfirmEmailVerificationRequestObject,
) (ConfirmEmailVerificationResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	// Require a verification token (email_verified=true, not a reset token)
	if !authInfo.EmailVerified || authInfo.PasswordReset {
		return ConfirmEmailVerification400Response{}, nil
	}

	q := db.New(s.pool)
	email, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get primary email", "error", err)
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}

	// Token email must match current primary email
	if authInfo.Email != email.Email {
		return ConfirmEmailVerification400Response{}, nil
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.VerifyEmail(time.Now()); err != nil {
		return ConfirmEmailVerification400Response{}, nil
	}

	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save user", "error", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return ConfirmEmailVerification204Response{}, nil
}

// SetupMFA implements [StrictServerInterface].
func (s *server) SetupMFA(ctx context.Context, _ SetupMFARequestObject) (SetupMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	q := db.New(s.pool)
	user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.MfaEnabled {
		return SetupMFA400Response{}, nil
	}

	email, err := q.GetPrimaryEmailForUser(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get email", "error", err)
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Datei",
		AccountName: email.Email,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		slog.Error("failed to generate TOTP key", "error", err)
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.InitiateMFASetup(key.Secret(), time.Now()); err != nil {
		return SetupMFA400Response{}, nil
	}
	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to save MFA secret", "error", err)
		return nil, fmt.Errorf("failed to save MFA secret: %w", err)
	}

	img, err := key.Image(200, 200)
	if err != nil {
		slog.Error("failed to generate QR code", "error", err)
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		slog.Error("failed to encode QR code", "error", err)
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	return SetupMFA200JSONResponse(api.SetupMFAResponse{
		Secret:    key.Secret(),
		QrCodeUrl: qrCode,
	}), nil
}

// EnableMFA implements [StrictServerInterface].
func (s *server) EnableMFA(ctx context.Context, request EnableMFARequestObject) (EnableMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil {
		return EnableMFA400Response{}, nil
	}

	q := db.New(s.pool)
	user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user.MfaEnabled {
		return EnableMFA400Response{}, nil
	}
	if user.MfaSecret == nil {
		return EnableMFA400Response{}, nil
	}
	if !totp.Validate(request.Body.Code, *user.MfaSecret) {
		return EnableMFA400Response{}, nil
	}

	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		slog.Error("failed to generate recovery codes", "error", err)
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	hashedCodes := make([]events.HashedRecoveryCode, len(codes))
	for i, code := range codes {
		hash, salt, err := security.HashRecoveryCode(code)
		if err != nil {
			slog.Error("failed to hash recovery code", "error", err)
			return nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		hashedCodes[i] = events.HashedRecoveryCode{
			ID:       uuid.New(),
			CodeHash: hash,
			CodeSalt: salt,
		}
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.EnableMFA(hashedCodes, time.Now()); err != nil {
		return EnableMFA400Response{}, nil
	}
	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to enable MFA", "error", err)
		return nil, fmt.Errorf("failed to enable MFA: %w", err)
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}
	return EnableMFA200JSONResponse(api.EnableMFAResponse{RecoveryCodes: formattedCodes}), nil
}

// DisableMFA implements [StrictServerInterface].
func (s *server) DisableMFA(ctx context.Context, request DisableMFARequestObject) (DisableMFAResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	if request.Body == nil {
		return DisableMFA400Response{}, nil
	}

	q := db.New(s.pool)
	user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !user.MfaEnabled {
		return DisableMFA400Response{}, nil
	}
	if err := security.VerifyPassword(request.Body.Password, user.PasswordHash, user.PasswordSalt); err != nil {
		return DisableMFA401Response{}, nil
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.DisableMFA(time.Now()); err != nil {
		return DisableMFA400Response{}, nil
	}
	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to disable MFA", "error", err)
		return nil, fmt.Errorf("failed to disable MFA: %w", err)
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

	q := db.New(s.pool)
	user, err := q.GetUserAccountByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to get user", "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !user.MfaEnabled {
		return RegenerateMFARecoveryCodes400Response{}, nil
	}
	if err := security.VerifyPassword(request.Body.Password, user.PasswordHash, user.PasswordSalt); err != nil {
		return RegenerateMFARecoveryCodes401Response{}, nil
	}

	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		slog.Error("failed to generate recovery codes", "error", err)
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	hashedCodes := make([]events.HashedRecoveryCode, len(codes))
	for i, code := range codes {
		hash, salt, err := security.HashRecoveryCode(code)
		if err != nil {
			slog.Error("failed to hash recovery code", "error", err)
			return nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		hashedCodes[i] = events.HashedRecoveryCode{
			ID:       uuid.New(),
			CodeHash: hash,
			CodeSalt: salt,
		}
	}

	agg, err := s.userRepo.LoadByID(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to load user", "error", err)
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.RegenerateRecoveryCodes(hashedCodes, time.Now()); err != nil {
		return RegenerateMFARecoveryCodes400Response{}, nil
	}
	if err := s.userRepo.Save(ctx, agg); err != nil {
		slog.Error("failed to regenerate recovery codes", "error", err)
		return nil, fmt.Errorf("failed to regenerate recovery codes: %w", err)
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}
	return RegenerateMFARecoveryCodes200JSONResponse(api.RegenerateMFARecoveryCodesResponse{
		RecoveryCodes: formattedCodes,
	}), nil
}

// GetMFARecoveryCodesStatus implements [StrictServerInterface].
func (s *server) GetMFARecoveryCodesStatus(
	ctx context.Context, _ GetMFARecoveryCodesStatusRequestObject,
) (GetMFARecoveryCodesStatusResponseObject, error) {
	authInfo := authn.RequireContext(ctx)

	q := db.New(s.pool)
	count, err := q.CountUnusedMFARecoveryCodes(ctx, authInfo.UserID)
	if err != nil {
		slog.Error("failed to count recovery codes", "error", err)
		return nil, fmt.Errorf("failed to count recovery codes: %w", err)
	}

	return GetMFARecoveryCodesStatus200JSONResponse(api.MFARecoveryCodesStatusResponse{
		RemainingCodes: int(count),
	}), nil
}

func (s *server) sendVerificationEmail(ctx context.Context, userID uuid.UUID, email string) {
	_, token, err := authjwt.GenerateVerificationToken(userID, email)
	if err != nil {
		slog.Error("failed to generate verification token", "error", err)
		return
	}
	verifyURL := fmt.Sprintf("%s/verify?jwt=%s", config.ServerHost(), token)
	if err := s.mailer.Send(ctx, mailer.EmailVerificationEmail(email, verifyURL)); err != nil {
		slog.Warn("failed to send verification email", "error", err)
	}
}

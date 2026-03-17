package users

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"time"

	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/security"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type SetupMFAOutput struct {
	Secret    string
	QrCodeUrl string
}

func (s *UserService) SetupMFA(ctx context.Context, userID uuid.UUID) (*SetupMFAOutput, error) {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.MfaEnabled {
		return nil, dateierrors.ErrMFAAlreadyEnabled
	}

	email, err := q.GetPrimaryEmailForUser(ctx, userID)
	if err != nil {
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
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.InitiateMFASetup(key.Secret(), time.Now()); err != nil {
		return nil, dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, fmt.Errorf("failed to save MFA secret: %w", err)
	}

	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	return &SetupMFAOutput{
		Secret:    key.Secret(),
		QrCodeUrl: qrCode,
	}, nil
}

type EnableMFAInput struct {
	UserID uuid.UUID
	Code   string
}

type EnableMFAOutput struct {
	RecoveryCodes []string
}

func (s *UserService) EnableMFA(ctx context.Context, input EnableMFAInput) (*EnableMFAOutput, error) {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user.MfaEnabled {
		return nil, dateierrors.ErrMFAAlreadyEnabled
	}
	if user.MfaSecret == nil {
		return nil, dateierrors.ErrMFANotSetUp
	}
	if !totp.Validate(input.Code, *user.MfaSecret) {
		return nil, dateierrors.ErrMFAInvalidCode
	}

	codes, hashedCodes, err := generateAndHashRecoveryCodes()
	if err != nil {
		return nil, err
	}

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.EnableMFA(hashedCodes, time.Now()); err != nil {
		return nil, dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, fmt.Errorf("failed to enable MFA: %w", err)
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}
	return &EnableMFAOutput{RecoveryCodes: formattedCodes}, nil
}

type DisableMFAInput struct {
	UserID   uuid.UUID
	Password string
}

func (s *UserService) DisableMFA(ctx context.Context, input DisableMFAInput) error {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if !user.MfaEnabled {
		return dateierrors.ErrMFANotEnabled
	}
	if err := security.VerifyPassword(input.Password, user.PasswordHash, user.PasswordSalt); err != nil {
		return dateierrors.ErrInvalidCredentials
	}

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.DisableMFA(time.Now()); err != nil {
		return dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to disable MFA: %w", err)
	}

	return nil
}

type RegenerateRecoveryCodesInput struct {
	UserID   uuid.UUID
	Password string
}

func (s *UserService) RegenerateMFARecoveryCodes(
	ctx context.Context, input RegenerateRecoveryCodesInput,
) ([]string, error) {
	q := s.queries()
	user, err := q.GetUserAccountByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !user.MfaEnabled {
		return nil, dateierrors.ErrMFANotEnabled
	}
	if err := security.VerifyPassword(input.Password, user.PasswordHash, user.PasswordSalt); err != nil {
		return nil, dateierrors.ErrInvalidCredentials
	}

	codes, hashedCodes, err := generateAndHashRecoveryCodes()
	if err != nil {
		return nil, err
	}

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if err := agg.RegenerateRecoveryCodes(hashedCodes, time.Now()); err != nil {
		return nil, dateierrors.ErrInvalidInput
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, fmt.Errorf("failed to regenerate recovery codes: %w", err)
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}
	return formattedCodes, nil
}

func (s *UserService) GetMFARecoveryCodesStatus(ctx context.Context, userID uuid.UUID) (int, error) {
	q := s.queries()
	count, err := q.CountUnusedMFARecoveryCodes(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count recovery codes: %w", err)
	}
	return int(count), nil
}

func generateAndHashRecoveryCodes() ([]string, []events.HashedRecoveryCode, error) {
	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	hashedCodes := make([]events.HashedRecoveryCode, len(codes))
	for i, code := range codes {
		hash, salt, err := security.HashRecoveryCode(code)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		hashedCodes[i] = events.HashedRecoveryCode{
			ID:       uuid.New(),
			CodeHash: hash,
			CodeSalt: salt,
		}
	}

	return codes, hashedCodes, nil
}

package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/security"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateAccessTokenInput struct {
	UserID    uuid.UUID
	Label     string
	ExpiresAt *time.Time
}

// CreateAccessToken mints a new personal access token. The returned response
// carries the plaintext token, which is shown to the caller exactly once; only
// its hash is persisted.
func (s *UserService) CreateAccessToken(
	ctx context.Context, input CreateAccessTokenInput,
) (*api.CreatePersonalAccessTokenResponse, error) {
	plaintext, hash, err := security.GenerateAccessToken()
	if err != nil {
		return nil, err
	}

	tokenID := uuid.New()
	now := time.Now()

	agg, err := s.repository.LoadByID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.CreateAccessToken(tokenID, input.Label, hash, input.ExpiresAt, now); err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrInvalidInput, err)
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	var label *string
	if input.Label != "" {
		label = &input.Label
	}

	return &api.CreatePersonalAccessTokenResponse{
		Token: plaintext,
		AccessToken: api.PersonalAccessToken{
			Id:        tokenID,
			Label:     label,
			ExpiresAt: input.ExpiresAt,
			CreatedAt: now,
		},
	}, nil
}

func (s *UserService) ListAccessTokens(ctx context.Context, userID uuid.UUID) ([]api.PersonalAccessToken, error) {
	rows, err := s.queries().ListAccessTokensForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list access tokens: %w", err)
	}
	return MapAccessTokenProjectionSliceToAPI(rows), nil
}

func (s *UserService) RevokeAccessToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	token, err := s.queries().GetAccessTokenByID(ctx, db.GetAccessTokenByIDParams{
		ID:            tokenID,
		UserAccountID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrNotFound
		}
		return fmt.Errorf("failed to get access token: %w", err)
	}
	if token.RevokedAt != nil {
		return apperrors.ErrNotFound
	}

	agg, err := s.repository.LoadByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	if err := agg.RevokeAccessToken(tokenID, time.Now()); err != nil {
		return apperrors.ErrInvalidInput
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

// AuthenticatedUser is the principal resolved from a credential: the account
// plus the primary email tied to it.
type AuthenticatedUser struct {
	UserAccount UserAccount
	Email       string
}

// AuthenticateAccessToken resolves a presented personal access token to its
// owning account. It returns apperrors.ErrInvalidCredentials when the value is
// not a token, does not match, or is revoked/expired (the underlying query
// already excludes revoked/expired tokens and archived accounts).
func (s *UserService) AuthenticateAccessToken(
	ctx context.Context, presented string,
) (*AuthenticatedUser, error) {
	hash, ok := security.HashPresentedAccessToken(presented)
	if !ok {
		return nil, apperrors.ErrInvalidCredentials
	}

	q := s.queries()
	row, err := q.GetActiveAccessTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to look up access token: %w", err)
	}

	primaryEmail, err := q.GetPrimaryEmailForUser(ctx, row.UserAccountProjection.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary email: %w", err)
	}

	return &AuthenticatedUser{
		UserAccount: userFromProjection(row.UserAccountProjection),
		Email:       primaryEmail.Email,
	}, nil
}

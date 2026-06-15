package link

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles the owner-facing plane of a link: creating, listing,
// updating, and revoking links for the authenticated user. The viewer-facing
// plane (unlock, public list/download) lives in PublicService.
type Service struct {
	db         *pgxpool.Pool
	repository Repository
}

func NewService(
	pool *pgxpool.Pool,
	repository Repository,
) *Service {
	return &Service{
		db:         pool,
		repository: repository,
	}
}

// generateKey returns a 12-byte random key encoded as base64-url (16 ASCII
// characters), suitable for use as a URL slug. 96 bits of entropy keeps keys
// unguessable while keeping share URLs reasonably short.
func generateKey() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

type CreateLinkInput struct {
	Name      string
	ExpiresAt *time.Time
	Code      *string
	FileIDs   []uuid.UUID
}

func (s *Service) CreateLink(ctx context.Context, input CreateLinkInput) (*api.LinkDetail, error) {
	userID := authn.RequireCurrentUser(ctx).ID

	input.Code = normalizeOptionalCode(input.Code)

	queries := db.New(s.db)
	if len(input.FileIDs) > 0 {
		count, err := queries.CountUntrashedFileByIDs(ctx, input.FileIDs)
		if err != nil {
			return nil, err
		}
		if int(count) != len(input.FileIDs) {
			return nil, apperrors.ErrInvalidInput
		}
	}

	key, err := generateKey()
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	now := time.Now()

	agg := &Aggregate{}
	if err := agg.Create(id, userID, input.Name, key, input.Code, input.ExpiresAt, input.FileIDs, now); err != nil {
		return nil, err
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.aggregateToLinkDetail(ctx, agg)
}

type ListLinksInput struct {
	// Status is "active", "expired", "revoked", or "" to return all.
	Status string
	Limit  int
	Offset int
}

type ListLinksOutput struct {
	Items []api.Link
	Total int
}

func (s *Service) ListLinks(ctx context.Context, input ListLinksInput) (*ListLinksOutput, error) {
	userID := authn.RequireCurrentUser(ctx).ID
	queries := db.New(s.db)

	limit := int32(input.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int32(max(input.Offset, 0))

	total, err := queries.CountLinkProjectionsByOwner(ctx, db.CountLinkProjectionsByOwnerParams{
		OwnerID: userID,
		Status:  input.Status,
	})
	if err != nil {
		return nil, err
	}

	projections, err := queries.ListLinkProjectionsByOwner(ctx, db.ListLinkProjectionsByOwnerParams{
		OwnerID: userID,
		Status:  input.Status,
		Lim:     limit,
		Off:     offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]api.Link, 0, len(projections))
	for i := range projections {
		c, err := queries.CountLinkContents(ctx, projections[i].ID)
		if err != nil {
			return nil, err
		}
		mapped := MapProjectionToLink(&projections[i], int(c.FileCount), int(c.FolderCount), int(c.OpenCount))
		if mapped != nil {
			items = append(items, *mapped)
		}
	}

	return &ListLinksOutput{Items: items, Total: int(total)}, nil
}

type UpdateLinkInput struct {
	ID              uuid.UUID
	Name            *string
	ExpiresAt       *time.Time
	ClearExpiration bool
	Code            *string
	ClearCode       bool
}

func (s *Service) UpdateLink(ctx context.Context, input UpdateLinkInput) (*api.LinkDetail, error) {
	agg, err := s.loadOwnedAggregate(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	input.Code = normalizeOptionalCode(input.Code)

	// Build the absolute desired state from the input. Fields the request
	// did not address fall back to the aggregate's current value.
	name := agg.Name
	if input.Name != nil {
		name = *input.Name
	}

	code := agg.Code
	switch {
	case input.ClearCode:
		code = nil
	case input.Code != nil:
		code = input.Code
	}

	expiresAt := agg.ExpiresAt
	switch {
	case input.ClearExpiration:
		expiresAt = nil
	case input.ExpiresAt != nil:
		expiresAt = input.ExpiresAt
	}

	if err := agg.Update(name, code, expiresAt, time.Now()); err != nil {
		return nil, err
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.aggregateToLinkDetail(ctx, agg)
}

func (s *Service) GetLink(ctx context.Context, id uuid.UUID) (*api.LinkDetail, error) {
	agg, err := s.loadOwnedAggregate(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.aggregateToLinkDetail(ctx, agg)
}

func (s *Service) RotateKey(ctx context.Context, id uuid.UUID) (*api.LinkDetail, error) {
	agg, err := s.loadOwnedAggregate(ctx, id)
	if err != nil {
		return nil, err
	}

	key, err := generateKey()
	if err != nil {
		return nil, err
	}

	if err := agg.RotateKey(key, time.Now()); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.aggregateToLinkDetail(ctx, agg)
}

func (s *Service) AddFileToLink(ctx context.Context, linkID, fileID uuid.UUID) (*api.LinkDetail, error) {
	agg, err := s.loadOwnedAggregate(ctx, linkID)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	count, err := queries.CountUntrashedFileByIDs(ctx, []uuid.UUID{fileID})
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, apperrors.ErrInvalidInput
	}

	if err := agg.AddFile(fileID, time.Now()); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.aggregateToLinkDetail(ctx, agg)
}

func (s *Service) RemoveFileFromLink(ctx context.Context, linkID, fileID uuid.UUID) error {
	agg, err := s.loadOwnedAggregate(ctx, linkID)
	if err != nil {
		return err
	}

	if err := agg.RemoveFile(fileID, time.Now()); err != nil {
		return err
	}
	return s.repository.Save(ctx, agg)
}

func (s *Service) RevokeLink(ctx context.Context, id uuid.UUID) error {
	agg, err := s.loadOwnedAggregate(ctx, id)
	if err != nil {
		return err
	}

	if err := agg.Revoke(time.Now()); err != nil {
		return err
	}
	return s.repository.Save(ctx, agg)
}

// ============================================================================
// helpers
// ============================================================================

func (s *Service) loadOwnedAggregate(ctx context.Context, id uuid.UUID) (*Aggregate, error) {
	userID := authn.RequireCurrentUser(ctx).ID
	agg, err := s.repository.LoadByID(ctx, id)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, apperrors.ErrLinkNotFound
		}
		return nil, err
	}
	if agg.OwnerID != userID {
		return nil, apperrors.ErrLinkNotFound
	}
	return agg, nil
}

func (s *Service) aggregateToLinkDetail(ctx context.Context, agg *Aggregate) (*api.LinkDetail, error) {
	queries := db.New(s.db)
	files, err := queries.ListFilesByLink(ctx, agg.ID)
	if err != nil {
		return nil, err
	}
	counts, err := queries.CountLinkContents(ctx, agg.ID)
	if err != nil {
		return nil, err
	}
	return MapAggregateToLinkDetail(
		agg, files, int(counts.FileCount), int(counts.FolderCount), int(counts.OpenCount),
	), nil
}

func normalizeOptionalCode(code *string) *string {
	if code == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*code)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

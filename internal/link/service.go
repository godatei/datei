package link

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db         *pgxpool.Pool
	store      storage.Store
	repository Repository
	dateiSvc   *datei.Service
}

func NewService(
	pool *pgxpool.Pool,
	store storage.Store,
	repository Repository,
	dateiSvc *datei.Service,
) *Service {
	return &Service{
		db:         pool,
		store:      store,
		repository: repository,
		dateiSvc:   dateiSvc,
	}
}

// generateAccessToken returns a 32-byte random token encoded as base64-url
// (43 ASCII characters) suitable for use as a URL slug.
func generateAccessToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// ============================================================================
// Owner-side operations
// ============================================================================

type CreateLinkInput struct {
	Name      string
	ExpiresAt *time.Time
	Code      *string
	DateiIDs  []uuid.UUID
}

func (s *Service) CreateLink(ctx context.Context, input CreateLinkInput) (*api.Link, error) {
	if input.Name == "" {
		return nil, dateierrors.ErrInvalidInput
	}

	userID := authn.RequireContext(ctx).UserID

	queries := db.New(s.db)
	for _, id := range input.DateiIDs {
		if _, err := queries.GetDateiProjectionByID(ctx, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, dateierrors.ErrInvalidInput
			}
			return nil, err
		}
	}

	token, err := generateAccessToken()
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	now := time.Now()

	agg := &Aggregate{}
	if err := agg.Create(id, userID, input.Name, token, input.Code, input.ExpiresAt, input.DateiIDs, now); err != nil {
		return nil, err
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.loadLinkAPI(ctx, id)
}

type ListLinksOutput struct {
	Items []api.Link
	Total int
}

func (s *Service) ListLinks(ctx context.Context) (*ListLinksOutput, error) {
	userID := authn.RequireContext(ctx).UserID
	queries := db.New(s.db)

	projections, err := queries.ListLinkProjectionsByOwner(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]api.Link, 0, len(projections))
	for i := range projections {
		dateien, err := queries.ListDateienByLink(ctx, projections[i].ID)
		if err != nil {
			return nil, err
		}
		counts, err := queries.CountLinkContents(ctx, projections[i].ID)
		if err != nil {
			return nil, err
		}
		mapped := MapProjectionToAPI(&projections[i], dateien, int(counts.FileCount), int(counts.FolderCount))
		if mapped != nil {
			items = append(items, *mapped)
		}
	}

	return &ListLinksOutput{Items: items, Total: len(items)}, nil
}

func (s *Service) GetLink(ctx context.Context, id uuid.UUID) (*api.Link, error) {
	userID := authn.RequireContext(ctx).UserID
	queries := db.New(s.db)

	projection, err := queries.GetLinkProjectionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrLinkNotFound
	} else if err != nil {
		return nil, err
	}
	if projection.OwnerID != userID {
		return nil, dateierrors.ErrLinkNotFound
	}

	return s.linkAPIFromProjection(ctx, &projection)
}

type UpdateLinkInput struct {
	ID              uuid.UUID
	Name            *string
	ExpiresAt       *time.Time
	ClearExpiration bool
	Code            *string
	ClearCode       bool
}

func (s *Service) UpdateLink(ctx context.Context, input UpdateLinkInput) (*api.Link, error) {
	userID := authn.RequireContext(ctx).UserID

	agg, err := s.loadOwnedAggregate(ctx, input.ID, userID)
	if err != nil {
		return nil, err
	}

	// Build the absolute desired state from the input. Fields the request
	// did not address fall back to the aggregate's current value.
	name := agg.Name
	if input.Name != nil && *input.Name != "" {
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

	return s.loadLinkAPI(ctx, agg.ID)
}

func (s *Service) RotateAccessToken(ctx context.Context, id uuid.UUID) (*api.Link, error) {
	userID := authn.RequireContext(ctx).UserID

	agg, err := s.loadOwnedAggregate(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	token, err := generateAccessToken()
	if err != nil {
		return nil, err
	}

	if err := agg.RotateAccessToken(token, time.Now()); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.loadLinkAPI(ctx, agg.ID)
}

func (s *Service) AddDateiToLink(ctx context.Context, linkID, dateiID uuid.UUID) (*api.Link, error) {
	userID := authn.RequireContext(ctx).UserID

	agg, err := s.loadOwnedAggregate(ctx, linkID, userID)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	if _, err := queries.GetDateiProjectionByID(ctx, dateiID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dateierrors.ErrInvalidInput
		}
		return nil, err
	}

	if _, exists := agg.DateiIDs[dateiID]; exists {
		return nil, dateierrors.ErrLinkDateiAlreadyAdded
	}

	if err := agg.AddDatei(dateiID, time.Now()); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.loadLinkAPI(ctx, agg.ID)
}

func (s *Service) RemoveDateiFromLink(ctx context.Context, linkID, dateiID uuid.UUID) error {
	userID := authn.RequireContext(ctx).UserID

	agg, err := s.loadOwnedAggregate(ctx, linkID, userID)
	if err != nil {
		return err
	}

	if err := agg.RemoveDatei(dateiID, time.Now()); err != nil {
		return err
	}
	return s.repository.Save(ctx, agg)
}

func (s *Service) RevokeLink(ctx context.Context, id uuid.UUID) error {
	userID := authn.RequireContext(ctx).UserID

	agg, err := s.loadOwnedAggregate(ctx, id, userID)
	if err != nil {
		return err
	}

	if err := agg.Revoke(time.Now()); err != nil {
		return err
	}
	return s.repository.Save(ctx, agg)
}

// ============================================================================
// Public-side operations
// ============================================================================

// verifyLinkAccess looks up a link by access token and validates that the
// caller is allowed to read its contents. Returns the projection on success.
func (s *Service) verifyLinkAccess(
	ctx context.Context,
	accessToken string,
	providedCode string,
) (*db.LinkProjection, error) {
	queries := db.New(s.db)

	projection, err := queries.GetLinkProjectionByAccessToken(ctx, accessToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrLinkNotFound
	} else if err != nil {
		return nil, err
	}
	if projection.RevokedAt != nil {
		return nil, dateierrors.ErrLinkRevoked
	}
	if projection.ExpiresAt != nil && projection.ExpiresAt.Before(time.Now()) {
		return nil, dateierrors.ErrLinkExpired
	}
	if projection.Code != nil {
		if providedCode == "" {
			return nil, dateierrors.ErrLinkCodeRequired
		}
		if providedCode != *projection.Code {
			return nil, dateierrors.ErrLinkCodeInvalid
		}
	}
	return &projection, nil
}

// ListPublicLinkDateienOutput holds a public link's display name, owner name,
// and the dateien accessible at the requested level.
type ListPublicLinkDateienOutput struct {
	Name      string
	OwnerName string
	Items     []api.Datei
}

// ListPublicLinkDateien returns the dateien visible to a public viewer.
// When parentID is nil, the link's top-level shared dateien are returned;
// otherwise the children of the parent are returned (the parent must be in
// the link's shared scope).
func (s *Service) ListPublicLinkDateien(
	ctx context.Context,
	accessToken string,
	parentID *uuid.UUID,
	code string,
) (*ListPublicLinkDateienOutput, error) {
	projection, err := s.verifyLinkAccess(ctx, accessToken, code)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	owner, err := queries.GetUserAccountByID(ctx, projection.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load link owner: %w", err)
	}

	if parentID == nil {
		dateien, err := queries.ListDateienByLink(ctx, projection.ID)
		if err != nil {
			return nil, err
		}
		return &ListPublicLinkDateienOutput{
			Name:      projection.Name,
			OwnerName: owner.Name,
			Items:     datei.MapProjectionSliceToAPI(dateien),
		}, nil
	}

	inScope, err := queries.IsDateiInLinkScope(ctx, db.IsDateiInLinkScopeParams{
		LinkID: projection.ID,
		ID:     *parentID,
	})
	if err != nil {
		return nil, err
	}
	if !inScope {
		return nil, dateierrors.ErrLinkDateiNotShared
	}

	children, err := queries.ListDateiProjectionsByParent(ctx, parentID)
	if err != nil {
		return nil, err
	}
	return &ListPublicLinkDateienOutput{
		Name:      projection.Name,
		OwnerName: owner.Name,
		Items:     datei.MapProjectionSliceToAPI(children),
	}, nil
}

func (s *Service) DownloadPublicLinkDatei(
	ctx context.Context,
	accessToken string,
	dateiID uuid.UUID,
	code string,
) (*datei.DownloadDateiOutput, error) {
	projection, err := s.verifyLinkAccess(ctx, accessToken, code)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	inScope, err := queries.IsDateiInLinkScope(ctx, db.IsDateiInLinkScopeParams{
		LinkID: projection.ID,
		ID:     dateiID,
	})
	if err != nil {
		return nil, err
	}
	if !inScope {
		return nil, dateierrors.ErrLinkDateiNotShared
	}

	return s.dateiSvc.DownloadDatei(ctx, dateiID)
}

// ============================================================================
// helpers
// ============================================================================

func (s *Service) loadOwnedAggregate(ctx context.Context, id, userID uuid.UUID) (*Aggregate, error) {
	agg, err := s.repository.LoadByID(ctx, id)
	if err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return nil, dateierrors.ErrLinkNotFound
		}
		return nil, err
	}
	if agg.OwnerID != userID {
		return nil, dateierrors.ErrLinkNotFound
	}
	return agg, nil
}

func (s *Service) loadLinkAPI(ctx context.Context, id uuid.UUID) (*api.Link, error) {
	queries := db.New(s.db)
	projection, err := queries.GetLinkProjectionByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrLinkNotFound
	} else if err != nil {
		return nil, err
	}
	return s.linkAPIFromProjection(ctx, &projection)
}

func (s *Service) linkAPIFromProjection(ctx context.Context, projection *db.LinkProjection) (*api.Link, error) {
	queries := db.New(s.db)
	dateien, err := queries.ListDateienByLink(ctx, projection.ID)
	if err != nil {
		return nil, err
	}
	counts, err := queries.CountLinkContents(ctx, projection.ID)
	if err != nil {
		return nil, err
	}
	return MapProjectionToAPI(projection, dateien, int(counts.FileCount), int(counts.FolderCount)), nil
}

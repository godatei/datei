package link

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db         *pgxpool.Pool
	repository Repository
	dateiSvc   *datei.Service
}

func NewService(
	pool *pgxpool.Pool,
	repository Repository,
	dateiSvc *datei.Service,
) *Service {
	return &Service{
		db:         pool,
		repository: repository,
		dateiSvc:   dateiSvc,
	}
}

// publicListChildrenLimit caps the number of children returned by a single
// public folder listing. Pagination at the public viewer is a future task; for
// now we cap at a generous value to avoid surprises.
const publicListChildrenLimit = 1000

// generateAccessToken returns a 12-byte random token encoded as base64-url
// (16 ASCII characters), suitable for use as a URL slug. 96 bits of entropy
// keeps tokens unguessable while keeping share URLs reasonably short.
func generateAccessToken() (string, error) {
	var b [12]byte
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

func (s *Service) CreateLink(ctx context.Context, input CreateLinkInput) (*api.LinkDetail, error) {
	userID := authn.RequireContext(ctx).UserID

	queries := db.New(s.db)
	if len(input.DateiIDs) > 0 {
		count, err := queries.CountDateiProjectionsByIDs(ctx, input.DateiIDs)
		if err != nil {
			return nil, err
		}
		if int(count) != len(input.DateiIDs) {
			return nil, dateierrors.ErrInvalidInput
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

	return s.aggregateToLinkDetail(ctx, agg)
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
		counts, err := queries.CountLinkContents(ctx, projections[i].ID)
		if err != nil {
			return nil, err
		}
		mapped := MapProjectionToLink(&projections[i], int(counts.FileCount), int(counts.FolderCount))
		if mapped != nil {
			items = append(items, *mapped)
		}
	}

	return &ListLinksOutput{Items: items, Total: len(items)}, nil
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

func (s *Service) RotateAccessToken(ctx context.Context, id uuid.UUID) (*api.LinkDetail, error) {
	agg, err := s.loadOwnedAggregate(ctx, id)
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

	return s.aggregateToLinkDetail(ctx, agg)
}

func (s *Service) AddDateiToLink(ctx context.Context, linkID, dateiID uuid.UUID) (*api.LinkDetail, error) {
	agg, err := s.loadOwnedAggregate(ctx, linkID)
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

	if err := agg.AddDatei(dateiID, time.Now()); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return s.aggregateToLinkDetail(ctx, agg)
}

func (s *Service) RemoveDateiFromLink(ctx context.Context, linkID, dateiID uuid.UUID) error {
	agg, err := s.loadOwnedAggregate(ctx, linkID)
	if err != nil {
		return err
	}

	if err := agg.RemoveDatei(dateiID, time.Now()); err != nil {
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
// Public-side operations
// ============================================================================

// verifyLinkAccess looks up a link by access token and validates that the
// caller is allowed to read its contents. Returns the projection + owner name
// on success.
func (s *Service) verifyLinkAccess(
	ctx context.Context,
	accessToken string,
	providedCode string,
) (*db.GetLinkProjectionByAccessTokenRow, error) {
	queries := db.New(s.db)

	row, err := queries.GetLinkProjectionByAccessToken(ctx, accessToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrLinkNotFound
	} else if err != nil {
		return nil, err
	}
	if row.RevokedAt != nil {
		return nil, dateierrors.ErrLinkRevoked
	}
	if row.ExpiresAt != nil && row.ExpiresAt.Before(time.Now()) {
		return nil, dateierrors.ErrLinkExpired
	}
	if row.Code != nil {
		if subtle.ConstantTimeCompare([]byte(providedCode), []byte(*row.Code)) != 1 {
			return nil, dateierrors.ErrLinkCodeRequired
		}
	}
	return &row, nil
}

// ListPublicLinkDateienOutput holds a public link's display name, owner name,
// expiration, and the dateien accessible at the requested level.
type ListPublicLinkDateienOutput struct {
	Name      string
	OwnerName string
	ExpiresAt *time.Time
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
	row, err := s.verifyLinkAccess(ctx, accessToken, code)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	if parentID == nil {
		dateien, err := queries.ListDateienByLink(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		return &ListPublicLinkDateienOutput{
			Name:      row.Name,
			OwnerName: row.OwnerName,
			ExpiresAt: row.ExpiresAt,
			Items:     datei.MapProjectionSliceToAPI(dateien),
		}, nil
	}

	inScope, err := queries.IsDateiInLinkScope(ctx, db.IsDateiInLinkScopeParams{
		LinkID: row.ID,
		ID:     *parentID,
	})
	if err != nil {
		return nil, err
	}
	if !inScope {
		return nil, dateierrors.ErrLinkDateiNotShared
	}

	// The public viewer renders the full folder contents in one shot — no
	// pagination yet — so request all rows.
	children, err := queries.ListDateiProjectionsByParent(ctx, db.ListDateiProjectionsByParentParams{
		ParentID: parentID,
		Limit:    publicListChildrenLimit,
		Offset:   0,
	})
	if err != nil {
		return nil, err
	}
	return &ListPublicLinkDateienOutput{
		Name:      row.Name,
		OwnerName: row.OwnerName,
		ExpiresAt: row.ExpiresAt,
		Items:     datei.MapProjectionSliceToAPI(children),
	}, nil
}

func (s *Service) DownloadPublicLinkDatei(
	ctx context.Context,
	accessToken string,
	dateiID uuid.UUID,
	code string,
) (*datei.DownloadDateiOutput, error) {
	row, err := s.verifyLinkAccess(ctx, accessToken, code)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	inScope, err := queries.IsDateiInLinkScope(ctx, db.IsDateiInLinkScopeParams{
		LinkID: row.ID,
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

func (s *Service) loadOwnedAggregate(ctx context.Context, id uuid.UUID) (*Aggregate, error) {
	userID := authn.RequireContext(ctx).UserID
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

func (s *Service) aggregateToLinkDetail(ctx context.Context, agg *Aggregate) (*api.LinkDetail, error) {
	queries := db.New(s.db)
	dateien, err := queries.ListDateienByLink(ctx, agg.ID)
	if err != nil {
		return nil, err
	}
	counts, err := queries.CountLinkContents(ctx, agg.ID)
	if err != nil {
		return nil, err
	}
	return MapAggregateToLinkDetail(agg, dateien, int(counts.FileCount), int(counts.FolderCount)), nil
}

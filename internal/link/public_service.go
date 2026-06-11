package link

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PublicService handles the viewer-facing plane of a link: anonymous unlock
// plus list/download calls authenticated by the public-link session JWT.
// Owner-side management lives in Service.
type PublicService struct {
	db         *pgxpool.Pool
	repository Repository
	dateiSvc   *datei.Service
}

func NewPublicService(
	pool *pgxpool.Pool,
	repository Repository,
	dateiSvc *datei.Service,
) *PublicService {
	return &PublicService{
		db:         pool,
		repository: repository,
		dateiSvc:   dateiSvc,
	}
}

// publicListChildrenLimit caps the number of children returned by a single
// public folder listing. Pagination at the public viewer is a future task; for
// now we cap at a generous value to avoid surprises.
const publicListChildrenLimit = 1000

// publicLinkSessionTTL is the default lifetime for the JWT issued by Unlock.
// The actual `exp` is the minimum of (now + this) and the link's own
// expiration, so a near-expiring link issues a correspondingly shorter JWT.
const publicLinkSessionTTL = time.Hour

// UnlockOutput is returned to the viewer after a successful unlock; the token
// is used as a Bearer credential on subsequent list/download calls.
type UnlockOutput struct {
	Token     string
	ExpiresAt time.Time
}

// UnlockPublicLink validates the key + optional code, records a
// LinkOpenedEvent (which the projection handler turns into an atomic
// open_count++), and issues a short-lived JWT bound to the link's UUID.
func (s *PublicService) UnlockPublicLink(ctx context.Context, key, code string) (*UnlockOutput, error) {
	row, err := s.lookupLinkByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if row.Code != nil {
		if subtle.ConstantTimeCompare([]byte(code), []byte(*row.Code)) != 1 {
			return nil, dateierrors.ErrLinkCodeRequired
		}
	}

	now := time.Now()

	agg, err := s.repository.LoadByID(ctx, row.ID)
	if err != nil {
		if errors.Is(err, dateierrors.ErrNotFound) {
			return nil, dateierrors.ErrLinkNotFound
		}
		return nil, err
	}
	if err := agg.RecordOpen(now); err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	exp := now.Add(publicLinkSessionTTL)
	if row.ExpiresAt != nil && row.ExpiresAt.Before(exp) {
		exp = *row.ExpiresAt
	}

	token, err := signSessionToken(row.ID, LinkFingerprint(row.Key, row.Code), now, exp)
	if err != nil {
		return nil, fmt.Errorf("failed to sign public-link token: %w", err)
	}

	return &UnlockOutput{Token: token, ExpiresAt: exp}, nil
}

// ListPublicLinkDateienOutput holds a public link's display name, owner name,
// expiration, and the dateien accessible at the requested level.
type ListPublicLinkDateienOutput struct {
	Name      string
	OwnerName string
	ExpiresAt *time.Time
	Items     []api.Datei
}

// ListPublicLinkDateien returns the dateien visible to a public viewer after
// unlock. The session (link ID + token-bound key) is read from the JWT context
// — the caller does not pass a key or code.
func (s *PublicService) ListPublicLinkDateien(
	ctx context.Context,
	session SessionClaims,
	parentID *uuid.UUID,
) (*ListPublicLinkDateienOutput, error) {
	row, err := s.verifyLinkActive(ctx, session.LinkID, session.Fingerprint)
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

	// Existence check before scope check so a missing parent returns 404
	// (distinct from "exists but not shared", which returns 403).
	if _, err := queries.GetDateiProjectionByID(ctx, *parentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dateierrors.ErrNotFound
		}
		return nil, err
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

func (s *PublicService) DownloadPublicLinkDatei(
	ctx context.Context,
	session SessionClaims,
	dateiID uuid.UUID,
) (*datei.DownloadDateiOutput, error) {
	row, err := s.verifyLinkActive(ctx, session.LinkID, session.Fingerprint)
	if err != nil {
		return nil, err
	}

	queries := db.New(s.db)
	// Existence check before scope check so a missing datei returns 404
	// (distinct from "exists but not shared", which returns 403).
	if _, err := queries.GetDateiProjectionByID(ctx, dateiID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dateierrors.ErrNotFound
		}
		return nil, err
	}

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

// lookupLinkByKey returns the projection row for unlock; it checks the link is
// not revoked and not past its expiration. Code is verified by the caller.
func (s *PublicService) lookupLinkByKey(ctx context.Context, key string) (*db.LinkProjection, error) {
	queries := db.New(s.db)
	row, err := queries.GetLinkProjectionByKey(ctx, key)
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
	return &row, nil
}

// verifyLinkActive re-validates the link's current state on every list/download
// call so that revoke, expire, key rotation, and code change take effect within
// JWT lifetime. The token's fingerprint is recomputed from the projection's
// current (key, code) and compared — a mismatch means the link's secret
// material has changed since unlock and the session must be re-established.
// Returns the join row that includes the owner's display name.
func (s *PublicService) verifyLinkActive(
	ctx context.Context, linkID uuid.UUID, tokenFingerprint string,
) (*db.GetLinkProjectionWithOwnerByIDRow, error) {
	queries := db.New(s.db)
	row, err := queries.GetLinkProjectionWithOwnerByID(ctx, linkID)
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
	currentFingerprint := LinkFingerprint(row.Key, row.Code)
	if subtle.ConstantTimeCompare([]byte(tokenFingerprint), []byte(currentFingerprint)) != 1 {
		return nil, dateierrors.ErrLinkUnauthorized
	}
	return &row, nil
}

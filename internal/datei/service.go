package datei

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/dateierrors"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/ocr"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/internal/thumbnail"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db         *pgxpool.Pool
	store      storage.Store
	repository Repository
	ocrClient  *ocr.Client
}

func NewService(
	db *pgxpool.Pool,
	store storage.Store,
	repository Repository,
	ocrClient *ocr.Client,
) *Service {
	return &Service{
		db:         db,
		store:      store,
		repository: repository,
		ocrClient:  ocrClient,
	}
}

func isOCRable(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") || contentType == "application/pdf"
}

func (s *Service) startOCR(ctx context.Context, dateiID uuid.UUID, s3Key, checksum, contentType string) {
	if s.ocrClient == nil || !isOCRable(contentType) {
		return
	}
	go func() {
		ctx := context.WithoutCancel(ctx)
		reader, err := s.store.GetObject(ctx, s3Key)
		if err != nil {
			slog.Warn("ocr: failed to get object from store", "error", err, "dateiID", dateiID)
			return
		}
		defer reader.Close()
		text, err := s.ocrClient.ExtractText(ctx, reader, contentType)
		if err != nil {
			slog.Warn("ocr: extraction failed", "error", err, "dateiID", dateiID)
			return
		}
		queries := db.New(s.db)
		if err := queries.UpdateDateiProjectionContentMD(ctx, db.UpdateDateiProjectionContentMDParams{
			ContentMd: &text,
			ID:        dateiID,
			Checksum:  &checksum,
		}); err != nil {
			slog.Warn("ocr: failed to update projection", "error", err, "dateiID", dateiID)
		}
	}()
}

// ListDateiInput contains parameters for listing datei records
type ListDateiInput struct {
	ParentID *uuid.UUID
	Limit    int
	Offset   int
}

// ListDateiOutput contains the response for listing datei records
type ListDateiOutput struct {
	Items []api.Datei
	Total int
}

// ListDatei retrieves all datei records with pagination
func (s *Service) ListDatei(ctx context.Context, input ListDateiInput) (*ListDateiOutput, error) {
	queries := db.New(s.db)

	limit := int32(input.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int32(max(input.Offset, 0))

	var projections []db.DateiProjection
	var total int64
	var err error

	if input.ParentID != nil {
		total, err = queries.CountDateiProjectionsByParent(ctx, input.ParentID)
		if err != nil {
			return nil, err
		}
		projections, err = queries.ListDateiProjectionsByParent(ctx, db.ListDateiProjectionsByParentParams{
			ParentID: input.ParentID,
			Limit:    limit,
			Offset:   offset,
		})
	} else {
		total, err = queries.CountRootDateiProjections(ctx)
		if err != nil {
			return nil, err
		}
		projections, err = queries.ListRootDateiProjections(ctx, db.ListRootDateiProjectionsParams{
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		return nil, err
	}

	return &ListDateiOutput{
		Items: MapProjectionSliceToAPI(projections),
		Total: int(total),
	}, nil
}

// CreateDateiInput contains parameters for creating a datei
type CreateDateiInput struct {
	ParentID    *uuid.UUID
	Reader      io.Reader
	FileName    string
	ContentType string
}

// CreateDatei creates a new datei record with optional file upload
func (s *Service) CreateDatei(ctx context.Context, input CreateDateiInput) (*api.Datei, error) {
	if input.ParentID != nil {
		queries := db.New(s.db)
		parent, err := queries.GetDateiProjectionByID(ctx, *input.ParentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dateierrors.ErrParentNotFound
		} else if err != nil {
			return nil, err
		}
		if !parent.IsDirectory {
			return nil, dateierrors.ErrParentNotDirectory
		}
		if parent.TrashedAt != nil {
			return nil, dateierrors.ErrParentTrashed
		}
	}

	isDirectory := input.Reader == nil
	id := uuid.New()
	now := time.Now()

	userID := authn.RequireContext(ctx).UserID

	agg := &Aggregate{}
	if err := agg.Create(id, input.ParentID, isDirectory, input.FileName, userID, now); err != nil {
		return nil, err
	}

	if input.Reader != nil && input.FileName != "" {
		putResult, err := s.store.PutObject(ctx, input.Reader, input.FileName, input.ContentType)
		if err != nil {
			return nil, err
		}

		if err = agg.UploadVersion(
			putResult.StorageKey, putResult.Size, putResult.Checksum, input.ContentType, nil, userID, now,
		); err != nil {
			return nil, err
		}
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	if input.Reader != nil && input.FileName != "" && agg.S3Key != nil && agg.Checksum != nil {
		s.startOCR(ctx, agg.ID, *agg.S3Key, *agg.Checksum, input.ContentType)
	}

	return MapAggregateToAPI(agg), nil
}

// DownloadDateiOutput contains the response for downloading a datei
type DownloadDateiOutput struct {
	Reader          io.Reader
	ContentType     string
	ContentLength   int64
	ContentFileName string
}

// DownloadDatei retrieves a file for download
func (s *Service) DownloadDatei(ctx context.Context, dateiID uuid.UUID) (*DownloadDateiOutput, error) {
	queries := db.New(s.db)

	projection, err := queries.GetDateiProjectionByID(ctx, dateiID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if projection.IsDirectory {
		return nil, dateierrors.ErrIsDirectory
	}

	if projection.S3Key == nil || projection.MimeType == nil || projection.Size == nil {
		return nil, dateierrors.ErrNoContent
	}

	reader, err := s.store.GetObject(ctx, *projection.S3Key)
	if err != nil {
		return nil, err
	}

	return &DownloadDateiOutput{
		Reader:          reader,
		ContentType:     *projection.MimeType,
		ContentLength:   *projection.Size,
		ContentFileName: projection.Name,
	}, nil
}

// UpdateDateiInput contains parameters for updating a datei
type UpdateDateiInput struct {
	ID            uuid.UUID
	Name          *string
	MoveRequested bool
	NewParentID   *uuid.UUID
	Reader        io.Reader
	FileName      string
	ContentType   string
}

// UpdateDatei updates a datei record with optional name, move, and/or file.
func (s *Service) UpdateDatei(ctx context.Context, input UpdateDateiInput) (*api.Datei, error) {
	agg, err := s.repository.LoadByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	userID := authn.RequireContext(ctx).UserID

	if input.Name != nil {
		if err := agg.Rename(*input.Name, userID, now); err != nil {
			return nil, err
		}
	}

	if input.MoveRequested {
		if input.NewParentID != nil {
			queries := db.New(s.db)
			if err := s.validateMoveTarget(ctx, queries, input.ID, *input.NewParentID, agg.IsDirectory); err != nil {
				return nil, err
			}
		}

		if err := agg.Move(input.NewParentID, userID, now); err != nil {
			return nil, err
		}
	}

	if input.Reader != nil && input.FileName != "" {
		putResult, err := s.store.PutObject(ctx, input.Reader, input.FileName, input.ContentType)
		if err != nil {
			return nil, err
		}

		if err = agg.UploadVersion(
			putResult.StorageKey, putResult.Size, putResult.Checksum, input.ContentType, nil, userID, now,
		); err != nil {
			return nil, err
		}
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	if input.Reader != nil && input.FileName != "" && agg.S3Key != nil && agg.Checksum != nil {
		s.startOCR(ctx, agg.ID, *agg.S3Key, *agg.Checksum, input.ContentType)
	}

	return MapAggregateToAPI(agg), nil
}

// GetThumbnailOutput contains the response for fetching a thumbnail.
type GetThumbnailOutput struct {
	Body          io.ReadCloser
	ContentLength int64
	ETag          string
}

// GetThumbnail returns a JPEG thumbnail for the given datei, generating and caching it on first call.
// If ifNoneMatch equals the current checksum, returns ErrNotModified to allow a 304 response.
func (s *Service) GetThumbnail(
	ctx context.Context, dateiID uuid.UUID, ifNoneMatch string,
) (*GetThumbnailOutput, error) {
	queries := db.New(s.db)

	projection, err := queries.GetDateiProjectionByID(ctx, dateiID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if projection.IsDirectory {
		return nil, dateierrors.ErrIsDirectory
	}
	if projection.Checksum == nil {
		return nil, dateierrors.ErrNoContent
	}
	if projection.MimeType == nil || projection.S3Key == nil {
		return nil, dateierrors.ErrNoContent
	}

	if ifNoneMatch == *projection.Checksum {
		return nil, dateierrors.ErrNotModified
	}

	thumbKey := "thumbs/" + *projection.Checksum

	exists, err := s.store.ObjectExists(ctx, thumbKey)
	if err != nil {
		return nil, fmt.Errorf("check thumbnail existence: %w", err)
	}

	if exists {
		rc, err := s.store.GetObject(ctx, thumbKey)
		if err != nil {
			return nil, fmt.Errorf("get cached thumbnail: %w", err)
		}
		return &GetThumbnailOutput{Body: rc, ETag: *projection.Checksum}, nil
	}

	original, err := s.store.GetObject(ctx, *projection.S3Key)
	if err != nil {
		return nil, fmt.Errorf("get original file: %w", err)
	}
	defer original.Close()

	thumbBytes, err := thumbnail.Generate(ctx, original, *projection.MimeType)
	if err != nil {
		return nil, err
	}

	if err := s.store.PutObjectAt(ctx, bytes.NewReader(thumbBytes), thumbKey, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("store thumbnail: %w", err)
	}

	return &GetThumbnailOutput{
		Body:          io.NopCloser(bytes.NewReader(thumbBytes)),
		ContentLength: int64(len(thumbBytes)),
		ETag:          *projection.Checksum,
	}, nil
}

// GetDateiPath returns the ancestor chain up to the given datei (inclusive), root-first.
// The chain is truncated at the first trashed ancestor: that ancestor is included but its own parents are not.
func (s *Service) GetDateiPath(ctx context.Context, dateiID uuid.UUID) ([]api.DateiPathItem, error) {
	queries := db.New(s.db)

	rows, err := queries.GetDateiPath(ctx, dateiID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, dateierrors.ErrNotFound
	}

	path := make([]api.DateiPathItem, len(rows))
	for i, row := range rows {
		path[i] = api.DateiPathItem{Id: row.ID, Name: row.Name}
		if row.TrashedAt != nil {
			path[i].Trashed = new(true)
		}
	}
	return path, nil
}

// FindDateiByPath resolves a slash-split path of segments to a single datei in one query.
// Returns dateierrors.ErrNotFound if any segment along the path does not exist.
func (s *Service) FindDateiByPath(ctx context.Context, segments []string) (*api.Datei, error) {
	proj, err := db.New(s.db).GetDateiProjectionByPath(ctx, segments)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return MapProjectionToAPI(&proj), nil
}

// FindDateiByName finds a non-trashed datei by name within a parent (or at root).
// Returns dateierrors.ErrNotFound if no match exists.
func (s *Service) FindDateiByName(ctx context.Context, parentID *uuid.UUID, name string) (*api.Datei, error) {
	queries := db.New(s.db)
	var proj db.DateiProjection
	var err error
	if parentID == nil {
		proj, err = queries.GetRootDateiProjectionByName(ctx, name)
	} else {
		proj, err = queries.GetDateiProjectionByParentAndName(ctx, db.GetDateiProjectionByParentAndNameParams{
			ParentID: parentID,
			Name:     name,
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, dateierrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return MapProjectionToAPI(&proj), nil
}

// ListDateiChildren returns all non-trashed children of the given parent (or root items if parentID is nil).
func (s *Service) ListDateiChildren(ctx context.Context, parentID *uuid.UUID) ([]api.Datei, error) {
	queries := db.New(s.db)
	var projs []db.DateiProjection
	var err error
	if parentID == nil {
		projs, err = queries.ListRootDateiProjections(ctx, db.ListRootDateiProjectionsParams{
			Limit:  math.MaxInt32,
			Offset: 0,
		})
	} else {
		projs, err = queries.ListDateiProjectionsByParent(ctx, db.ListDateiProjectionsByParentParams{
			ParentID: parentID,
			Limit:    math.MaxInt32,
			Offset:   0,
		})
	}
	if err != nil {
		return nil, err
	}
	return MapProjectionSliceToAPI(projs), nil
}

// ListTrashInput contains parameters for listing trashed datei records.
type ListTrashInput struct {
	Limit  int
	Offset int
}

// ListTrashOutput contains the response for listing trashed datei records.
type ListTrashOutput struct {
	Items []api.TrashedDatei
	Total int
}

// ListTrash retrieves root-level trashed datei records with pagination.
func (s *Service) ListTrash(ctx context.Context, input ListTrashInput) (*ListTrashOutput, error) {
	queries := db.New(s.db)

	limit := int32(input.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int32(max(input.Offset, 0))

	total, err := queries.CountTrashedDatei(ctx)
	if err != nil {
		return nil, err
	}

	projections, err := queries.ListTrashedDatei(ctx, db.ListTrashedDateiParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]api.TrashedDatei, 0, len(projections))
	for i := range projections {
		p := &projections[i]
		var originPath *[]api.DateiPathItem
		if p.ParentID != nil {
			rows, err := queries.GetDateiPathIncludingTrashed(ctx, *p.ParentID)
			if err != nil {
				return nil, fmt.Errorf("get origin path for %s: %w", p.ID, err)
			}
			path := make([]api.DateiPathItem, len(rows))
			for j, row := range rows {
				path[j] = api.DateiPathItem{Id: row.ID, Name: row.Name}
			}
			originPath = &path
		} else {
			empty := []api.DateiPathItem{}
			originPath = &empty
		}
		if mapped := MapProjectionToTrashedDatei(p, originPath); mapped != nil {
			items = append(items, *mapped)
		}
	}

	return &ListTrashOutput{Items: items, Total: int(total)}, nil
}

// ListTrashChildrenInput contains parameters for listing contents of a trashed directory.
type ListTrashChildrenInput struct {
	ParentID uuid.UUID
	Limit    int
	Offset   int
}

// ListTrashChildren lists the direct children of a trashed directory.
func (s *Service) ListTrashChildren(ctx context.Context, input ListTrashChildrenInput) (*ListDateiOutput, error) {
	queries := db.New(s.db)

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := max(input.Offset, 0)

	parent, err := queries.GetDateiProjectionByID(ctx, input.ParentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dateierrors.ErrParentNotFound
		}
		return nil, err
	}

	// A directory is browsable in trash if it is directly trashed or has a trashed ancestor.
	inTrash := parent.TrashedAt != nil
	if !inTrash {
		path, err := queries.GetDateiPath(ctx, input.ParentID)
		if err != nil {
			return nil, err
		}
		for _, row := range path {
			if row.TrashedAt != nil {
				inTrash = true
				break
			}
		}
	}
	if !inTrash {
		return nil, dateierrors.ErrParentNotTrashed
	}

	total, err := queries.CountDateiProjectionsByParent(ctx, &input.ParentID)
	if err != nil {
		return nil, err
	}

	projections, err := queries.ListDateiProjectionsByParent(ctx, db.ListDateiProjectionsByParentParams{
		ParentID: &input.ParentID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	return &ListDateiOutput{
		Items: MapProjectionSliceToAPI(projections),
		Total: int(total),
	}, nil
}

// DeleteDatei soft-deletes a datei record
func (s *Service) DeleteDatei(ctx context.Context, dateiID uuid.UUID) error {
	agg, err := s.repository.LoadByID(ctx, dateiID)
	if err != nil {
		return err
	}

	userID := authn.RequireContext(ctx).UserID
	if err := agg.Trash(userID, time.Now()); err != nil {
		return err
	}

	return s.repository.Save(ctx, agg)
}

type RestoreDateiInput struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
}

// RestoreDatei restores a trashed datei or moves a descendant of a trashed datei to a new parent.
// parentId == nil moves the item to root.
func (s *Service) RestoreDatei(ctx context.Context, input RestoreDateiInput) error {
	queries := db.New(s.db)
	projection, err := queries.GetDateiProjectionByID(ctx, input.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dateierrors.ErrNotFound
	} else if err != nil {
		return err
	}

	userID := authn.RequireContext(ctx).UserID
	now := time.Now()

	if input.ParentID != nil {
		if err := s.validateMoveTarget(ctx, queries, input.ID, *input.ParentID, projection.IsDirectory); err != nil {
			return err
		}
	}

	directlyTrashed := projection.TrashedAt != nil
	if !directlyTrashed {
		if projection.ParentID == nil {
			return dateierrors.ErrNotInTrash
		}
		parentPath, err := queries.GetDateiPath(ctx, *projection.ParentID)
		if err != nil {
			return err
		}
		hasTrashedAncestor := false
		for _, row := range parentPath {
			if row.TrashedAt != nil {
				hasTrashedAncestor = true
				break
			}
		}
		if !hasTrashedAncestor {
			return dateierrors.ErrNotInTrash
		}
	}

	agg, err := s.repository.LoadByID(ctx, input.ID)
	if err != nil {
		return err
	}
	if directlyTrashed {
		if err := agg.Restore(userID, now); err != nil {
			return err
		}
	}
	if err := agg.Move(input.ParentID, userID, now); err != nil {
		return err
	}
	return s.repository.Save(ctx, agg)
}

// validateMoveTarget checks that targetParentID is a valid, accessible (non-trashed) directory,
// and (for directory items) that moving itemID there would not create a cycle.
func (s *Service) validateMoveTarget(
	ctx context.Context,
	queries *db.Queries,
	itemID uuid.UUID,
	targetParentID uuid.UUID,
	itemIsDirectory bool,
) error {
	parent, err := queries.GetDateiProjectionByID(ctx, targetParentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return dateierrors.ErrParentNotFound
	} else if err != nil {
		return err
	}
	if !parent.IsDirectory {
		return dateierrors.ErrParentNotDirectory
	}

	// Walk the target's ancestor path to detect both trashed ancestors and cycles.
	// GetDateiPath includes the target itself and stops above the first trashed node,
	// but includes that node — so any trashed entry means the target is inside trash.
	pathRows, err := queries.GetDateiPath(ctx, targetParentID)
	if err != nil {
		return err
	}
	for _, row := range pathRows {
		if row.TrashedAt != nil {
			return dateierrors.ErrParentTrashed
		}
		if itemIsDirectory && row.ID == itemID {
			return dateierrors.ErrCycleDetected
		}
	}
	return nil
}

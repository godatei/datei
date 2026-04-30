package datei

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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

	var allProjections []db.DateiProjection
	var err error
	if input.ParentID != nil {
		allProjections, err = queries.ListDateiProjectionsByParent(ctx, input.ParentID)
	} else {
		allProjections, err = queries.ListRootDateiProjections(ctx)
	}
	if err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	offset := max(input.Offset, 0)
	total := len(allProjections)
	start := min(offset, len(allProjections))
	end := min(offset+limit, len(allProjections))

	items := MapProjectionSliceToAPI(allProjections[start:end])

	return &ListDateiOutput{
		Items: items,
		Total: total,
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
	ID          uuid.UUID
	Name        *string
	Reader      io.Reader
	FileName    string
	ContentType string
}

// UpdateDatei updates a datei record with optional name and/or file
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

// GetDateiPath returns the ancestor chain from root to the given datei (inclusive), root-first.
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
	}
	return path, nil
}

// ListTrashInput contains parameters for listing trashed datei records.
type ListTrashInput struct {
	ParentID *uuid.UUID
	Limit    int
	Offset   int
}

// ListTrashOutput contains the response for listing trashed datei records.
type ListTrashOutput struct {
	Items []api.TrashedDatei
	Total int
}

// ListTrash retrieves trashed datei records with pagination.
// Without a ParentID it lists root-level trash items and includes origin paths.
// With a ParentID it lists all direct children of that trashed directory.
func (s *Service) ListTrash(ctx context.Context, input ListTrashInput) (*ListTrashOutput, error) {
	queries := db.New(s.db)

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := max(input.Offset, 0)

	if input.ParentID != nil {
		projections, err := queries.ListDateiProjectionsByParent(ctx, input.ParentID)
		if err != nil {
			return nil, err
		}
		total := len(projections)
		start := min(offset, total)
		end := min(offset+limit, total)
		items := make([]api.TrashedDatei, 0, end-start)
		for i := range projections[start:end] {
			if mapped := MapProjectionToTrashedDatei(&projections[start+i], nil); mapped != nil {
				items = append(items, *mapped)
			}
		}
		return &ListTrashOutput{Items: items, Total: total}, nil
	}

	projections, err := queries.ListTrashedDatei(ctx)
	if err != nil {
		return nil, err
	}
	total := len(projections)
	start := min(offset, total)
	end := min(offset+limit, total)

	items := make([]api.TrashedDatei, 0, end-start)
	for i := range projections[start:end] {
		p := &projections[start+i]
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

	return &ListTrashOutput{Items: items, Total: total}, nil
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

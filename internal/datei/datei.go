package datei

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/mapping"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DateiService struct {
	db         *pgxpool.Pool
	store      storage.Store
	repository aggregate.DateiRepository
}

func NewDateiService(
	db *pgxpool.Pool,
	store storage.Store,
	repository aggregate.DateiRepository,
) *DateiService {
	return &DateiService{
		db:         db,
		store:      store,
		repository: repository,
	}
}

// ListDateiInput contains parameters for listing datei records
type ListDateiInput struct {
	Limit  int
	Offset int
}

// ListDateiOutput contains the response for listing datei records
type ListDateiOutput struct {
	Items []api.Datei
	Total int
}

// ListDatei retrieves all datei records with pagination
func (s *DateiService) ListDatei(ctx context.Context, input ListDateiInput) (*ListDateiOutput, error) {
	queries := db.New(s.db)

	allProjections, err := queries.ListDateiProjections(ctx)
	if err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	total := len(allProjections)
	start := min(offset, len(allProjections))
	end := min(offset+limit, len(allProjections))

	items := mapping.MapDateiProjectionSliceToAPI(allProjections[start:end])

	return &ListDateiOutput{
		Items: items,
		Total: total,
	}, nil
}

// CreateDateiInput contains parameters for creating a datei
type CreateDateiInput struct {
	Name        string
	Reader      io.Reader
	FileName    string
	ContentType string
}

// CreateDatei creates a new datei record with optional file upload
func (s *DateiService) CreateDatei(ctx context.Context, input CreateDateiInput) (*api.Datei, error) {
	isDirectory := input.Reader == nil
	id := uuid.New()
	now := time.Now()

	agg := &aggregate.DateiAggregate{}
	if err := agg.Create(id, nil, isDirectory, input.Name, uuid.Nil, now); err != nil {
		return nil, err
	}

	if input.Reader != nil && input.FileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, input.Reader, input.ContentType)
		if err != nil {
			return nil, err
		}

		if err = agg.UploadVersion(
			hash, fileSize, hash, input.ContentType, nil, uuid.Nil, now,
		); err != nil {
			return nil, err
		}
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return mapping.MapAggregateToAPI(agg), nil
}

// DownloadDateiOutput contains the response for downloading a datei
type DownloadDateiOutput struct {
	Reader          io.Reader
	ContentType     string
	ContentLength   int64
	ContentFileName string
}

// DownloadDatei retrieves a file for download
func (s *DateiService) DownloadDatei(ctx context.Context, dateiID uuid.UUID) (*DownloadDateiOutput, error) {
	queries := db.New(s.db)

	projection, err := queries.GetDateiProjectionByID(ctx, dateiID)
	if err != nil {
		return nil, err
	}

	if projection.IsDirectory {
		return nil, ErrIsDirectory
	}

	if projection.LatestVersionS3Key == nil {
		return nil, errors.New("no version available")
	}

	reader, err := s.store.GetObject(ctx, *projection.LatestVersionS3Key)
	if err != nil {
		return nil, err
	}

	return &DownloadDateiOutput{
		Reader:          reader,
		ContentType:     *projection.LatestVersionMimeType,
		ContentLength:   *projection.LatestVersionFileSize,
		ContentFileName: projection.LatestName,
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
func (s *DateiService) UpdateDatei(ctx context.Context, input UpdateDateiInput) (*api.Datei, error) {
	agg, err := s.repository.LoadByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if input.Name != nil {
		if err := agg.Rename(*input.Name, uuid.Nil, now); err != nil {
			return nil, err
		}
	}

	if input.Reader != nil && input.FileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, input.Reader, input.ContentType)
		if err != nil {
			return nil, err
		}

		if err = agg.UploadVersion(
			hash, fileSize, hash, input.ContentType, nil, uuid.Nil, now,
		); err != nil {
			return nil, err
		}
	}

	if err := s.repository.Save(ctx, agg); err != nil {
		return nil, err
	}

	return mapping.MapAggregateToAPI(agg), nil
}

// DeleteDatei soft-deletes a datei record
func (s *DateiService) DeleteDatei(ctx context.Context, dateiID uuid.UUID) error {
	agg, err := s.repository.LoadByID(ctx, dateiID)
	if err != nil {
		return err
	}

	if err := agg.Trash(uuid.Nil, time.Now()); err != nil {
		return err
	}

	return s.repository.Save(ctx, agg)
}

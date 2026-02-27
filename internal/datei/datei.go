package datei

import (
	"context"
	"io"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/mapping"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DateiService struct {
	db         *pgxpool.Pool
	store      storage.Store
	repository DateiRepository
	publisher  events.EventPublisher
}

func NewDateiService(db *pgxpool.Pool, store storage.Store, repository DateiRepository, publisher events.EventPublisher) *DateiService {
	return &DateiService{
		db:         db,
		store:      store,
		repository: repository,
		publisher:  publisher,
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

	// Get all Datei records with details in a single query
	allDateiWithDetails, err := queries.ListDateiWithDetails(ctx)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	total := len(allDateiWithDetails)
	start := min(offset, len(allDateiWithDetails))
	end := min(offset+limit, len(allDateiWithDetails))

	paginatedDatei := allDateiWithDetails[start:end]

	// Map to API response
	items := mapping.MapDBDateiDetailsSliceToAPI(paginatedDatei)

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
	queries := db.New(s.db)

	// Create Datei record
	isDirectory := input.Reader == nil
	datei, err := queries.CreateDatei(ctx, isDirectory)
	if err != nil {
		return nil, err
	}

	// Create DateiName
	nameRecord, err := queries.CreateDateiName(ctx, db.CreateDateiNameParams{
		DateiID: datei.ID,
		Name:    input.Name,
	})
	if err != nil {
		return nil, err
	}

	// Update Datei with latest name ID
	datei, err = queries.UpdateDateiLatestNameID(ctx, db.UpdateDateiLatestNameIDParams{
		ID:           datei.ID,
		LatestNameID: &nameRecord.ID,
	})
	if err != nil {
		return nil, err
	}

	// Handle file upload if provided
	var latestVersion *db.DateiVersion
	if input.Reader != nil && input.FileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, input.Reader, input.ContentType)
		if err != nil {
			return nil, err
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    hash,
			FileSize: fileSize,
			Checksum: hash,
			MimeType: input.ContentType,
		})
		if err != nil {
			return nil, err
		}

		// Update Datei with latest version ID
		datei, err = queries.UpdateDateiLatestVersionID(ctx, db.UpdateDateiLatestVersionIDParams{
			ID:              datei.ID,
			LatestVersionID: &versionRecord.ID,
		})
		if err != nil {
			return nil, err
		}

		latestVersion = &versionRecord
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&datei, latestVersion, &input.Name)
	return response, nil
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

	// Get Datei with details to check if it exists and has a version
	details, err := queries.GetDateiByIDWithDetails(ctx, dateiID)
	if err != nil {
		return nil, err
	}

	// Check if it's a directory
	if details.Datei.IsDirectory {
		return nil, ErrIsDirectory
	}

	// Get the file from storage
	reader, err := s.store.GetObject(ctx, details.DateiVersion.S3Key)
	if err != nil {
		return nil, err
	}

	return &DownloadDateiOutput{
		Reader:          reader,
		ContentType:     details.DateiVersion.MimeType,
		ContentLength:   details.DateiVersion.FileSize,
		ContentFileName: details.DateiName.Name,
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
	queries := db.New(s.db)

	// Get existing Datei
	datei, err := queries.GetDateiByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	// Update name if provided
	if input.Name != nil {
		nameRecord, err := queries.CreateDateiName(ctx, db.CreateDateiNameParams{
			DateiID: datei.ID,
			Name:    *input.Name,
		})
		if err != nil {
			return nil, err
		}

		datei, err = queries.UpdateDateiLatestNameID(ctx, db.UpdateDateiLatestNameIDParams{
			ID:           datei.ID,
			LatestNameID: &nameRecord.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	// Update file if provided
	if input.Reader != nil && input.FileName != "" {
		hash, fileSize, err := s.store.PutObject(ctx, input.Reader, input.ContentType)
		if err != nil {
			return nil, err
		}

		versionRecord, err := queries.CreateDateiVersion(ctx, db.CreateDateiVersionParams{
			DateiID:  datei.ID,
			S3Key:    hash,
			FileSize: fileSize,
			Checksum: hash,
			MimeType: input.ContentType,
		})
		if err != nil {
			return nil, err
		}

		datei, err = queries.UpdateDateiLatestVersionID(ctx, db.UpdateDateiLatestVersionIDParams{
			ID:              datei.ID,
			LatestVersionID: &versionRecord.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	// Fetch updated details for response
	details, err := queries.GetDateiByIDWithDetails(ctx, datei.ID)
	if err != nil {
		return nil, err
	}

	// Map to API response
	response := mapping.MapDBDateiToAPI(&details.Datei, &details.DateiVersion, &details.DateiName.Name)
	return response, nil
}

// DeleteDatei soft-deletes a datei record
func (s *DateiService) DeleteDatei(ctx context.Context, dateiID uuid.UUID) error {
	queries := db.New(s.db)

	// Get Datei to verify it exists
	_, err := queries.GetDateiByID(ctx, dateiID)
	if err != nil {
		return err
	}

	// Soft delete by setting trashed_at
	_, err = queries.SetDateiTrashedAt(ctx, dateiID)
	if err != nil {
		return err
	}

	return nil
}

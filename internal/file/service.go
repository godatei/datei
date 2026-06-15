package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/godatei/datei/internal/apperrors"
	"github.com/godatei/datei/internal/authn"
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

func (s *Service) startOCR(ctx context.Context, fileID uuid.UUID, s3Key, checksum, contentType string) {
	if s.ocrClient == nil || !isOCRable(contentType) {
		return
	}
	go func() {
		ctx := context.WithoutCancel(ctx)
		reader, err := s.store.GetObject(ctx, s3Key)
		if err != nil {
			slog.Warn("ocr: failed to get object from store", "error", err, "fileID", fileID)
			return
		}
		defer reader.Close()
		text, err := s.ocrClient.ExtractText(ctx, reader, contentType)
		if err != nil {
			slog.Warn("ocr: extraction failed", "error", err, "fileID", fileID)
			return
		}
		queries := db.New(s.db)
		if err := queries.UpdateFileProjectionContentMD(ctx, db.UpdateFileProjectionContentMDParams{
			ContentMd: &text,
			ID:        fileID,
			Checksum:  &checksum,
		}); err != nil {
			slog.Warn("ocr: failed to update projection", "error", err, "fileID", fileID)
		}
	}()
}

// ListFilesInput contains parameters for listing file records
type ListFilesInput struct {
	ParentID *uuid.UUID
	Limit    int
	Offset   int
}

// ListFilesOutput contains the response for listing file records
type ListFilesOutput struct {
	Items []api.File
	Total int
}

// ListFiles retrieves all file records with pagination
func (s *Service) ListFiles(ctx context.Context, input ListFilesInput) (*ListFilesOutput, error) {
	queries := db.New(s.db)

	limit := int32(input.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int32(max(input.Offset, 0))

	var projections []db.FileProjection
	var total int64
	var err error

	if input.ParentID != nil {
		total, err = queries.CountFileProjectionsByParent(ctx, input.ParentID)
		if err != nil {
			return nil, err
		}
		projections, err = queries.ListFileProjectionsByParent(ctx, db.ListFileProjectionsByParentParams{
			ParentID: input.ParentID,
			Limit:    limit,
			Offset:   offset,
		})
	} else {
		total, err = queries.CountRootFileProjections(ctx)
		if err != nil {
			return nil, err
		}
		projections, err = queries.ListRootFileProjections(ctx, db.ListRootFileProjectionsParams{
			Limit:  limit,
			Offset: offset,
		})
	}
	if err != nil {
		return nil, err
	}

	return &ListFilesOutput{
		Items: MapProjectionSliceToAPI(projections),
		Total: int(total),
	}, nil
}

// CreateFileInput contains parameters for creating a file
type CreateFileInput struct {
	ParentID    *uuid.UUID
	Reader      io.Reader
	FileName    string
	ContentType string
}

// CreateFile creates a new file record with optional file upload
func (s *Service) CreateFile(ctx context.Context, input CreateFileInput) (*api.File, error) {
	if input.ParentID != nil {
		queries := db.New(s.db)
		parent, err := queries.GetFileProjectionByID(ctx, *input.ParentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrParentNotFound
		} else if err != nil {
			return nil, err
		}
		if !parent.IsDirectory {
			return nil, apperrors.ErrParentNotDirectory
		}
		if parent.TrashedAt != nil {
			return nil, apperrors.ErrParentTrashed
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

// DownloadFileOutput contains the response for downloading a file
type DownloadFileOutput struct {
	Reader          io.Reader
	ContentType     string
	ContentLength   int64
	ContentFileName string
}

// DownloadFile retrieves a file for download
func (s *Service) DownloadFile(ctx context.Context, fileID uuid.UUID) (*DownloadFileOutput, error) {
	queries := db.New(s.db)

	projection, err := queries.GetFileProjectionByID(ctx, fileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if projection.IsDirectory {
		return nil, apperrors.ErrIsDirectory
	}

	if projection.S3Key == nil || projection.MimeType == nil || projection.Size == nil {
		return nil, apperrors.ErrNoContent
	}

	reader, err := s.store.GetObject(ctx, *projection.S3Key)
	if err != nil {
		return nil, err
	}

	return &DownloadFileOutput{
		Reader:          reader,
		ContentType:     *projection.MimeType,
		ContentLength:   *projection.Size,
		ContentFileName: projection.Name,
	}, nil
}

// UpdateFileInput contains parameters for updating a file
type UpdateFileInput struct {
	ID            uuid.UUID
	Name          *string
	MoveRequested bool
	NewParentID   *uuid.UUID
	Reader        io.Reader
	FileName      string
	ContentType   string
}

// UpdateFile updates a file record with optional name, move, and/or file.
func (s *Service) UpdateFile(ctx context.Context, input UpdateFileInput) (*api.File, error) {
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

// GetThumbnail returns a JPEG thumbnail for the given file, generating and caching it on first call.
// If ifNoneMatch equals the current checksum, returns ErrNotModified to allow a 304 response.
func (s *Service) GetThumbnail(
	ctx context.Context, fileID uuid.UUID, ifNoneMatch string,
) (*GetThumbnailOutput, error) {
	queries := db.New(s.db)

	projection, err := queries.GetFileProjectionByID(ctx, fileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if projection.IsDirectory {
		return nil, apperrors.ErrIsDirectory
	}
	if projection.Checksum == nil {
		return nil, apperrors.ErrNoContent
	}
	if projection.MimeType == nil || projection.S3Key == nil {
		return nil, apperrors.ErrNoContent
	}

	if ifNoneMatch == *projection.Checksum {
		return nil, apperrors.ErrNotModified
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

// GetFilePath returns the ancestor chain up to the given file (inclusive), root-first.
// The chain is truncated at the first trashed ancestor: that ancestor is included but its own parents are not.
func (s *Service) GetFilePath(ctx context.Context, fileID uuid.UUID) ([]api.FilePathItem, error) {
	queries := db.New(s.db)

	rows, err := queries.GetFilePath(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, apperrors.ErrNotFound
	}

	path := make([]api.FilePathItem, len(rows))
	for i, row := range rows {
		path[i] = api.FilePathItem{Id: row.ID, Name: row.Name}
		if row.TrashedAt != nil {
			path[i].Trashed = new(true)
		}
	}
	return path, nil
}

// FindFileByPath resolves a slash-split path of segments to a single file in one query.
// Returns apperrors.ErrNotFound if any segment along the path does not exist.
func (s *Service) FindFileByPath(ctx context.Context, segments []string) (*api.File, error) {
	proj, err := db.New(s.db).GetFileProjectionByPath(ctx, segments)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return MapProjectionToAPI(&proj), nil
}

// FindFileByName finds a non-trashed file by name within a parent (or at root).
// Returns apperrors.ErrNotFound if no match exists.
func (s *Service) FindFileByName(ctx context.Context, parentID *uuid.UUID, name string) (*api.File, error) {
	queries := db.New(s.db)
	var proj db.FileProjection
	var err error
	if parentID == nil {
		proj, err = queries.GetRootFileProjectionByName(ctx, name)
	} else {
		proj, err = queries.GetFileProjectionByParentAndName(ctx, db.GetFileProjectionByParentAndNameParams{
			ParentID: parentID,
			Name:     name,
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return MapProjectionToAPI(&proj), nil
}

const maxDirChildren = 10_000

// ListFileChildren returns up to 10 000 non-trashed children of the given
// parent (or root items if parentID is nil).
func (s *Service) ListFileChildren(ctx context.Context, parentID *uuid.UUID) ([]api.File, error) {
	queries := db.New(s.db)
	var projs []db.FileProjection
	var err error
	if parentID == nil {
		projs, err = queries.ListRootFileProjections(ctx, db.ListRootFileProjectionsParams{
			Limit:  maxDirChildren,
			Offset: 0,
		})
	} else {
		projs, err = queries.ListFileProjectionsByParent(ctx, db.ListFileProjectionsByParentParams{
			ParentID: parentID,
			Limit:    maxDirChildren,
			Offset:   0,
		})
	}
	if err != nil {
		return nil, err
	}
	return MapProjectionSliceToAPI(projs), nil
}

// ListTrashInput contains parameters for listing trashed file records.
type ListTrashInput struct {
	Limit  int
	Offset int
}

// ListTrashOutput contains the response for listing trashed file records.
type ListTrashOutput struct {
	Items []api.TrashedFile
	Total int
}

// ListTrash retrieves root-level trashed file records with pagination.
func (s *Service) ListTrash(ctx context.Context, input ListTrashInput) (*ListTrashOutput, error) {
	queries := db.New(s.db)

	limit := int32(input.Limit)
	if limit <= 0 {
		limit = 100
	}
	offset := int32(max(input.Offset, 0))

	total, err := queries.CountTrashedFile(ctx)
	if err != nil {
		return nil, err
	}

	projections, err := queries.ListTrashedFile(ctx, db.ListTrashedFileParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]api.TrashedFile, 0, len(projections))
	for i := range projections {
		p := &projections[i]
		var originPath *[]api.FilePathItem
		if p.ParentID != nil {
			rows, err := queries.GetFilePathIncludingTrashed(ctx, *p.ParentID)
			if err != nil {
				return nil, fmt.Errorf("get origin path for %s: %w", p.ID, err)
			}
			path := make([]api.FilePathItem, len(rows))
			for j, row := range rows {
				path[j] = api.FilePathItem{Id: row.ID, Name: row.Name}
			}
			originPath = &path
		} else {
			empty := []api.FilePathItem{}
			originPath = &empty
		}
		if mapped := MapProjectionToTrashedFile(p, originPath); mapped != nil {
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
func (s *Service) ListTrashChildren(ctx context.Context, input ListTrashChildrenInput) (*ListFilesOutput, error) {
	queries := db.New(s.db)

	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := max(input.Offset, 0)

	parent, err := queries.GetFileProjectionByID(ctx, input.ParentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrParentNotFound
		}
		return nil, err
	}

	if !parent.IsDirectory {
		return nil, apperrors.ErrParentNotDirectory
	}

	// A directory is browsable in trash if it is directly trashed or has a trashed ancestor.
	inTrash := parent.TrashedAt != nil
	if !inTrash {
		path, err := queries.GetFilePath(ctx, input.ParentID)
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
		return nil, apperrors.ErrParentNotTrashed
	}

	total, err := queries.CountFileProjectionsByParent(ctx, &input.ParentID)
	if err != nil {
		return nil, err
	}

	projections, err := queries.ListFileProjectionsByParent(ctx, db.ListFileProjectionsByParentParams{
		ParentID: &input.ParentID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}

	return &ListFilesOutput{
		Items: MapProjectionSliceToAPI(projections),
		Total: int(total),
	}, nil
}

// DeleteFile soft-deletes a file record
func (s *Service) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	agg, err := s.repository.LoadByID(ctx, fileID)
	if err != nil {
		return err
	}

	userID := authn.RequireContext(ctx).UserID
	if err := agg.Trash(userID, time.Now()); err != nil {
		return err
	}

	return s.repository.Save(ctx, agg)
}

type RestoreFileInput struct {
	ID       uuid.UUID
	ParentID *uuid.UUID
}

// RestoreFile restores a trashed file or moves a descendant of a trashed file to a new parent.
// parentId == nil moves the item to root.
func (s *Service) RestoreFile(ctx context.Context, input RestoreFileInput) error {
	queries := db.New(s.db)
	projection, err := queries.GetFileProjectionByID(ctx, input.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrNotFound
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
			return apperrors.ErrNotInTrash
		}
		parentPath, err := queries.GetFilePath(ctx, *projection.ParentID)
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
			return apperrors.ErrNotInTrash
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
	parent, err := queries.GetFileProjectionByID(ctx, targetParentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.ErrParentNotFound
	} else if err != nil {
		return err
	}
	if !parent.IsDirectory {
		return apperrors.ErrParentNotDirectory
	}

	// Walk the target's ancestor path to detect both trashed ancestors and cycles.
	// GetFilePath includes the target itself and stops above the first trashed node,
	// but includes that node — so any trashed entry means the target is inside trash.
	pathRows, err := queries.GetFilePath(ctx, targetParentID)
	if err != nil {
		return err
	}
	for _, row := range pathRows {
		if row.TrashedAt != nil {
			return apperrors.ErrParentTrashed
		}
		if itemIsDirectory && row.ID == itemID {
			return apperrors.ErrCycleDetected
		}
	}
	return nil
}

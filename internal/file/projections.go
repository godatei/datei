package file

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
)

func updateProjectionForFileCreated(ctx context.Context, q *db.Queries, event *FileCreatedEvent) error {
	err := q.InsertFileProjection(ctx, db.InsertFileProjectionParams{
		ID:          event.ID,
		ParentID:    event.ParentID,
		IsDirectory: event.IsDirectory,
		Name:        event.Name,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
		CreatedBy:   &event.CreatedBy,
		UpdatedBy:   &event.CreatedBy,
	})
	if err != nil {
		return fmt.Errorf("failed to insert file_projection: %w", err)
	}

	// Ownership is modeled as a file_permission_projection row rather than a
	// column on file_projection, so it can later carry shared read/write grants.
	// For now the creator gets the sole entry; access checks treat any entry as
	// access (see GetFileProjectionForUser). The row is keyed by its natural
	// (file_id, user_account_id), so this handler is deterministic on replay.
	if err := q.InsertFilePermissionProjection(ctx, db.InsertFilePermissionProjectionParams{
		FileID:        event.ID,
		UserAccountID: event.CreatedBy,
		CreatedAt:     event.CreatedAt,
	}); err != nil {
		return fmt.Errorf("failed to insert owner file_permission_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileRenamed(ctx context.Context, q *db.Queries, event *FileRenamedEvent) error {
	err := q.UpdateFileProjectionName(ctx, db.UpdateFileProjectionNameParams{
		Name:      event.NewName,
		UpdatedAt: event.RenamedAt,
		UpdatedBy: &event.RenamedBy,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileVersionUploaded(
	ctx context.Context,
	q *db.Queries,
	event *FileVersionUploadedEvent,
) error {
	err := q.UpdateFileProjectionVersion(ctx, db.UpdateFileProjectionVersionParams{
		S3Key:     &event.S3Key,
		Size:      &event.FileSize,
		Checksum:  &event.Checksum,
		MimeType:  &event.MimeType,
		ContentMd: event.ContentMD,
		UpdatedAt: event.UploadedAt,
		UpdatedBy: &event.UploadedBy,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileMoved(ctx context.Context, q *db.Queries, event *FileMovedEvent) error {
	err := q.UpdateFileProjectionParent(ctx, db.UpdateFileProjectionParentParams{
		ParentID:  event.NewParentID,
		UpdatedAt: event.MovedAt,
		UpdatedBy: &event.MovedBy,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileTrashed(ctx context.Context, q *db.Queries, event *FileTrashedEvent) error {
	err := q.UpdateFileProjectionTrashed(ctx, db.UpdateFileProjectionTrashedParams{
		TrashedAt: &event.TrashedAt,
		TrashedBy: &event.TrashedBy,
		UpdatedAt: event.TrashedAt,
		UpdatedBy: &event.TrashedBy,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileRestored(ctx context.Context, q *db.Queries, event *FileRestoredEvent) error {
	err := q.UpdateFileProjectionRestored(ctx, db.UpdateFileProjectionRestoredParams{
		UpdatedAt: event.RestoredAt,
		UpdatedBy: &event.RestoredBy,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileLinked(ctx context.Context, q *db.Queries, event *FileLinkedEvent) error {
	err := q.UpdateFileProjectionLinked(ctx, db.UpdateFileProjectionLinkedParams{
		LinkedFileID: &event.LinkedFileID,
		UpdatedAt:    event.LinkedAt,
		UpdatedBy:    &event.LinkedBy,
		ID:           event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileUnlinked(ctx context.Context, q *db.Queries, event *FileUnlinkedEvent) error {
	err := q.UpdateFileProjectionUnlinked(ctx, db.UpdateFileProjectionUnlinkedParams{
		UpdatedAt: event.UnlinkedAt,
		UpdatedBy: &event.UnlinkedBy,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

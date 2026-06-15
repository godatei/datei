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
	})
	if err != nil {
		return fmt.Errorf("failed to insert file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFileRenamed(ctx context.Context, q *db.Queries, event *FileRenamedEvent) error {
	err := q.UpdateFileProjectionName(ctx, db.UpdateFileProjectionNameParams{
		Name:      event.NewName,
		UpdatedAt: event.RenamedAt,
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
		UpdatedAt: event.TrashedAt,
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
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update file_projection: %w", err)
	}

	return nil
}

func updateProjectionForFilePermissionGranted(
	ctx context.Context,
	q *db.Queries,
	event *FilePermissionGrantedEvent,
) error {
	err := q.InsertFilePermissionProjection(ctx, db.InsertFilePermissionProjectionParams{
		ID:             event.ID,
		FileID:         event.FileID,
		UserAccountID:  event.UserAccountID,
		UserGroupID:    event.UserGroupID,
		PermissionType: db.FilePermissionType(event.PermissionType),
		CreatedAt:      event.GrantedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to insert file_permission_projection: %w", err)
	}

	return nil
}

func updateProjectionForFilePermissionRevoked(
	ctx context.Context,
	q *db.Queries,
	event *FilePermissionRevokedEvent,
) error {
	err := q.DeleteFilePermissionProjection(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to delete from file_permission_projection: %w", err)
	}

	return nil
}

package datei

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
)

func updateProjectionForDateiCreated(ctx context.Context, q *db.Queries, event *DateiCreatedEvent) error {
	err := q.InsertDateiProjection(ctx, db.InsertDateiProjectionParams{
		ID:          event.ID,
		ParentID:    event.ParentID,
		IsDirectory: event.IsDirectory,
		Name:        event.Name,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to insert datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiRenamed(ctx context.Context, q *db.Queries, event *DateiRenamedEvent) error {
	err := q.UpdateDateiProjectionName(ctx, db.UpdateDateiProjectionNameParams{
		Name:      event.NewName,
		UpdatedAt: event.RenamedAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiVersionUploaded(
	ctx context.Context,
	q *db.Queries,
	event *DateiVersionUploadedEvent,
) error {
	err := q.UpdateDateiProjectionVersion(ctx, db.UpdateDateiProjectionVersionParams{
		S3Key:     &event.S3Key,
		Size:      &event.FileSize,
		Checksum:  &event.Checksum,
		MimeType:  &event.MimeType,
		ContentMd: event.ContentMD,
		UpdatedAt: event.UploadedAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiMoved(ctx context.Context, q *db.Queries, event *DateiMovedEvent) error {
	err := q.UpdateDateiProjectionParent(ctx, db.UpdateDateiProjectionParentParams{
		ParentID:  event.NewParentID,
		UpdatedAt: event.MovedAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiTrashed(ctx context.Context, q *db.Queries, event *DateiTrashedEvent) error {
	err := q.UpdateDateiProjectionTrashed(ctx, db.UpdateDateiProjectionTrashedParams{
		TrashedAt: &event.TrashedAt,
		UpdatedAt: event.TrashedAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiRestored(ctx context.Context, q *db.Queries, event *DateiRestoredEvent) error {
	err := q.UpdateDateiProjectionRestored(ctx, db.UpdateDateiProjectionRestoredParams{
		UpdatedAt: event.RestoredAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiLinked(ctx context.Context, q *db.Queries, event *DateiLinkedEvent) error {
	err := q.UpdateDateiProjectionLinked(ctx, db.UpdateDateiProjectionLinkedParams{
		LinkedDateiID: &event.LinkedDateiID,
		UpdatedAt:     event.LinkedAt,
		ID:            event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiUnlinked(ctx context.Context, q *db.Queries, event *DateiUnlinkedEvent) error {
	err := q.UpdateDateiProjectionUnlinked(ctx, db.UpdateDateiProjectionUnlinkedParams{
		UpdatedAt: event.UnlinkedAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiPermissionGranted(
	ctx context.Context,
	q *db.Queries,
	event *DateiPermissionGrantedEvent,
) error {
	err := q.InsertDateiPermissionProjection(ctx, db.InsertDateiPermissionProjectionParams{
		ID:             event.ID,
		DateiID:        event.DateiID,
		UserAccountID:  event.UserAccountID,
		UserGroupID:    event.UserGroupID,
		PermissionType: db.DateiPermissionType(event.PermissionType),
		CreatedAt:      event.GrantedAt,
	})
	if err != nil {
		return fmt.Errorf("failed to insert datei_permission_projection: %w", err)
	}

	return nil
}

func updateProjectionForDateiPermissionRevoked(
	ctx context.Context,
	q *db.Queries,
	event *DateiPermissionRevokedEvent,
) error {
	err := q.DeleteDateiPermissionProjection(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to delete from datei_permission_projection: %w", err)
	}

	return nil
}

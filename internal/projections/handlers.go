package projections

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/events"
)

// ============================================================================
// Datei Projection Handlers
// ============================================================================

// UpdateProjectionForDateiCreated updates projections after a datei is created
func UpdateProjectionForDateiCreated(ctx context.Context, q *db.Queries, event *events.DateiCreatedEvent) error {
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

// UpdateProjectionForDateiRenamed updates projections after a datei is renamed
func UpdateProjectionForDateiRenamed(ctx context.Context, q *db.Queries, event *events.DateiRenamedEvent) error {
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

// UpdateProjectionForDateiVersionUploaded updates projections after a new version is uploaded
func UpdateProjectionForDateiVersionUploaded(
	ctx context.Context,
	q *db.Queries,
	event *events.DateiVersionUploadedEvent,
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

// UpdateProjectionForDateiMoved updates projections after a datei is moved
func UpdateProjectionForDateiMoved(ctx context.Context, q *db.Queries, event *events.DateiMovedEvent) error {
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

// UpdateProjectionForDateiTrashed updates projections after a datei is trashed
func UpdateProjectionForDateiTrashed(ctx context.Context, q *db.Queries, event *events.DateiTrashedEvent) error {
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

// UpdateProjectionForDateiRestored updates projections after a datei is restored
func UpdateProjectionForDateiRestored(ctx context.Context, q *db.Queries, event *events.DateiRestoredEvent) error {
	err := q.UpdateDateiProjectionRestored(ctx, db.UpdateDateiProjectionRestoredParams{
		UpdatedAt: event.RestoredAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiLinked updates projections after a datei is linked
func UpdateProjectionForDateiLinked(ctx context.Context, q *db.Queries, event *events.DateiLinkedEvent) error {
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

// UpdateProjectionForDateiUnlinked updates projections after a datei is unlinked
func UpdateProjectionForDateiUnlinked(ctx context.Context, q *db.Queries, event *events.DateiUnlinkedEvent) error {
	err := q.UpdateDateiProjectionUnlinked(ctx, db.UpdateDateiProjectionUnlinkedParams{
		UpdatedAt: event.UnlinkedAt,
		ID:        event.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// ============================================================================
// Permission Projection Handlers
// ============================================================================

// UpdateProjectionForDateiPermissionGranted updates projections after a permission is granted
func UpdateProjectionForDateiPermissionGranted(
	ctx context.Context,
	q *db.Queries,
	event *events.DateiPermissionGrantedEvent,
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

// UpdateProjectionForDateiPermissionRevoked updates projections after a permission is revoked
func UpdateProjectionForDateiPermissionRevoked(
	ctx context.Context,
	q *db.Queries,
	event *events.DateiPermissionRevokedEvent,
) error {
	err := q.DeleteDateiPermissionProjection(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to delete from datei_permission_projection: %w", err)
	}

	return nil
}

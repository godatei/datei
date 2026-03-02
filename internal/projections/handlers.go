package projections

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/events"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Datei Projection Handlers
// ============================================================================

// UpdateProjectionForDateiCreated updates projections after a datei is created
func UpdateProjectionForDateiCreated(ctx context.Context, tx pgx.Tx, event *events.DateiCreatedEvent) error {
	// Create datei_projection record with embedded initial name data
	_, err := tx.Exec(ctx,
		`INSERT INTO datei_projection
		 (id, parent_id, is_directory, latest_name,
		  created_by, created_at, updated_at, projection_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 1)`,
		event.ID,
		event.ParentID,
		event.IsDirectory,
		event.Name,
		event.CreatedBy,
		event.CreatedAt,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiRenamed updates projections after a datei is renamed
func UpdateProjectionForDateiRenamed(ctx context.Context, tx pgx.Tx, event *events.DateiRenamedEvent) error {
	// Update datei projection with new name
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection
		 SET latest_name = $1, updated_at = $2, projection_version = projection_version + 1
		 WHERE id = $3`,
		event.NewName,
		event.RenamedAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiVersionUploaded updates projections after a new version is uploaded
func UpdateProjectionForDateiVersionUploaded(
	ctx context.Context,
	tx pgx.Tx,
	event *events.DateiVersionUploadedEvent,
) error {
	// Update datei projection with new version data
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection
		 SET latest_version_s3_key = $1, latest_version_file_size = $2,
		     latest_version_checksum = $3, latest_version_mime_type = $4,
		     latest_version_content_md = $5, updated_at = $6,
		     projection_version = projection_version + 1
		 WHERE id = $7`,
		event.S3Key,
		event.FileSize,
		event.Checksum,
		event.MimeType,
		event.ContentMD,
		event.UploadedAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiMoved updates projections after a datei is moved
func UpdateProjectionForDateiMoved(ctx context.Context, tx pgx.Tx, event *events.DateiMovedEvent) error {
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection SET parent_id = $1, updated_at = $2, projection_version = projection_version + 1
		 WHERE id = $3`,
		event.NewParentID,
		event.MovedAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiTrashed updates projections after a datei is trashed
func UpdateProjectionForDateiTrashed(ctx context.Context, tx pgx.Tx, event *events.DateiTrashedEvent) error {
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection
		 SET trashed_at = $1, trashed_by = $2, updated_at = $3,
		     projection_version = projection_version + 1
		 WHERE id = $4`,
		event.TrashedAt,
		event.TrashedBy,
		event.TrashedAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiRestored updates projections after a datei is restored
func UpdateProjectionForDateiRestored(ctx context.Context, tx pgx.Tx, event *events.DateiRestoredEvent) error {
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection
		 SET trashed_at = NULL, trashed_by = NULL, updated_at = $1,
		     projection_version = projection_version + 1
		 WHERE id = $2`,
		event.RestoredAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiLinked updates projections after a datei is linked
func UpdateProjectionForDateiLinked(ctx context.Context, tx pgx.Tx, event *events.DateiLinkedEvent) error {
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection SET linked_datei_id = $1, updated_at = $2, projection_version = projection_version + 1
		 WHERE id = $3`,
		event.LinkedDateiID,
		event.LinkedAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiUnlinked updates projections after a datei is unlinked
func UpdateProjectionForDateiUnlinked(ctx context.Context, tx pgx.Tx, event *events.DateiUnlinkedEvent) error {
	_, err := tx.Exec(ctx,
		`UPDATE datei_projection SET linked_datei_id = NULL, updated_at = $1, projection_version = projection_version + 1
		 WHERE id = $2`,
		event.UnlinkedAt,
		event.ID,
	)
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
	tx pgx.Tx,
	event *events.DateiPermissionGrantedEvent,
) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO datei_permission_projection
		 (id, datei_id, user_account_id, user_group_id, permission_type, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		event.ID,
		event.DateiID,
		event.UserAccountID,
		event.UserGroupID,
		event.PermissionType,
		event.GrantedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert datei_permission_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiPermissionRevoked updates projections after a permission is revoked
func UpdateProjectionForDateiPermissionRevoked(
	ctx context.Context,
	tx pgx.Tx,
	event *events.DateiPermissionRevokedEvent,
) error {
	_, err := tx.Exec(ctx,
		`DELETE FROM datei_permission_projection
		 WHERE id = $1`,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete from datei_permission_projection: %w", err)
	}

	return nil
}

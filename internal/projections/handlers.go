package projections

import (
	"context"
	"fmt"

	"github.com/godatei/datei/internal/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Datei Projection Handlers
// ============================================================================

// UpdateProjectionForDateiCreated updates projections after a datei is created
func UpdateProjectionForDateiCreated(ctx context.Context, tx pgx.Tx, event *events.DateiCreatedEvent) error {
	// Create datei_name record
	nameID := uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO datei_name_projection (id, datei_id, name, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		nameID,
		event.ID,
		event.Name,
		event.CreatedBy,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert datei_name_projection: %w", err)
	}

	// Create datei_projection record
	_, err = tx.Exec(ctx,
		`INSERT INTO datei_projection (id, parent_id, is_directory, latest_name_id, created_by, created_at, updated_at, projection_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 1)`,
		event.ID,
		event.ParentID,
		event.IsDirectory,
		nameID,
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
	// Create new name record
	nameID := uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO datei_name_projection (id, datei_id, name, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		nameID,
		event.ID,
		event.NewName,
		event.RenamedBy,
		event.RenamedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert datei_name_projection: %w", err)
	}

	// Update datei projection to point to new name
	_, err = tx.Exec(ctx,
		`UPDATE datei_projection SET latest_name_id = $1, updated_at = $2, projection_version = projection_version + 1
		 WHERE id = $3`,
		nameID,
		event.RenamedAt,
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update datei_projection: %w", err)
	}

	return nil
}

// UpdateProjectionForDateiVersionUploaded updates projections after a new version is uploaded
func UpdateProjectionForDateiVersionUploaded(ctx context.Context, tx pgx.Tx, event *events.DateiVersionUploadedEvent) error {
	// Create new version record
	_, err := tx.Exec(ctx,
		`INSERT INTO datei_version_projection (id, datei_id, s3_key, file_size, checksum, mime_type, content_md, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		event.VersionID,
		event.ID,
		event.S3Key,
		event.FileSize,
		event.Checksum,
		event.MimeType,
		event.ContentMD,
		event.UploadedBy,
		event.UploadedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert datei_version_projection: %w", err)
	}

	// Update datei projection to point to new version
	_, err = tx.Exec(ctx,
		`UPDATE datei_projection SET latest_version_id = $1, updated_at = $2, projection_version = projection_version + 1
		 WHERE id = $3`,
		event.VersionID,
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
		`UPDATE datei_projection SET trashed_at = $1, trashed_by = $2, updated_at = $3, projection_version = projection_version + 1
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
		`UPDATE datei_projection SET trashed_at = NULL, trashed_by = NULL, updated_at = $1, projection_version = projection_version + 1
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
func UpdateProjectionForDateiPermissionGranted(ctx context.Context, tx pgx.Tx, event *events.DateiPermissionGrantedEvent) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO datei_permission_projection (id, datei_id, user_account_id, user_group_id, permission_type, created_at)
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
func UpdateProjectionForDateiPermissionRevoked(ctx context.Context, tx pgx.Tx, event *events.DateiPermissionRevokedEvent) error {
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

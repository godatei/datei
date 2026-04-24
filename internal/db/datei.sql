-- name: GetDateiProjectionByID :one
SELECT * FROM datei_projection WHERE id = $1;

-- name: ListDateiProjections :many
SELECT * FROM datei_projection ORDER BY created_at DESC;

-- name: ListRootDateiProjections :many
SELECT * FROM datei_projection WHERE parent_id IS NULL AND trashed_at IS NULL ORDER BY is_directory DESC, name ASC;

-- name: ListDateiProjectionsByParent :many
SELECT * FROM datei_projection WHERE parent_id = $1 AND trashed_at IS NULL ORDER BY is_directory DESC, name ASC;

-- name: InsertDateiProjection :exec
INSERT INTO datei_projection
 (id, parent_id, is_directory, name, created_at, updated_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateDateiProjectionName :exec
UPDATE datei_projection
 SET name = $1, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateDateiProjectionVersion :exec
UPDATE datei_projection
 SET s3_key = $1, size = $2, checksum = $3, mime_type = $4,
     content_md = $5, updated_at = $6, updated_by = NULL
 WHERE id = $7;

-- name: UpdateDateiProjectionParent :exec
UPDATE datei_projection
 SET parent_id = $1, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateDateiProjectionTrashed :exec
UPDATE datei_projection
 SET trashed_at = $1, trashed_by = NULL, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateDateiProjectionRestored :exec
UPDATE datei_projection
 SET trashed_at = NULL, trashed_by = NULL, updated_at = $1, updated_by = NULL
 WHERE id = $2;

-- name: UpdateDateiProjectionLinked :exec
UPDATE datei_projection
 SET linked_datei_id = $1, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateDateiProjectionUnlinked :exec
UPDATE datei_projection
 SET linked_datei_id = NULL, updated_at = $1, updated_by = NULL
 WHERE id = $2;

-- name: InsertDateiPermissionProjection :exec
INSERT INTO datei_permission_projection
 (id, datei_id, user_account_id, user_group_id, permission_type, created_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteDateiPermissionProjection :exec
DELETE FROM datei_permission_projection
 WHERE id = $1;

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

-- name: UpdateDateiProjectionContentMD :exec
UPDATE datei_projection
 SET content_md = $1
 WHERE id = $2 AND checksum = $3;

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

-- name: GetDateiPath :many
WITH RECURSIVE ancestors(id, parent_id, name, trashed_at, depth) AS (
  SELECT d.id, d.parent_id, d.name, d.trashed_at, 0 FROM datei_projection d WHERE d.id = $1
  UNION ALL
  SELECT p.id, p.parent_id, p.name, p.trashed_at, a.depth + 1
  FROM datei_projection p
  INNER JOIN ancestors a ON p.id = a.parent_id
)
SELECT id, name FROM ancestors
WHERE NOT EXISTS (SELECT 1 FROM ancestors WHERE trashed_at IS NOT NULL)
ORDER BY depth DESC;

-- name: InsertDateiPermissionProjection :exec
INSERT INTO datei_permission_projection
 (id, datei_id, user_account_id, user_group_id, permission_type, created_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteDateiPermissionProjection :exec
DELETE FROM datei_permission_projection
 WHERE id = $1;

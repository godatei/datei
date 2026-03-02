-- name: CreateDatei :one
INSERT INTO datei (is_directory) VALUES ($1) RETURNING *;

-- name: CreateDateiName :one
INSERT INTO datei_name (datei_id, name) VALUES ($1, $2) RETURNING *;

-- name: CreateDateiVersion :one
INSERT INTO datei_version (datei_id, s3_key, file_size, checksum, mime_type) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateDateiLatestNameID :one
UPDATE datei SET latest_name_id = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: UpdateDateiLatestVersionID :one
UPDATE datei SET latest_version_id = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: SetDateiTrashedAt :one
UPDATE datei SET trashed_at = now(), updated_at = now() WHERE id = $1 RETURNING *;

-- name: GetDateiByID :one
SELECT * FROM datei WHERE id = $1;

-- name: GetDateiByIDWithDetails :one
SELECT sqlc.embed(d), sqlc.embed(ln), sqlc.embed(lv)
FROM datei d
LEFT JOIN datei_name ln ON d.latest_name_id = ln.id
LEFT JOIN datei_version lv ON d.latest_version_id = lv.id
WHERE d.id = $1;

-- name: ListDateiWithDetails :many
SELECT sqlc.embed(d), sqlc.embed(ln), sqlc.embed(lv)
FROM datei d
LEFT JOIN datei_name ln ON d.latest_name_id = ln.id
LEFT JOIN datei_version lv ON d.latest_version_id = lv.id
ORDER BY d.created_at DESC;

-- name: InsertDateiProjection :exec
INSERT INTO datei_projection
 (id, parent_id, is_directory, latest_name,
  created_by, created_at, updated_at, projection_version)
 VALUES ($1, $2, $3, $4, $5, $6, $7, 1);

-- name: UpdateDateiProjectionName :exec
UPDATE datei_projection
 SET latest_name = $1, updated_at = $2, projection_version = projection_version + 1
 WHERE id = $3;

-- name: UpdateDateiProjectionVersion :exec
UPDATE datei_projection
 SET latest_version_s3_key = $1, latest_version_file_size = $2,
     latest_version_checksum = $3, latest_version_mime_type = $4,
     latest_version_content_md = $5, updated_at = $6,
     projection_version = projection_version + 1
 WHERE id = $7;

-- name: UpdateDateiProjectionParent :exec
UPDATE datei_projection
 SET parent_id = $1, updated_at = $2, projection_version = projection_version + 1
 WHERE id = $3;

-- name: UpdateDateiProjectionTrashed :exec
UPDATE datei_projection
 SET trashed_at = $1, trashed_by = $2, updated_at = $3,
     projection_version = projection_version + 1
 WHERE id = $4;

-- name: UpdateDateiProjectionRestored :exec
UPDATE datei_projection
 SET trashed_at = NULL, trashed_by = NULL, updated_at = $1,
     projection_version = projection_version + 1
 WHERE id = $2;

-- name: UpdateDateiProjectionLinked :exec
UPDATE datei_projection
 SET linked_datei_id = $1, updated_at = $2, projection_version = projection_version + 1
 WHERE id = $3;

-- name: UpdateDateiProjectionUnlinked :exec
UPDATE datei_projection
 SET linked_datei_id = NULL, updated_at = $1, projection_version = projection_version + 1
 WHERE id = $2;

-- name: InsertDateiPermissionProjection :exec
INSERT INTO datei_permission_projection
 (id, datei_id, user_account_id, user_group_id, permission_type, created_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteDateiPermissionProjection :exec
DELETE FROM datei_permission_projection
 WHERE id = $1;

-- name: GetFileProjectionByID :one
SELECT * FROM file_projection WHERE id = $1;

-- name: CountUntrashedFileByIDs :one
-- Counts how many of the given UUIDs refer to files that exist AND are not
-- trashed. Callers compare against len(input) to reject requests that point
-- at missing or trashed rows.
SELECT COUNT(*)::int FROM file_projection
 WHERE id = ANY($1::uuid[]) AND trashed_at IS NULL;

-- name: ListFileProjections :many
SELECT * FROM file_projection ORDER BY created_at DESC;

-- name: ListRootFileProjections :many
SELECT * FROM file_projection WHERE parent_id IS NULL AND trashed_at IS NULL ORDER BY is_directory DESC, name ASC
LIMIT $1 OFFSET $2;

-- name: CountRootFileProjections :one
SELECT COUNT(*) FROM file_projection WHERE parent_id IS NULL AND trashed_at IS NULL;

-- name: ListTrashedFile :many
SELECT * FROM file_projection WHERE trashed_at IS NOT NULL ORDER BY trashed_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTrashedFile :one
SELECT COUNT(*) FROM file_projection WHERE trashed_at IS NOT NULL;

-- name: ListFileProjectionsByParent :many
SELECT * FROM file_projection WHERE parent_id = $1 AND trashed_at IS NULL ORDER BY is_directory DESC, name ASC
LIMIT $2 OFFSET $3;

-- name: CountFileProjectionsByParent :one
SELECT COUNT(*) FROM file_projection WHERE parent_id = $1 AND trashed_at IS NULL;

-- name: InsertFileProjection :exec
INSERT INTO file_projection
 (id, parent_id, is_directory, name, created_at, updated_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateFileProjectionName :exec
UPDATE file_projection
 SET name = $1, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateFileProjectionVersion :exec
UPDATE file_projection
 SET s3_key = $1, size = $2, checksum = $3, mime_type = $4,
     content_md = $5, updated_at = $6, updated_by = NULL
 WHERE id = $7;

-- name: UpdateFileProjectionParent :exec
UPDATE file_projection
 SET parent_id = $1, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateFileProjectionContentMD :exec
UPDATE file_projection
 SET content_md = $1
 WHERE id = $2 AND checksum = $3;

-- name: UpdateFileProjectionTrashed :exec
UPDATE file_projection
 SET trashed_at = $1, trashed_by = NULL, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateFileProjectionRestored :exec
UPDATE file_projection
 SET trashed_at = NULL, trashed_by = NULL, updated_at = $1, updated_by = NULL
 WHERE id = $2;

-- name: UpdateFileProjectionLinked :exec
UPDATE file_projection
 SET linked_file_id = $1, updated_at = $2, updated_by = NULL
 WHERE id = $3;

-- name: UpdateFileProjectionUnlinked :exec
UPDATE file_projection
 SET linked_file_id = NULL, updated_at = $1, updated_by = NULL
 WHERE id = $2;

-- name: GetRootFileProjectionByName :one
SELECT * FROM file_projection WHERE parent_id IS NULL AND name = $1 AND trashed_at IS NULL;

-- name: GetFileProjectionByParentAndName :one
SELECT * FROM file_projection WHERE parent_id = $1 AND name = $2 AND trashed_at IS NULL;

-- name: GetFileProjectionByPath :one
WITH RECURSIVE path_walk AS (
  SELECT d.id, 1::int AS depth
  FROM file_projection d
  WHERE d.parent_id IS NULL
    AND d.name = ($1::text[])[1]
    AND d.trashed_at IS NULL
  UNION ALL
  SELECT d.id, pw.depth + 1
  FROM file_projection d
  JOIN path_walk pw ON d.parent_id = pw.id
  WHERE d.name = ($1::text[])[pw.depth + 1]
    AND d.trashed_at IS NULL
)
SELECT dp.* FROM file_projection dp
JOIN path_walk pw ON dp.id = pw.id
WHERE pw.depth = array_length($1::text[], 1);

-- name: GetFilePath :many
WITH RECURSIVE ancestors(id, parent_id, name, trashed_at, depth) AS (
  SELECT d.id, d.parent_id, d.name, d.trashed_at, 0 FROM file_projection d WHERE d.id = $1
  UNION ALL
  SELECT p.id, p.parent_id, p.name, p.trashed_at, a.depth + 1
  FROM file_projection p
  INNER JOIN ancestors a ON p.id = a.parent_id
  WHERE a.trashed_at IS NULL
)
SELECT id, name, trashed_at FROM ancestors
ORDER BY depth DESC;

-- name: GetFilePathIncludingTrashed :many
WITH RECURSIVE ancestors(id, parent_id, name, depth) AS (
  SELECT d.id, d.parent_id, d.name, 0 FROM file_projection d WHERE d.id = $1
  UNION ALL
  SELECT p.id, p.parent_id, p.name, a.depth + 1
  FROM file_projection p
  INNER JOIN ancestors a ON p.id = a.parent_id
)
SELECT id, name FROM ancestors
ORDER BY depth DESC;

-- name: InsertFilePermissionProjection :exec
INSERT INTO file_permission_projection
 (id, file_id, user_account_id, user_group_id, permission_type, created_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteFilePermissionProjection :exec
DELETE FROM file_permission_projection
 WHERE id = $1;

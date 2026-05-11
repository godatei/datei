-- name: GetDateiProjectionByID :one
SELECT * FROM datei_projection WHERE id = $1;

-- name: CountDateiProjectionsByIDs :one
SELECT COUNT(*)::int FROM datei_projection WHERE id = ANY($1::uuid[]);

-- name: CountDateiProjectionsByIDsOwnedBy :one
-- Counts dateien from the given set that the specified user owns
-- (permission_type = 'owner'). Used by the link service so create/add-datei
-- can't be tricked into sharing another user's files.
SELECT COUNT(*)::int
FROM datei_projection d
INNER JOIN datei_permission_projection p ON p.datei_id = d.id
WHERE d.id = ANY($1::uuid[])
  AND p.user_account_id = $2
  AND p.permission_type = 'owner';

-- name: IsDateiOwnedBy :one
-- Single-id ownership predicate, same semantics as CountDateiProjectionsByIDsOwnedBy.
SELECT EXISTS(
  SELECT 1 FROM datei_permission_projection
   WHERE datei_id = $1
     AND user_account_id = $2
     AND permission_type = 'owner'
);

-- name: ListDateiProjections :many
SELECT * FROM datei_projection ORDER BY created_at DESC;

-- name: ListRootDateiProjections :many
SELECT * FROM datei_projection WHERE parent_id IS NULL AND trashed_at IS NULL ORDER BY is_directory DESC, name ASC
LIMIT $1 OFFSET $2;

-- name: CountRootDateiProjections :one
SELECT COUNT(*) FROM datei_projection WHERE parent_id IS NULL AND trashed_at IS NULL;

-- name: ListTrashedDatei :many
SELECT * FROM datei_projection WHERE trashed_at IS NOT NULL ORDER BY trashed_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTrashedDatei :one
SELECT COUNT(*) FROM datei_projection WHERE trashed_at IS NOT NULL;

-- name: ListDateiProjectionsByParent :many
SELECT * FROM datei_projection WHERE parent_id = $1 AND trashed_at IS NULL ORDER BY is_directory DESC, name ASC
LIMIT $2 OFFSET $3;

-- name: CountDateiProjectionsByParent :one
SELECT COUNT(*) FROM datei_projection WHERE parent_id = $1 AND trashed_at IS NULL;

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

-- name: GetRootDateiProjectionByName :one
SELECT * FROM datei_projection WHERE parent_id IS NULL AND name = $1 AND trashed_at IS NULL;

-- name: GetDateiProjectionByParentAndName :one
SELECT * FROM datei_projection WHERE parent_id = $1 AND name = $2 AND trashed_at IS NULL;

-- name: GetDateiProjectionByPath :one
WITH RECURSIVE path_walk AS (
  SELECT d.id, 1::int AS depth
  FROM datei_projection d
  WHERE d.parent_id IS NULL
    AND d.name = ($1::text[])[1]
    AND d.trashed_at IS NULL
  UNION ALL
  SELECT d.id, pw.depth + 1
  FROM datei_projection d
  JOIN path_walk pw ON d.parent_id = pw.id
  WHERE d.name = ($1::text[])[pw.depth + 1]
    AND d.trashed_at IS NULL
)
SELECT dp.* FROM datei_projection dp
JOIN path_walk pw ON dp.id = pw.id
WHERE pw.depth = array_length($1::text[], 1);

-- name: GetDateiPath :many
WITH RECURSIVE ancestors(id, parent_id, name, trashed_at, depth) AS (
  SELECT d.id, d.parent_id, d.name, d.trashed_at, 0 FROM datei_projection d WHERE d.id = $1
  UNION ALL
  SELECT p.id, p.parent_id, p.name, p.trashed_at, a.depth + 1
  FROM datei_projection p
  INNER JOIN ancestors a ON p.id = a.parent_id
  WHERE a.trashed_at IS NULL
)
SELECT id, name, trashed_at FROM ancestors
ORDER BY depth DESC;

-- name: GetDateiPathIncludingTrashed :many
WITH RECURSIVE ancestors(id, parent_id, name, depth) AS (
  SELECT d.id, d.parent_id, d.name, 0 FROM datei_projection d WHERE d.id = $1
  UNION ALL
  SELECT p.id, p.parent_id, p.name, a.depth + 1
  FROM datei_projection p
  INNER JOIN ancestors a ON p.id = a.parent_id
)
SELECT id, name FROM ancestors
ORDER BY depth DESC;

-- name: InsertDateiPermissionProjection :exec
INSERT INTO datei_permission_projection
 (id, datei_id, user_account_id, user_group_id, permission_type, created_at)
 VALUES ($1, $2, $3, $4, $5, $6);

-- name: DeleteDateiPermissionProjection :exec
DELETE FROM datei_permission_projection
 WHERE id = $1;

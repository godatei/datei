-- name: GetFileProjectionByID :one
-- Access-agnostic lookup by id (no permission check). Used by the public-link
-- viewer plane, which authorizes via link scope rather than file ownership.
SELECT * FROM file_projection WHERE id = $1;

-- name: GetFileProjectionForUser :one
-- Ownership-scoped lookup: returns the file only if the given user has a
-- file_permission_projection entry for it. Any entry currently means "owner"
-- (see InsertFilePermissionProjection on create); the join is the single
-- authorization gate and extends naturally to shared read/write grants later.
SELECT f.* FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.id = @id;

-- name: CountUntrashedFileByIDs :one
-- Counts how many of the given UUIDs refer to files that exist, are accessible
-- to the given user, AND are not trashed. Callers compare against len(input) to
-- reject requests that point at missing, trashed, or non-accessible rows.
SELECT COUNT(*)::int FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.id = ANY(@ids::uuid[]) AND f.trashed_at IS NULL;

-- name: ListFileProjections :many
SELECT * FROM file_projection ORDER BY created_at DESC;

-- name: ListRootFileProjections :many
SELECT f.* FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.parent_id IS NULL AND f.trashed_at IS NULL ORDER BY f.is_directory DESC, f.name ASC
LIMIT @lim OFFSET @off;

-- name: CountRootFileProjections :one
SELECT COUNT(*) FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.parent_id IS NULL AND f.trashed_at IS NULL;

-- name: ListTrashedFile :many
SELECT f.* FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.trashed_at IS NOT NULL ORDER BY f.trashed_at DESC
LIMIT @lim OFFSET @off;

-- name: CountTrashedFile :one
SELECT COUNT(*) FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.trashed_at IS NOT NULL;

-- name: ListFileProjectionsByParent :many
SELECT f.* FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.parent_id = @parent_id AND f.trashed_at IS NULL ORDER BY f.is_directory DESC, f.name ASC
LIMIT @lim OFFSET @off;

-- name: CountFileProjectionsByParent :one
SELECT COUNT(*) FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.parent_id = @parent_id AND f.trashed_at IS NULL;

-- name: InsertFileProjection :exec
INSERT INTO file_projection
 (id, parent_id, is_directory, name, created_at, updated_at, created_by, updated_by)
 VALUES (@id, @parent_id, @is_directory, @name, @created_at, @updated_at, @created_by, @updated_by);

-- name: UpdateFileProjectionName :exec
UPDATE file_projection
 SET name = $1, updated_at = $2, updated_by = $3
 WHERE id = $4;

-- name: UpdateFileProjectionVersion :exec
UPDATE file_projection
 SET s3_key = $1, size = $2, checksum = $3, mime_type = $4,
     content_md = $5, updated_at = $6, updated_by = $7
 WHERE id = $8;

-- name: UpdateFileProjectionParent :exec
UPDATE file_projection
 SET parent_id = $1, updated_at = $2, updated_by = $3
 WHERE id = $4;

-- name: UpdateFileProjectionContentMD :exec
UPDATE file_projection
 SET content_md = $1
 WHERE id = $2 AND checksum = $3;

-- name: UpdateFileProjectionTrashed :exec
UPDATE file_projection
 SET trashed_at = $1, trashed_by = $2, updated_at = $3, updated_by = $4
 WHERE id = $5;

-- name: UpdateFileProjectionRestored :exec
UPDATE file_projection
 SET trashed_at = NULL, trashed_by = NULL, updated_at = $1, updated_by = $2
 WHERE id = $3;

-- name: UpdateFileProjectionLinked :exec
UPDATE file_projection
 SET linked_file_id = $1, updated_at = $2, updated_by = $3
 WHERE id = $4;

-- name: UpdateFileProjectionUnlinked :exec
UPDATE file_projection
 SET linked_file_id = NULL, updated_at = $1, updated_by = $2
 WHERE id = $3;

-- name: GetRootFileProjectionByName :one
SELECT f.* FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.parent_id IS NULL AND f.name = @name AND f.trashed_at IS NULL;

-- name: GetFileProjectionByParentAndName :one
SELECT f.* FROM file_projection f
 JOIN file_permission_projection p ON p.file_id = f.id AND p.user_account_id = @user_id::uuid
 WHERE f.parent_id = @parent_id AND f.name = @name AND f.trashed_at IS NULL;

-- name: GetFileProjectionByPath :one
-- Resolves a name path to a file the user can access. The recursive walk matches
-- purely on parent/name (trees are single-owner and disjoint per user); the final
-- permission join restricts the result to the requesting user's own subtree.
WITH RECURSIVE path_walk AS (
  SELECT d.id, 1::int AS depth
  FROM file_projection d
  WHERE d.parent_id IS NULL
    AND d.name = (@segments::text[])[1]
    AND d.trashed_at IS NULL
  UNION ALL
  SELECT d.id, pw.depth + 1
  FROM file_projection d
  JOIN path_walk pw ON d.parent_id = pw.id
  WHERE d.name = (@segments::text[])[pw.depth + 1]
    AND d.trashed_at IS NULL
)
SELECT dp.* FROM file_projection dp
JOIN path_walk pw ON dp.id = pw.id
JOIN file_permission_projection p ON p.file_id = dp.id AND p.user_account_id = @user_id::uuid
WHERE pw.depth = array_length(@segments::text[], 1);

-- name: GetFilePath :many
-- Walks ancestors of a file the user can access. The anchor is permission-checked;
-- ancestors share the same owner (single-owner trees) so the chain needs no recheck.
WITH RECURSIVE ancestors(id, parent_id, name, trashed_at, depth) AS (
  SELECT d.id, d.parent_id, d.name, d.trashed_at, 0
  FROM file_projection d
  JOIN file_permission_projection perm ON perm.file_id = d.id AND perm.user_account_id = @user_id::uuid
  WHERE d.id = @file_id
  UNION ALL
  SELECT parent.id, parent.parent_id, parent.name, parent.trashed_at, a.depth + 1
  FROM file_projection parent
  INNER JOIN ancestors a ON parent.id = a.parent_id
  WHERE a.trashed_at IS NULL
)
SELECT id, name, trashed_at FROM ancestors
ORDER BY depth DESC;

-- name: GetFilePathIncludingTrashed :many
WITH RECURSIVE ancestors(id, parent_id, name, depth) AS (
  SELECT d.id, d.parent_id, d.name, 0
  FROM file_projection d
  JOIN file_permission_projection perm ON perm.file_id = d.id AND perm.user_account_id = @user_id::uuid
  WHERE d.id = @file_id
  UNION ALL
  SELECT parent.id, parent.parent_id, parent.name, a.depth + 1
  FROM file_projection parent
  INNER JOIN ancestors a ON parent.id = a.parent_id
)
SELECT id, name FROM ancestors
ORDER BY depth DESC;

-- name: InsertFilePermissionProjection :exec
INSERT INTO file_permission_projection
 (file_id, user_account_id, created_at)
 VALUES (@file_id, @user_account_id, @created_at);

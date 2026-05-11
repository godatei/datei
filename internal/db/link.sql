-- name: InsertLinkProjection :exec
INSERT INTO link_projection
 (id, owner_id, name, key, code, expires_at, created_at, updated_at)
 VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateLinkProjection :exec
UPDATE link_projection
 SET name = $1, code = $2, expires_at = $3, updated_at = $4
 WHERE id = $5;

-- name: UpdateLinkProjectionKey :exec
UPDATE link_projection SET key = $1, updated_at = $2 WHERE id = $3;

-- name: TouchLinkProjection :exec
UPDATE link_projection SET updated_at = $1 WHERE id = $2;

-- name: UpdateLinkProjectionRevoked :exec
UPDATE link_projection SET revoked_at = $1, updated_at = $2 WHERE id = $3;

-- name: IncrementLinkProjectionOpenCount :exec
-- Atomic counter increment driven by the public unlock endpoint. Each
-- successful unlock counts as one access.
UPDATE link_projection SET open_count = open_count + 1, updated_at = $1 WHERE id = $2;

-- name: GetLinkProjectionWithOwnerByID :one
-- Returns the link projection joined with the owner's display name in one
-- round-trip. Used by the public list/download endpoints to populate the
-- response shape without a second user lookup.
SELECT l.*, u.name AS owner_name
FROM link_projection l
INNER JOIN user_account_projection u ON u.id = l.owner_id
WHERE l.id = $1;

-- name: GetLinkProjectionByKey :one
SELECT l.*, u.name AS owner_name
FROM link_projection l
INNER JOIN user_account_projection u ON u.id = l.owner_id
WHERE l.key = $1;

-- name: ListLinkProjectionsByOwner :many
-- status filter values: 'active', 'expired', 'revoked', or '' to return all.
SELECT * FROM link_projection
 WHERE owner_id = @owner_id
   AND CASE @status::text
     WHEN 'active'  THEN revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
     WHEN 'expired' THEN revoked_at IS NULL AND expires_at IS NOT NULL AND expires_at <= NOW()
     WHEN 'revoked' THEN revoked_at IS NOT NULL
     ELSE TRUE
   END
 ORDER BY created_at DESC
 LIMIT @lim OFFSET @off;

-- name: CountLinkProjectionsByOwner :one
-- Total row count for the same filter as ListLinkProjectionsByOwner; used to
-- populate the response's `total` field for pagination.
SELECT COUNT(*)::bigint FROM link_projection
 WHERE owner_id = @owner_id
   AND CASE @status::text
     WHEN 'active'  THEN revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())
     WHEN 'expired' THEN revoked_at IS NULL AND expires_at IS NOT NULL AND expires_at <= NOW()
     WHEN 'revoked' THEN revoked_at IS NOT NULL
     ELSE TRUE
   END;

-- name: InsertLinkDateiProjection :exec
INSERT INTO link_datei_projection (link_id, datei_id, added_at)
 VALUES ($1, $2, $3)
 ON CONFLICT (link_id, datei_id) DO NOTHING;

-- name: DeleteLinkDateiProjection :exec
DELETE FROM link_datei_projection
 WHERE link_id = $1 AND datei_id = $2;

-- name: ListDateienByLink :many
SELECT d.* FROM datei_projection d
 INNER JOIN link_datei_projection ld ON ld.datei_id = d.id
 WHERE ld.link_id = $1 AND d.trashed_at IS NULL
 ORDER BY d.is_directory DESC, d.name ASC;

-- name: IsDateiInLinkScope :one
-- Returns true iff dateiID is one of the link's directly-shared dateien OR is
-- a descendant of any shared directory in the link, AND no ancestor in the
-- chain (up to and including the shared root) is trashed.
WITH RECURSIVE
shared_roots(id) AS (
  SELECT datei_id FROM link_datei_projection WHERE link_id = $1
),
ancestors(id, parent_id, trashed_at, depth) AS (
  SELECT d.id, d.parent_id, d.trashed_at, 0
    FROM datei_projection d
    WHERE d.id = $2
  UNION ALL
  SELECT p.id, p.parent_id, p.trashed_at, a.depth + 1
    FROM datei_projection p
    INNER JOIN ancestors a ON p.id = a.parent_id
)
SELECT EXISTS(
  SELECT 1 FROM ancestors a
   WHERE a.id IN (SELECT id FROM shared_roots)
     AND NOT EXISTS (SELECT 1 FROM ancestors WHERE trashed_at IS NOT NULL)
);

-- name: CountLinkContents :one
-- Recursively counts files and folders reachable from the link's shared roots,
-- including the shared roots themselves. Trashed dateien are excluded. Also
-- returns the link's lifetime open count so the response includes everything
-- the owner needs in a single round-trip.
WITH RECURSIVE
roots AS (
  SELECT d.id, d.is_directory
    FROM datei_projection d
    INNER JOIN link_datei_projection ld ON ld.datei_id = d.id
    WHERE ld.link_id = $1 AND d.trashed_at IS NULL
),
descendants(id, is_directory) AS (
  SELECT id, is_directory FROM roots
  UNION
  SELECT child.id, child.is_directory
    FROM datei_projection child
    INNER JOIN descendants d ON child.parent_id = d.id
    WHERE child.trashed_at IS NULL
)
SELECT
  COALESCE(SUM(CASE WHEN is_directory THEN 0 ELSE 1 END), 0)::bigint AS file_count,
  COALESCE(SUM(CASE WHEN is_directory THEN 1 ELSE 0 END), 0)::bigint AS folder_count,
  (SELECT lp.open_count FROM link_projection lp WHERE lp.id = $1) AS open_count
FROM descendants;

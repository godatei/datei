-- ============================================================================
-- Personal Access Token Projection Writes (called inside event handler TX)
-- ============================================================================

-- name: InsertAccessTokenProjection :exec
INSERT INTO user_account_access_token_projection
  (id, user_account_id, label, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: RevokeAccessTokenProjection :exec
UPDATE user_account_access_token_projection
SET revoked_at = $1
WHERE id = $2 AND revoked_at IS NULL;

-- ============================================================================
-- Personal Access Token Read Queries
-- ============================================================================

-- name: ListAccessTokensForUser :many
-- Active (non-revoked) tokens for the settings list. Expired tokens are still
-- returned so the owner can see and clean them up; the auth path filters expiry
-- separately.
SELECT * FROM user_account_access_token_projection
WHERE user_account_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: GetAccessTokenByID :one
SELECT * FROM user_account_access_token_projection
WHERE id = $1 AND user_account_id = $2;

-- name: GetActiveAccessTokenByHash :one
-- Auth hot path: resolve a presented token to its owning account in a single
-- round-trip, excluding revoked/expired tokens and archived accounts.
SELECT sqlc.embed(ua)
FROM user_account_access_token_projection t
JOIN user_account_projection ua ON ua.id = t.user_account_id
WHERE t.token_hash = $1
  AND t.revoked_at IS NULL
  AND (t.expires_at IS NULL OR t.expires_at > now())
  AND ua.archived_at IS NULL;

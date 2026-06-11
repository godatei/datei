-- ============================================================================
-- User Account Projection Writes (called inside event handler TX)
-- ============================================================================

-- name: InsertUserAccountProjection :exec
INSERT INTO user_account_projection (id, name, password_hash, password_salt, is_admin, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $6);

-- name: UpdateUserAccountProjectionName :exec
UPDATE user_account_projection SET name = $1, updated_at = $2 WHERE id = $3;

-- name: UpdateUserAccountProjectionIsAdmin :exec
UPDATE user_account_projection SET is_admin = $1, updated_at = $2 WHERE id = $3;

-- name: UpdateUserAccountProjectionPassword :exec
UPDATE user_account_projection SET password_hash = $1, password_salt = $2, updated_at = $3 WHERE id = $4;

-- name: UpdateUserAccountProjectionMFASecret :exec
UPDATE user_account_projection SET mfa_secret = $1, updated_at = $2 WHERE id = $3;

-- name: UpdateUserAccountProjectionMFAEnabled :exec
UPDATE user_account_projection SET mfa_enabled = true, mfa_enabled_at = $1, updated_at = $1 WHERE id = $2;

-- name: UpdateUserAccountProjectionMFADisabled :exec
UPDATE user_account_projection SET mfa_enabled = false, mfa_secret = NULL, mfa_enabled_at = NULL, updated_at = $1 WHERE id = $2;

-- name: UpdateUserAccountProjectionArchived :exec
UPDATE user_account_projection SET archived_at = $1, updated_at = $2 WHERE id = $3;

-- name: UpdateUserAccountProjectionLoggedIn :exec
UPDATE user_account_projection SET last_logged_in_at = $1 WHERE id = $2;

-- ============================================================================
-- User Email Projection Writes
-- ============================================================================

-- name: InsertUserAccountEmailProjection :exec
INSERT INTO user_account_email_projection (id, user_account_id, email, is_primary, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateUserAccountEmailProjectionEmail :exec
UPDATE user_account_email_projection SET email = $1, verified_at = NULL
WHERE user_account_id = $2 AND is_primary = true;

-- name: UpdateUserAccountEmailProjectionVerified :exec
UPDATE user_account_email_projection SET verified_at = $1
WHERE user_account_id = $2 AND is_primary = true;

-- name: DeleteUserAccountEmailProjection :exec
DELETE FROM user_account_email_projection WHERE id = $1;

-- name: SetUserAccountEmailPrimaryProjection :exec
UPDATE user_account_email_projection SET is_primary = (id = $1)
WHERE user_account_id = $2;

-- ============================================================================
-- MFA Recovery Code Projection Writes
-- ============================================================================

-- name: InsertMFARecoveryCodeProjection :exec
INSERT INTO user_account_mfa_recovery_code_projection (id, user_account_id, code_hash, code_salt)
VALUES ($1, $2, $3, $4);

-- name: MarkMFARecoveryCodeUsedProjection :exec
UPDATE user_account_mfa_recovery_code_projection SET used_at = now()
WHERE id = $1;

-- name: DeleteAllMFARecoveryCodesProjection :exec
DELETE FROM user_account_mfa_recovery_code_projection
WHERE user_account_id = $1;

-- ============================================================================
-- Read Queries (for handlers, outside TX)
-- ============================================================================

-- name: GetUserAccountByID :one
SELECT * FROM user_account_projection WHERE id = $1;

-- name: ListUserAccountProjections :many
SELECT
  ua.id,
  ua.name,
  ua.is_admin,
  ua.mfa_enabled,
  ua.archived_at,
  ua.created_at,
  ua.last_logged_in_at,
  ue.email AS primary_email,
  ue.verified_at AS primary_email_verified_at
FROM user_account_projection ua
LEFT JOIN user_account_email_projection ue
  ON ue.user_account_id = ua.id AND ue.is_primary = true
ORDER BY (ua.archived_at IS NOT NULL), NOT ua.is_admin, ua.name ASC
LIMIT $1 OFFSET $2;

-- name: CountUserAccountProjections :one
SELECT COUNT(*) FROM user_account_projection;

-- name: GetUserAccountByEmail :one
SELECT ua.* FROM user_account_projection ua
JOIN user_account_email_projection ue ON ue.user_account_id = ua.id
WHERE ue.email = $1 AND ua.archived_at IS NULL;

-- name: GetPrimaryEmailForUser :one
SELECT * FROM user_account_email_projection
WHERE user_account_id = $1 AND is_primary = true;

-- name: GetEmailsForUser :many
SELECT * FROM user_account_email_projection
WHERE user_account_id = $1
ORDER BY is_primary DESC, created_at;

-- name: GetEmailByID :one
SELECT * FROM user_account_email_projection
WHERE id = $1 AND user_account_id = $2;

-- name: UserAccountEmailExists :one
SELECT EXISTS(SELECT 1 FROM user_account_email_projection WHERE email = $1);

-- name: GetUnusedMFARecoveryCodes :many
SELECT * FROM user_account_mfa_recovery_code_projection
WHERE user_account_id = $1 AND used_at IS NULL;

-- name: CountUnusedMFARecoveryCodes :one
SELECT COUNT(*)::int FROM user_account_mfa_recovery_code_projection
WHERE user_account_id = $1 AND used_at IS NULL;

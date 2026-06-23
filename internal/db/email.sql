-- ============================================================================
-- Mail Account Projection
-- ============================================================================

-- name: InsertMailAccountProjection :exec
INSERT INTO mail_account_projection
 (id, owner_id, name, imap_host, imap_port, username, password_encrypted, security, created_at, updated_at)
 VALUES (@id, @owner_id, @name, @imap_host, @imap_port, @username, @password_encrypted, @security, @created_at, @updated_at);

-- name: UpdateMailAccountProjection :exec
UPDATE mail_account_projection
 SET name = @name,
     imap_host = @imap_host,
     imap_port = @imap_port,
     username = @username,
     password_encrypted = @password_encrypted,
     security = @security,
     updated_at = @updated_at
 WHERE id = @id;

-- name: DeleteMailAccountProjection :exec
DELETE FROM mail_account_projection WHERE id = @id;

-- name: GetMailAccountProjection :one
SELECT * FROM mail_account_projection WHERE id = @id;

-- name: GetMailAccountProjectionForOwner :one
SELECT * FROM mail_account_projection WHERE id = @id AND owner_id = @owner_id;

-- name: ListMailAccountProjectionsByOwner :many
SELECT * FROM mail_account_projection
 WHERE owner_id = @owner_id
 ORDER BY name ASC
 LIMIT @lim OFFSET @off;

-- name: CountMailAccountProjectionsByOwner :one
SELECT COUNT(*) FROM mail_account_projection WHERE owner_id = @owner_id;

-- name: ListMailAccountProjectionsAll :many
SELECT * FROM mail_account_projection ORDER BY id ASC;

-- ============================================================================
-- Mail Rule Projection
-- ============================================================================

-- name: InsertMailRuleProjection :exec
INSERT INTO mail_rule_projection
 (id, account_id, name, sort_order, enabled, folder, filter_from, filter_subject,
  max_age_days, attachment_pattern, action, target_directory_id, created_at, updated_at)
 VALUES (@id, @account_id, @name, @sort_order, @enabled, @folder, @filter_from, @filter_subject,
  @max_age_days, @attachment_pattern, @action, @target_directory_id, @created_at, @updated_at);

-- name: UpdateMailRuleProjection :exec
UPDATE mail_rule_projection
 SET account_id = @account_id,
     name = @name,
     sort_order = @sort_order,
     enabled = @enabled,
     folder = @folder,
     filter_from = @filter_from,
     filter_subject = @filter_subject,
     max_age_days = @max_age_days,
     attachment_pattern = @attachment_pattern,
     action = @action,
     target_directory_id = @target_directory_id,
     updated_at = @updated_at
 WHERE id = @id;

-- name: DeleteMailRuleProjection :exec
DELETE FROM mail_rule_projection WHERE id = @id;

-- name: GetMailRuleProjection :one
SELECT * FROM mail_rule_projection WHERE id = @id;

-- name: ListMailRuleProjectionsByOwner :many
SELECT r.* FROM mail_rule_projection r
 JOIN mail_account_projection a ON a.id = r.account_id
 WHERE a.owner_id = @owner_id
 ORDER BY a.name ASC, r.sort_order ASC, r.created_at ASC
 LIMIT @lim OFFSET @off;

-- name: CountMailRuleProjectionsByOwner :one
SELECT COUNT(*) FROM mail_rule_projection r
 JOIN mail_account_projection a ON a.id = r.account_id
 WHERE a.owner_id = @owner_id;

-- name: ListEnabledMailRuleProjectionsByAccount :many
SELECT * FROM mail_rule_projection
 WHERE account_id = @account_id AND enabled = true
 ORDER BY sort_order ASC, created_at ASC;

-- ============================================================================
-- Processed Message Bookkeeping
-- ============================================================================

-- name: InsertProcessedMessage :exec
INSERT INTO mail_processed_message (account_id, folder, uid_validity, imap_uid)
 VALUES (@account_id, @folder, @uid_validity, @imap_uid)
 ON CONFLICT (account_id, folder, uid_validity, imap_uid) DO NOTHING;

-- name: ProcessedMessageExists :one
SELECT EXISTS (
  SELECT 1 FROM mail_processed_message
   WHERE account_id = @account_id AND folder = @folder
     AND uid_validity = @uid_validity AND imap_uid = @imap_uid
);

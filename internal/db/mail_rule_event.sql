-- name: GetMailRuleStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM mail_rule_event WHERE stream_id = $1;

-- name: InsertMailRuleEvent :exec
INSERT INTO mail_rule_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetMailRuleEventsByStreamID :many
SELECT * FROM mail_rule_event
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

-- name: GetMailAccountStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM mail_account_event WHERE stream_id = $1;

-- name: InsertMailAccountEvent :exec
INSERT INTO mail_account_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetMailAccountEventsByStreamID :many
SELECT * FROM mail_account_event
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

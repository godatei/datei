-- name: GetUserAccountStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM user_account_event WHERE stream_id = $1;

-- name: InsertUserAccountEvent :exec
INSERT INTO user_account_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetUserAccountEventsByStreamID :many
SELECT * FROM user_account_event
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

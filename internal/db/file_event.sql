-- name: GetStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM file_event WHERE stream_id = $1;

-- name: InsertFileEvent :exec
INSERT INTO file_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetFileEventsByStreamID :many
SELECT * FROM file_event
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

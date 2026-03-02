-- name: GetStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM event_store WHERE stream_id = $1;

-- name: InsertEventStoreEvent :exec
INSERT INTO event_store (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetEventsByStreamID :many
SELECT event_type, event_data FROM event_store
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

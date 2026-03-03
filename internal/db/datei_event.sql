-- name: GetStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM datei_event WHERE stream_id = $1;

-- name: InsertDateiEvent :exec
INSERT INTO datei_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetDateiEventsByStreamID :many
SELECT * FROM datei_event
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

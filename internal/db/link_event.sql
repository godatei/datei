-- name: GetLinkStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM link_event WHERE stream_id = $1;

-- name: InsertLinkEvent :exec
INSERT INTO link_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES ($1, $2, $3, $4, NOW());

-- name: GetLinkEventsByStreamID :many
SELECT * FROM link_event
 WHERE stream_id = $1 AND stream_version >= $2
 ORDER BY stream_version ASC;

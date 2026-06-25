-- name: GetMailAccountStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM mail_account_event WHERE stream_id = @stream_id;

-- name: InsertMailAccountEvent :exec
INSERT INTO mail_account_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES (@stream_id, @stream_version, @event_type, @event_data, NOW());

-- name: GetMailAccountEventsByStreamID :many
SELECT * FROM mail_account_event
 WHERE stream_id = @stream_id AND stream_version >= @stream_version
 ORDER BY stream_version ASC;

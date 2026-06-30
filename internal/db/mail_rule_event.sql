-- name: GetMailRuleStreamVersion :one
SELECT COALESCE(MAX(stream_version), 0)::int FROM mail_rule_event WHERE stream_id = @stream_id;

-- name: InsertMailRuleEvent :exec
INSERT INTO mail_rule_event (stream_id, stream_version, event_type, event_data, created_at)
 VALUES (@stream_id, @stream_version, @event_type, @event_data, NOW());

-- name: GetMailRuleEventsByStreamID :many
SELECT * FROM mail_rule_event
 WHERE stream_id = @stream_id AND stream_version >= @stream_version
 ORDER BY stream_version ASC;

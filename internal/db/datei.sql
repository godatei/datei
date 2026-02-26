-- name: CreateDatei :one
INSERT INTO datei (is_directory) VALUES ($1) RETURNING *;

-- name: CreateDateiName :one
INSERT INTO datei_name (datei_id, name) VALUES ($1, $2) RETURNING *;

-- name: CreateDateiVersion :one
INSERT INTO datei_version (datei_id, s3_key, file_size, checksum, mime_type) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: UpdateDateiLatestNameID :one
UPDATE datei SET latest_name_id = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: UpdateDateiLatestVersionID :one
UPDATE datei SET latest_version_id = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: SetDateiTrashedAt :one
UPDATE datei SET trashed_at = now(), updated_at = now() WHERE id = $1 RETURNING *;

-- name: GetDateiByID :one
SELECT * FROM datei WHERE id = $1;

-- name: GetDateiByIDWithDetails :one
SELECT sqlc.embed(d), sqlc.embed(ln), sqlc.embed(lv)
FROM datei d
LEFT JOIN datei_name ln ON d.latest_name_id = ln.id
LEFT JOIN datei_version lv ON d.latest_version_id = lv.id
WHERE d.id = $1;

-- name: ListDateiWithDetails :many
SELECT sqlc.embed(d), sqlc.embed(ln), sqlc.embed(lv)
FROM datei d
LEFT JOIN datei_name ln ON d.latest_name_id = ln.id
LEFT JOIN datei_version lv ON d.latest_version_id = lv.id
ORDER BY d.created_at DESC;

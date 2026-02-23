-- name: ListUserAccounts :many
SELECT * from user_account ORDER BY name; 

-- name: GetUserAccountByID :one
SELECT * FROM user_account WHERE id = $1;

-- name: CreateUserAccount :one
INSERT INTO user_account (name) VALUES ($1) RETURNING *;

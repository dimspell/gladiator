-- name: GetAuthor :one
SELECT *
FROM users
WHERE id = ? LIMIT 1;

-- name: ListAuthors :many
SELECT *
FROM users
ORDER BY username;

-- name: CreateAuthor :one
INSERT INTO users (username, password)
VALUES (?, ?) RETURNING *;

-- name: UpdateAuthor :exec
UPDATE users
set username = ?,
    password  = ?
WHERE id = ?;

-- name: DeleteAuthor :exec
DELETE
FROM users
WHERE id = ?;

-- name: GetUser :one
SELECT *
FROM users
WHERE username = ? LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (username, password)
VALUES (?, ?) RETURNING *;

-- name: CreateGameRoom :one
INSERT INTO game_rooms (name, password, host_ip_address)
VALUES (?, ?, ?) RETURNING *;

-- name: ListGameRooms :many
SELECT *
FROM game_rooms;

-- name: ListCharacters :many
SELECT *
FROM characters
WHERE user_id = ?
ORDER BY slot_order;
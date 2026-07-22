-- name: CreateChirp :one
INSERT INTO chirp (id, created_at, updated_at, body, userId) values (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: DeleteAllChirps :exec
TRUNCATE TABLE chirp;
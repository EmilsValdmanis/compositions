-- name: UpsertUser :exec
INSERT INTO users (
    id,
    name,
    email,
    image_url
)
VALUES (
    $1,
    $2,
    $3,
    $4
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    image_url = EXCLUDED.image_url,
    updated_at = NOW();

-- name: GetUserByID :one
SELECT id, name, email, image_url, created_at, updated_at
FROM users
WHERE id = $1;

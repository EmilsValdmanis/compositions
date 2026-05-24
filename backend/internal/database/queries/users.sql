-- name: UpsertUser :exec
WITH updated_by_id AS (
    UPDATE users
    SET
        name = $2,
        email = $3,
        image_url = $4,
        updated_at = NOW()
    WHERE id = $1
    RETURNING id
)
INSERT INTO users (
    id,
    name,
    email,
    image_url
)
SELECT
    $1,
    $2,
    $3,
    $4
WHERE NOT EXISTS (
    SELECT 1
    FROM updated_by_id
)
ON CONFLICT (LOWER(email)) WHERE email <> '' DO UPDATE SET
    id = EXCLUDED.id,
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    image_url = EXCLUDED.image_url,
    updated_at = NOW();

-- name: GetUserByID :one
SELECT id, name, email, image_url, created_at, updated_at
FROM users
WHERE id = $1;

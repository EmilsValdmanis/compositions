-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    image_url
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING
    id::text AS id,
    name,
    email,
    image_url,
    created_at,
    updated_at;

-- name: CreateAccount :exec
INSERT INTO accounts (
    user_id,
    provider,
    provider_account_id
)
VALUES (
    sqlc.arg(user_id)::uuid,
    sqlc.arg(provider),
    sqlc.arg(provider_account_id)
);

-- name: GetUserByAccount :one
SELECT
    users.id::text AS id,
    users.name,
    users.email,
    users.image_url,
    users.created_at,
    users.updated_at
FROM accounts
JOIN users ON users.id = accounts.user_id
WHERE accounts.provider = $1
  AND accounts.provider_account_id = $2;

-- name: GetUserByEmail :one
SELECT
    id::text AS id,
    name,
    email,
    image_url,
    created_at,
    updated_at
FROM users
WHERE LOWER(email) = LOWER($1)
  AND email <> '';

-- name: UpsertUserByID :one
INSERT INTO users (
    id,
    name,
    email,
    image_url
)
VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(name),
    sqlc.arg(email),
    sqlc.arg(image_url)
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    email = EXCLUDED.email,
    image_url = EXCLUDED.image_url,
    updated_at = NOW()
RETURNING
    id::text AS id,
    name,
    email,
    image_url,
    created_at,
    updated_at;

-- name: GetUserByID :one
SELECT
    id::text AS id,
    name,
    email,
    image_url,
    created_at,
    updated_at
FROM users
WHERE id = sqlc.arg(id)::uuid;

-- name: UpdateUserByID :exec
UPDATE users
SET
    name = sqlc.arg(name),
    email = sqlc.arg(email),
    image_url = sqlc.arg(image_url),
    updated_at = NOW()
WHERE id = sqlc.arg(id)::uuid;

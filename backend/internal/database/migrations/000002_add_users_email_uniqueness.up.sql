UPDATE users
SET
    name = BTRIM(name),
    email = LOWER(BTRIM(email)),
    image_url = BTRIM(image_url)
WHERE
    name <> BTRIM(name)
    OR email <> LOWER(BTRIM(email))
    OR image_url <> BTRIM(image_url);

WITH ranked_users AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY LOWER(email)
            ORDER BY updated_at DESC, created_at DESC, id DESC
        ) AS duplicate_rank
    FROM users
    WHERE email <> ''
)
DELETE FROM users
WHERE id IN (
    SELECT id
    FROM ranked_users
    WHERE duplicate_rank > 1
);

CREATE UNIQUE INDEX users_email_key ON users (LOWER(email)) WHERE email <> '';

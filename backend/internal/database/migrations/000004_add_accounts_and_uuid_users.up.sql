CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE users RENAME COLUMN id TO legacy_id;
ALTER TABLE users ADD COLUMN id UUID;

UPDATE users
SET id = gen_random_uuid()
WHERE id IS NULL;

ALTER TABLE users ALTER COLUMN id SET NOT NULL;
ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE sessions DROP CONSTRAINT sessions_user_id_fkey;
DROP INDEX IF EXISTS sessions_user_id_idx;

ALTER TABLE sessions RENAME COLUMN user_id TO legacy_user_id;
ALTER TABLE sessions ADD COLUMN user_id UUID;

UPDATE sessions AS sessions_table
SET user_id = users.id
FROM users
WHERE users.legacy_id = sessions_table.legacy_user_id;

ALTER TABLE users DROP CONSTRAINT users_pkey;
ALTER TABLE users ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE sessions ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE sessions DROP COLUMN legacy_user_id;
ALTER TABLE sessions ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_account_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX accounts_provider_account_key ON accounts (provider, provider_account_id);
CREATE INDEX accounts_user_id_idx ON accounts (user_id);

INSERT INTO accounts (
    user_id,
    provider,
    provider_account_id,
    created_at,
    updated_at
)
SELECT
    users.id,
    CASE
        WHEN users.legacy_id LIKE 'google_%' THEN 'google'
        ELSE 'legacy'
    END,
    CASE
        WHEN users.legacy_id LIKE 'google_%' THEN SUBSTRING(users.legacy_id FROM 8)
        ELSE users.legacy_id
    END,
    users.created_at,
    users.updated_at
FROM users;

ALTER TABLE users DROP COLUMN legacy_id;

ALTER TABLE users ADD COLUMN legacy_id TEXT;

WITH preferred_accounts AS (
    SELECT DISTINCT ON (accounts.user_id)
        accounts.user_id,
        CASE
            WHEN accounts.provider = 'google' THEN 'google_' || accounts.provider_account_id
            ELSE accounts.provider || '_' || accounts.provider_account_id
        END AS legacy_id
    FROM accounts
    ORDER BY accounts.user_id, accounts.created_at, accounts.id
)
UPDATE users
SET legacy_id = COALESCE(preferred_accounts.legacy_id, users.id::text)
FROM preferred_accounts
WHERE preferred_accounts.user_id = users.id;

UPDATE users
SET legacy_id = id::text
WHERE legacy_id IS NULL;

ALTER TABLE users ADD CONSTRAINT users_legacy_id_key UNIQUE (legacy_id);

ALTER TABLE sessions DROP CONSTRAINT sessions_user_id_fkey;
DROP INDEX IF EXISTS sessions_user_id_idx;

ALTER TABLE sessions ADD COLUMN legacy_user_id TEXT;

UPDATE sessions AS sessions_table
SET legacy_user_id = users.legacy_id
FROM users
WHERE users.id = sessions_table.user_id;

ALTER TABLE sessions ALTER COLUMN legacy_user_id SET NOT NULL;
ALTER TABLE sessions DROP COLUMN user_id;
ALTER TABLE sessions RENAME COLUMN legacy_user_id TO user_id;

DROP TABLE accounts;

ALTER TABLE users DROP CONSTRAINT users_pkey;
ALTER TABLE users DROP CONSTRAINT users_legacy_id_key;
ALTER TABLE users DROP COLUMN id;
ALTER TABLE users RENAME COLUMN legacy_id TO id;
ALTER TABLE users ADD CONSTRAINT users_pkey PRIMARY KEY (id);

ALTER TABLE sessions ADD CONSTRAINT sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

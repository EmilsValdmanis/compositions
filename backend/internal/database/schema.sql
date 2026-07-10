CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_key ON users (LOWER(email)) WHERE email <> '';

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

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE lobby_state (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    state JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lobby_state_singleton CHECK (id)
);

CREATE TABLE game_bug_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_code TEXT NOT NULL,
    reporter_player_id TEXT NOT NULL,
    reporter_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    description TEXT NOT NULL,
    game_state JSONB NOT NULL,
    round INTEGER NOT NULL,
    turn INTEGER NOT NULL,
    requested_abort BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT game_bug_reports_room_code_not_blank CHECK (BTRIM(room_code) <> ''),
    CONSTRAINT game_bug_reports_reporter_not_blank CHECK (BTRIM(reporter_player_id) <> ''),
    CONSTRAINT game_bug_reports_description_length CHECK (
        BTRIM(description) <> '' AND CHAR_LENGTH(description) <= 500
    ),
    CONSTRAINT game_bug_reports_round_positive CHECK (round > 0),
    CONSTRAINT game_bug_reports_turn_positive CHECK (turn > 0)
);

CREATE INDEX game_bug_reports_created_at_idx ON game_bug_reports (created_at DESC);
CREATE INDEX game_bug_reports_room_code_idx ON game_bug_reports (room_code, created_at DESC);

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

CREATE TABLE friendships (
    user_a_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_a_id, user_b_id),
    CONSTRAINT friendships_canonical_order CHECK (user_a_id < user_b_id)
);

CREATE INDEX friendships_user_b_id_idx ON friendships (user_b_id);

CREATE TABLE friend_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT friend_requests_different_users CHECK (sender_id <> recipient_id)
);

CREATE UNIQUE INDEX friend_requests_pair_key
    ON friend_requests (LEAST(sender_id, recipient_id), GREATEST(sender_id, recipient_id));
CREATE INDEX friend_requests_recipient_created_idx
    ON friend_requests (recipient_id, created_at DESC);
CREATE INDEX friend_requests_sender_idx ON friend_requests (sender_id);

CREATE TABLE game_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_code TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT game_invites_different_users CHECK (sender_id <> recipient_id),
    CONSTRAINT game_invites_room_code_not_blank CHECK (BTRIM(room_code) <> ''),
    CONSTRAINT game_invites_expiry_after_creation CHECK (expires_at > created_at),
    UNIQUE (sender_id, recipient_id, room_code)
);

CREATE INDEX game_invites_recipient_expiry_idx
    ON game_invites (recipient_id, expires_at, created_at DESC);

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

CREATE TABLE games (
    id UUID PRIMARY KEY,
    room_code TEXT NOT NULL,
    status TEXT NOT NULL,
    rounds_played INTEGER NOT NULL,
    player_count INTEGER NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    active_playtime_seconds BIGINT NOT NULL DEFAULT 0,
	game_mode TEXT NOT NULL DEFAULT 'full',
	ranked BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE game_player_statistics (
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    placement INTEGER,
    won BOOLEAN,
    forfeited BOOLEAN NOT NULL,
    total_points INTEGER NOT NULL,
    rounds_played INTEGER NOT NULL,
    rounds_won INTEGER NOT NULL,
    same_suit_wins INTEGER NOT NULL,
    six_pairs_wins INTEGER NOT NULL,
    turns_taken INTEGER NOT NULL,
    cards_drawn_from_deck INTEGER NOT NULL,
    cards_drawn_from_discard INTEGER NOT NULL,
    cards_discarded INTEGER NOT NULL,
    cards_played INTEGER NOT NULL,
    compositions_created INTEGER NOT NULL,
    sets_created INTEGER NOT NULL,
    runs_created INTEGER NOT NULL,
    additions_done INTEGER NOT NULL,
    compositions_completed INTEGER NOT NULL,
    sets_completed INTEGER NOT NULL,
    runs_completed INTEGER NOT NULL,
    jokers_played INTEGER NOT NULL,
    jokers_reclaimed INTEGER NOT NULL,
    cards_remaining INTEGER NOT NULL,
    hand_points INTEGER NOT NULL,
    penalty_points INTEGER NOT NULL,
    points_inflicted INTEGER NOT NULL,
    largest_round_penalty INTEGER NOT NULL,
    largest_round_points_inflicted INTEGER NOT NULL,
    most_cards_remaining INTEGER NOT NULL,
    rounds_opened INTEGER NOT NULL,
    fastest_opening_turn INTEGER NOT NULL,
    starting_round_win_streak INTEGER NOT NULL,
    ending_round_win_streak INTEGER NOT NULL,
    longest_round_win_streak INTEGER NOT NULL,
    PRIMARY KEY (game_id, user_id)
);

CREATE TABLE player_statistics (
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	game_mode TEXT NOT NULL DEFAULT 'full',
	ranked BOOLEAN NOT NULL DEFAULT TRUE,
    games_played BIGINT NOT NULL DEFAULT 0,
    games_won BIGINT NOT NULL DEFAULT 0,
    total_placement BIGINT NOT NULL DEFAULT 0,
    rounds_played BIGINT NOT NULL DEFAULT 0,
    rounds_won BIGINT NOT NULL DEFAULT 0,
    same_suit_wins BIGINT NOT NULL DEFAULT 0,
    six_pairs_wins BIGINT NOT NULL DEFAULT 0,
    forfeits BIGINT NOT NULL DEFAULT 0,
    turns_taken BIGINT NOT NULL DEFAULT 0,
    cards_drawn_from_deck BIGINT NOT NULL DEFAULT 0,
    cards_drawn_from_discard BIGINT NOT NULL DEFAULT 0,
    cards_discarded BIGINT NOT NULL DEFAULT 0,
    cards_played BIGINT NOT NULL DEFAULT 0,
    compositions_created BIGINT NOT NULL DEFAULT 0,
    sets_created BIGINT NOT NULL DEFAULT 0,
    runs_created BIGINT NOT NULL DEFAULT 0,
    additions_done BIGINT NOT NULL DEFAULT 0,
    compositions_completed BIGINT NOT NULL DEFAULT 0,
    sets_completed BIGINT NOT NULL DEFAULT 0,
    runs_completed BIGINT NOT NULL DEFAULT 0,
    jokers_played BIGINT NOT NULL DEFAULT 0,
    jokers_reclaimed BIGINT NOT NULL DEFAULT 0,
    cards_remaining BIGINT NOT NULL DEFAULT 0,
    hand_points BIGINT NOT NULL DEFAULT 0,
    penalty_points BIGINT NOT NULL DEFAULT 0,
    points_inflicted BIGINT NOT NULL DEFAULT 0,
    largest_round_penalty INTEGER NOT NULL DEFAULT 0,
    largest_round_points_inflicted INTEGER NOT NULL DEFAULT 0,
    most_cards_remaining INTEGER NOT NULL DEFAULT 0,
    rounds_opened BIGINT NOT NULL DEFAULT 0,
    fastest_opening_turn INTEGER NOT NULL DEFAULT 0,
    current_game_win_streak INTEGER NOT NULL DEFAULT 0,
    longest_game_win_streak INTEGER NOT NULL DEFAULT 0,
    current_round_win_streak INTEGER NOT NULL DEFAULT 0,
    longest_round_win_streak INTEGER NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (user_id, game_mode, ranked)
);

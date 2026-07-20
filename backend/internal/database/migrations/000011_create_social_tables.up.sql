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

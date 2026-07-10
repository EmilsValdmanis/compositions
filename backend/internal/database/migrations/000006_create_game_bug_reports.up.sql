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

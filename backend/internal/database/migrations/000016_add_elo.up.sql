-- Elo starts fresh at launch; historical statistics are not replayed.
CREATE TABLE player_ratings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL DEFAULT 1000 CHECK (rating >= 0),
    games_played BIGINT NOT NULL DEFAULT 0 CHECK (games_played >= 0)
);

CREATE TABLE game_rating_changes (
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating_before INTEGER NOT NULL CHECK (rating_before >= 0),
    rating_after INTEGER NOT NULL CHECK (rating_after >= 0),
    PRIMARY KEY (game_id, user_id)
);

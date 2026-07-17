ALTER TABLE games
    ADD COLUMN game_mode TEXT NOT NULL DEFAULT 'full',
    ADD COLUMN ranked BOOLEAN NOT NULL DEFAULT TRUE,
    ADD CONSTRAINT games_game_mode_check CHECK (game_mode IN ('quick', 'full'));

ALTER TABLE player_statistics
    ADD COLUMN game_mode TEXT NOT NULL DEFAULT 'full',
    ADD COLUMN ranked BOOLEAN NOT NULL DEFAULT TRUE,
    DROP CONSTRAINT player_statistics_pkey,
    ADD CONSTRAINT player_statistics_game_mode_check CHECK (game_mode IN ('quick', 'full')),
    ADD PRIMARY KEY (user_id, game_mode, ranked);

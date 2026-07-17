DELETE FROM player_statistics
WHERE game_mode <> 'full' OR NOT ranked;

ALTER TABLE player_statistics
    DROP CONSTRAINT player_statistics_pkey,
    DROP CONSTRAINT player_statistics_game_mode_check,
    DROP COLUMN game_mode,
    DROP COLUMN ranked,
    ADD PRIMARY KEY (user_id);

ALTER TABLE games
    DROP CONSTRAINT games_game_mode_check,
    DROP COLUMN game_mode,
    DROP COLUMN ranked;

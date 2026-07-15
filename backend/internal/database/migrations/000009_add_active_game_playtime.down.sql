ALTER TABLE games
    DROP CONSTRAINT games_active_playtime_nonnegative,
    DROP COLUMN active_playtime_seconds;

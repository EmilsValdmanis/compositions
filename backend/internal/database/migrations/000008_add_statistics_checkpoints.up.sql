ALTER TABLE games ADD COLUMN status TEXT;

UPDATE games
SET status = CASE completion_kind
    WHEN 'normal' THEN 'completed'
    ELSE 'forfeit'
END;

ALTER TABLE games
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN completed_at DROP NOT NULL,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    DROP CONSTRAINT games_completion_kind_check,
    DROP COLUMN completion_kind,
    ADD CONSTRAINT games_status_check CHECK (
        status IN ('in_progress', 'completed', 'forfeit', 'mutual_end', 'technical_abort', 'abandoned')
    );

DROP INDEX games_completed_at_idx;
CREATE INDEX games_status_completed_at_idx ON games (status, completed_at DESC);

ALTER TABLE game_player_statistics
    ALTER COLUMN placement DROP NOT NULL,
    ALTER COLUMN won DROP NOT NULL,
    DROP CONSTRAINT game_player_statistics_placement_positive,
    ADD CONSTRAINT game_player_statistics_placement_positive CHECK (placement IS NULL OR placement > 0);

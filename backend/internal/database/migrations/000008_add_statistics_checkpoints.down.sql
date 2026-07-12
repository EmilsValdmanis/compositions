-- In-progress and unranked rows did not exist before checkpoint support and
-- cannot be represented by the old non-null outcome columns.
DELETE FROM games WHERE status NOT IN ('completed', 'forfeit');

DROP INDEX games_status_completed_at_idx;

ALTER TABLE games ADD COLUMN completion_kind TEXT;

UPDATE games
SET completion_kind = CASE status
    WHEN 'completed' THEN 'normal'
    ELSE 'forfeit'
END;

ALTER TABLE games
    ALTER COLUMN completion_kind SET NOT NULL,
    ALTER COLUMN completed_at SET NOT NULL,
    DROP CONSTRAINT games_status_check,
    DROP COLUMN status,
    DROP COLUMN updated_at,
    ADD CONSTRAINT games_completion_kind_check CHECK (completion_kind IN ('normal', 'forfeit'));

CREATE INDEX games_completed_at_idx ON games (completed_at DESC);

ALTER TABLE game_player_statistics
    ALTER COLUMN placement SET NOT NULL,
    ALTER COLUMN won SET NOT NULL,
    DROP CONSTRAINT game_player_statistics_placement_positive,
    ADD CONSTRAINT game_player_statistics_placement_positive CHECK (placement > 0);

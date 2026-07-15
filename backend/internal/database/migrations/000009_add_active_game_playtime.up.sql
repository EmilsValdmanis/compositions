ALTER TABLE games
    ADD COLUMN active_playtime_seconds BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT games_active_playtime_nonnegative CHECK (active_playtime_seconds >= 0);

-- Exact active intervals were not recorded before this migration. Preserve
-- plausible completed history, but discard 12+ hour legacy sessions because
-- they cannot be separated from overnight pauses. All new and resumed games
-- overwrite this field with their explicitly accumulated active time.
UPDATE games
SET active_playtime_seconds = CASE
	WHEN completed_at - started_at < INTERVAL '12 hours' THEN GREATEST(
		0,
		EXTRACT(EPOCH FROM (completed_at - started_at))::bigint
	)
	ELSE 0
END
WHERE completed_at IS NOT NULL;

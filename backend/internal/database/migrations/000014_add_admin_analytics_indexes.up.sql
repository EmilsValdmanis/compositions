CREATE INDEX games_started_at_idx ON games (started_at DESC);

CREATE INDEX game_bug_reports_completed_at_idx
ON game_bug_reports (completed_at DESC)
WHERE completed_at IS NOT NULL;

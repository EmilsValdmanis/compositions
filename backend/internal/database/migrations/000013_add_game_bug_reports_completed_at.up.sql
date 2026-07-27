ALTER TABLE game_bug_reports
ADD COLUMN completed_at TIMESTAMPTZ;

CREATE INDEX game_bug_reports_open_created_at_idx
ON game_bug_reports (created_at DESC)
WHERE completed_at IS NULL;

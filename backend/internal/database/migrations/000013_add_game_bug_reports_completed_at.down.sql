DROP INDEX IF EXISTS game_bug_reports_open_created_at_idx;

ALTER TABLE game_bug_reports
DROP COLUMN IF EXISTS completed_at;

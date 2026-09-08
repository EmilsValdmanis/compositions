-- Existing changes were all calculated with the original rules.
ALTER TABLE game_rating_changes
    ADD COLUMN rules_version TEXT NOT NULL DEFAULT 'elo-v1' CHECK (rules_version <> '');
-- Require writers to identify their rules explicitly after the backfill.
ALTER TABLE game_rating_changes ALTER COLUMN rules_version DROP DEFAULT;
-- The application owns the starting rating; avoid a second tunable default.
ALTER TABLE player_ratings ALTER COLUMN rating DROP DEFAULT;

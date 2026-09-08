ALTER TABLE game_rating_changes DROP COLUMN rules_version;
ALTER TABLE player_ratings ALTER COLUMN rating SET DEFAULT 1000;

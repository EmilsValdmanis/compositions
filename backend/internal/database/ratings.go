package database

import (
	"context"

	"github.com/EmilsValdmanis/compositions/internal/rating"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Called before lifetime updates, with players sorted by canonical UUID. All
// participating ratings are locked before calculating any changes, preventing
// lost updates and lock-order deadlocks between overlapping games.
func saveRatings(ctx context.Context, tx pgx.Tx, gameID pgtype.UUID, players []CompletedGamePlayerRecord) error {
	rules := rating.Current()
	before := make([]rating.Player, len(players))
	for i, player := range players {
		if _, err := tx.Exec(ctx, `INSERT INTO player_ratings (user_id, rating) VALUES ($1, $2) ON CONFLICT DO NOTHING`, player.UserID, rules.Initial); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT rating FROM player_ratings WHERE user_id = $1 FOR UPDATE`, player.UserID).Scan(&before[i].Rating); err != nil {
			return err
		}
		before[i].Placement = player.Placement
	}
	after, err := rules.Calculate(before)
	if err != nil {
		return err
	}
	for i, player := range players {
		if _, err := tx.Exec(ctx, `INSERT INTO game_rating_changes (game_id, user_id, rating_before, rating_after, rules_version) VALUES ($1, $2, $3, $4, $5)`, gameID, player.UserID, before[i].Rating, after[i], rules.Version); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE player_ratings SET rating = $2, games_played = games_played + 1 WHERE user_id = $1`, player.UserID, after[i]); err != nil {
			return err
		}
	}
	return nil
}

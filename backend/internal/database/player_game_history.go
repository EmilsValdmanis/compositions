package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type PlayerGameHistoryRecord struct {
	GameID, Status                       string
	CompletedAt                          time.Time
	Placement, PlayerCount               int
	Won, Forfeited                       bool
	TotalPoints, RoundsPlayed, RoundsWon int
	PlaytimeSeconds                      int64
}

type PlayerGameHistoryPage struct {
	Games []PlayerGameHistoryRecord
	Total int64
}

// GetPlayerGameHistory returns ranked games newest first. Limit and offset are
// supplied by the HTTP layer, which bounds them before reaching the database.
func (s *UserStore) GetPlayerGameHistory(ctx context.Context, userID string, limit, offset int) (PlayerGameHistoryPage, error) {
	if s == nil || s.pool == nil {
		return PlayerGameHistoryPage{}, errors.New("user store is not configured")
	}
	if limit <= 0 || offset < 0 {
		return PlayerGameHistoryPage{}, errors.New("invalid game history pagination")
	}

	uuidID, err := parseUUID(userID)
	if err != nil {
		return PlayerGameHistoryPage{}, err
	}

	var total int64
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(g.id)
		FROM users u
		LEFT JOIN game_player_statistics gps ON gps.user_id = u.id
		LEFT JOIN games g ON g.id = gps.game_id
			AND g.status IN ('completed', 'forfeit')
			AND g.completed_at IS NOT NULL
		WHERE u.id = $1
		GROUP BY u.id
	`, uuidID).Scan(&total)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlayerGameHistoryPage{}, ErrPlayerProfileNotFound
	}
	if err != nil {
		return PlayerGameHistoryPage{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT g.id::text, g.status, g.completed_at,
			gps.placement, g.player_count, gps.won, gps.forfeited,
			gps.total_points, gps.rounds_played, gps.rounds_won,
			g.active_playtime_seconds
		FROM games g
		JOIN game_player_statistics gps ON gps.game_id = g.id
		WHERE gps.user_id = $1
			AND g.status IN ('completed', 'forfeit')
			AND g.completed_at IS NOT NULL
		ORDER BY g.completed_at DESC, g.id DESC
		LIMIT $2 OFFSET $3
	`, uuidID, limit, offset)
	if err != nil {
		return PlayerGameHistoryPage{}, err
	}
	defer rows.Close()

	page := PlayerGameHistoryPage{Games: make([]PlayerGameHistoryRecord, 0), Total: total}
	for rows.Next() {
		var game PlayerGameHistoryRecord
		if err := rows.Scan(
			&game.GameID, &game.Status, &game.CompletedAt,
			&game.Placement, &game.PlayerCount, &game.Won, &game.Forfeited,
			&game.TotalPoints, &game.RoundsPlayed, &game.RoundsWon,
			&game.PlaytimeSeconds,
		); err != nil {
			return PlayerGameHistoryPage{}, err
		}
		page.Games = append(page.Games, game)
	}
	if err := rows.Err(); err != nil {
		return PlayerGameHistoryPage{}, err
	}
	return page, nil
}

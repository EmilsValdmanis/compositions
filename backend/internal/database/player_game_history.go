package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type PlayerGameHistoryRecord struct {
	GameID, Status                       string
	GameMode                             string
	Ranked                               bool
	CompletedAt                          time.Time
	Placement, PlayerCount               int
	Won, Forfeited                       bool
	TotalPoints, RoundsPlayed, RoundsWon int
	PlaytimeSeconds                      int64
}

type GameHistoryFilter string

const (
	GameHistoryAll   GameHistoryFilter = "all"
	GameHistoryFull  GameHistoryFilter = "full"
	GameHistoryQuick GameHistoryFilter = "quick"
)

func ParseGameHistoryFilter(value string) (GameHistoryFilter, bool) {
	filter := GameHistoryFilter(value)
	if filter == "" {
		return GameHistoryAll, true
	}
	switch filter {
	case GameHistoryAll, GameHistoryFull, GameHistoryQuick:
		return filter, true
	default:
		return "", false
	}
}

type PlayerGameHistoryPage struct {
	Games []PlayerGameHistoryRecord
	Total int64
}

// GetPlayerGameHistory returns completed games newest first. Limit, offset,
// and the optional mode filter are bounded by the HTTP layer.
func (s *UserStore) GetPlayerGameHistory(ctx context.Context, userID string, limit, offset int, filters ...GameHistoryFilter) (PlayerGameHistoryPage, error) {
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
	filter := GameHistoryAll
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter != GameHistoryAll && filter != GameHistoryFull && filter != GameHistoryQuick {
		return PlayerGameHistoryPage{}, errors.New("invalid game history filter")
	}

	var total int64
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(g.id)
		FROM users u
		LEFT JOIN game_player_statistics gps ON gps.user_id = u.id
		LEFT JOIN games g ON g.id = gps.game_id
			AND g.status IN ('completed', 'forfeit')
			AND g.completed_at IS NOT NULL
			AND ($2 = 'all' OR g.game_mode = $2)
		WHERE u.id = $1
		GROUP BY u.id
	`, uuidID, filter).Scan(&total)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlayerGameHistoryPage{}, ErrPlayerProfileNotFound
	}
	if err != nil {
		return PlayerGameHistoryPage{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT g.id::text, g.status, g.game_mode, g.ranked, g.completed_at,
			gps.placement, g.player_count, gps.won, gps.forfeited,
			gps.total_points, gps.rounds_played, gps.rounds_won,
			g.active_playtime_seconds
		FROM games g
		JOIN game_player_statistics gps ON gps.game_id = g.id
		WHERE gps.user_id = $1
			AND g.status IN ('completed', 'forfeit')
			AND g.completed_at IS NOT NULL
			AND ($4 = 'all' OR g.game_mode = $4)
		ORDER BY g.completed_at DESC, g.id DESC
		LIMIT $2 OFFSET $3
	`, uuidID, limit, offset, filter)
	if err != nil {
		return PlayerGameHistoryPage{}, err
	}
	defer rows.Close()

	page := PlayerGameHistoryPage{Games: make([]PlayerGameHistoryRecord, 0), Total: total}
	for rows.Next() {
		var game PlayerGameHistoryRecord
		if err := rows.Scan(
			&game.GameID, &game.Status, &game.GameMode, &game.Ranked, &game.CompletedAt,
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

package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/EmilsValdmanis/compositions/internal/rating"
	"github.com/jackc/pgx/v5/pgtype"
)

type LeaderboardMetric string

type LeaderboardScope string

const (
	LeaderboardMetricElo      LeaderboardMetric = "elo"
	LeaderboardMetricWins     LeaderboardMetric = "wins"
	LeaderboardMetricGames    LeaderboardMetric = "games"
	LeaderboardMetricPlaytime LeaderboardMetric = "playtime"
	LeaderboardMetricRounds   LeaderboardMetric = "rounds"
	LeaderboardMetricPoints   LeaderboardMetric = "points"
)

const (
	LeaderboardScopeFriends LeaderboardScope = "friends"
	LeaderboardScopeGlobal  LeaderboardScope = "global"
)

func ParseLeaderboardMetric(value string) (LeaderboardMetric, bool) {
	metric := LeaderboardMetric(strings.TrimSpace(value))
	if metric == "" {
		return LeaderboardMetricElo, true
	}
	switch metric {
	case LeaderboardMetricElo, LeaderboardMetricWins, LeaderboardMetricGames, LeaderboardMetricPlaytime,
		LeaderboardMetricRounds, LeaderboardMetricPoints:
		return metric, true
	default:
		return "", false
	}
}

func ParseLeaderboardScope(value string) (LeaderboardScope, bool) {
	scope := LeaderboardScope(strings.TrimSpace(value))
	if scope == "" {
		return LeaderboardScopeFriends, true
	}
	switch scope {
	case LeaderboardScopeFriends, LeaderboardScopeGlobal:
		return scope, true
	default:
		return "", false
	}
}

func (m LeaderboardMetric) scoreExpression() (string, error) {
	switch m {
	case LeaderboardMetricElo:
		return fmt.Sprintf("COALESCE(pr.rating, %d)", rating.Current().Initial), nil
	case LeaderboardMetricWins:
		return "ps.games_won", nil
	case LeaderboardMetricGames:
		return "ps.games_played", nil
	case LeaderboardMetricPlaytime:
		return "COALESCE(playtimes.total_playtime_seconds, 0)", nil
	case LeaderboardMetricRounds:
		return "ps.rounds_won", nil
	case LeaderboardMetricPoints:
		return "ps.points_inflicted", nil
	default:
		return "", errors.New("invalid leaderboard metric")
	}
}

type LeaderboardCursor struct {
	Revision string
	Score    int64
	PlayerID string
}

type LeaderboardPlayerRecord struct {
	Rank                 int64
	Score                int64
	PlayerID             string
	Name                 string
	ImageURL             string
	Wins                 int64
	GamesPlayed          int64
	RoundsWon            int64
	PointsInflicted      int64
	TotalPlaytimeSeconds int64
}

type LeaderboardPage struct {
	Reset      bool
	Players    []LeaderboardPlayerRecord
	NextCursor *LeaderboardCursor
	Placement  *LeaderboardPlayerRecord
}

func leaderboardRankedPlayers(metric LeaderboardMetric) (string, error) {
	score, err := metric.scoreExpression()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`
	WITH playtimes AS (
		SELECT gps.user_id, COALESCE(SUM(g.active_playtime_seconds), 0)::bigint AS total_playtime_seconds
		FROM game_player_statistics gps
		JOIN games g ON g.id = gps.game_id
		WHERE g.status IN ('completed', 'forfeit') AND g.completed_at IS NOT NULL
			AND g.game_mode = 'full' AND g.ranked
		GROUP BY gps.user_id
	), ranked AS (
		SELECT
			ROW_NUMBER() OVER (ORDER BY %[1]s DESC, ps.user_id ASC)::bigint AS rank,
			%[1]s::bigint AS score,
			u.id AS player_id,
			u.name,
			u.image_url,
			ps.games_won AS wins,
			ps.games_played,
			ps.rounds_won,
			ps.points_inflicted,
			COALESCE(playtimes.total_playtime_seconds, 0)::bigint AS total_playtime_seconds
		FROM player_statistics ps
		JOIN users u ON u.id = ps.user_id
		LEFT JOIN player_ratings pr ON pr.user_id = ps.user_id
		LEFT JOIN playtimes ON playtimes.user_id = ps.user_id
		WHERE ps.games_played > 0 AND ps.game_mode = 'full' AND ps.ranked
			AND (
				$1::text = 'global'
				OR ps.user_id = $2
				OR EXISTS (
					SELECT 1
					FROM friendships f
					WHERE (f.user_a_id = $2 AND f.user_b_id = ps.user_id)
						OR (f.user_b_id = $2 AND f.user_a_id = ps.user_id)
				)
			)
	)
`, score), nil
}

// GetLeaderboard ranks players by the selected all-time statistic. The UUID
// tie-breaker makes every cursor boundary deterministic even when scores match.
// Fingerprint membership and scores in the same SQL snapshot as the page. A
// changed ranking restarts pagination instead of mixing rows from two rankings.
func (s *UserStore) GetLeaderboard(ctx context.Context, cursor *LeaderboardCursor, limit int, viewerUserID string, metric LeaderboardMetric, scope LeaderboardScope) (LeaderboardPage, error) {
	if s == nil || s.pool == nil {
		return LeaderboardPage{}, errors.New("user store is not configured")
	}
	if limit <= 0 || limit > 50 {
		return LeaderboardPage{}, errors.New("leaderboard limit must be between 1 and 50")
	}
	viewerUserID = strings.TrimSpace(viewerUserID)
	viewerID := pgtype.UUID{}
	if viewerUserID != "" {
		var err error
		viewerID, err = parseUUID(viewerUserID)
		if err != nil {
			return LeaderboardPage{}, err
		}
	}
	if scope == LeaderboardScopeFriends && !viewerID.Valid {
		return LeaderboardPage{}, errors.New("viewer user id is required for friends leaderboard")
	}
	if scope != LeaderboardScopeFriends && scope != LeaderboardScopeGlobal {
		return LeaderboardPage{}, errors.New("invalid leaderboard scope")
	}
	rankedPlayers, err := leaderboardRankedPlayers(metric)
	if err != nil {
		return LeaderboardPage{}, err
	}

	hasCursor := cursor != nil
	cursorScore := int64(0)
	cursorRevision := ""
	cursorPlayerID := pgtype.UUID{}
	if cursor != nil {
		if cursor.Score < 0 {
			return LeaderboardPage{}, errors.New("leaderboard cursor score cannot be negative")
		}
		parsedID, err := parseUUID(cursor.PlayerID)
		if err != nil {
			return LeaderboardPage{}, err
		}
		cursorScore = cursor.Score
		cursorRevision = cursor.Revision
		cursorPlayerID = parsedID
	}

	var playersJSON, placementJSON json.RawMessage
	var revision string
	var reset bool
	err = s.pool.QueryRow(ctx, rankedPlayers+`, revision AS (
		SELECT md5(COALESCE(string_agg(player_id::text || ':' || score::text, ',' ORDER BY player_id), '')) AS value
		FROM ranked
	), page AS (
		SELECT rank, score, player_id, name, image_url, wins, games_played,
			rounds_won, points_inflicted, total_playtime_seconds
		FROM ranked
		WHERE NOT $3::boolean
			OR $7::text <> (SELECT value FROM revision)
			OR score < $4
			OR (score = $4 AND player_id > $5)
		ORDER BY score DESC, player_id ASC
		LIMIT $6
	)
	SELECT COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'Rank', rank, 'Score', score, 'PlayerID', player_id::text,
			'Name', name, 'ImageURL', image_url, 'Wins', wins,
			'GamesPlayed', games_played, 'RoundsWon', rounds_won,
			'PointsInflicted', points_inflicted,
			'TotalPlaytimeSeconds', total_playtime_seconds
		) ORDER BY score DESC, player_id ASC) FROM page
	), '[]'::jsonb),
	COALESCE((
		SELECT jsonb_build_object(
			'Rank', rank, 'Score', score, 'PlayerID', player_id::text,
			'Name', name, 'ImageURL', image_url, 'Wins', wins,
			'GamesPlayed', games_played, 'RoundsWon', rounds_won,
			'PointsInflicted', points_inflicted,
			'TotalPlaytimeSeconds', total_playtime_seconds
		) FROM ranked WHERE player_id = $2
	), 'null'::jsonb),
	(SELECT value FROM revision),
	$3::boolean AND $7::text <> (SELECT value FROM revision)
	`, string(scope), viewerID, hasCursor, cursorScore, cursorPlayerID, limit+1, cursorRevision).Scan(&playersJSON, &placementJSON, &revision, &reset)
	if err != nil {
		return LeaderboardPage{}, err
	}
	page := LeaderboardPage{Reset: reset}
	if err := json.Unmarshal(playersJSON, &page.Players); err != nil {
		return LeaderboardPage{}, err
	}

	if len(page.Players) > limit {
		page.Players = page.Players[:limit]
		last := page.Players[len(page.Players)-1]
		page.NextCursor = &LeaderboardCursor{Score: last.Score, PlayerID: last.PlayerID, Revision: revision}
	}

	if viewerUserID == "" {
		return page, nil
	}
	var placement *LeaderboardPlayerRecord
	if err := json.Unmarshal(placementJSON, &placement); err != nil {
		return LeaderboardPage{}, err
	}
	page.Placement = placement
	return page, nil
}

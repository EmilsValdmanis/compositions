package database

import (
	"context"
	"errors"
	"time"
)

const adminAnalyticsTimezone = "Europe/Riga"

type AdminAnalyticsRange struct {
	From, To                 time.Time
	PreviousFrom, PreviousTo time.Time
}

type AdminAnalyticsTotalsRecord struct {
	Games                      int64   `json:"games"`
	ActivePlayers              int64   `json:"activePlayers"`
	ActivePlaytimeSeconds      int64   `json:"activePlaytimeSeconds"`
	HealthyFinishRate          float64 `json:"healthyFinishRate"`
	BugReports                 int64   `json:"bugReports"`
	BugsResolved               int64   `json:"bugsResolved"`
	MedianBugResolutionSeconds float64 `json:"medianBugResolutionSeconds"`
}

type AdminAnalyticsPointRecord struct {
	Date             string `json:"date"`
	Games            int64  `json:"games"`
	ActivePlayers    int64  `json:"activePlayers"`
	NewPlayers       int64  `json:"newPlayers"`
	ReturningPlayers int64  `json:"returningPlayers"`
	Completed        int64  `json:"completed"`
	Forfeit          int64  `json:"forfeit"`
	MutualEnd        int64  `json:"mutualEnd"`
	TechnicalAbort   int64  `json:"technicalAbort"`
	Abandoned        int64  `json:"abandoned"`
	InProgress       int64  `json:"inProgress"`
	BugReports       int64  `json:"bugReports"`
	BugsResolved     int64  `json:"bugsResolved"`
}

type AdminAnalyticsRecord struct {
	Current  AdminAnalyticsTotalsRecord
	Previous AdminAnalyticsTotalsRecord
	Points   []AdminAnalyticsPointRecord
}

func (s *UserStore) GetAdminAnalytics(ctx context.Context, dateRange AdminAnalyticsRange) (AdminAnalyticsRecord, error) {
	if s == nil || s.pool == nil {
		return AdminAnalyticsRecord{}, errors.New("user store is not configured")
	}
	if dateRange.From.IsZero() || dateRange.To.IsZero() || !dateRange.From.Before(dateRange.To) {
		return AdminAnalyticsRecord{}, errors.New("invalid analytics date range")
	}
	if dateRange.PreviousFrom.IsZero() || dateRange.PreviousTo.IsZero() ||
		!dateRange.PreviousFrom.Before(dateRange.PreviousTo) || !dateRange.PreviousTo.Equal(dateRange.From) {
		return AdminAnalyticsRecord{}, errors.New("invalid previous analytics date range")
	}

	current, err := s.adminAnalyticsTotals(ctx, dateRange.From, dateRange.To)
	if err != nil {
		return AdminAnalyticsRecord{}, err
	}
	previous, err := s.adminAnalyticsTotals(ctx, dateRange.PreviousFrom, dateRange.PreviousTo)
	if err != nil {
		return AdminAnalyticsRecord{}, err
	}
	points, err := s.adminAnalyticsPoints(ctx, dateRange.From, dateRange.To)
	if err != nil {
		return AdminAnalyticsRecord{}, err
	}
	return AdminAnalyticsRecord{Current: current, Previous: previous, Points: points}, nil
}

func (s *UserStore) adminAnalyticsTotals(ctx context.Context, from, to time.Time) (AdminAnalyticsTotalsRecord, error) {
	var totals AdminAnalyticsTotalsRecord
	err := s.pool.QueryRow(ctx, `
		WITH selected_games AS (
			SELECT id, status, active_playtime_seconds
			FROM games
			WHERE started_at >= $1 AND started_at < $2
		)
		SELECT
			COUNT(*)::bigint,
			COALESCE((
				SELECT COUNT(DISTINCT gps.user_id)::bigint
				FROM game_player_statistics gps
				JOIN selected_games selected ON selected.id = gps.game_id
			), 0),
			COALESCE(SUM(active_playtime_seconds), 0)::bigint,
			COALESCE(
				COUNT(*) FILTER (WHERE status = 'completed')::double precision /
				NULLIF(COUNT(*) FILTER (WHERE status <> 'in_progress'), 0),
				0
			),
			(SELECT COUNT(*)::bigint FROM game_bug_reports WHERE created_at >= $1 AND created_at < $2),
			(SELECT COUNT(*)::bigint FROM game_bug_reports WHERE completed_at >= $1 AND completed_at < $2),
			COALESCE((
				SELECT percentile_cont(0.5) WITHIN GROUP (
					ORDER BY EXTRACT(EPOCH FROM (completed_at - created_at))
				)::double precision
				FROM game_bug_reports
				WHERE completed_at >= $1 AND completed_at < $2
			), 0)
		FROM selected_games
	`, from, to).Scan(
		&totals.Games,
		&totals.ActivePlayers,
		&totals.ActivePlaytimeSeconds,
		&totals.HealthyFinishRate,
		&totals.BugReports,
		&totals.BugsResolved,
		&totals.MedianBugResolutionSeconds,
	)
	return totals, err
}

func (s *UserStore) adminAnalyticsPoints(ctx context.Context, from, to time.Time) ([]AdminAnalyticsPointRecord, error) {
	rows, err := s.pool.Query(ctx, `
		WITH days AS (
			SELECT generate_series(
				($1::timestamptz AT TIME ZONE $3)::date,
				(($2::timestamptz AT TIME ZONE $3)::date - 1),
				INTERVAL '1 day'
			)::date AS bucket
		), range_games AS (
			SELECT id, status, (started_at AT TIME ZONE $3)::date AS bucket
			FROM games
			WHERE started_at >= $1 AND started_at < $2
		), daily_games AS (
			SELECT
				bucket,
				COUNT(*)::bigint AS games,
				COUNT(*) FILTER (WHERE status = 'completed')::bigint AS completed,
				COUNT(*) FILTER (WHERE status = 'forfeit')::bigint AS forfeit,
				COUNT(*) FILTER (WHERE status = 'mutual_end')::bigint AS mutual_end,
				COUNT(*) FILTER (WHERE status = 'technical_abort')::bigint AS technical_abort,
				COUNT(*) FILTER (WHERE status = 'abandoned')::bigint AS abandoned,
				COUNT(*) FILTER (WHERE status = 'in_progress')::bigint AS in_progress
			FROM range_games
			GROUP BY bucket
		), range_players AS (
			SELECT DISTINCT range_games.bucket, gps.user_id
			FROM range_games
			JOIN game_player_statistics gps ON gps.game_id = range_games.id
		), first_games AS (
			SELECT gps.user_id, MIN((games.started_at AT TIME ZONE $3)::date) AS first_bucket
			FROM game_player_statistics gps
			JOIN games ON games.id = gps.game_id
			WHERE gps.user_id IN (SELECT DISTINCT user_id FROM range_players)
			GROUP BY gps.user_id
		), daily_players AS (
			SELECT
				range_players.bucket,
				COUNT(*)::bigint AS active_players,
				COUNT(*) FILTER (WHERE first_games.first_bucket = range_players.bucket)::bigint AS new_players,
				COUNT(*) FILTER (WHERE first_games.first_bucket < range_players.bucket)::bigint AS returning_players
			FROM range_players
			JOIN first_games ON first_games.user_id = range_players.user_id
			GROUP BY range_players.bucket
		), daily_bug_reports AS (
			SELECT (created_at AT TIME ZONE $3)::date AS bucket, COUNT(*)::bigint AS bug_reports
			FROM game_bug_reports
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY bucket
		), daily_bugs_resolved AS (
			SELECT (completed_at AT TIME ZONE $3)::date AS bucket, COUNT(*)::bigint AS bugs_resolved
			FROM game_bug_reports
			WHERE completed_at >= $1 AND completed_at < $2
			GROUP BY bucket
		)
		SELECT
			days.bucket::text,
			COALESCE(daily_games.games, 0),
			COALESCE(daily_players.active_players, 0),
			COALESCE(daily_players.new_players, 0),
			COALESCE(daily_players.returning_players, 0),
			COALESCE(daily_games.completed, 0),
			COALESCE(daily_games.forfeit, 0),
			COALESCE(daily_games.mutual_end, 0),
			COALESCE(daily_games.technical_abort, 0),
			COALESCE(daily_games.abandoned, 0),
			COALESCE(daily_games.in_progress, 0),
			COALESCE(daily_bug_reports.bug_reports, 0),
			COALESCE(daily_bugs_resolved.bugs_resolved, 0)
		FROM days
		LEFT JOIN daily_games ON daily_games.bucket = days.bucket
		LEFT JOIN daily_players ON daily_players.bucket = days.bucket
		LEFT JOIN daily_bug_reports ON daily_bug_reports.bucket = days.bucket
		LEFT JOIN daily_bugs_resolved ON daily_bugs_resolved.bucket = days.bucket
		ORDER BY days.bucket
	`, from, to, adminAnalyticsTimezone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]AdminAnalyticsPointRecord, 0)
	for rows.Next() {
		var point AdminAnalyticsPointRecord
		if err := rows.Scan(
			&point.Date,
			&point.Games,
			&point.ActivePlayers,
			&point.NewPlayers,
			&point.ReturningPlayers,
			&point.Completed,
			&point.Forfeit,
			&point.MutualEnd,
			&point.TechnicalAbort,
			&point.Abandoned,
			&point.InProgress,
			&point.BugReports,
			&point.BugsResolved,
		); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

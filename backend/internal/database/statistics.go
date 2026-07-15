package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxStoredStatistic = int64(1<<31 - 1)

type GameCheckpointRecord struct {
	ID, RoomCode              string
	RoundsPlayed, PlayerCount int
	StartedAt                 time.Time
	PlaytimeSeconds           int64
	Players                   []CompletedGamePlayerRecord
}

type CompletedGameRecord struct {
	ID, RoomCode              string
	CompletionKind            string
	RoundsPlayed, PlayerCount int
	StartedAt, CompletedAt    time.Time
	PlaytimeSeconds           int64
	Players                   []CompletedGamePlayerRecord
}

type CompletedGamePlayerRecord struct {
	UserID                                                                             string
	Placement                                                                          int
	Won, Forfeited                                                                     bool
	TotalPoints                                                                        int
	RoundsPlayed, RoundsWon, SameSuitWins, SixPairsWins                                int
	TurnsTaken, CardsDrawnFromDeck, CardsDrawnFromDiscard, CardsDiscarded, CardsPlayed int
	CompositionsCreated, SetsCreated, RunsCreated, AdditionsDone                       int
	CompositionsCompleted, SetsCompleted, RunsCompleted                                int
	JokersPlayed, JokersReclaimed                                                      int
	CardsRemaining, HandPoints, PenaltyPoints, PointsInflicted                         int
	LargestRoundPenalty, LargestRoundPointsInflicted, MostCardsRemaining               int
	RoundsOpened, FastestOpeningTurn                                                   int
	StartingRoundWinStreak, EndingRoundWinStreak, LongestRoundWinStreak                int
}

// SaveGameCheckpoint replaces the cumulative per-game counters without
// touching lifetime competitive statistics.
func (s *UserStore) SaveGameCheckpoint(ctx context.Context, checkpoint GameCheckpointRecord) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}
	gameID, err := validateCheckpoint(checkpoint)
	if err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	active, err := saveCheckpointHeader(ctx, tx, gameID, checkpoint)
	if err != nil {
		return err
	}
	if active {
		for _, player := range checkpoint.Players {
			if err := saveGamePlayer(ctx, tx, gameID, player, false); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

// SaveUnrankedGame preserves the latest checkpoint but deliberately does not
// update lifetime totals, placements, wins, losses, or streaks.
func (s *UserStore) SaveUnrankedGame(ctx context.Context, checkpoint GameCheckpointRecord, status string, completedAt time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}
	if status != "mutual_end" && status != "technical_abort" && status != "abandoned" {
		return errors.New("invalid unranked game status")
	}
	if completedAt.IsZero() || completedAt.Before(checkpoint.StartedAt) {
		return errors.New("invalid unranked completion time")
	}
	gameID, err := validateCheckpoint(checkpoint)
	if err != nil {
		return err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	active, err := saveCheckpointHeader(ctx, tx, gameID, checkpoint)
	if err != nil {
		return err
	}
	if !active {
		return tx.Commit(ctx)
	}
	for _, player := range checkpoint.Players {
		if err := saveGamePlayer(ctx, tx, gameID, player, false); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE games
		SET status = $2, completed_at = $3,
			active_playtime_seconds = GREATEST(active_playtime_seconds, $4), updated_at = NOW()
		WHERE id = $1 AND status = 'in_progress'
	`, gameID, status, completedAt, checkpoint.PlaytimeSeconds)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SaveCompletedGame finalizes checkpoint rows and updates lifetime totals in
// one transaction. Transitioning from in_progress makes retries idempotent.
func (s *UserStore) SaveCompletedGame(ctx context.Context, completed CompletedGameRecord) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}
	status := completed.CompletionKind
	if status == "normal" {
		status = "completed"
	}
	if status != "completed" && status != "forfeit" {
		return errors.New("invalid game completion kind")
	}
	if completed.CompletedAt.IsZero() || completed.CompletedAt.Before(completed.StartedAt) {
		return errors.New("invalid game completion time")
	}
	checkpoint := GameCheckpointRecord{
		ID: completed.ID, RoomCode: completed.RoomCode, RoundsPlayed: completed.RoundsPlayed,
		PlayerCount: completed.PlayerCount, StartedAt: completed.StartedAt,
		PlaytimeSeconds: completed.PlaytimeSeconds, Players: completed.Players,
	}
	gameID, err := validateCheckpoint(checkpoint)
	if err != nil {
		return fmt.Errorf("completed game statistics are incomplete: %w", err)
	}
	winners := 0
	for _, player := range completed.Players {
		if player.Placement <= 0 || player.Placement > completed.PlayerCount {
			return errors.New("completed game placement is invalid")
		}
		if player.Won != (player.Placement == 1) {
			return errors.New("completed game winner and placement disagree")
		}
		if player.Won {
			winners++
			if player.Forfeited {
				return errors.New("completed game winner cannot be forfeited")
			}
		}
	}
	if winners > 1 {
		return errors.New("completed game has multiple winners")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	active, err := saveCheckpointHeader(ctx, tx, gameID, checkpoint)
	if err != nil {
		return err
	}
	if !active {
		return tx.Commit(ctx)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE games
		SET status = $2, completed_at = $3,
			active_playtime_seconds = GREATEST(active_playtime_seconds, $4), updated_at = NOW()
		WHERE id = $1 AND status = 'in_progress'
	`, gameID, status, completed.CompletedAt, completed.PlaytimeSeconds)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	for _, player := range completed.Players {
		if err := saveGamePlayer(ctx, tx, gameID, player, true); err != nil {
			return err
		}
		if err := addLifetimeStatistics(ctx, tx, player); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func validateCheckpoint(checkpoint GameCheckpointRecord) (pgtype.UUID, error) {
	if strings.TrimSpace(checkpoint.ID) == "" || strings.TrimSpace(checkpoint.RoomCode) == "" {
		return pgtype.UUID{}, errors.New("game id and room code are required")
	}
	if checkpoint.RoundsPlayed <= 0 || checkpoint.PlayerCount < 2 || len(checkpoint.Players) == 0 {
		return pgtype.UUID{}, errors.New("game checkpoint statistics are incomplete")
	}
	if checkpoint.PlayerCount > 4 || len(checkpoint.Players) > checkpoint.PlayerCount || checkpoint.StartedAt.IsZero() {
		return pgtype.UUID{}, errors.New("game checkpoint metadata is invalid")
	}
	if checkpoint.PlaytimeSeconds < 0 {
		return pgtype.UUID{}, errors.New("game playtime cannot be negative")
	}
	var gameID pgtype.UUID
	if err := gameID.Scan(checkpoint.ID); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid game id: %w", err)
	}
	seenUsers := make(map[string]bool, len(checkpoint.Players))
	for _, player := range checkpoint.Players {
		userID := strings.TrimSpace(player.UserID)
		var parsed pgtype.UUID
		if err := parsed.Scan(userID); err != nil {
			return pgtype.UUID{}, fmt.Errorf("invalid statistics user id: %w", err)
		}
		if seenUsers[userID] {
			return pgtype.UUID{}, errors.New("duplicate statistics user id")
		}
		seenUsers[userID] = true
		if err := validatePlayerStatistics(player); err != nil {
			return pgtype.UUID{}, err
		}
		if player.RoundsPlayed > checkpoint.RoundsPlayed {
			return pgtype.UUID{}, errors.New("player rounds exceed game rounds")
		}
	}
	return gameID, nil
}

func validatePlayerStatistics(player CompletedGamePlayerRecord) error {
	values := []int{
		player.TotalPoints, player.RoundsPlayed, player.RoundsWon, player.SameSuitWins, player.SixPairsWins,
		player.TurnsTaken, player.CardsDrawnFromDeck, player.CardsDrawnFromDiscard, player.CardsDiscarded,
		player.CardsPlayed, player.CompositionsCreated, player.SetsCreated, player.RunsCreated,
		player.AdditionsDone, player.CompositionsCompleted, player.SetsCompleted, player.RunsCompleted,
		player.JokersPlayed, player.JokersReclaimed, player.CardsRemaining, player.HandPoints,
		player.PenaltyPoints, player.PointsInflicted, player.LargestRoundPenalty,
		player.LargestRoundPointsInflicted, player.MostCardsRemaining, player.RoundsOpened,
		player.FastestOpeningTurn, player.StartingRoundWinStreak, player.EndingRoundWinStreak,
		player.LongestRoundWinStreak,
	}
	for _, value := range values {
		if value < 0 || int64(value) > maxStoredStatistic {
			return errors.New("player statistics are outside the supported range")
		}
	}
	if player.RoundsWon > player.RoundsPlayed || player.RoundsOpened > player.RoundsPlayed {
		return errors.New("player round statistics are inconsistent")
	}
	if player.SameSuitWins > player.RoundsWon || player.SixPairsWins > player.RoundsWon {
		return errors.New("player special wins are inconsistent")
	}
	if player.StartingRoundWinStreak > player.RoundsWon || player.EndingRoundWinStreak > player.RoundsWon || player.LongestRoundWinStreak > player.RoundsWon {
		return errors.New("player round streak is inconsistent")
	}
	if (player.RoundsOpened == 0) != (player.FastestOpeningTurn == 0) {
		return errors.New("player opening statistics are inconsistent")
	}
	return nil
}

func saveCheckpointHeader(ctx context.Context, tx pgx.Tx, gameID pgtype.UUID, checkpoint GameCheckpointRecord) (bool, error) {
	tag, err := tx.Exec(ctx, `
		INSERT INTO games (id, room_code, status, rounds_played, player_count, started_at, active_playtime_seconds)
		VALUES ($1, $2, 'in_progress', $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			room_code = EXCLUDED.room_code,
			rounds_played = EXCLUDED.rounds_played,
			player_count = EXCLUDED.player_count,
			active_playtime_seconds = GREATEST(games.active_playtime_seconds, EXCLUDED.active_playtime_seconds),
			updated_at = NOW()
		WHERE games.status = 'in_progress'
	`, gameID, strings.TrimSpace(checkpoint.RoomCode), checkpoint.RoundsPlayed, checkpoint.PlayerCount, checkpoint.StartedAt, checkpoint.PlaytimeSeconds)
	return tag.RowsAffected() > 0, err
}

func saveGamePlayer(ctx context.Context, tx pgx.Tx, gameID pgtype.UUID, p CompletedGamePlayerRecord, finalized bool) error {
	var userID pgtype.UUID
	if err := userID.Scan(p.UserID); err != nil {
		return fmt.Errorf("invalid statistics user id: %w", err)
	}
	var placement any
	var won any
	if finalized {
		placement = p.Placement
		won = p.Won
	}
	values := []any{gameID, userID, placement, won, p.Forfeited, p.TotalPoints,
		p.RoundsPlayed, p.RoundsWon, p.SameSuitWins, p.SixPairsWins, p.TurnsTaken,
		p.CardsDrawnFromDeck, p.CardsDrawnFromDiscard, p.CardsDiscarded, p.CardsPlayed,
		p.CompositionsCreated, p.SetsCreated, p.RunsCreated, p.AdditionsDone,
		p.CompositionsCompleted, p.SetsCompleted, p.RunsCompleted, p.JokersPlayed, p.JokersReclaimed,
		p.CardsRemaining, p.HandPoints, p.PenaltyPoints, p.PointsInflicted, p.LargestRoundPenalty,
		p.LargestRoundPointsInflicted, p.MostCardsRemaining, p.RoundsOpened, p.FastestOpeningTurn,
		p.StartingRoundWinStreak, p.EndingRoundWinStreak, p.LongestRoundWinStreak}
	_, err := tx.Exec(ctx, `
		INSERT INTO game_player_statistics (
			game_id, user_id, placement, won, forfeited, total_points, rounds_played, rounds_won,
			same_suit_wins, six_pairs_wins, turns_taken, cards_drawn_from_deck, cards_drawn_from_discard,
			cards_discarded, cards_played, compositions_created, sets_created, runs_created, additions_done,
			compositions_completed, sets_completed, runs_completed, jokers_played, jokers_reclaimed,
			cards_remaining, hand_points, penalty_points, points_inflicted, largest_round_penalty,
			largest_round_points_inflicted, most_cards_remaining, rounds_opened, fastest_opening_turn,
			starting_round_win_streak, ending_round_win_streak, longest_round_win_streak
		) VALUES (`+placeholders(len(values))+`)
		ON CONFLICT (game_id, user_id) DO UPDATE SET
			placement = COALESCE(EXCLUDED.placement, game_player_statistics.placement),
			won = COALESCE(EXCLUDED.won, game_player_statistics.won),
			forfeited = EXCLUDED.forfeited, total_points = EXCLUDED.total_points,
			rounds_played = EXCLUDED.rounds_played, rounds_won = EXCLUDED.rounds_won,
			same_suit_wins = EXCLUDED.same_suit_wins, six_pairs_wins = EXCLUDED.six_pairs_wins,
			turns_taken = EXCLUDED.turns_taken, cards_drawn_from_deck = EXCLUDED.cards_drawn_from_deck,
			cards_drawn_from_discard = EXCLUDED.cards_drawn_from_discard, cards_discarded = EXCLUDED.cards_discarded,
			cards_played = EXCLUDED.cards_played, compositions_created = EXCLUDED.compositions_created,
			sets_created = EXCLUDED.sets_created, runs_created = EXCLUDED.runs_created,
			additions_done = EXCLUDED.additions_done, compositions_completed = EXCLUDED.compositions_completed,
			sets_completed = EXCLUDED.sets_completed, runs_completed = EXCLUDED.runs_completed,
			jokers_played = EXCLUDED.jokers_played, jokers_reclaimed = EXCLUDED.jokers_reclaimed,
			cards_remaining = EXCLUDED.cards_remaining, hand_points = EXCLUDED.hand_points,
			penalty_points = EXCLUDED.penalty_points, points_inflicted = EXCLUDED.points_inflicted,
			largest_round_penalty = EXCLUDED.largest_round_penalty,
			largest_round_points_inflicted = EXCLUDED.largest_round_points_inflicted,
			most_cards_remaining = EXCLUDED.most_cards_remaining, rounds_opened = EXCLUDED.rounds_opened,
			fastest_opening_turn = EXCLUDED.fastest_opening_turn,
			starting_round_win_streak = EXCLUDED.starting_round_win_streak,
			ending_round_win_streak = EXCLUDED.ending_round_win_streak,
			longest_round_win_streak = EXCLUDED.longest_round_win_streak
	`, values...)
	return err
}

func addLifetimeStatistics(ctx context.Context, tx pgx.Tx, p CompletedGamePlayerRecord) error {
	var userID pgtype.UUID
	if err := userID.Scan(p.UserID); err != nil {
		return fmt.Errorf("invalid statistics user id: %w", err)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO player_statistics (
			user_id, games_played, games_won, total_placement, rounds_played, rounds_won, same_suit_wins,
			six_pairs_wins, forfeits, turns_taken, cards_drawn_from_deck, cards_drawn_from_discard,
			cards_discarded, cards_played, compositions_created, sets_created, runs_created, additions_done,
			compositions_completed, sets_completed, runs_completed, jokers_played, jokers_reclaimed,
			cards_remaining, hand_points, penalty_points, points_inflicted, largest_round_penalty,
			largest_round_points_inflicted, most_cards_remaining, rounds_opened, fastest_opening_turn,
			current_game_win_streak, longest_game_win_streak, current_round_win_streak, longest_round_win_streak
		) VALUES (
			$1, 1, $2::bigint, $3, $4, $5, $6, $7, $8::bigint, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31,
			$2::int, $2::int, $32, $33
		)
		ON CONFLICT (user_id) DO UPDATE SET
			games_played = player_statistics.games_played + 1,
			games_won = player_statistics.games_won + EXCLUDED.games_won,
			total_placement = player_statistics.total_placement + EXCLUDED.total_placement,
			rounds_played = player_statistics.rounds_played + EXCLUDED.rounds_played,
			rounds_won = player_statistics.rounds_won + EXCLUDED.rounds_won,
			same_suit_wins = player_statistics.same_suit_wins + EXCLUDED.same_suit_wins,
			six_pairs_wins = player_statistics.six_pairs_wins + EXCLUDED.six_pairs_wins,
			forfeits = player_statistics.forfeits + EXCLUDED.forfeits,
			turns_taken = player_statistics.turns_taken + EXCLUDED.turns_taken,
			cards_drawn_from_deck = player_statistics.cards_drawn_from_deck + EXCLUDED.cards_drawn_from_deck,
			cards_drawn_from_discard = player_statistics.cards_drawn_from_discard + EXCLUDED.cards_drawn_from_discard,
			cards_discarded = player_statistics.cards_discarded + EXCLUDED.cards_discarded,
			cards_played = player_statistics.cards_played + EXCLUDED.cards_played,
			compositions_created = player_statistics.compositions_created + EXCLUDED.compositions_created,
			sets_created = player_statistics.sets_created + EXCLUDED.sets_created,
			runs_created = player_statistics.runs_created + EXCLUDED.runs_created,
			additions_done = player_statistics.additions_done + EXCLUDED.additions_done,
			compositions_completed = player_statistics.compositions_completed + EXCLUDED.compositions_completed,
			sets_completed = player_statistics.sets_completed + EXCLUDED.sets_completed,
			runs_completed = player_statistics.runs_completed + EXCLUDED.runs_completed,
			jokers_played = player_statistics.jokers_played + EXCLUDED.jokers_played,
			jokers_reclaimed = player_statistics.jokers_reclaimed + EXCLUDED.jokers_reclaimed,
			cards_remaining = player_statistics.cards_remaining + EXCLUDED.cards_remaining,
			hand_points = player_statistics.hand_points + EXCLUDED.hand_points,
			penalty_points = player_statistics.penalty_points + EXCLUDED.penalty_points,
			points_inflicted = player_statistics.points_inflicted + EXCLUDED.points_inflicted,
			largest_round_penalty = GREATEST(player_statistics.largest_round_penalty, EXCLUDED.largest_round_penalty),
			largest_round_points_inflicted = GREATEST(player_statistics.largest_round_points_inflicted, EXCLUDED.largest_round_points_inflicted),
			most_cards_remaining = GREATEST(player_statistics.most_cards_remaining, EXCLUDED.most_cards_remaining),
			rounds_opened = player_statistics.rounds_opened + EXCLUDED.rounds_opened,
			fastest_opening_turn = CASE WHEN player_statistics.fastest_opening_turn = 0 THEN EXCLUDED.fastest_opening_turn WHEN EXCLUDED.fastest_opening_turn = 0 THEN player_statistics.fastest_opening_turn ELSE LEAST(player_statistics.fastest_opening_turn, EXCLUDED.fastest_opening_turn) END,
			current_game_win_streak = CASE WHEN EXCLUDED.games_won = 1 THEN player_statistics.current_game_win_streak + 1 ELSE 0 END,
			longest_game_win_streak = GREATEST(player_statistics.longest_game_win_streak, CASE WHEN EXCLUDED.games_won = 1 THEN player_statistics.current_game_win_streak + 1 ELSE 0 END),
			current_round_win_streak = CASE WHEN $32 = 0 THEN 0 WHEN $5 = $4 THEN player_statistics.current_round_win_streak + $32 ELSE $32 END,
			longest_round_win_streak = GREATEST(player_statistics.longest_round_win_streak, $33, player_statistics.current_round_win_streak + $34),
			updated_at = NOW()
	`, userID, boolInt(p.Won), p.Placement, p.RoundsPlayed, p.RoundsWon, p.SameSuitWins, p.SixPairsWins,
		boolInt(p.Forfeited), p.TurnsTaken, p.CardsDrawnFromDeck, p.CardsDrawnFromDiscard, p.CardsDiscarded,
		p.CardsPlayed, p.CompositionsCreated, p.SetsCreated, p.RunsCreated, p.AdditionsDone,
		p.CompositionsCompleted, p.SetsCompleted, p.RunsCompleted, p.JokersPlayed, p.JokersReclaimed,
		p.CardsRemaining, p.HandPoints, p.PenaltyPoints, p.PointsInflicted, p.LargestRoundPenalty,
		p.LargestRoundPointsInflicted, p.MostCardsRemaining, p.RoundsOpened, p.FastestOpeningTurn,
		p.EndingRoundWinStreak, p.LongestRoundWinStreak, p.StartingRoundWinStreak)
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func placeholders(count int) string {
	parts := make([]string, count)
	for i := range parts {
		parts[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(parts, ", ")
}

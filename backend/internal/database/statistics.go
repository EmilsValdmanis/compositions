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

type CompletedGameRecord struct {
	ID             string
	RoomCode       string
	CompletionKind string
	RoundsPlayed   int
	PlayerCount    int
	StartedAt      time.Time
	CompletedAt    time.Time
	Players        []CompletedGamePlayerRecord
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

// SaveCompletedGame stores the per-game source rows and updates lifetime totals
// in one transaction. The games primary key makes retries safe after restarts.
func (s *UserStore) SaveCompletedGame(ctx context.Context, game CompletedGameRecord) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}
	if strings.TrimSpace(game.ID) == "" || strings.TrimSpace(game.RoomCode) == "" {
		return errors.New("game id and room code are required")
	}
	if game.CompletionKind != "normal" && game.CompletionKind != "forfeit" {
		return errors.New("invalid game completion kind")
	}
	if game.RoundsPlayed <= 0 || game.PlayerCount < 2 || len(game.Players) == 0 {
		return errors.New("completed game statistics are incomplete")
	}

	var gameID pgtype.UUID
	if err := gameID.Scan(game.ID); err != nil {
		return fmt.Errorf("invalid game id: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		INSERT INTO games (id, room_code, completion_kind, rounds_played, player_count, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`, gameID, strings.TrimSpace(game.RoomCode), game.CompletionKind, game.RoundsPlayed, game.PlayerCount, game.StartedAt, game.CompletedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}

	for _, player := range game.Players {
		if err := saveCompletedGamePlayer(ctx, tx, gameID, player); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func saveCompletedGamePlayer(ctx context.Context, tx pgx.Tx, gameID pgtype.UUID, p CompletedGamePlayerRecord) error {
	var userID pgtype.UUID
	if err := userID.Scan(p.UserID); err != nil {
		return fmt.Errorf("invalid statistics user id: %w", err)
	}
	values := []any{gameID, userID, p.Placement, p.Won, p.Forfeited, p.TotalPoints,
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
	`, values...)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
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

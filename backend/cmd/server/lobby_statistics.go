package main

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/EmilsValdmanis/compositions/internal/game"
)

// gameStatisticsStore is intentionally separate from userStore so local/no-op
// servers can run without pretending to persist competitive statistics.
type gameStatisticsStore interface {
	SaveGameCheckpoint(ctx context.Context, checkpoint database.GameCheckpointRecord) error
	SaveUnrankedGame(ctx context.Context, checkpoint database.GameCheckpointRecord, status string, completedAt time.Time) error
	SaveCompletedGame(ctx context.Context, game database.CompletedGameRecord) error
}

// Terminal statistics survive room reset and restart until their idempotent write succeeds.
type pendingGameStatistics struct {
	Sequence    uint64                        `json:"sequence,omitempty"`
	Kind        string                        `json:"kind"`
	Status      string                        `json:"status,omitempty"`
	Checkpoint  database.GameCheckpointRecord `json:"checkpoint"`
	Completed   database.CompletedGameRecord  `json:"completed"`
	CompletedAt time.Time                     `json:"completedAt"`
}

type statisticsSaveJob struct {
	sequence    uint64
	room        *room
	roomCode    string
	gameID      string
	phase       game.GamePhase
	kind        string
	status      string
	checkpoint  database.GameCheckpointRecord
	completed   database.CompletedGameRecord
	completedAt time.Time
}

func (l *lobbyServer) statisticsJobsLocked() ([]statisticsSaveJob, bool) {
	if _, ok := l.store.(gameStatisticsStore); !ok {
		return nil, false
	}
	changed := false
	jobs := make([]statisticsSaveJob, 0)
	for _, room := range l.rooms {
		if room == nil || room.statisticsSaved || room.statisticsGameID == "" {
			continue
		}
		phase := room.gameStatePhase()
		completed := room.gameState.CompletedPlayerStatistics()
		results := completed
		if len(results) == 0 {
			results = room.gameState.PlayerStatistics()
		}
		players := make([]database.CompletedGamePlayerRecord, 0, len(results))
		for _, result := range results {
			roomPlayer := room.playerByID(result.PlayerID)
			if roomPlayer == nil {
				continue
			}
			session := l.sessions[roomPlayer.sessionID]
			if session == nil || !session.authenticated || session.authUserID == "" {
				continue
			}
			s := result.Statistics
			players = append(players, database.CompletedGamePlayerRecord{
				UserID: session.authUserID, Placement: result.Placement,
				Won: result.Winner, Forfeited: result.Forfeited, TotalPoints: result.TotalPoints,
				RoundsPlayed: s.RoundsPlayed, RoundsWon: s.RoundsWon, SameSuitWins: s.SameSuitWins, SixPairsWins: s.SixPairsWins,
				TurnsTaken: s.TurnsTaken, CardsDrawnFromDeck: s.CardsDrawnFromDeck, CardsDrawnFromDiscard: s.CardsDrawnFromDiscard,
				CardsDiscarded: s.CardsDiscarded, CardsPlayed: s.CardsPlayed, CompositionsCreated: s.CompositionsCreated,
				SetsCreated: s.SetsCreated, RunsCreated: s.RunsCreated, AdditionsDone: s.AdditionsDone,
				CompositionsCompleted: s.CompositionsCompleted, SetsCompleted: s.SetsCompleted, RunsCompleted: s.RunsCompleted,
				JokersPlayed: s.JokersPlayed, JokersReclaimed: s.JokersReclaimed, CardsRemaining: s.CardsRemaining,
				HandPoints: s.HandPoints, PenaltyPoints: s.PenaltyPoints, PointsInflicted: s.PointsInflicted,
				LargestRoundPenalty: s.LargestRoundPenalty, LargestRoundPointsInflicted: s.LargestRoundPointsInflicted,
				MostCardsRemaining: s.MostCardsRemaining, RoundsOpened: s.RoundsOpened, FastestOpeningTurn: s.FastestOpeningTurn,
				StartingRoundWinStreak: s.StartingRoundWinStreak, EndingRoundWinStreak: s.CurrentRoundWinStreak,
				LongestRoundWinStreak: s.LongestRoundWinStreak,
			})
		}
		if len(players) == 0 {
			room.statisticsDirty = false
			if phase == game.PhaseGameOver {
				room.statisticsSaved = true
			}
			changed = true
			continue
		}
		startedAt := room.statisticsStartedAt
		if startedAt.IsZero() {
			startedAt = time.Now().UTC()
		}
		checkpoint := database.GameCheckpointRecord{
			ID: room.statisticsGameID, RoomCode: room.code, RoundsPlayed: room.gameState.RoundNumber(),
			GameMode: string(room.gameState.GameMode()), Ranked: room.gameState.GameMode() == game.GameModeFull,
			PlayerCount: len(results), StartedAt: startedAt,
			PlaytimeSeconds: int64(room.statisticsPlaytime / time.Second), Players: players,
		}
		job := statisticsSaveJob{
			room: room, roomCode: room.code, gameID: room.statisticsGameID,
			phase: phase, checkpoint: checkpoint, completedAt: time.Now().UTC(),
		}
		switch {
		case phase == game.PhaseGameOver && len(completed) > 0:
			kind := "normal"
			if room.conclusion != nil && room.conclusion.kind == "forfeit" {
				kind = "forfeit"
			}
			job.kind = "completed"
			job.completed = database.CompletedGameRecord{
				ID: checkpoint.ID, RoomCode: checkpoint.RoomCode, CompletionKind: kind,
				GameMode: checkpoint.GameMode, Ranked: checkpoint.Ranked,
				RoundsPlayed: checkpoint.RoundsPlayed, PlayerCount: checkpoint.PlayerCount,
				StartedAt: checkpoint.StartedAt, CompletedAt: job.completedAt,
				PlaytimeSeconds: checkpoint.PlaytimeSeconds, Players: players,
			}
		case phase == game.PhaseGameOver:
			job.kind = "unranked"
			job.status = "abandoned"
			if room.conclusion != nil && (room.conclusion.kind == "mutual_end" || room.conclusion.kind == "technical_abort") {
				job.status = room.conclusion.kind
			}
		case room.statisticsDirty:
			job.kind = "checkpoint"
		default:
			continue
		}
		if phase == game.PhaseGameOver {
			if l.pendingStatistics == nil {
				l.pendingStatistics = make(map[string]pendingGameStatistics)
			}
			if _, exists := l.pendingStatistics[job.gameID]; !exists {
				var sequence uint64
				for _, pending := range l.pendingStatistics {
					sequence = max(sequence, pending.Sequence)
				}
				l.pendingStatistics[job.gameID] = pendingGameStatistics{
					Sequence: sequence + 1,
					Kind:     job.kind, Status: job.status, Checkpoint: job.checkpoint,
					Completed: job.completed, CompletedAt: job.completedAt,
				}
			}
		} else {
			jobs = append(jobs, job)
		}
	}
	terminalJobs := make([]statisticsSaveJob, 0, len(l.pendingStatistics))
	for gameID, record := range l.pendingStatistics {
		terminalJobs = append(terminalJobs, statisticsSaveJob{
			sequence: record.Sequence,
			gameID:   gameID, roomCode: record.Checkpoint.RoomCode, phase: game.PhaseGameOver,
			kind: record.Kind, status: record.Status, checkpoint: record.Checkpoint,
			completed: record.Completed, completedAt: record.CompletedAt,
		})
	}
	// Legacy records have sequence zero; replay them first in completion order.
	sort.Slice(terminalJobs, func(i, j int) bool {
		a, b := terminalJobs[i], terminalJobs[j]
		if a.sequence != b.sequence {
			return a.sequence < b.sequence
		}
		if !a.completedAt.Equal(b.completedAt) {
			return a.completedAt.Before(b.completedAt)
		}
		return a.gameID < b.gameID
	})
	return append(jobs, terminalJobs...), changed
}

func (l *lobbyServer) saveStatisticsJobsLocked(ctx context.Context, jobs []statisticsSaveJob) bool {
	store, ok := l.store.(gameStatisticsStore)
	if !ok {
		return false
	}
	changed := false
	blockedPlayers := make(map[string]bool)

	for _, job := range jobs {
		players := job.checkpoint.Players
		if job.kind == "completed" {
			players = job.completed.Players
		}
		blocked := false
		for _, player := range players {
			blocked = blocked || blockedPlayers[player.UserID]
		}
		if job.phase == game.PhaseGameOver && blocked {
			// Propagate dependencies: a skipped A/B game also blocks B/C.
			for _, player := range players {
				blockedPlayers[player.UserID] = true
			}
			continue
		}
		l.mu.Unlock()
		var err error
		switch job.kind {
		case "completed":
			err = store.SaveCompletedGame(ctx, job.completed)
		case "unranked":
			err = store.SaveUnrankedGame(ctx, job.checkpoint, job.status, job.completedAt)
		case "checkpoint":
			err = store.SaveGameCheckpoint(ctx, job.checkpoint)
		}
		l.mu.Lock()
		if err != nil {
			slog.Error("persist game statistics failed", "roomCode", job.roomCode, "gameID", job.gameID, "error", err)
			if job.phase == game.PhaseGameOver {
				for _, player := range players {
					blockedPlayers[player.UserID] = true
				}
			}
			continue
		}
		if job.phase == game.PhaseGameOver {
			delete(l.pendingStatistics, job.gameID)
			changed = true
			// The active room may already be reset or running another game.
			for _, room := range l.rooms {
				if room != nil && room.statisticsGameID == job.gameID {
					job.room = room
					break
				}
			}
		}
		if job.room == nil {
			continue
		}
		job.room.statisticsDirty = false
		if job.phase == game.PhaseGameOver {
			job.room.statisticsSaved = true
		}
		changed = true
	}
	return changed
}

// Restored rooms start offline so a restart never counts toward active playtime.
func (r *room) updateStatisticsPlaytime(now time.Time) {
	if r == nil {
		return
	}
	if !r.statisticsActiveSince.IsZero() {
		if elapsed := now.Sub(r.statisticsActiveSince); elapsed > 0 {
			r.statisticsPlaytime += min(elapsed, statisticsIdleLimit)
		}
		r.statisticsActiveSince = time.Time{}
	}
	phase := r.gameStatePhase()
	if r.statisticsGameID != "" && phase != game.PhaseLobby && phase != game.PhaseGameOver && r.allPlayersConnected() {
		r.statisticsActiveSince = now
	}
}

func (l *lobbyServer) retryPendingStatistics() {
	l.commandMu.Lock()
	defer l.commandMu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.pendingStatistics) == 0 {
		return
	}
	_ = l.persistLocked("retry completed statistics")
}

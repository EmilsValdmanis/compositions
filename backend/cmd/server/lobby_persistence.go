package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sort"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/EmilsValdmanis/compositions/internal/game"
)

const persistedLobbyStateVersion = 1

type persistedLobbyState struct {
	Version           int                              `json:"version"`
	Sessions          []persistedPlayerSession         `json:"sessions"`
	Rooms             []persistedRoom                  `json:"rooms"`
	PendingStatistics map[string]pendingGameStatistics `json:"pendingStatistics,omitempty"`
}

type persistedPlayerSession struct {
	SessionID     string `json:"sessionId"`
	PlayerID      string `json:"playerId"`
	RoomCode      string `json:"roomCode,omitempty"`
	AuthUserID    string `json:"authUserId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

type persistedRoom struct {
	Code                     string                      `json:"code"`
	GameState                game.PersistenceSnapshot    `json:"gameState"`
	Players                  []persistedRoomPlayer       `json:"players"`
	HostID                   string                      `json:"hostId"`
	PendingDealChoice        *persistedPendingDealChoice `json:"pendingDealChoice,omitempty"`
	TurnBaseline             *game.GameSnapshot          `json:"turnBaseline,omitempty"`
	TurnActivity             *game.TurnActivitySnapshot  `json:"turnActivity,omitempty"`
	EndProposal              *persistedEndGameProposal   `json:"endProposal,omitempty"`
	Conclusion               *persistedGameConclusion    `json:"conclusion,omitempty"`
	StatisticsGameID         string                      `json:"statisticsGameId,omitempty"`
	StatisticsStartedAt      time.Time                   `json:"statisticsStartedAt,omitempty"`
	StatisticsPlaytime       time.Duration               `json:"statisticsPlaytime,omitempty"`
	StatisticsActiveSince    time.Time                   `json:"statisticsActiveSince,omitempty"`
	StatisticsSaved          bool                        `json:"statisticsSaved,omitempty"`
	StatisticsDirty          bool                        `json:"statisticsDirty,omitempty"`
	EndProposalCooldownUntil time.Time                   `json:"endProposalCooldownUntil,omitempty"`
}

type persistedPendingDealChoice struct {
	DealerIndex  int           `json:"dealerIndex"`
	ChooserIndex int           `json:"chooserIndex"`
	GameMode     game.GameMode `json:"gameMode,omitempty"`
}

type persistedRoomPlayer struct {
	PlayerID  string `json:"playerId"`
	Name      string `json:"name"`
	ImageURL  string `json:"imageUrl,omitempty"`
	SessionID string `json:"sessionId"`
	Connected bool   `json:"connected"`
	Seat      int    `json:"seat"`
	Host      bool   `json:"host"`
	Forfeited bool   `json:"forfeited,omitempty"`
}

type persistedEndGameProposal struct {
	ID                string    `json:"id"`
	Kind              string    `json:"kind"`
	ProposerPlayerID  string    `json:"proposerPlayerId"`
	Description       string    `json:"description,omitempty"`
	ReportID          string    `json:"reportId,omitempty"`
	EligiblePlayerIDs []string  `json:"eligiblePlayerIds"`
	AgreedPlayerIDs   []string  `json:"agreedPlayerIds"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
}

type persistedGameConclusion struct {
	Kind           string `json:"kind"`
	WinnerPlayerID string `json:"winnerPlayerId,omitempty"`
	ReportID       string `json:"reportId,omitempty"`
}

func (l *lobbyServer) restorePersistedState(ctx context.Context) error {
	if l == nil || l.store == nil {
		return nil
	}

	data, err := l.store.LoadLobbyState(ctx)
	if errors.Is(err, database.ErrLobbyStateNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var state persistedLobbyState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.Version == 0 && len(state.Rooms) == 0 && len(state.Sessions) == 0 {
		return nil
	}
	if state.Version != persistedLobbyStateVersion {
		return errors.New("unsupported lobby persistence snapshot version")
	}

	sessions := make(map[string]*playerSession, len(state.Sessions))
	for _, persistedSession := range state.Sessions {
		if persistedSession.SessionID == "" || persistedSession.PlayerID == "" {
			return errors.New("persisted session requires session and player ids")
		}
		if persistedSession.Authenticated && persistedSession.AuthUserID == "" {
			return errors.New("persisted authenticated session requires auth user id")
		}
		sessions[persistedSession.SessionID] = &playerSession{
			sessionID:     persistedSession.SessionID,
			playerID:      persistedSession.PlayerID,
			roomCode:      normalizeRoomCode(persistedSession.RoomCode),
			authUserID:    persistedSession.AuthUserID,
			displayName:   persistedSession.DisplayName,
			imageURL:      persistedSession.ImageURL,
			authenticated: persistedSession.Authenticated,
		}
	}

	rooms := make(map[string]*room, len(state.Rooms))
	for _, persistedRoom := range state.Rooms {
		roomCode := normalizeRoomCode(persistedRoom.Code)
		if roomCode == "" {
			return errors.New("persisted room requires code")
		}
		gameState, err := game.RestoreGameState(persistedRoom.GameState)
		if err != nil {
			return err
		}
		restoredRoom := &room{
			code:                     roomCode,
			gameState:                gameState,
			players:                  make([]*roomPlayer, 0, len(persistedRoom.Players)),
			hostID:                   persistedRoom.HostID,
			turnBaseline:             persistedRoom.TurnBaseline,
			turnActivity:             persistedRoom.TurnActivity,
			endProposalCooldownUntil: persistedRoom.EndProposalCooldownUntil,
			statisticsGameID:         persistedRoom.StatisticsGameID,
			statisticsStartedAt:      persistedRoom.StatisticsStartedAt,
			statisticsPlaytime:       persistedRoom.StatisticsPlaytime,
			statisticsSaved:          persistedRoom.StatisticsSaved,
			statisticsDirty:          persistedRoom.StatisticsDirty,
		}
		if persistedRoom.Conclusion != nil {
			restoredRoom.conclusion = &gameConclusion{
				kind:           persistedRoom.Conclusion.Kind,
				winnerPlayerID: persistedRoom.Conclusion.WinnerPlayerID,
				reportID:       persistedRoom.Conclusion.ReportID,
			}
		}
		if persistedRoom.EndProposal != nil {
			agreedPlayerIDs := make(map[string]bool, len(persistedRoom.EndProposal.AgreedPlayerIDs))
			for _, playerID := range persistedRoom.EndProposal.AgreedPlayerIDs {
				agreedPlayerIDs[playerID] = true
			}
			restoredRoom.endProposal = &endGameProposal{
				id:                persistedRoom.EndProposal.ID,
				kind:              persistedRoom.EndProposal.Kind,
				proposerPlayerID:  persistedRoom.EndProposal.ProposerPlayerID,
				description:       persistedRoom.EndProposal.Description,
				reportID:          persistedRoom.EndProposal.ReportID,
				eligiblePlayerIDs: append([]string(nil), persistedRoom.EndProposal.EligiblePlayerIDs...),
				agreedPlayerIDs:   agreedPlayerIDs,
				createdAt:         persistedRoom.EndProposal.CreatedAt,
				expiresAt:         persistedRoom.EndProposal.ExpiresAt,
			}
		}
		if persistedRoom.PendingDealChoice != nil {
			gameMode := persistedRoom.PendingDealChoice.GameMode
			if gameMode == "" {
				gameMode = game.GameModeFull
			}
			if !gameMode.Valid() {
				return errors.New("persisted deal choice has invalid game mode")
			}
			restoredRoom.pendingDealChoice = &pendingDealChoice{
				dealerIndex:  persistedRoom.PendingDealChoice.DealerIndex,
				chooserIndex: persistedRoom.PendingDealChoice.ChooserIndex,
				gameMode:     gameMode,
			}
		}
		for _, persistedPlayer := range persistedRoom.Players {
			if persistedPlayer.PlayerID == "" || persistedPlayer.SessionID == "" {
				return errors.New("persisted room player requires player and session ids")
			}
			if _, ok := sessions[persistedPlayer.SessionID]; !ok {
				return errors.New("persisted room player references missing session")
			}
			restoredRoom.players = append(restoredRoom.players, &roomPlayer{
				player:     newPlayerWithID(persistedPlayer.PlayerID),
				authUserID: sessions[persistedPlayer.SessionID].authUserID,
				name:       persistedPlayer.Name,
				imageURL:   persistedPlayer.ImageURL,
				sessionID:  persistedPlayer.SessionID,
				connected:  false,
				seat:       persistedPlayer.Seat,
				host:       persistedPlayer.Host,
				forfeited:  persistedPlayer.Forfeited,
			})
			if !persistedPlayer.Forfeited {
				sessions[persistedPlayer.SessionID].roomCode = roomCode
			}
		}
		if restoredRoom.hostID == "" && len(restoredRoom.players) > 0 {
			restoredRoom.hostID = restoredRoom.players[0].player.ID
			restoredRoom.players[0].host = true
		}
		if restoredRoom.pendingDealChoice != nil {
			if restoredRoom.pendingDealChoice.dealerIndex < 0 ||
				restoredRoom.pendingDealChoice.dealerIndex >= len(restoredRoom.players) ||
				restoredRoom.pendingDealChoice.chooserIndex < 0 ||
				restoredRoom.pendingDealChoice.chooserIndex >= len(restoredRoom.players) {
				return errors.New("invalid persisted deal choice")
			}
		}
		rooms[roomCode] = restoredRoom
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.sessions = sessions
	l.rooms = rooms
	l.pendingStatistics = state.PendingStatistics
	// Older snapshots may have a finished room but no independent retry record.
	l.statisticsJobsLocked()

	slog.Info("lobby state restored", "rooms", len(rooms), "sessions", len(sessions))
	return nil
}

func (l *lobbyServer) persistLocked(reason string) error {
	if l == nil || l.store == nil {
		if l != nil {
			l.mutationBefore = nil
		}
		return nil
	}

	now := time.Now().UTC()
	for _, room := range l.rooms {
		room.updateStatisticsPlaytime(now)
	}
	// Capture terminal records before the authoritative write, including their
	// original completion time and replay order. Never derive new jobs after I/O.
	jobs, changed := l.statisticsJobsLocked()
	snapshot := l.persistenceSnapshotLocked()
	tentative := lobbyMutationState{sessions: l.sessions, rooms: l.rooms, pendingStatistics: l.pendingStatistics}
	if l.mutationBefore != nil {
		l.sessions, l.rooms = l.mutationBefore.sessions, l.mutationBefore.rooms
		l.pendingStatistics = l.mutationBefore.pendingStatistics
	}
	// Persist the authoritative game state with its dirty checkpoint marker
	// before writing derived statistics. If this write fails, a statistics row
	// must not get ahead of the state we would restore after a restart.
	if err := l.saveLobbySnapshotLocked(snapshot); err != nil {
		slog.Error("persist lobby state failed", "reason", reason, "error", err)
		return fmt.Errorf("persist lobby state: %w", err)
	}
	l.sessions, l.rooms = tentative.sessions, tentative.rooms
	l.pendingStatistics = tentative.pendingStatistics
	l.mutationBefore = nil
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	if !l.saveStatisticsJobsLocked(ctx, jobs) && !changed {
		return nil
	}
	// Statistics are idempotent. Saving the cleared dirty/finalized marker in a
	// second write means a failure here merely causes a safe retry on restart.
	if err := l.saveLobbySnapshotLocked(l.persistenceSnapshotLocked()); err != nil {
		slog.Error("persist statistics marker failed", "reason", reason, "error", err)
	}
	return nil
}

// The caller holds commandMu, so reads can use mu during I/O without allowing another command to commit.
func (l *lobbyServer) saveLobbySnapshotLocked(snapshot persistedLobbyState) error {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	l.mu.Unlock()
	defer l.mu.Lock()
	return l.store.SaveLobbyState(ctx, data)
}

func (l *lobbyServer) persistenceSnapshotLocked() persistedLobbyState {
	state := persistedLobbyState{
		PendingStatistics: maps.Clone(l.pendingStatistics),
		Version:           persistedLobbyStateVersion,
		Sessions:          make([]persistedPlayerSession, 0, len(l.sessions)),
		Rooms:             make([]persistedRoom, 0, len(l.rooms)),
	}

	for _, session := range l.sessions {
		if session == nil {
			continue
		}
		state.Sessions = append(state.Sessions, persistedPlayerSession{
			SessionID:     session.sessionID,
			PlayerID:      session.playerID,
			RoomCode:      session.roomCode,
			AuthUserID:    session.authUserID,
			DisplayName:   session.displayName,
			ImageURL:      session.imageURL,
			Authenticated: session.authenticated,
		})
	}
	sort.Slice(state.Sessions, func(i, j int) bool {
		return state.Sessions[i].SessionID < state.Sessions[j].SessionID
	})

	for _, room := range l.rooms {
		if room == nil || room.gameState == nil {
			continue
		}
		persistedRoom := persistedRoom{
			Code:                     room.code,
			GameState:                room.gameState.PersistenceSnapshot(),
			Players:                  make([]persistedRoomPlayer, 0, len(room.players)),
			HostID:                   room.hostID,
			TurnBaseline:             cloneGameSnapshot(room.turnBaseline),
			TurnActivity:             cloneTurnActivitySnapshot(room.turnActivity),
			EndProposalCooldownUntil: room.endProposalCooldownUntil,
			StatisticsGameID:         room.statisticsGameID,
			StatisticsStartedAt:      room.statisticsStartedAt,
			StatisticsPlaytime:       room.statisticsPlaytime,
			StatisticsActiveSince:    room.statisticsActiveSince,
			StatisticsSaved:          room.statisticsSaved,
			StatisticsDirty:          room.statisticsDirty,
		}
		if room.endProposal != nil {
			agreedPlayerIDs := make([]string, 0, len(room.endProposal.agreedPlayerIDs))
			for _, playerID := range room.endProposal.eligiblePlayerIDs {
				if room.endProposal.agreedPlayerIDs[playerID] {
					agreedPlayerIDs = append(agreedPlayerIDs, playerID)
				}
			}
			persistedRoom.EndProposal = &persistedEndGameProposal{
				ID:                room.endProposal.id,
				Kind:              room.endProposal.kind,
				ProposerPlayerID:  room.endProposal.proposerPlayerID,
				Description:       room.endProposal.description,
				ReportID:          room.endProposal.reportID,
				EligiblePlayerIDs: append([]string(nil), room.endProposal.eligiblePlayerIDs...),
				AgreedPlayerIDs:   agreedPlayerIDs,
				CreatedAt:         room.endProposal.createdAt,
				ExpiresAt:         room.endProposal.expiresAt,
			}
		}
		if room.conclusion != nil {
			persistedRoom.Conclusion = &persistedGameConclusion{
				Kind:           room.conclusion.kind,
				WinnerPlayerID: room.conclusion.winnerPlayerID,
				ReportID:       room.conclusion.reportID,
			}
		}
		if room.pendingDealChoice != nil {
			gameMode := room.pendingDealChoice.gameMode
			if gameMode == "" {
				gameMode = game.GameModeFull
			}
			persistedRoom.PendingDealChoice = &persistedPendingDealChoice{
				DealerIndex:  room.pendingDealChoice.dealerIndex,
				ChooserIndex: room.pendingDealChoice.chooserIndex,
				GameMode:     gameMode,
			}
		}
		for _, player := range room.players {
			if player == nil || player.player == nil {
				continue
			}
			persistedRoom.Players = append(persistedRoom.Players, persistedRoomPlayer{
				PlayerID:  player.player.ID,
				Name:      player.name,
				ImageURL:  player.imageURL,
				SessionID: player.sessionID,
				Connected: player.connected,
				Seat:      player.seat,
				Host:      player.host,
				Forfeited: player.forfeited,
			})
		}
		sort.Slice(persistedRoom.Players, func(i, j int) bool {
			return persistedRoom.Players[i].Seat < persistedRoom.Players[j].Seat
		})
		state.Rooms = append(state.Rooms, persistedRoom)
	}
	sort.Slice(state.Rooms, func(i, j int) bool {
		return state.Rooms[i].Code < state.Rooms[j].Code
	})

	return state
}

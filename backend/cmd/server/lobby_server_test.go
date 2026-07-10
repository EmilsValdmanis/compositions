package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/gorilla/websocket"
)

type jsonLobbyStateStore struct {
	data         []byte
	saves        int
	saveErr      error
	loadErr      error
	bugReports   []database.GameBugReportRecord
	bugReportErr error
}

func (s *jsonLobbyStateStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	return user, nil
}

func (s *jsonLobbyStateStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (s *jsonLobbyStateStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (s *jsonLobbyStateStore) DeleteSession(context.Context, string) error {
	return database.ErrSessionNotFound
}

func (s *jsonLobbyStateStore) SaveLobbyState(_ context.Context, state persistedLobbyState) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	s.data = data
	s.saves++
	return nil
}

func (s *jsonLobbyStateStore) LoadLobbyState(context.Context) (persistedLobbyState, error) {
	if s.loadErr != nil {
		return persistedLobbyState{}, s.loadErr
	}
	if len(s.data) == 0 {
		return persistedLobbyState{}, nil
	}
	var state persistedLobbyState
	if err := json.Unmarshal(s.data, &state); err != nil {
		return persistedLobbyState{}, err
	}
	return state, nil
}

func (s *jsonLobbyStateStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	if s.bugReportErr != nil {
		return database.GameBugReportRecord{}, s.bugReportErr
	}
	s.bugReports = append(s.bugReports, report)
	return report, nil
}

func (s *jsonLobbyStateStore) Close() error { return nil }

func setGameStatePhaseForTest(t *testing.T, state *game.GameState, phase game.GamePhase) {
	t.Helper()

	field := reflect.ValueOf(state).Elem().FieldByName("phase")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().SetInt(int64(phase))
}

func newActiveLobbyForExitTests(t *testing.T, playerCount int) (*lobbyServer, []connectedEvent, string) {
	t.Helper()
	lobby := newLobbyServer()
	events := make([]connectedEvent, 0, playerCount)
	for index := range playerCount {
		event, _, _, err := lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect player %d error = %v", index, err)
		}
		events = append(events, event)
	}
	roomState, _, err := lobby.createRoom(events[0].SessionID, "Player 1")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	for index := 1; index < playerCount; index++ {
		if _, _, err := lobby.joinRoom(events[index].SessionID, roomState.Code, fmt.Sprintf("Player %d", index+1)); err != nil {
			t.Fatalf("joinRoom(player %d) error = %v", index, err)
		}
	}
	if _, _, err := lobby.startGame(events[0].SessionID, 0); err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	cutSize := 0
	if _, _, err := lobby.chooseDealing(events[playerCount-1].SessionID, "round_robin", dealingChoiceOptions{cutSize: &cutSize}); err != nil {
		t.Fatalf("chooseDealing() error = %v", err)
	}
	return lobby, events, roomState.Code
}

func TestLobbyForfeitGameKeepsCardsAvailableAndTransfersHost(t *testing.T) {
	lobby, events, roomCode := newActiveLobbyForExitTests(t, 3)
	room := lobby.rooms[roomCode]
	before := room.gameState.PersistenceSnapshot()
	hostHandCount := len(before.Players[0].Hand)
	beforeDrawCount := len(before.DrawPile)

	roomState, _, result, _, err := lobby.forfeitGame(events[0].SessionID)
	if err != nil {
		t.Fatalf("forfeitGame() error = %v", err)
	}
	if !result.OK || result.Action != "forfeit_game" {
		t.Fatalf("forfeit result = %#v", result)
	}
	if roomState.Players[0].Forfeited != true || roomState.Players[0].CanReconnect {
		t.Fatalf("forfeited player snapshot = %#v", roomState.Players[0])
	}
	if roomState.HostPlayerID != events[1].PlayerID {
		t.Fatalf("new host = %q; want %q", roomState.HostPlayerID, events[1].PlayerID)
	}
	if lobby.sessions[events[0].SessionID].roomCode != "" {
		t.Fatal("forfeited session remained attached to room")
	}
	after := room.gameState.PersistenceSnapshot()
	if len(after.DrawPile) != beforeDrawCount+hostHandCount {
		t.Fatalf("draw pile count = %d; want %d", len(after.DrawPile), beforeDrawCount+hostHandCount)
	}
	if !after.Players[0].Forfeited || len(after.Players[0].Hand) != 0 {
		t.Fatalf("persisted forfeited player = %#v", after.Players[0])
	}
}

func TestLobbyEndGameRequiresUnanimousApproval(t *testing.T) {
	lobby, events, _ := newActiveLobbyForExitTests(t, 3)
	roomState, _, _, err := lobby.requestEndGame(events[0].SessionID, "mutual_end")
	if err != nil {
		t.Fatalf("requestEndGame() error = %v", err)
	}
	if roomState.EndProposal == nil || len(roomState.EndProposal.AgreedPlayerIDs) != 1 {
		t.Fatalf("end proposal = %#v", roomState.EndProposal)
	}

	proposalID := roomState.EndProposal.ID
	roomState, _, _, err = lobby.voteEndGame(events[1].SessionID, proposalID, true)
	if err != nil {
		t.Fatalf("first voteEndGame() error = %v", err)
	}
	if roomState.Phase != "in_progress" {
		t.Fatalf("phase after partial vote = %q; want in_progress", roomState.Phase)
	}

	roomState, _, _, err = lobby.voteEndGame(events[2].SessionID, proposalID, true)
	if err != nil {
		t.Fatalf("second voteEndGame() error = %v", err)
	}
	if roomState.Phase != "game_over" || roomState.Conclusion == nil || roomState.Conclusion.Kind != "mutual_end" {
		t.Fatalf("unanimous result = %#v", roomState)
	}
}

func TestLobbyRejectedEndGameVoteStartsCooldown(t *testing.T) {
	lobby, events, _ := newActiveLobbyForExitTests(t, 2)
	roomState, _, _, err := lobby.requestEndGame(events[0].SessionID, "mutual_end")
	if err != nil {
		t.Fatalf("requestEndGame() error = %v", err)
	}
	if _, _, _, err := lobby.voteEndGame(events[1].SessionID, roomState.EndProposal.ID, false); err != nil {
		t.Fatalf("voteEndGame(reject) error = %v", err)
	}
	if _, _, _, err := lobby.requestEndGame(events[0].SessionID, "mutual_end"); err == nil || !strings.Contains(err.Error(), "wait before") {
		t.Fatalf("requestEndGame during cooldown error = %v", err)
	}
}

func TestLobbyTechnicalAbortRetainsIssueReport(t *testing.T) {
	lobby, events, roomCode := newActiveLobbyForExitTests(t, 2)
	store := &jsonLobbyStateStore{}
	lobby.store = store
	roomState, _, _, err := lobby.reportIssue(events[0].SessionID, "Discard pile became stuck", true)
	if err != nil {
		t.Fatalf("reportIssue() error = %v", err)
	}
	if roomState.EndProposal == nil || roomState.EndProposal.Kind != "technical_abort" {
		t.Fatalf("technical abort proposal = %#v", roomState.EndProposal)
	}
	if len(store.bugReports) != 1 {
		t.Fatalf("saved bug report count = %d; want 1", len(store.bugReports))
	}
	savedReport := store.bugReports[0]
	if savedReport.RoomCode != roomCode || savedReport.Description != "Discard pile became stuck" || !json.Valid(savedReport.GameState) || !savedReport.RequestedAbort {
		t.Fatalf("saved bug report = %#v", savedReport)
	}

	roomState, _, _, err = lobby.voteEndGame(events[1].SessionID, roomState.EndProposal.ID, true)
	if err != nil {
		t.Fatalf("voteEndGame() error = %v", err)
	}
	if roomState.Conclusion == nil || roomState.Conclusion.Kind != "technical_abort" || roomState.Conclusion.ReportID == "" {
		t.Fatalf("technical abort conclusion = %#v", roomState.Conclusion)
	}
}

func TestLobbyReportIssueDoesNotCreateProposalWhenDatabaseSaveFails(t *testing.T) {
	lobby, events, roomCode := newActiveLobbyForExitTests(t, 2)
	lobby.store = &jsonLobbyStateStore{bugReportErr: errors.New("database unavailable")}

	if _, _, _, err := lobby.reportIssue(events[0].SessionID, "Broken game", true); err == nil || !strings.Contains(err.Error(), "save bug report: database unavailable") {
		t.Fatalf("reportIssue() error = %v", err)
	}
	if lobby.rooms[roomCode].endProposal != nil {
		t.Fatal("end proposal created after bug report save failed")
	}
}

func TestLobbyServerRestoresPersistedRoomAndReconnectsPlayers(t *testing.T) {
	store := &jsonLobbyStateStore{}
	lobby := newLobbyServerWithStore(store)

	hostUser := authenticatedUser{ID: "auth-host", Name: "Host"}
	guestUser := authenticatedUser{ID: "auth-guest", Name: "Guest"}
	hostEvent, _, _, err := lobby.connectWithUser("", hostUser, nil)
	if err != nil {
		t.Fatalf("host connect() error = %v", err)
	}
	roomState, _, err := lobby.createRoom(hostEvent.SessionID, "ignored")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	guestEvent, _, _, err := lobby.connectWithUser("", guestUser, nil)
	if err != nil {
		t.Fatalf("guest connect() error = %v", err)
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, roomState.Code, "ignored"); err != nil {
		t.Fatalf("joinRoom() error = %v", err)
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	if store.saves == 0 {
		t.Fatal("store.saves = 0; want persisted lobby state")
	}

	restoredPending := newLobbyServerWithStore(store)
	if err := restoredPending.restorePersistedState(context.Background()); err != nil {
		t.Fatalf("restorePersistedState(pending) error = %v", err)
	}
	pendingRoom := restoredPending.rooms[roomState.Code]
	if pendingRoom == nil || pendingRoom.pendingDealChoice == nil {
		t.Fatalf("restored pending room = %#v; want pending deal choice", pendingRoom)
	}
	if pendingRoom.pendingDealChoice.dealerIndex != 0 || pendingRoom.pendingDealChoice.chooserIndex != 1 {
		t.Fatalf("pendingDealChoice = %#v; want dealer 0 chooser 1", pendingRoom.pendingDealChoice)
	}

	cutSize := 0
	if _, _, err := restoredPending.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: &cutSize}); err != nil {
		t.Fatalf("chooseDealing(restored pending) error = %v", err)
	}

	restoredGame := newLobbyServerWithStore(store)
	if err := restoredGame.restorePersistedState(context.Background()); err != nil {
		t.Fatalf("restorePersistedState(game) error = %v", err)
	}
	restoredRoom := restoredGame.rooms[roomState.Code]
	if restoredRoom == nil {
		t.Fatal("restored game room = nil; want room")
	}
	if restoredRoom.players[0].connected || restoredRoom.players[1].connected {
		t.Fatalf("restored connected flags = %v/%v; want both false before reconnect", restoredRoom.players[0].connected, restoredRoom.players[1].connected)
	}

	hostReconnect, hostRoomState, _, err := restoredGame.connectWithUser("", hostUser, nil)
	if err != nil {
		t.Fatalf("host reconnect error = %v", err)
	}
	if hostReconnect.PlayerID != hostEvent.PlayerID {
		t.Fatalf("host reconnect playerID = %q; want %q", hostReconnect.PlayerID, hostEvent.PlayerID)
	}
	if hostRoomState == nil || hostRoomState.Code != roomState.Code || hostRoomState.Phase != "in_progress" {
		t.Fatalf("host reconnect room = %#v; want in-progress room %s", hostRoomState, roomState.Code)
	}
	hostGameState, err := restoredGame.gameStateForSession(hostReconnect.SessionID, *hostRoomState)
	if err != nil {
		t.Fatalf("gameStateForSession(host) error = %v", err)
	}
	if hostGameState == nil || hostGameState.Game.Phase != game.PhaseInProgress || len(hostGameState.Game.Hand) != game.InitialHandSize {
		t.Fatalf("host game state = %#v; want in-progress private hand", hostGameState)
	}

	_, guestRoomState, _, err := restoredGame.connectWithUser("", guestUser, nil)
	if err != nil {
		t.Fatalf("guest reconnect error = %v", err)
	}
	if guestRoomState == nil || !guestRoomState.Players[0].Connected || !guestRoomState.Players[1].Connected {
		t.Fatalf("guest reconnect room = %#v; want both players connected", guestRoomState)
	}
}

func TestLobbyServerRestorePersistedStateValidation(t *testing.T) {
	validGameSnapshot := game.NewGameState().PersistenceSnapshot()
	validState := func() persistedLobbyState {
		return persistedLobbyState{
			Version: persistedLobbyStateVersion,
			Sessions: []persistedPlayerSession{
				{SessionID: "session-1", PlayerID: "player-1"},
			},
			Rooms: []persistedRoom{
				{
					Code:      "room1",
					GameState: validGameSnapshot,
					Players: []persistedRoomPlayer{
						{PlayerID: "player-1", SessionID: "session-1", Name: "Player One"},
					},
				},
			},
		}
	}

	tests := []struct {
		name   string
		mutate func(*persistedLobbyState)
	}{
		{"load error", func(state *persistedLobbyState) { *state = persistedLobbyState{} }},
		{"version", func(state *persistedLobbyState) { state.Version = 99 }},
		{"session ids", func(state *persistedLobbyState) { state.Sessions[0].SessionID = "" }},
		{"authenticated user id", func(state *persistedLobbyState) {
			state.Sessions[0].Authenticated = true
		}},
		{"room code", func(state *persistedLobbyState) { state.Rooms[0].Code = "" }},
		{"game state", func(state *persistedLobbyState) { state.Rooms[0].GameState.Version = 99 }},
		{"room player ids", func(state *persistedLobbyState) { state.Rooms[0].Players[0].PlayerID = "" }},
		{"missing session", func(state *persistedLobbyState) { state.Rooms[0].Players[0].SessionID = "missing" }},
		{"deal choice", func(state *persistedLobbyState) {
			state.Rooms[0].PendingDealChoice = &persistedPendingDealChoice{DealerIndex: 0, ChooserIndex: 9}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &jsonLobbyStateStore{}
			if test.name == "load error" {
				store.loadErr = errors.New("load boom")
			} else {
				state := validState()
				test.mutate(&state)
				data, err := json.Marshal(state)
				if err != nil {
					t.Fatalf("Marshal() error = %v", err)
				}
				store.data = data
			}
			lobby := newLobbyServerWithStore(store)
			if err := lobby.restorePersistedState(context.Background()); err == nil {
				t.Fatal("restorePersistedState() error = nil; want error")
			}
		})
	}
}

func TestLobbyServerPersistenceEdgeBranches(t *testing.T) {
	if err := (*lobbyServer)(nil).restorePersistedState(context.Background()); err != nil {
		t.Fatalf("nil restorePersistedState() error = %v", err)
	}
	(*lobbyServer)(nil).persistLocked("nil receiver")

	lobby := newLobbyServerWithStore(nil)
	if _, ok := lobby.store.(noopUserStore); !ok {
		t.Fatalf("store type = %T; want noopUserStore", lobby.store)
	}

	errorStore := &jsonLobbyStateStore{saveErr: errors.New("save boom")}
	lobby = newLobbyServerWithStore(errorStore)
	lobby.sessions["nil-session"] = nil
	lobby.sessions["session-1"] = &playerSession{sessionID: "session-1", playerID: "player-1"}
	lobby.rooms["nil-room"] = nil
	lobby.rooms["no-game"] = &room{code: "no-game"}
	lobby.rooms["with-skips"] = &room{
		code:      "with-skips",
		gameState: game.NewGameState(),
		players: []*roomPlayer{
			nil,
			{},
			{player: newPlayerWithID("player-1"), sessionID: "session-1", name: "Player One"},
		},
	}

	snapshot := lobby.persistenceSnapshotLocked()
	if len(snapshot.Sessions) != 1 {
		t.Fatalf("len(snapshot.Sessions) = %d; want 1", len(snapshot.Sessions))
	}
	if len(snapshot.Rooms) != 1 || len(snapshot.Rooms[0].Players) != 1 {
		t.Fatalf("snapshot.Rooms = %#v; want one room with one player", snapshot.Rooms)
	}
	lobby.persistLocked("save error is logged")
}

func TestLobbyServerAuthenticatedReconnectWithExistingSessionID(t *testing.T) {
	lobby := newLobbyServer()
	user := authenticatedUser{ID: "user-1", Name: "Player One"}
	event, _, _, err := lobby.connectWithUser("", user, nil)
	if err != nil {
		t.Fatalf("connectWithUser() error = %v", err)
	}

	reconnected, _, _, err := lobby.connectWithUser(event.SessionID, user, nil)
	if err != nil {
		t.Fatalf("connectWithUser(existing authenticated) error = %v", err)
	}
	if reconnected.SessionID != event.SessionID {
		t.Fatalf("reconnected sessionID = %q; want %q", reconnected.SessionID, event.SessionID)
	}

	if _, _, _, err := lobby.connectExistingSessionWithUser(event.SessionID, authenticatedUser{ID: "user-2", Name: "Player Two"}, nil); err == nil || err.Error() != "session belongs to a different user" {
		t.Fatalf("connectExistingSessionWithUser(different user) error = %v; want session belongs to a different user", err)
	}
}

func TestLobbyServerChooseDealingRejectsMissingGameStateAndUnexpectedPhase(t *testing.T) {
	t.Run("missing game state", func(t *testing.T) {
		lobby := newLobbyServer()
		hostEvent, _, _, err := lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect(host) error = %v", err)
		}
		roomState, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		guestEvent, _, _, err := lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect(guest) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestEvent.SessionID, roomState.Code, "Guest"); err != nil {
			t.Fatalf("joinRoom() error = %v", err)
		}
		if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
			t.Fatalf("startGame() error = %v", err)
		}
		lobby.rooms[roomState.Code].gameState = nil
		if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err == nil || err.Error() != "game state not initialized" {
			t.Fatalf("chooseDealing() error = %v; want game state not initialized", err)
		}
	})

	t.Run("unexpected phase", func(t *testing.T) {
		lobby := newLobbyServer()
		hostEvent, _, _, err := lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect(host) error = %v", err)
		}
		roomState, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		guestEvent, _, _, err := lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect(guest) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestEvent.SessionID, roomState.Code, "Guest"); err != nil {
			t.Fatalf("joinRoom() error = %v", err)
		}
		if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
			t.Fatalf("startGame() error = %v", err)
		}
		setGameStatePhaseForTest(t, lobby.rooms[roomState.Code].gameState, game.PhaseGameOver)
		if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err == nil || err.Error() != "game is not waiting for a dealing choice" {
			t.Fatalf("chooseDealing() error = %v; want game is not waiting for a dealing choice", err)
		}
	})
}

func TestLobbyServerSendEmoteErrorBranchesAndExpiredSnapshot(t *testing.T) {
	lobby := newLobbyServer()
	if _, _, err := lobby.sendEmote("missing", "👋"); err == nil {
		t.Fatal("sendEmote(missing session) error = nil; want error")
	}

	event, _, _, err := lobby.connect("", nil)
	if err != nil {
		t.Fatalf("connect() error = %v", err)
	}
	if _, _, err := lobby.sendEmote(event.SessionID, "👋"); err == nil || err.Error() != "join a room first" {
		t.Fatalf("sendEmote(no room) error = %v; want join a room first", err)
	}
	roomState, _, err := lobby.createRoom(event.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	room := lobby.rooms[roomState.Code]
	room.players[0].activeEmote = &playerEmote{id: "expired", emoji: "👋", expiresAt: time.Now().Add(-time.Second)}
	if snapshot := room.snapshot(); len(snapshot.Players) != 1 || snapshot.Players[0].ActiveEmote != nil {
		t.Fatalf("expired emote snapshot = %#v; want no active emote", snapshot.Players)
	}
	if room.players[0].activeEmote != nil {
		t.Fatal("expired activeEmote was not cleared")
	}
}

func TestLobbyServerCoverage(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()
	originalAddPlayer := addPlayerToGameState
	defer func() { addPlayerToGameState = originalAddPlayer }()

	lobby := newLobbyServer()
	if _, err := lobby.requireSession("missing"); err == nil {
		t.Fatal("requireSession(missing) error = nil; want error")
	}

	hostConn, _, closeHostPair := newSocketPair(t)
	defer closeHostPair()
	hostEvent, _, _, err := lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(new) error = %v", err)
	}
	if _, _, _, err := lobby.connect("missing", hostConn); err == nil {
		t.Fatal("connect(missing) error = nil; want error")
	}

	if _, _, err := lobby.createRoom(hostEvent.SessionID, "   "); err == nil {
		t.Fatal("createRoom(blank name) error = nil; want error")
	}
	hostRoom, recipients, err := lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	if len(recipients) != 1 || recipients[0] != hostConn {
		t.Fatalf("createRoom recipients = %v; want [%p]", recipients, hostConn)
	}
	if _, _, err := lobby.createRoom(hostEvent.SessionID, "Host Again"); err == nil {
		t.Fatal("createRoom(second room) error = nil; want error")
	}
	if _, _, err := lobby.createRoom("missing", "No Session"); err == nil {
		t.Fatal("createRoom(missing session) error = nil; want error")
	}

	guestConn, _, closeGuestPair := newSocketPair(t)
	defer closeGuestPair()
	guestEvent, _, _, err := lobby.connect("", guestConn)
	if err != nil {
		t.Fatalf("guest connect(new) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, "NOPE", "Guest"); err == nil {
		t.Fatal("joinRoom(missing room) error = nil; want error")
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "   "); err == nil {
		t.Fatal("joinRoom(blank name) error = nil; want error")
	}
	joinedRoom, _, err := lobby.joinRoom(guestEvent.SessionID, strings.ToLower(hostRoom.Code), "Guest")
	if err != nil {
		t.Fatalf("joinRoom() error = %v", err)
	}
	if len(joinedRoom.Players) != 2 {
		t.Fatalf("len(joinedRoom.Players) = %d; want 2", len(joinedRoom.Players))
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "Guest Again"); err == nil {
		t.Fatal("joinRoom(second room) error = nil; want error")
	}
	if _, _, err := lobby.joinRoom("missing", hostRoom.Code, "Ghost"); err == nil {
		t.Fatal("joinRoom(missing session) error = nil; want error")
	}

	thirdConn, _, closeThirdPair := newSocketPair(t)
	defer closeThirdPair()
	thirdEvent, _, _, err := lobby.connect("", thirdConn)
	if err != nil {
		t.Fatalf("third connect(new) error = %v", err)
	}
	fourthConn, _, closeFourthPair := newSocketPair(t)
	defer closeFourthPair()
	fourthEvent, _, _, err := lobby.connect("", fourthConn)
	if err != nil {
		t.Fatalf("fourth connect(new) error = %v", err)
	}
	fifthConn, _, closeFifthPair := newSocketPair(t)
	defer closeFifthPair()
	fifthEvent, _, _, err := lobby.connect("", fifthConn)
	if err != nil {
		t.Fatalf("fifth connect(new) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(thirdEvent.SessionID, hostRoom.Code, "Third"); err != nil {
		t.Fatalf("joinRoom(third) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(fourthEvent.SessionID, hostRoom.Code, "Fourth"); err != nil {
		t.Fatalf("joinRoom(fourth) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(fifthEvent.SessionID, hostRoom.Code, "Fifth"); err == nil {
		t.Fatal("joinRoom(room full) error = nil; want error")
	}

	notHostConn, _, closeNotHostPair := newSocketPair(t)
	defer closeNotHostPair()
	notHostEvent, _, _, err := lobby.connect("", notHostConn)
	if err != nil {
		t.Fatalf("notHost connect(new) error = %v", err)
	}
	if _, _, err := lobby.startGame(notHostEvent.SessionID, 0); err == nil {
		t.Fatal("startGame(no room) error = nil; want error")
	}
	if _, _, err := lobby.startGame("missing", 0); err == nil {
		t.Fatal("startGame(missing session) error = nil; want error")
	}
	if _, _, err := lobby.startGame(guestEvent.SessionID, 0); err == nil {
		t.Fatal("startGame(non host) error = nil; want error")
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 99); !errors.Is(err, game.ErrInvalidDealer) {
		t.Fatalf("startGame(invalid dealer) error = %v; want ErrInvalidDealer", err)
	}

	startedRoom, startRecipients, err := lobby.startGame(hostEvent.SessionID, 0)
	if err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	if startedRoom.Phase != "lobby" {
		t.Fatalf("startedRoom.Phase = %q; want lobby", startedRoom.Phase)
	}
	if startedRoom.PendingDealChoice == nil {
		t.Fatal("startedRoom.PendingDealChoice = nil; want pending dealing choice")
	}
	if startedRoom.PendingDealChoice.ChooserPlayerID != fourthEvent.PlayerID {
		t.Fatalf("startedRoom.PendingDealChoice.ChooserPlayerID = %q; want %q", startedRoom.PendingDealChoice.ChooserPlayerID, fourthEvent.PlayerID)
	}
	if len(startRecipients) != 4 {
		t.Fatalf("len(startRecipients) = %d; want 4", len(startRecipients))
	}
	if _, _, err := lobby.joinRoom(notHostEvent.SessionID, hostRoom.Code, "Late"); err == nil {
		t.Fatal("joinRoom(while starting) error = nil; want error")
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err == nil {
		t.Fatal("startGame(second start request) error = nil; want error")
	}
	if _, _, err := lobby.chooseDealing(hostEvent.SessionID, "round_robin"); err == nil {
		t.Fatal("chooseDealing(non chooser) error = nil; want error")
	}
	if _, _, err := lobby.chooseDealing(fourthEvent.SessionID, "tap", dealingChoiceOptions{cutSize: intPtr(0)}); !errors.Is(err, game.ErrInvalidDealingOrder) {
		t.Fatalf("chooseDealing(tap without order) error = %v; want ErrInvalidDealingOrder", err)
	}
	startedRoom, gameRecipients, err := lobby.chooseDealing(fourthEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)})
	if err != nil {
		t.Fatalf("chooseDealing() error = %v", err)
	}
	if startedRoom.Phase != "in_progress" {
		t.Fatalf("startedRoom.Phase = %q; want in_progress", startedRoom.Phase)
	}
	if startedRoom.PendingDealChoice != nil {
		t.Fatalf("startedRoom.PendingDealChoice = %#v; want nil", startedRoom.PendingDealChoice)
	}
	if len(gameRecipients) != 4 {
		t.Fatalf("len(gameRecipients) = %d; want 4", len(gameRecipients))
	}

	afterReconnectEvent, reconnectRoom, reconnectRecipients, err := lobby.connect(guestEvent.SessionID, guestConn)
	if err != nil {
		t.Fatalf("connect(existing) error = %v", err)
	}
	if afterReconnectEvent.PlayerID != guestEvent.PlayerID {
		t.Fatalf("afterReconnectEvent.PlayerID = %q; want %q", afterReconnectEvent.PlayerID, guestEvent.PlayerID)
	}
	if reconnectRoom == nil || reconnectRoom.Phase != "in_progress" {
		t.Fatalf("reconnectRoom = %#v; want in-progress room", reconnectRoom)
	}
	if len(reconnectRecipients) != 4 {
		t.Fatalf("len(reconnectRecipients) = %d; want 4", len(reconnectRecipients))
	}

	if _, _, err := lobby.joinRoom(notHostEvent.SessionID, hostRoom.Code, "Late"); err == nil {
		t.Fatal("joinRoom(after start) error = nil; want error")
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 99); err == nil {
		t.Fatal("startGame(second start) error = nil; want error")
	}

	gameRoom := lobby.rooms[hostRoom.Code]
	if gameRoom == nil {
		t.Fatal("room = nil; want room")
	}
	if player := gameRoom.playerByID("missing"); player != nil {
		t.Fatalf("room.playerByID(missing) = %#v; want nil", player)
	}
	if !gameRoom.allPlayersConnected() {
		t.Fatal("room.allPlayersConnected() = false; want true")
	}
	if len(gameRoom.connectedConns(lobby.sessions)) != 4 {
		t.Fatalf("len(room.connectedConns()) = %d; want 4", len(gameRoom.connectedConns(lobby.sessions)))
	}

	lobby.disconnect(guestEvent.SessionID, notHostConn)
	lobby.disconnect("missing", guestConn)
	lobby.disconnect(guestEvent.SessionID, guestConn)
	if gameRoom.players[1].connected {
		t.Fatal("room.players[1].connected = true; want false")
	}
	if gameRoom.allPlayersConnected() {
		t.Fatal("room.allPlayersConnected() = true; want false")
	}
	if len(gameRoom.connectedConns(lobby.sessions)) != 3 {
		t.Fatalf("len(room.connectedConns()) = %d; want 3", len(gameRoom.connectedConns(lobby.sessions)))
	}

	makeGameState = func() *game.GameState { return nil }
	if _, _, err := lobby.createRoom(hostEvent.SessionID, "Broken"); err == nil {
		t.Fatal("createRoom(nil game state) error = nil; want error")
	}
	makeGameState = originalMakeGameState
	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return errors.New("add player boom")
	}
	if _, _, err := lobby.createRoom(hostEvent.SessionID, "Broken Add"); err == nil {
		t.Fatal("createRoom(add player error) error = nil; want error")
	}
	addPlayerToGameState = originalAddPlayer

	joinTestConn, _, closeJoinTestPair := newSocketPair(t)
	defer closeJoinTestPair()
	joinTestEvent, _, _, err := lobby.connect("", joinTestConn)
	if err != nil {
		t.Fatalf("connect(join test) error = %v", err)
	}
	joinTestRoom, _, err := lobby.createRoom(joinTestEvent.SessionID, "Join Test")
	if err != nil {
		t.Fatalf("createRoom(join test) error = %v", err)
	}
	joinTargetConn, _, closeJoinTargetPair := newSocketPair(t)
	defer closeJoinTargetPair()
	joinTargetEvent, _, _, err := lobby.connect("", joinTargetConn)
	if err != nil {
		t.Fatalf("connect(join target) error = %v", err)
	}
	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return errors.New("add player boom")
	}
	if _, _, err := lobby.joinRoom(joinTargetEvent.SessionID, joinTestRoom.Code, "Join Target"); err == nil {
		t.Fatal("joinRoom(add player error) error = nil; want error")
	}

	broadcastConn, _, closeBroadcastPair := newSocketPair(t)
	defer closeBroadcastPair()
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		return errors.New("emit boom")
	}
	lobby.broadcastDisconnect(hostRoom, []*websocket.Conn{nil, broadcastConn})
}

func TestLobbyChooseDealingOptions(t *testing.T) {
	cases := []struct {
		name     string
		dealType string
		options  dealingChoiceOptions
	}{
		{
			name:     "round robin with cut size",
			dealType: "round_robin",
			options:  dealingChoiceOptions{cutSize: intPtr(7)},
		},
		{
			name:     "tap with player order",
			dealType: "tap",
			options:  dealingChoiceOptions{order: []int{2, 0, 1}, cutSize: intPtr(5)},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			lobby := newLobbyServer()
			hostEvent, _, _, err := lobby.connect("", nil)
			if err != nil {
				t.Fatalf("connect(host) error = %v", err)
			}
			hostRoom, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
			if err != nil {
				t.Fatalf("createRoom() error = %v", err)
			}
			firstGuestEvent, _, _, err := lobby.connect("", nil)
			if err != nil {
				t.Fatalf("connect(first guest) error = %v", err)
			}
			if _, _, err := lobby.joinRoom(firstGuestEvent.SessionID, hostRoom.Code, "First Guest"); err != nil {
				t.Fatalf("joinRoom(first guest) error = %v", err)
			}
			chooserEvent, _, _, err := lobby.connect("", nil)
			if err != nil {
				t.Fatalf("connect(chooser) error = %v", err)
			}
			if _, _, err := lobby.joinRoom(chooserEvent.SessionID, hostRoom.Code, "Chooser"); err != nil {
				t.Fatalf("joinRoom(chooser) error = %v", err)
			}
			if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
				t.Fatalf("startGame() error = %v", err)
			}

			startedRoom, _, err := lobby.chooseDealing(chooserEvent.SessionID, tt.dealType, tt.options)
			if err != nil {
				t.Fatalf("chooseDealing() error = %v", err)
			}
			if startedRoom.Phase != "in_progress" {
				t.Fatalf("startedRoom.Phase = %q; want in_progress", startedRoom.Phase)
			}
			if startedRoom.PendingDealChoice != nil {
				t.Fatalf("startedRoom.PendingDealChoice = %#v; want nil", startedRoom.PendingDealChoice)
			}
		})
	}
}

func intPtr(value int) *int {
	return &value
}

func TestCreateRoomAddPlayerErrorWithFreshSession(t *testing.T) {
	t.Run("require active session connection", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()

		if err := lobby.requireActiveSessionConnection("missing", conn); err == nil || err.Error() != "session not found" {
			t.Fatalf("requireActiveSessionConnection(missing) error = %v; want session not found", err)
		}

		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		if err := lobby.requireActiveSessionConnection(event.SessionID, conn); err != nil {
			t.Fatalf("requireActiveSessionConnection(active) error = %v", err)
		}
	})

	t.Run("cloneDraftCompositionSnapshots clones insert index", func(t *testing.T) {
		tableIndex := 2
		insertIndex := 1
		source := []game.DraftCompositionSnapshot{{
			TableIndex:        &tableIndex,
			InsertIndex:       &insertIndex,
			CardInsertIndices: map[string]int{"ace-1": 0},
			ReclaimTargets:    map[string]int{"king-1": 2},
			Cards:             []game.CardSnapshot{{Rank: game.Ace, Suit: game.Spades}},
		}}
		cloned := cloneDraftCompositionSnapshots(source)

		if len(cloned) != 1 {
			t.Fatalf("len(cloned) = %d; want 1", len(cloned))
		}
		if cloned[0].TableIndex == nil || *cloned[0].TableIndex != tableIndex {
			t.Fatalf("cloned[0].TableIndex = %#v; want %d", cloned[0].TableIndex, tableIndex)
		}
		if cloned[0].InsertIndex == nil || *cloned[0].InsertIndex != insertIndex {
			t.Fatalf("cloned[0].InsertIndex = %#v; want %d", cloned[0].InsertIndex, insertIndex)
		}
		if cloned[0].CardInsertIndices["ace-1"] != 0 {
			t.Fatalf("cloned[0].CardInsertIndices = %#v; want ace-1 preserved", cloned[0].CardInsertIndices)
		}
		if cloned[0].ReclaimTargets["king-1"] != 2 {
			t.Fatalf("cloned[0].ReclaimTargets = %#v; want king-1 preserved", cloned[0].ReclaimTargets)
		}
		if &cloned[0].Cards[0] == &source[0].Cards[0] {
			t.Fatal("cloneDraftCompositionSnapshots() reused card backing array")
		}
		cloned[0].CardInsertIndices["ace-1"] = 3
		cloned[0].ReclaimTargets["king-1"] = 4
		if source[0].CardInsertIndices["ace-1"] != 0 || source[0].ReclaimTargets["king-1"] != 2 {
			t.Fatal("cloneDraftCompositionSnapshots() reused draft metadata maps")
		}
	})

	t.Run("authenticated reconnect requires authenticated user", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()

		event, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "Host User"}, conn)
		if err != nil {
			t.Fatalf("connectWithUser() error = %v", err)
		}

		reconnectConn, _, reconnectCleanup := newSocketPair(t)
		defer reconnectCleanup()

		if _, _, _, err := lobby.connectExistingSessionWithUser(event.SessionID, authenticatedUser{}, reconnectConn); !errors.Is(err, errAuthenticationRequired) {
			t.Fatalf("connectExistingSessionWithUser() error = %v; want errAuthenticationRequired", err)
		}
	})

	t.Run("authenticated connect ignores another users supplied session id", func(t *testing.T) {
		lobby := newLobbyServer()
		firstConn, _, firstCleanup := newSocketPair(t)
		defer firstCleanup()

		firstEvent, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "First User"}, firstConn)
		if err != nil {
			t.Fatalf("connectWithUser(first) error = %v", err)
		}
		firstRoom, _, err := lobby.createRoom(firstEvent.SessionID, "ignored")
		if err != nil {
			t.Fatalf("createRoom(first) error = %v", err)
		}

		secondConn, _, secondCleanup := newSocketPair(t)
		defer secondCleanup()

		secondEvent, secondRoomState, secondRecipients, err := lobby.connectWithUser(firstEvent.SessionID, authenticatedUser{ID: "user-2", Name: "Second User"}, secondConn)
		if err != nil {
			t.Fatalf("connectWithUser(second with stolen session id) error = %v", err)
		}
		if secondEvent.SessionID == firstEvent.SessionID {
			t.Fatal("secondEvent.SessionID reused first user's session; want independent session")
		}
		if secondEvent.PlayerID == firstEvent.PlayerID {
			t.Fatal("secondEvent.PlayerID reused first user's player; want independent player")
		}
		if secondRoomState != nil {
			t.Fatalf("secondRoomState = %#v; want nil", secondRoomState)
		}
		if len(secondRecipients) != 0 {
			t.Fatalf("len(secondRecipients) = %d; want 0", len(secondRecipients))
		}
		if lobby.sessions[firstEvent.SessionID].roomCode != firstRoom.Code {
			t.Fatalf("first user's roomCode = %q; want %q", lobby.sessions[firstEvent.SessionID].roomCode, firstRoom.Code)
		}
		if lobby.sessions[secondEvent.SessionID].authUserID != "user-2" {
			t.Fatalf("second session authUserID = %q; want user-2", lobby.sessions[secondEvent.SessionID].authUserID)
		}
	})

	t.Run("authenticated connect replaces second live socket", func(t *testing.T) {
		lobby := newLobbyServer()
		firstConn, _, firstCleanup := newSocketPair(t)
		defer firstCleanup()

		firstEvent, firstRoomState, firstRecipients, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "First Name", Image: "https://cdn.example.com/first.png"}, firstConn)
		if err != nil {
			t.Fatalf("connectWithUser(first) error = %v", err)
		}
		if firstRoomState != nil {
			t.Fatalf("firstRoomState = %#v; want nil", firstRoomState)
		}
		if len(firstRecipients) != 0 {
			t.Fatalf("len(firstRecipients) = %d; want 0", len(firstRecipients))
		}
		if len(lobby.sessions) != 1 {
			t.Fatalf("len(lobby.sessions) = %d; want 1", len(lobby.sessions))
		}

		secondConn, _, secondCleanup := newSocketPair(t)
		defer secondCleanup()

		secondEvent, secondRoomState, secondRecipients, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "Updated Name", Image: "https://cdn.example.com/updated.png"}, secondConn)
		if err != nil {
			t.Fatalf("connectWithUser(second) error = %v", err)
		}
		if secondEvent.SessionID != firstEvent.SessionID {
			t.Fatalf("secondEvent.SessionID = %q; want %q", secondEvent.SessionID, firstEvent.SessionID)
		}
		if secondEvent.PlayerID != firstEvent.PlayerID {
			t.Fatalf("secondEvent.PlayerID = %q; want %q", secondEvent.PlayerID, firstEvent.PlayerID)
		}
		if secondRoomState != nil {
			t.Fatalf("secondRoomState = %#v; want nil", secondRoomState)
		}
		if len(secondRecipients) != 0 {
			t.Fatalf("len(secondRecipients) = %d; want 0", len(secondRecipients))
		}
		if len(lobby.sessions) != 1 {
			t.Fatalf("len(lobby.sessions) = %d; want 1 after replacing second socket", len(lobby.sessions))
		}

		session := lobby.sessions[firstEvent.SessionID]
		if session == nil {
			t.Fatal("session = nil; want existing session")
		}
		if session.conn != secondConn {
			t.Fatal("session.conn changed; want second connection to become active")
		}
		if session.displayName != "Updated Name" {
			t.Fatalf("session.displayName = %q; want Updated Name", session.displayName)
		}
		if session.imageURL != "https://cdn.example.com/updated.png" {
			t.Fatalf("session.imageURL = %q; want https://cdn.example.com/updated.png", session.imageURL)
		}
		if err := lobby.requireActiveSessionConnection(firstEvent.SessionID, firstConn); err == nil || err.Error() != "session not active on this connection" {
			t.Fatalf("requireActiveSessionConnection(stale) error = %v; want session not active on this connection", err)
		}
	})

	t.Run("authenticated connect reuses session after disconnect", func(t *testing.T) {
		lobby := newLobbyServer()
		firstConn, _, firstCleanup := newSocketPair(t)
		defer firstCleanup()

		firstEvent, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "First Name", Image: "https://cdn.example.com/first.png"}, firstConn)
		if err != nil {
			t.Fatalf("connectWithUser(first) error = %v", err)
		}

		lobby.disconnect(firstEvent.SessionID, firstConn)

		secondConn, _, secondCleanup := newSocketPair(t)
		defer secondCleanup()

		secondEvent, secondRoomState, secondRecipients, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "Updated Name", Image: "https://cdn.example.com/updated.png"}, secondConn)
		if err != nil {
			t.Fatalf("connectWithUser(second) error = %v", err)
		}
		if secondEvent.SessionID != firstEvent.SessionID {
			t.Fatalf("secondEvent.SessionID = %q; want %q", secondEvent.SessionID, firstEvent.SessionID)
		}
		if secondEvent.PlayerID != firstEvent.PlayerID {
			t.Fatalf("secondEvent.PlayerID = %q; want %q", secondEvent.PlayerID, firstEvent.PlayerID)
		}
		if secondRoomState != nil {
			t.Fatalf("secondRoomState = %#v; want nil", secondRoomState)
		}
		if len(secondRecipients) != 0 {
			t.Fatalf("len(secondRecipients) = %d; want 0", len(secondRecipients))
		}

		session := lobby.sessions[firstEvent.SessionID]
		if session == nil {
			t.Fatal("session = nil; want existing session")
		}
		if session.conn != secondConn {
			t.Fatal("session.conn was not updated to reconnected connection")
		}
		if session.displayName != "Updated Name" {
			t.Fatalf("session.displayName = %q; want Updated Name", session.displayName)
		}
		if session.imageURL != "https://cdn.example.com/updated.png" {
			t.Fatalf("session.imageURL = %q; want https://cdn.example.com/updated.png", session.imageURL)
		}
	})

	t.Run("authenticated reconnect updates room player name", func(t *testing.T) {
		lobby := newLobbyServer()
		firstConn, _, firstCleanup := newSocketPair(t)
		defer firstCleanup()

		firstEvent, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "First Name", Image: "https://cdn.example.com/first.png"}, firstConn)
		if err != nil {
			t.Fatalf("connectWithUser(first) error = %v", err)
		}
		roomState, _, err := lobby.createRoom(firstEvent.SessionID, "ignored")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		if roomState.Players[0].Name != "First Name" {
			t.Fatalf("roomState.Players[0].Name = %q; want First Name", roomState.Players[0].Name)
		}
		if roomState.Players[0].ImageURL != "https://cdn.example.com/first.png" {
			t.Fatalf("roomState.Players[0].ImageURL = %q; want https://cdn.example.com/first.png", roomState.Players[0].ImageURL)
		}

		lobby.disconnect(firstEvent.SessionID, firstConn)

		secondConn, _, secondCleanup := newSocketPair(t)
		defer secondCleanup()

		secondEvent, secondRoomState, secondRecipients, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "Updated Name", Image: "https://cdn.example.com/updated.png"}, secondConn)
		if err != nil {
			t.Fatalf("connectWithUser(second) error = %v", err)
		}
		if secondEvent.SessionID != firstEvent.SessionID {
			t.Fatalf("secondEvent.SessionID = %q; want %q", secondEvent.SessionID, firstEvent.SessionID)
		}
		if secondRoomState == nil {
			t.Fatal("secondRoomState = nil; want room snapshot")
		}
		if secondRoomState.Players[0].Name != "Updated Name" {
			t.Fatalf("secondRoomState.Players[0].Name = %q; want Updated Name", secondRoomState.Players[0].Name)
		}
		if secondRoomState.Players[0].ImageURL != "https://cdn.example.com/updated.png" {
			t.Fatalf("secondRoomState.Players[0].ImageURL = %q; want https://cdn.example.com/updated.png", secondRoomState.Players[0].ImageURL)
		}
		if len(secondRecipients) != 1 || secondRecipients[0] != secondConn {
			t.Fatalf("secondRecipients = %v; want [%p]", secondRecipients, secondConn)
		}

		if err := lobby.requireActiveSessionConnection(firstEvent.SessionID, firstConn); err == nil || err.Error() != "session not active on this connection" {
			t.Fatalf("requireActiveSessionConnection(stale) error = %v; want session not active on this connection", err)
		}
	})

	t.Run("authenticated room snapshot includes image url", func(t *testing.T) {
		lobby := newLobbyServer()
		hostConn, _, closeHostPair := newSocketPair(t)
		defer closeHostPair()

		hostEvent, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "Host User", Image: "https://cdn.example.com/host.png"}, hostConn)
		if err != nil {
			t.Fatalf("connectWithUser(host) error = %v", err)
		}

		roomState, _, err := lobby.createRoom(hostEvent.SessionID, "ignored")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		if roomState.Players[0].ImageURL != "https://cdn.example.com/host.png" {
			t.Fatalf("roomState.Players[0].ImageURL = %q; want https://cdn.example.com/host.png", roomState.Players[0].ImageURL)
		}

		guestConn, _, closeGuestPair := newSocketPair(t)
		defer closeGuestPair()

		guestEvent, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "user-2", Name: "Guest User", Image: "https://cdn.example.com/guest.png"}, guestConn)
		if err != nil {
			t.Fatalf("connectWithUser(guest) error = %v", err)
		}

		joinedRoom, _, err := lobby.joinRoom(guestEvent.SessionID, roomState.Code, "ignored")
		if err != nil {
			t.Fatalf("joinRoom() error = %v", err)
		}
		if joinedRoom.Players[1].ImageURL != "https://cdn.example.com/guest.png" {
			t.Fatalf("joinedRoom.Players[1].ImageURL = %q; want https://cdn.example.com/guest.png", joinedRoom.Players[1].ImageURL)
		}
	})

	t.Run("stale room membership is cleared", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()

		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		createdRoom, _, err := lobby.createRoom(event.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}

		delete(lobby.rooms, createdRoom.Code)

		_, roomState, recipients, err := lobby.connect(event.SessionID, conn)
		if err != nil {
			t.Fatalf("connect(existing) error = %v", err)
		}
		if roomState != nil {
			t.Fatalf("roomState = %#v; want nil", roomState)
		}
		if len(recipients) != 0 {
			t.Fatalf("len(recipients) = %d; want 0", len(recipients))
		}
		if lobby.sessions[event.SessionID].roomCode != "" {
			t.Fatalf("session.roomCode = %q; want empty", lobby.sessions[event.SessionID].roomCode)
		}

		if _, _, err := lobby.createRoom(event.SessionID, "Host Again"); err != nil {
			t.Fatalf("createRoom() after stale room cleanup error = %v", err)
		}
	})

	t.Run("nil game state", func(t *testing.T) {
		originalMakeGameState := makeGameState
		defer func() { makeGameState = originalMakeGameState }()
		makeGameState = func() *game.GameState { return nil }

		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		if _, _, err := lobby.createRoom(event.SessionID, "Host"); err == nil {
			t.Fatal("createRoom(nil game state) error = nil; want error")
		}
	})

	t.Run("add player error", func(t *testing.T) {
		originalAddPlayer := addPlayerToGameState
		defer func() { addPlayerToGameState = originalAddPlayer }()
		addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
			return errors.New("add player boom")
		}

		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		if _, _, err := lobby.createRoom(event.SessionID, "Host"); err == nil {
			t.Fatal("createRoom(add player error) error = nil; want error")
		}
	})
}

func TestRoomTurnTrackingAndActivitySnapshots(t *testing.T) {
	room := &room{}
	room.clearTurnTracking()
	if baseline, activity := room.turnContextForPlayer("other"); baseline != nil || activity != nil {
		t.Fatalf("turnContextForPlayer() = %#v %#v; want nil nil without tracking", baseline, activity)
	}

	state := game.NewGameStateWithDeck([]game.Card{})
	current := newPlayerWithID("current")
	other := newPlayerWithID("other")
	state.AddPlayer(current)
	state.AddPlayer(other)
	setGameStatePhaseForTest(t, state, game.PhaseInProgress)
	setField := func(field string, value int) {
		t.Helper()
		v := reflect.ValueOf(state).Elem().FieldByName(field)
		reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().SetInt(int64(value))
	}
	setField("round", 3)
	turnField := reflect.ValueOf(state).Elem().FieldByName("turn")
	turn := reflect.NewAt(turnField.Type(), unsafe.Pointer(turnField.UnsafeAddr())).Elem()
	turnNumberField := turn.FieldByName("number")
	reflect.NewAt(turnNumberField.Type(), unsafe.Pointer(turnNumberField.UnsafeAddr())).Elem().SetInt(9)
	turnPlayerIndexField := turn.FieldByName("playerIndex")
	reflect.NewAt(turnPlayerIndexField.Type(), unsafe.Pointer(turnPlayerIndexField.UnsafeAddr())).Elem().SetInt(0)
	comp := mustSetComposition(t, game.NewCard(game.Seven, game.Hearts), game.NewCard(game.Seven, game.Diamonds), game.NewCard(game.Seven, game.Clubs))
	activeField := reflect.ValueOf(state).Elem().FieldByName("activeCompositions")
	reflect.NewAt(activeField.Type(), unsafe.Pointer(activeField.UnsafeAddr())).Elem().Set(reflect.ValueOf([]*game.Composition{comp}))

	room.gameState = state
	room.resetTurnTracking("current")
	if room.turnBaseline == nil || room.turnActivity == nil {
		t.Fatal("turn tracking was not initialized")
	}
	if room.turnActivity.PlayerID != "current" || room.turnActivity.Round != 3 || room.turnActivity.TurnNumber != 9 {
		t.Fatalf("turnActivity = %#v; want current player round metadata", room.turnActivity)
	}
	if baseline, activity := room.turnContextForPlayer("current"); baseline != nil || activity != nil {
		t.Fatalf("active player should not receive spectator context, got %#v %#v", baseline, activity)
	}

	newComp := mustSetComposition(t, game.NewCard(game.King, game.Hearts), game.NewCard(game.King, game.Diamonds), game.NewCard(game.King, game.Spades))
	room.applySubmittedTurnActivity(
		"current",
		[]*game.Composition{newComp},
		[]game.CompositionAddition{{CompositionIndex: 0, Cards: []game.Card{game.NewCard(game.Ace, game.Hearts)}}},
		[]game.JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: game.NewCard(game.Queen, game.Hearts)}},
	)
	if len(room.turnActivity.DraftCompositions) != 2 {
		t.Fatalf("len(turnActivity.DraftCompositions) = %d; want 2", len(room.turnActivity.DraftCompositions))
	}
	if room.turnActivity.DraftCompositions[1].TableIndex == nil || *room.turnActivity.DraftCompositions[1].TableIndex != 0 {
		t.Fatalf("addition draft table index = %#v; want 0", room.turnActivity.DraftCompositions[1].TableIndex)
	}
	baseline, activity := room.turnContextForPlayer("other")
	if baseline == nil || activity == nil {
		t.Fatal("spectator turn context missing")
	}
	if len(baseline.ActiveCompositions) != 1 {
		t.Fatalf("len(baseline.ActiveCompositions) = %d; want 1", len(baseline.ActiveCompositions))
	}
	if len(activity.CompositionActivities) != 1 {
		t.Fatalf("len(activity.CompositionActivities) = %d; want 1", len(activity.CompositionActivities))
	}
	if activity.CompositionActivities[0].Kind != "new_composition" || activity.CompositionActivities[0].PlayerID != "current" {
		t.Fatalf("new composition activity = %#v; want current player new_composition", activity.CompositionActivities[0])
	}
	if activity.CompositionActivities[0].CardActivities[0].Kind != "addition" {
		t.Fatalf("addition activity = %#v; want addition marker", activity.CompositionActivities[0])
	}
	if activity.CompositionActivities[0].CardActivities[1].Kind != "joker_reclaim" {
		t.Fatalf("reclaim activity = %#v; want joker reclaim marker", activity.CompositionActivities[0])
	}
	activity.CompositionActivities[0].CardActivities[0] = game.CardActivitySnapshot{Kind: "mutated"}
	if room.turnActivity.CompositionActivities[0].CardActivities[0].Kind != "addition" {
		t.Fatal("turnContextForPlayer should clone activity maps")
	}

	room.clearTurnTracking()
	if room.turnBaseline != nil || room.turnActivity != nil {
		t.Fatalf("turn tracking after clear = %#v %#v; want nil nil", room.turnBaseline, room.turnActivity)
	}
}

func TestTurnTrackingEdgeCasesAndDraftActivityCoverage(t *testing.T) {
	t.Run("turn tracking nil and missing player guards", func(t *testing.T) {
		(*room)(nil).clearTurnTracking()
		var nilRoom *room
		nilRoom.resetTurnTracking("player")

		r := &room{gameState: game.NewGameState()}
		r.resetTurnTracking("missing")
		if r.turnBaseline != nil || r.turnActivity != nil {
			t.Fatalf("unexpected tracking for missing player: %#v %#v", r.turnBaseline, r.turnActivity)
		}

		r.applySubmittedTurnActivity("player", nil, nil, nil)
	})

	t.Run("merge helpers and submitted draft snapshots", func(t *testing.T) {
		if got := buildDraftSnapshotsFromSubmitted(nil, nil); len(got) != 0 {
			t.Fatalf("buildDraftSnapshotsFromSubmitted(nil,nil) len = %d; want 0", len(got))
		}
		if got := buildDraftSnapshotsFromSubmitted([]*game.Composition{nil}, nil); len(got) != 0 {
			t.Fatalf("buildDraftSnapshotsFromSubmitted([nil],nil) len = %d; want 0", len(got))
		}

		activity := &game.TurnActivitySnapshot{}
		mergeCompositionActivities(activity, nil)
		mergeCompositionActivities(nil, []game.CompositionActivitySnapshot{{TableIndex: 1}})
		mergeCompositionActivities(activity, []game.CompositionActivitySnapshot{{
			TableIndex:     2,
			CardActivities: map[int]game.CardActivitySnapshot{0: {Kind: "addition", PlayerID: "p1"}},
		}})
		mergeCompositionActivities(activity, []game.CompositionActivitySnapshot{{
			TableIndex: 2,
			Kind:       "new_composition",
			PlayerID:   "p1",
			CardActivities: map[int]game.CardActivitySnapshot{
				1: {Kind: "joker_reclaim", PlayerID: "p1"},
			},
		}})
		if len(activity.CompositionActivities) != 1 {
			t.Fatalf("len(CompositionActivities) = %d; want 1", len(activity.CompositionActivities))
		}
		if activity.CompositionActivities[0].Kind != "new_composition" || activity.CompositionActivities[0].PlayerID != "p1" {
			t.Fatalf("merged activity = %#v; want merged kind/player", activity.CompositionActivities[0])
		}
		if len(activity.CompositionActivities[0].CardActivities) != 2 {
			t.Fatalf("merged card activities = %#v; want 2 entries", activity.CompositionActivities[0].CardActivities)
		}
		mergeCompositionActivities(activity, []game.CompositionActivitySnapshot{{
			TableIndex: 2,
			Kind:       "ignored_kind",
			PlayerID:   "ignored_player",
		}})
		if activity.CompositionActivities[0].Kind != "new_composition" || activity.CompositionActivities[0].PlayerID != "p1" {
			t.Fatalf("merge should preserve existing kind/player, got %#v", activity.CompositionActivities[0])
		}
		mergeCompositionActivities(activity, []game.CompositionActivitySnapshot{{
			TableIndex:     2,
			CardActivities: map[int]game.CardActivitySnapshot{2: {Kind: "new", PlayerID: "p1"}},
		}})
		if activity.CompositionActivities[0].CardActivities[2].Kind != "new" {
			t.Fatalf("merge should add card activities into existing map, got %#v", activity.CompositionActivities[0].CardActivities)
		}
		emptyCardMapActivity := &game.TurnActivitySnapshot{CompositionActivities: []game.CompositionActivitySnapshot{{TableIndex: 7}}}
		mergeCompositionActivities(emptyCardMapActivity, []game.CompositionActivitySnapshot{{
			TableIndex:     7,
			CardActivities: map[int]game.CardActivitySnapshot{1: {Kind: "addition", PlayerID: "p2"}},
		}})
		if emptyCardMapActivity.CompositionActivities[0].CardActivities[1].Kind != "addition" {
			t.Fatalf("merge should initialize missing card activity map, got %#v", emptyCardMapActivity.CompositionActivities[0])
		}
		negativeNewCountComp := mustSetComposition(t, game.NewCard(game.Four, game.Hearts), game.NewCard(game.Four, game.Diamonds), game.NewCard(game.Four, game.Clubs))
		activities := buildCompositionActivities("p1", []*game.Composition{nil, negativeNewCountComp}, nil, nil, nil)
		if len(activities) != 1 || activities[0].TableIndex != 1 {
			t.Fatalf("buildCompositionActivities(new comps) = %#v; want one activity at table index 1", activities)
		}
		additionActivities := buildCompositionActivities(
			"p1",
			nil,
			[]game.CompositionAddition{{CompositionIndex: 0, Cards: []game.Card{game.NewCard(game.Two, game.Spades), game.NewCard(game.Three, game.Spades)}}},
			nil,
			[]*game.Composition{mustSetComposition(t, game.NewCard(game.Nine, game.Hearts), game.NewCard(game.Nine, game.Diamonds), game.NewCard(game.Nine, game.Clubs))},
		)
		if len(additionActivities) != 1 || additionActivities[0].CardActivities[1].Kind != "addition" {
			t.Fatalf("buildCompositionActivities(addition overflow) = %#v; want addition markers starting at zero", additionActivities)
		}
		prependIndex := 0
		prependActivities := buildCompositionActivities(
			"p1",
			nil,
			[]game.CompositionAddition{{
				CompositionIndex: 0,
				InsertIndex:      &prependIndex,
				Cards:            []game.Card{game.NewCard(game.Two, game.Spades)},
			}},
			nil,
			[]*game.Composition{mustSetComposition(t, game.NewCard(game.Nine, game.Hearts), game.NewCard(game.Nine, game.Diamonds), game.NewCard(game.Nine, game.Clubs))},
		)
		if len(prependActivities) != 1 || prependActivities[0].CardActivities[0].Kind != "addition" {
			t.Fatalf("buildCompositionActivities(prepend) = %#v; want addition marker at insert index", prependActivities)
		}
		negativeInsertIndex := -2
		negativeStartActivities := buildCompositionActivities(
			"p1",
			nil,
			[]game.CompositionAddition{{CompositionIndex: 0, InsertIndex: &negativeInsertIndex, Cards: []game.Card{
				game.NewCard(game.Two, game.Spades),
				game.NewCard(game.Three, game.Spades),
				game.NewCard(game.Four, game.Spades),
				game.NewCard(game.Five, game.Spades),
			}}},
			nil,
			[]*game.Composition{mustSetComposition(t, game.NewCard(game.Nine, game.Hearts), game.NewCard(game.Nine, game.Diamonds), game.NewCard(game.Nine, game.Clubs))},
		)
		if len(negativeStartActivities) != 1 || negativeStartActivities[0].CardActivities[0].Kind != "addition" || negativeStartActivities[0].CardActivities[3].Kind != "addition" {
			t.Fatalf("buildCompositionActivities(negative start index) = %#v; want addition markers clamped to zero", negativeStartActivities)
		}
		oversizedInsertIndex := 99
		oversizedStartActivities := buildCompositionActivities(
			"p1",
			nil,
			[]game.CompositionAddition{{
				CompositionIndex: 0,
				InsertIndex:      &oversizedInsertIndex,
				Cards:            []game.Card{game.NewCard(game.Six, game.Spades)},
			}},
			nil,
			[]*game.Composition{mustSetComposition(t, game.NewCard(game.Nine, game.Hearts), game.NewCard(game.Nine, game.Diamonds), game.NewCard(game.Nine, game.Clubs))},
		)
		if len(oversizedStartActivities) != 1 || oversizedStartActivities[0].CardActivities[3].Kind != "addition" {
			t.Fatalf("buildCompositionActivities(oversized start index) = %#v; want addition marker clamped to composition end", oversizedStartActivities)
		}
		reclaimActivities := buildCompositionActivities("p1", nil, nil, []game.JokerReclaim{{CompositionIndex: 3, JokerIndex: 2}}, nil)
		if len(reclaimActivities) != 1 || reclaimActivities[0].CardActivities[2].Kind != "joker_reclaim" {
			t.Fatalf("buildCompositionActivities(reclaim only) = %#v; want reclaim activity", reclaimActivities)
		}

		state := game.NewGameStateWithDeck([]game.Card{})
		player := newPlayerWithID("p1")
		state.AddPlayer(player)
		setGameStatePhaseForTest(t, state, game.PhaseInProgress)
		r := &room{gameState: state}
		r.applySubmittedTurnActivity("p1", nil, nil, nil)
		if r.turnActivity != nil {
			t.Fatalf("turnActivity = %#v; want nil without resetTurnTracking", r.turnActivity)
		}
		r.turnBaseline = &game.GameSnapshot{}
		r.turnActivity = &game.TurnActivitySnapshot{PlayerID: "other"}
		r.applySubmittedTurnActivity("p1", []*game.Composition{mustSetComposition(t, game.NewCard(game.Ace, game.Hearts), game.NewCard(game.Ace, game.Diamonds), game.NewCard(game.Ace, game.Clubs))}, nil, nil)
		if len(r.turnActivity.DraftCompositions) != 0 {
			t.Fatalf("wrong-player applySubmittedTurnActivity should not mutate drafts: %#v", r.turnActivity.DraftCompositions)
		}
	})

	t.Run("updateDraftActivity guard paths and resets tracking", func(t *testing.T) {
		lobby := newLobbyServer()
		if _, _, err := lobby.updateDraftActivity("missing", nil); err == nil || err.Error() != "session not found" {
			t.Fatalf("updateDraftActivity(missing session) error = %v; want session not found", err)
		}

		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		if _, _, err := lobby.updateDraftActivity(event.SessionID, nil); err == nil || err.Error() != "join a room first" {
			t.Fatalf("updateDraftActivity(no room) error = %v; want join a room first", err)
		}

		lobby.sessions[event.SessionID].roomCode = "ROOM"
		lobby.rooms["ROOM"] = &room{code: "ROOM", players: []*roomPlayer{{player: newPlayerWithID(event.PlayerID), sessionID: event.SessionID, connected: true, seat: 0}}}
		if _, _, err := lobby.updateDraftActivity(event.SessionID, nil); err == nil || err.Error() != "game state not initialized" {
			t.Fatalf("updateDraftActivity(nil game) error = %v; want game state not initialized", err)
		}

		state := game.NewGameState()
		player := newPlayerWithID(event.PlayerID)
		if err := state.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
		lobby.rooms["ROOM"].gameState = state
		if _, _, err := lobby.updateDraftActivity(event.SessionID, nil); err == nil || err != game.ErrGameNotInProgress {
			t.Fatalf("updateDraftActivity(lobby) error = %v; want ErrGameNotInProgress", err)
		}

		second := newPlayerWithID("other")
		if err := state.AddPlayer(second); err != nil {
			t.Fatalf("AddPlayer(second) error = %v", err)
		}
		setGameStatePhaseForTest(t, state, game.PhaseInProgress)
		turnField := reflect.ValueOf(state).Elem().FieldByName("turn")
		turn := reflect.NewAt(turnField.Type(), unsafe.Pointer(turnField.UnsafeAddr())).Elem()
		turnPlayerIndexField := turn.FieldByName("playerIndex")
		reflect.NewAt(turnPlayerIndexField.Type(), unsafe.Pointer(turnPlayerIndexField.UnsafeAddr())).Elem().SetInt(1)
		if _, _, err := lobby.updateDraftActivity(event.SessionID, nil); err == nil || err.Error() != "not your turn" {
			t.Fatalf("updateDraftActivity(not turn) error = %v; want not your turn", err)
		}

		reflect.NewAt(turnPlayerIndexField.Type(), unsafe.Pointer(turnPlayerIndexField.UnsafeAddr())).Elem().SetInt(0)
		roomState, recipients, err := lobby.updateDraftActivity(event.SessionID, []game.DraftCompositionSnapshot{{Cards: []game.CardSnapshot{{Rank: game.Ace, Suit: game.Hearts}}}})
		if err != nil {
			t.Fatalf("updateDraftActivity(success) error = %v", err)
		}
		if roomState.Code != "ROOM" || len(recipients) != 1 {
			t.Fatalf("updateDraftActivity success = %#v recipients=%d; want ROOM and 1 recipient", roomState, len(recipients))
		}
		if lobby.rooms["ROOM"].turnActivity == nil || len(lobby.rooms["ROOM"].turnActivity.DraftCompositions) != 1 {
			t.Fatalf("turnActivity after draft update = %#v; want 1 draft", lobby.rooms["ROOM"].turnActivity)
		}
		roomState, recipients, err = lobby.updateDraftActivity(event.SessionID, []game.DraftCompositionSnapshot{})
		if err != nil || roomState.Code != "ROOM" || len(recipients) != 1 {
			t.Fatalf("updateDraftActivity(second success) = %#v %d %v; want ROOM 1 nil", roomState, len(recipients), err)
		}
		otherConn, _, otherCleanup := newSocketPair(t)
		defer otherCleanup()
		otherEvent, _, _, err := lobby.connect("", otherConn)
		if err != nil {
			t.Fatalf("connect(other) error = %v", err)
		}
		lobby.sessions[otherEvent.SessionID].roomCode = "ROOM"
		lobby.rooms["ROOM"].players = append(lobby.rooms["ROOM"].players, &roomPlayer{
			player:    newPlayerWithID(otherEvent.PlayerID),
			sessionID: otherEvent.SessionID,
			connected: true,
			seat:      1,
		})
		if _, _, err := lobby.updateDraftActivity(event.SessionID, nil); err == nil || err.Error() != "game state snapshot failed" {
			t.Fatalf("updateDraftActivity(snapshot failure) error = %v; want game state snapshot failed", err)
		}
	})

	t.Run("applyGameAction afterMutate error and wrapper no-op branches", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		state := game.NewGameState()
		player := newPlayerWithID(event.PlayerID)
		other := newPlayerWithID("other")
		state.AddPlayer(player)
		state.AddPlayer(other)
		setGameStatePhaseForTest(t, state, game.PhaseInProgress)
		lobby.rooms["ROOM"] = &room{code: "ROOM", gameState: state, players: []*roomPlayer{{player: player, sessionID: event.SessionID, connected: true, seat: 0}}}
		lobby.sessions[event.SessionID].roomCode = "ROOM"

		if _, _, _, err := lobby.applyGameAction(event.SessionID, "test", func(*game.GameState) error { return nil }, func(*room, *playerSession) error { return errors.New("after boom") }); err == nil || err.Error() != "after boom" {
			t.Fatalf("applyGameAction(after error) = %v; want after boom", err)
		}

		if _, _, _, err := lobby.draw(event.SessionID, "sideways"); err == nil || err.Error() != "unknown draw source" {
			t.Fatalf("draw(sideways) error = %v; want unknown draw source", err)
		}
		if _, _, _, err := lobby.play(event.SessionID, nil, nil, nil); err == nil {
			t.Fatal("play() error = nil; want invalid composition path")
		}
	})
}

func mustSetComposition(t *testing.T, cards ...game.Card) *game.Composition {
	t.Helper()
	comp, ok := game.NewSet(cards)
	if !ok {
		t.Fatalf("NewSet(%#v) returned false", cards)
	}
	return comp
}

func TestLobbyLeaveRoomCoverage(t *testing.T) {
	originalMakeGameState := makeGameState
	originalAddPlayer := addPlayerToGameState
	defer func() { makeGameState = originalMakeGameState }()
	defer func() { addPlayerToGameState = originalAddPlayer }()

	t.Run("missing room", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		lobby.sessions[event.SessionID].roomCode = "NOPE"
		if _, _, _, err := lobby.leaveRoom(event.SessionID); err == nil || err.Error() != "join a room first" {
			t.Fatalf("leaveRoom(missing room) error = %v; want join a room first", err)
		}
		if lobby.sessions[event.SessionID].roomCode != "" {
			t.Fatalf("session.roomCode = %q; want empty", lobby.sessions[event.SessionID].roomCode)
		}
	})

	t.Run("missing session", func(t *testing.T) {
		lobby := newLobbyServer()
		if _, _, _, err := lobby.leaveRoom("missing"); err == nil || err.Error() != "session not found" {
			t.Fatalf("leaveRoom(missing session) error = %v; want session not found", err)
		}
	})

	t.Run("nil player entries are skipped", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}
		currentPlayer := &roomPlayer{player: newPlayerWithID(event.PlayerID), name: "Solo", sessionID: event.SessionID, connected: true, seat: 0, host: true}
		lobby.sessions[event.SessionID].roomCode = "ROOM"
		lobby.rooms["ROOM"] = &room{code: "ROOM", gameState: game.NewGameState(), players: []*roomPlayer{nil, currentPlayer}, hostID: currentPlayer.player.ID}

		snapshot, recipients, roomCode, err := lobby.leaveRoom(event.SessionID)
		if err != nil {
			t.Fatalf("leaveRoom() error = %v", err)
		}
		if snapshot != nil {
			t.Fatalf("snapshot = %#v; want nil", snapshot)
		}
		if len(recipients) != 0 {
			t.Fatalf("len(recipients) = %d; want 0", len(recipients))
		}
		if roomCode != "ROOM" {
			t.Fatalf("roomCode = %q; want ROOM", roomCode)
		}
		if _, exists := lobby.rooms["ROOM"]; exists {
			t.Fatal("room still exists after last player left")
		}
	})

	t.Run("nil game state creation", func(t *testing.T) {
		lobby := newLobbyServer()
		hostConn, _, closeHost := newSocketPair(t)
		defer closeHost()
		hostEvent, _, _, err := lobby.connect("", hostConn)
		if err != nil {
			t.Fatalf("connect(host) error = %v", err)
		}
		roomState, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		guestConn, _, closeGuest := newSocketPair(t)
		defer closeGuest()
		guestEvent, _, _, err := lobby.connect("", guestConn)
		if err != nil {
			t.Fatalf("connect(guest) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestEvent.SessionID, roomState.Code, "Guest"); err != nil {
			t.Fatalf("joinRoom() error = %v", err)
		}

		makeGameState = func() *game.GameState { return nil }
		if _, _, _, err := lobby.leaveRoom(guestEvent.SessionID); err == nil || err.Error() != "game state not initialized" {
			t.Fatalf("leaveRoom(nil game state) error = %v; want game state not initialized", err)
		}
		makeGameState = originalMakeGameState
	})

	t.Run("add player error rebuilding room", func(t *testing.T) {
		lobby := newLobbyServer()
		hostConn, _, closeHost := newSocketPair(t)
		defer closeHost()
		hostEvent, _, _, err := lobby.connect("", hostConn)
		if err != nil {
			t.Fatalf("connect(host) error = %v", err)
		}
		roomState, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		guestConn, _, closeGuest := newSocketPair(t)
		defer closeGuest()
		guestEvent, _, _, err := lobby.connect("", guestConn)
		if err != nil {
			t.Fatalf("connect(guest) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestEvent.SessionID, roomState.Code, "Guest"); err != nil {
			t.Fatalf("joinRoom() error = %v", err)
		}

		addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
			return errors.New("add player boom")
		}
		if _, _, _, err := lobby.leaveRoom(guestEvent.SessionID); err == nil || err.Error() != "add player boom" {
			t.Fatalf("leaveRoom(add player error) error = %v; want add player boom", err)
		}
		addPlayerToGameState = originalAddPlayer
	})

	t.Run("host reassignment and seat compaction", func(t *testing.T) {
		lobby := newLobbyServer()
		hostConn, _, closeHost := newSocketPair(t)
		defer closeHost()
		hostEvent, _, _, err := lobby.connect("", hostConn)
		if err != nil {
			t.Fatalf("connect(host) error = %v", err)
		}
		roomState, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}

		guestAConn, _, closeGuestA := newSocketPair(t)
		defer closeGuestA()
		guestAEvent, _, _, err := lobby.connect("", guestAConn)
		if err != nil {
			t.Fatalf("connect(guestA) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestAEvent.SessionID, roomState.Code, "Guest A"); err != nil {
			t.Fatalf("joinRoom(guestA) error = %v", err)
		}

		guestBConn, _, closeGuestB := newSocketPair(t)
		defer closeGuestB()
		guestBEvent, _, _, err := lobby.connect("", guestBConn)
		if err != nil {
			t.Fatalf("connect(guestB) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestBEvent.SessionID, roomState.Code, "Guest B"); err != nil {
			t.Fatalf("joinRoom(guestB) error = %v", err)
		}

		snapshot, recipients, roomCode, err := lobby.leaveRoom(hostEvent.SessionID)
		if err != nil {
			t.Fatalf("leaveRoom(host) error = %v", err)
		}
		if snapshot == nil {
			t.Fatal("leaveRoom(host) snapshot = nil; want snapshot")
		}
		if roomCode != roomState.Code {
			t.Fatalf("roomCode = %q; want %q", roomCode, roomState.Code)
		}
		if len(recipients) != 2 {
			t.Fatalf("len(recipients) = %d; want 2", len(recipients))
		}
		if snapshot.HostPlayerID != guestAEvent.PlayerID {
			t.Fatalf("snapshot.HostPlayerID = %q; want %q", snapshot.HostPlayerID, guestAEvent.PlayerID)
		}
		if len(snapshot.Players) != 2 {
			t.Fatalf("len(snapshot.Players) = %d; want 2", len(snapshot.Players))
		}
		if !snapshot.Players[0].IsHost || snapshot.Players[0].Seat != 0 {
			t.Fatalf("first player = %#v; want host at seat 0", snapshot.Players[0])
		}
		if snapshot.Players[1].IsHost || snapshot.Players[1].Seat != 1 {
			t.Fatalf("second player = %#v; want non-host at seat 1", snapshot.Players[1])
		}
		if got := lobby.sessions[hostEvent.SessionID].roomCode; got != "" {
			t.Fatalf("host session roomCode = %q; want empty", got)
		}
	})

	t.Run("fallback host when current host missing from players", func(t *testing.T) {
		lobby := newLobbyServer()
		conn, _, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := lobby.connect("", conn)
		if err != nil {
			t.Fatalf("connect() error = %v", err)
		}

		remaining := &roomPlayer{player: newPlayerWithID("remaining"), name: "Remaining", sessionID: event.SessionID, connected: true, seat: 3}
		lobby.sessions[event.SessionID].playerID = remaining.player.ID
		lobby.sessions[event.SessionID].roomCode = "ROOM"
		lobby.rooms["ROOM"] = &room{
			code:      "ROOM",
			gameState: game.NewGameState(),
			players: []*roomPlayer{
				{player: newPlayerWithID("ghost"), name: "Ghost", sessionID: "ghost-session", connected: false, seat: 0},
				remaining,
			},
			hostID: "missing-host",
		}

		snapshot, _, _, err := lobby.leaveRoom(event.SessionID)
		if err != nil {
			t.Fatalf("leaveRoom() error = %v", err)
		}
		if snapshot == nil || snapshot.HostPlayerID != "ghost" {
			t.Fatalf("snapshot.HostPlayerID = %v; want ghost", snapshot)
		}
		if !snapshot.Players[0].IsHost || snapshot.Players[0].Seat != 0 {
			t.Fatalf("snapshot.Players[0] = %#v; want host seat 0", snapshot.Players[0])
		}
	})
}

func BenchmarkLobbyServerCreateJoinStartGame(b *testing.B) {
	for b.Loop() {
		lobby := newLobbyServer()

		hostEvent, _, _, err := lobby.connect("", nil)
		if err != nil {
			b.Fatalf("connect(host) error = %v", err)
		}
		hostRoom, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			b.Fatalf("createRoom() error = %v", err)
		}

		guestEvent, _, _, err := lobby.connect("", nil)
		if err != nil {
			b.Fatalf("connect(guest) error = %v", err)
		}
		if _, _, err := lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "Guest"); err != nil {
			b.Fatalf("joinRoom() error = %v", err)
		}

		if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
			b.Fatalf("startGame() error = %v", err)
		}
		if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err != nil {
			b.Fatalf("chooseDealing() error = %v", err)
		}
	}
}

func TestRoomHelpersAndSnapshotCoverage(t *testing.T) {
	phaseCases := map[game.GamePhase]string{
		game.PhaseLobby:      "lobby",
		game.PhaseInProgress: "in_progress",
		game.PhaseRoundOver:  "round_over",
		game.PhaseGameOver:   "game_over",
		game.GamePhase(999):  "unknown",
	}
	for phase, want := range phaseCases {
		if got := phaseName(phase); got != want {
			t.Fatalf("phaseName(%v) = %q; want %q", phase, got, want)
		}
	}

	nilRoom := (*room)(nil)
	if nilRoom.gameStatePhase() != game.PhaseLobby {
		t.Fatalf("nilRoom.gameStatePhase() = %v; want %v", nilRoom.gameStatePhase(), game.PhaseLobby)
	}
	if nilRoom.gameStateDealerIndex() != 0 {
		t.Fatalf("nilRoom.gameStateDealerIndex() = %d; want 0", nilRoom.gameStateDealerIndex())
	}
	stubRoom := &room{}
	if stubRoom.gameStatePhase() != game.PhaseLobby {
		t.Fatalf("stubRoom.gameStatePhase() = %v; want %v", stubRoom.gameStatePhase(), game.PhaseLobby)
	}
	if stubRoom.gameStateDealerIndex() != 0 {
		t.Fatalf("stubRoom.gameStateDealerIndex() = %d; want 0", stubRoom.gameStateDealerIndex())
	}

	newPlayerValue := newPlayer()
	if newPlayerValue == nil || newPlayerValue.ID == "" {
		t.Fatalf("newPlayer() = %#v; want player with ID", newPlayerValue)
	}
	playerWithID := newPlayerWithID("fixed")
	if playerWithID.ID != "fixed" {
		t.Fatalf("newPlayerWithID() ID = %q; want fixed", playerWithID.ID)
	}

	lobby := newLobbyServer()
	brokenRoom := &room{code: "BROKEN", players: []*roomPlayer{{player: newPlayerWithID("p1"), connected: true, sessionID: "missing", seat: 0, host: true}, nil}, hostID: "p1"}
	brokenRoomSnapshot := brokenRoom.snapshot()
	if brokenRoomSnapshot.Phase != "lobby" {
		t.Fatalf("brokenRoomSnapshot.Phase = %q; want lobby", brokenRoomSnapshot.Phase)
	}
	if brokenRoomSnapshot.DealerIndex != 0 {
		t.Fatalf("brokenRoomSnapshot.DealerIndex = %d; want 0", brokenRoomSnapshot.DealerIndex)
	}
	if len(brokenRoom.connectedConns(lobby.sessions)) != 0 {
		t.Fatalf("len(brokenRoom.connectedConns()) = %d; want 0", len(brokenRoom.connectedConns(lobby.sessions)))
	}
	brokenRoom.pendingDealChoice = &pendingDealChoice{dealerIndex: 0, chooserIndex: 0}
	brokenRoomSnapshot = brokenRoom.snapshot()
	if brokenRoomSnapshot.PendingDealChoice == nil || brokenRoomSnapshot.PendingDealChoice.ChooserPlayerID != "p1" {
		t.Fatalf("brokenRoomSnapshot.PendingDealChoice = %#v; want chooser p1", brokenRoomSnapshot.PendingDealChoice)
	}
}

func TestDisconnectHandlesMissingRoomAndMissingPlayer(t *testing.T) {
	lobby := newLobbyServer()
	connA, _, closePairA := newSocketPair(t)
	defer closePairA()
	eventA, _, _, err := lobby.connect("", connA)
	if err != nil {
		t.Fatalf("connect(A) error = %v", err)
	}
	sessionA := lobby.sessions[eventA.SessionID]
	sessionA.roomCode = "NOPE"
	lobby.disconnect(eventA.SessionID, connA)

	connB, _, closePairB := newSocketPair(t)
	defer closePairB()
	eventB, _, _, err := lobby.connect("", connB)
	if err != nil {
		t.Fatalf("connect(B) error = %v", err)
	}
	lobby.rooms["ROOM"] = &room{code: "ROOM", gameState: game.NewGameState(), players: []*roomPlayer{}}
	sessionB := lobby.sessions[eventB.SessionID]
	sessionB.roomCode = "ROOM"
	lobby.disconnect(eventB.SessionID, connB)
}

func TestLobbyGameActionCoverage(t *testing.T) {
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()
	makeGameState = func() *game.GameState {
		return game.NewGameStateWithDeck(roundRobinDeckForServerTest(
			[]game.Card{
				game.NewCard(game.King, game.Hearts),
				game.NewCard(game.King, game.Diamonds),
				game.NewCard(game.King, game.Clubs),
				game.NewCard(game.Ace, game.Spades),
				game.NewCard(game.Two, game.Spades),
				game.NewCard(game.Three, game.Spades),
				game.NewCard(game.Five, game.Hearts),
				game.NewJoker(),
				game.NewCard(game.Seven, game.Hearts),
				game.NewCard(game.Six, game.Hearts),
				game.NewCard(game.Five, game.Spades),
				game.NewCard(game.Two, game.Clubs),
			},
			[]game.Card{
				game.NewCard(game.Ace, game.Clubs),
				game.NewCard(game.Ace, game.Diamonds),
				game.NewCard(game.Two, game.Clubs),
				game.NewCard(game.Four, game.Clubs),
				game.NewCard(game.Five, game.Diamonds),
				game.NewCard(game.Six, game.Clubs),
				game.NewCard(game.Seven, game.Diamonds),
				game.NewCard(game.Eight, game.Clubs),
				game.NewCard(game.Nine, game.Diamonds),
				game.NewCard(game.Five, game.Spades),
				game.NewCard(game.Two, game.Hearts),
				game.NewCard(game.Three, game.Spades),
			},
			game.NewCard(game.Four, game.Spades),
			game.NewCard(game.Three, game.Diamonds),
			game.NewCard(game.Ace, game.Diamonds),
		))
	}

	lobby := newLobbyServer()
	if _, _, _, err := lobby.draw("missing", "deck"); err == nil || err.Error() != "session not found" {
		t.Fatalf("draw(missing session) error = %v; want session not found", err)
	}

	hostConn, _, closeHost := newSocketPair(t)
	defer closeHost()
	hostEvent, _, _, err := lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(host) error = %v", err)
	}
	if _, _, _, err := lobby.draw(hostEvent.SessionID, "deck"); err == nil || err.Error() != "join a room first" {
		t.Fatalf("draw(no room) error = %v; want join a room first", err)
	}
	hostRoom, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	if _, _, _, err := lobby.draw(hostEvent.SessionID, "deck"); !errors.Is(err, game.ErrGameNotInProgress) {
		t.Fatalf("draw(lobby phase) error = %v; want ErrGameNotInProgress", err)
	}

	guestConn, _, closeGuest := newSocketPair(t)
	defer closeGuest()
	guestEvent, _, _, err := lobby.connect("", guestConn)
	if err != nil {
		t.Fatalf("connect(guest) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "Guest"); err != nil {
		t.Fatalf("joinRoom() error = %v", err)
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err != nil {
		t.Fatalf("chooseDealing() error = %v", err)
	}
	if _, _, _, err := lobby.draw(hostEvent.SessionID, "deck"); err == nil || err.Error() != "not your turn" {
		t.Fatalf("draw(not your turn) error = %v; want not your turn", err)
	}

	roomState, recipients, result, err := lobby.draw(guestEvent.SessionID, "discard")
	if err != nil {
		t.Fatalf("draw(discard) error = %v", err)
	}
	if roomState.Phase != "in_progress" || len(recipients) != 2 || result.Action != "draw" || result.PlayerID != guestEvent.PlayerID || !result.OK {
		t.Fatalf("draw result = %#v, recipients=%d, room=%#v; want successful broadcast", result, len(recipients), roomState)
	}
	if _, _, _, err := lobby.draw(guestEvent.SessionID, "deck"); !errors.Is(err, game.ErrPlayerAlreadyDrew) {
		t.Fatalf("draw(twice) error = %v; want ErrPlayerAlreadyDrew", err)
	}
	if _, _, _, err := lobby.play(guestEvent.SessionID, []*game.Composition{}, nil, nil); !errors.Is(err, game.ErrInvalidComposition) {
		t.Fatalf("play(empty) error = %v; want ErrInvalidComposition", err)
	}

	setComp, ok := game.NewSet([]game.Card{game.NewCard(game.King, game.Hearts), game.NewCard(game.King, game.Diamonds), game.NewCard(game.King, game.Clubs)})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	spadeRun, ok := game.NewRun([]game.Card{game.NewCard(game.Ace, game.Spades), game.NewCard(game.Two, game.Spades), game.NewCard(game.Three, game.Spades), game.NewCard(game.Four, game.Spades)})
	if !ok {
		t.Fatal("NewRun(spade) returned false; want true")
	}
	heartRun, ok := game.NewRun([]game.Card{game.NewCard(game.Five, game.Hearts), game.NewJoker(), game.NewCard(game.Seven, game.Hearts)})
	if !ok {
		t.Fatal("NewRun(heart) returned false; want true")
	}
	if _, _, result, err := lobby.play(guestEvent.SessionID, []*game.Composition{setComp, spadeRun, heartRun}, nil, nil); err != nil || result.Action != "play" {
		t.Fatalf("play(valid) result = %#v, error = %v; want play success", result, err)
	}
	if _, _, _, err := lobby.play(
		guestEvent.SessionID,
		nil,
		[]game.CompositionAddition{{CompositionIndex: 1, Cards: []game.Card{game.NewCard(game.Five, game.Spades)}}},
		[]game.JokerReclaim{{CompositionIndex: 2, JokerIndex: 1, ReplacementCard: game.NewCard(game.Six, game.Hearts)}},
	); err != nil {
		t.Fatalf("play(addition+reclaim) error = %v", err)
	}
	if _, _, _, err := lobby.discard(guestEvent.SessionID, 1); err != nil {
		t.Fatalf("discard() error = %v", err)
	}

	activeRoom := lobby.rooms[hostRoom.Code]
	if activeRoom == nil {
		t.Fatal("active room = nil; want room")
	}
	originalHostPlayer := activeRoom.players[0].player
	activeRoom.players[0].player = newPlayerWithID("not-in-state")
	lobby.sessions[hostEvent.SessionID].playerID = "not-in-state"
	if _, _, _, err := lobby.draw(hostEvent.SessionID, "deck"); err == nil || err.Error() != "game state snapshot failed" {
		t.Fatalf("draw(snapshot failure) error = %v; want game state snapshot failed", err)
	}
	activeRoom.players[0].player = originalHostPlayer
	lobby.sessions[hostEvent.SessionID].playerID = hostEvent.PlayerID

	activeRoom.gameState = nil
	if _, _, _, err := lobby.draw(hostEvent.SessionID, "deck"); err == nil || err.Error() != "game state not initialized" {
		t.Fatalf("draw(nil game state) error = %v; want game state not initialized", err)
	}
}

func TestLobbyStartGameGameStateRecipientsError(t *testing.T) {
	lobby := newLobbyServer()
	gameState := game.NewGameState()
	if err := gameState.AddPlayer(newPlayerWithID("engine-player-a")); err != nil {
		t.Fatalf("AddPlayer(engine-player-a) error = %v", err)
	}
	if err := gameState.AddPlayer(newPlayerWithID("engine-player-b")); err != nil {
		t.Fatalf("AddPlayer(engine-player-b) error = %v", err)
	}

	hostSessionID := "host-session"
	hostPlayerID := "room-player-a"
	guestSessionID := "guest-session"
	guestPlayerID := "room-player-b"
	lobby.sessions[hostSessionID] = &playerSession{sessionID: hostSessionID, playerID: hostPlayerID, conn: &websocket.Conn{}, roomCode: "ROOM"}
	lobby.sessions[guestSessionID] = &playerSession{sessionID: guestSessionID, playerID: guestPlayerID, conn: &websocket.Conn{}, roomCode: "ROOM"}
	lobby.rooms["ROOM"] = &room{
		code:      "ROOM",
		gameState: gameState,
		hostID:    hostPlayerID,
		players: []*roomPlayer{
			{player: newPlayerWithID(hostPlayerID), sessionID: hostSessionID, connected: true, seat: 0, host: true},
			{player: newPlayerWithID(guestPlayerID), sessionID: guestSessionID, connected: true, seat: 1},
		},
	}

	if roomState, recipients, err := lobby.startGame(hostSessionID, 0); err != nil {
		t.Fatalf("startGame() error = %v", err)
	} else if roomState.PendingDealChoice == nil || len(recipients) != 2 {
		t.Fatalf("startGame() = room:%#v recipients:%d; want pending choice and 2 recipients", roomState, len(recipients))
	}
	if _, _, err := lobby.chooseDealing(guestSessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err == nil || err.Error() != "game state snapshot failed" {
		t.Fatalf("chooseDealing(snapshot failure) error = %v; want game state snapshot failed", err)
	}
}

func TestLobbyChooseDealingValidationErrors(t *testing.T) {
	lobby := newLobbyServer()
	if _, _, err := lobby.chooseDealing("missing", "round_robin"); err == nil || err.Error() != "session not found" {
		t.Fatalf("chooseDealing(missing session) error = %v; want session not found", err)
	}

	hostConn, _, closeHost := newSocketPair(t)
	defer closeHost()
	hostEvent, _, _, err := lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(host) error = %v", err)
	}
	if _, _, err := lobby.chooseDealing(hostEvent.SessionID, "round_robin"); err == nil || err.Error() != "join a room first" {
		t.Fatalf("chooseDealing(no room) error = %v; want join a room first", err)
	}

	hostRoom, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	if _, _, err := lobby.chooseDealing(hostEvent.SessionID, "round_robin"); err == nil || err.Error() != "no dealing choice is pending" {
		t.Fatalf("chooseDealing(no pending) error = %v; want no dealing choice is pending", err)
	}

	guestConn, _, closeGuest := newSocketPair(t)
	defer closeGuest()
	guestEvent, _, _, err := lobby.connect("", guestConn)
	if err != nil {
		t.Fatalf("connect(guest) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "Guest"); err != nil {
		t.Fatalf("joinRoom() error = %v", err)
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "banana", dealingChoiceOptions{cutSize: intPtr(0)}); !errors.Is(err, game.ErrInvalidDealingType) {
		t.Fatalf("chooseDealing(invalid type) error = %v; want ErrInvalidDealingType", err)
	}
	if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin"); err == nil || err.Error() != "cut size is required" {
		t.Fatalf("chooseDealing(missing cut size) error = %v; want cut size is required", err)
	}
	room := lobby.rooms[hostRoom.Code]
	room.pendingDealChoice = &pendingDealChoice{dealerIndex: 99, chooserIndex: 1}
	if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); !errors.Is(err, game.ErrInvalidDealer) {
		t.Fatalf("chooseDealing(start game failure) error = %v; want ErrInvalidDealer", err)
	}
}

func TestLobbyStartNextRoundCoverage(t *testing.T) {
	lobby := newLobbyServer()
	if _, _, err := lobby.startNextRound("missing"); err == nil || err.Error() != "session not found" {
		t.Fatalf("startNextRound(missing session) error = %v; want session not found", err)
	}

	hostConn, _, closeHost := newSocketPair(t)
	defer closeHost()
	hostEvent, _, _, err := lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(host) error = %v", err)
	}
	if _, _, err := lobby.startNextRound(hostEvent.SessionID); err == nil || err.Error() != "join a room first" {
		t.Fatalf("startNextRound(no room) error = %v; want join a room first", err)
	}

	hostRoom, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}

	guestConn, _, closeGuest := newSocketPair(t)
	defer closeGuest()
	guestEvent, _, _, err := lobby.connect("", guestConn)
	if err != nil {
		t.Fatalf("connect(guest) error = %v", err)
	}
	if _, _, err := lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "Guest"); err != nil {
		t.Fatalf("joinRoom() error = %v", err)
	}
	if _, _, err := lobby.startGame(hostEvent.SessionID, 0); err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	if _, _, err := lobby.chooseDealing(guestEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err != nil {
		t.Fatalf("chooseDealing() error = %v", err)
	}

	if _, _, err := lobby.startNextRound(hostEvent.SessionID); !errors.Is(err, game.ErrCannotStartNextRound) {
		t.Fatalf("startNextRound(in progress) error = %v; want ErrCannotStartNextRound", err)
	}
	if _, _, err := lobby.startNextRound(guestEvent.SessionID); err == nil || err.Error() != "only the host can start the next round" {
		t.Fatalf("startNextRound(non host) error = %v; want only the host can start the next round", err)
	}

	room := lobby.rooms[hostRoom.Code]
	room.players[1].connected = false
	setGameStatePhaseForTest(t, room.gameState, game.PhaseRoundOver)
	if _, _, err := lobby.startNextRound(hostEvent.SessionID); err == nil || err.Error() != "all players must be connected" {
		t.Fatalf("startNextRound(disconnected player) error = %v; want all players must be connected", err)
	}
	room.players[1].connected = true

	room.gameState = nil
	if _, _, err := lobby.startNextRound(hostEvent.SessionID); err == nil || err.Error() != "game state not initialized" {
		t.Fatalf("startNextRound(nil game state) error = %v; want game state not initialized", err)
	}

	room.gameState = game.NewGameState()
	if err := room.gameState.AddPlayer(newPlayerWithID(hostEvent.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(host) error = %v", err)
	}
	if err := room.gameState.AddPlayer(newPlayerWithID(guestEvent.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(guest) error = %v", err)
	}
	setGameStatePhaseForTest(t, room.gameState, game.PhaseRoundOver)

	roomState, recipients, err := lobby.startNextRound(hostEvent.SessionID)
	if err != nil {
		t.Fatalf("startNextRound() error = %v", err)
	}
	if roomState.Phase != "round_over" {
		t.Fatalf("roomState.Phase = %q; want round_over", roomState.Phase)
	}
	if roomState.PendingDealChoice == nil {
		t.Fatal("roomState.PendingDealChoice = nil; want pending dealing choice")
	}
	if roomState.PendingDealChoice.DealerIndex != 1 {
		t.Fatalf("roomState.PendingDealChoice.DealerIndex = %d; want 1", roomState.PendingDealChoice.DealerIndex)
	}
	if len(recipients) != 2 {
		t.Fatalf("len(recipients) = %d; want 2", len(recipients))
	}
	if _, _, err := lobby.startNextRound(hostEvent.SessionID); err == nil || err.Error() != "dealing choice already pending" {
		t.Fatalf("startNextRound(pending choice) error = %v; want dealing choice already pending", err)
	}

	startedRoom, gameRecipients, err := lobby.chooseDealing(hostEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)})
	if err != nil {
		t.Fatalf("chooseDealing(next round) error = %v", err)
	}
	if startedRoom.Phase != "in_progress" {
		t.Fatalf("startedRoom.Phase = %q; want in_progress", startedRoom.Phase)
	}
	if gameRecipients[0].event.Game.Round != 2 {
		t.Fatalf("gameRecipients[0].event.Game.Round = %d; want 2", gameRecipients[0].event.Game.Round)
	}

	room.gameState = game.NewGameState()
	if err := room.gameState.AddPlayer(newPlayerWithID("state-host")); err != nil {
		t.Fatalf("AddPlayer(state-host) error = %v", err)
	}
	if err := room.gameState.AddPlayer(newPlayerWithID("state-guest")); err != nil {
		t.Fatalf("AddPlayer(state-guest) error = %v", err)
	}
	setGameStatePhaseForTest(t, room.gameState, game.PhaseRoundOver)
	if _, _, err := lobby.startNextRound(hostEvent.SessionID); err != nil {
		t.Fatalf("startNextRound(snapshot setup) error = %v", err)
	}
	if _, _, err := lobby.chooseDealing(hostEvent.SessionID, "round_robin", dealingChoiceOptions{cutSize: intPtr(0)}); err == nil || err.Error() != "game state snapshot failed" {
		t.Fatalf("chooseDealing(snapshot failure) error = %v; want game state snapshot failed", err)
	}
}

func TestLobbyResetRoomAfterGameOverCoverage(t *testing.T) {
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()
	originalAddPlayer := addPlayerToGameState
	defer func() { addPlayerToGameState = originalAddPlayer }()

	lobby := newLobbyServer()
	if _, _, err := lobby.resetRoomAfterGameOver("missing"); err == nil || err.Error() != "room not found" {
		t.Fatalf("resetRoomAfterGameOver(missing room) error = %v; want room not found", err)
	}

	hostConn, _, closeHost := newSocketPair(t)
	defer closeHost()
	hostEvent, _, _, err := lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(host) error = %v", err)
	}
	hostRoom, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	room := lobby.rooms[hostRoom.Code]

	room.gameState = nil
	if _, _, err := lobby.resetRoomAfterGameOver(hostRoom.Code); err == nil || err.Error() != "game state not initialized" {
		t.Fatalf("resetRoomAfterGameOver(nil game state) error = %v; want game state not initialized", err)
	}

	room.gameState = game.NewGameState()
	if err := room.gameState.AddPlayer(newPlayerWithID(hostEvent.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(host) error = %v", err)
	}
	if _, _, err := lobby.resetRoomAfterGameOver(hostRoom.Code); err == nil || err.Error() != "game is not over" {
		t.Fatalf("resetRoomAfterGameOver(not over) error = %v; want game is not over", err)
	}

	setGameStatePhaseForTest(t, room.gameState, game.PhaseGameOver)
	makeGameState = func() *game.GameState { return nil }
	if _, _, err := lobby.resetRoomAfterGameOver(hostRoom.Code); err == nil || err.Error() != "game state not initialized" {
		t.Fatalf("resetRoomAfterGameOver(reset error) error = %v; want game state not initialized", err)
	}

	makeGameState = game.NewGameState
	setGameStatePhaseForTest(t, room.gameState, game.PhaseGameOver)
	snapshot, recipients, err := lobby.resetRoomAfterGameOver(hostRoom.Code)
	if err != nil {
		t.Fatalf("resetRoomAfterGameOver() error = %v", err)
	}
	if snapshot == nil || snapshot.Phase != "lobby" {
		t.Fatalf("snapshot = %#v; want lobby snapshot", snapshot)
	}
	if len(snapshot.Players) != 1 || snapshot.Players[0].PlayerID != hostEvent.PlayerID {
		t.Fatalf("snapshot.Players = %#v; want host retained", snapshot.Players)
	}
	if len(recipients) != 1 || recipients[0] != hostConn {
		t.Fatalf("recipients = %v; want [%p]", recipients, hostConn)
	}

	room.players = append(room.players, nil)
	if err := room.resetForLobby(); err != nil {
		t.Fatalf("resetForLobby(with nil player) error = %v", err)
	}

	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return errors.New("add failed")
	}
	if err := room.resetForLobby(); err == nil || err.Error() != "add failed" {
		t.Fatalf("resetForLobby(add failure) error = %v; want add failed", err)
	}
}

func TestLobbyGameStateForSessionMissingSession(t *testing.T) {
	lobby := newLobbyServer()

	state, err := lobby.gameStateForSession("missing", roomSnapshot{})
	if err == nil || err.Error() != "session not found" {
		t.Fatalf("gameStateForSession(missing) error = %v; want session not found", err)
	}
	if state != nil {
		t.Fatalf("gameStateForSession(missing) state = %#v; want nil", state)
	}
}

func TestRoomGameStateRecipientAndCurrentTurnCoverage(t *testing.T) {
	if (&room{}).isCurrentTurn("p1") {
		t.Fatal("empty room isCurrentTurn() = true; want false")
	}
	if ((*room)(nil)).isCurrentTurn("p1") {
		t.Fatal("nil room isCurrentTurn() = true; want false")
	}

	state := game.NewGameState()
	statePlayer := newPlayerWithID("state-player")
	if err := state.AddPlayer(statePlayer); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
	roomWithBadTurn := &room{gameState: state, players: []*roomPlayer{}}
	if roomWithBadTurn.isCurrentTurn("state-player") {
		t.Fatal("out-of-range turn isCurrentTurn() = true; want false")
	}
	roomWithNilPlayerTurn := &room{gameState: state, players: []*roomPlayer{nil}}
	if roomWithNilPlayerTurn.isCurrentTurn("state-player") {
		t.Fatal("nil player turn isCurrentTurn() = true; want false")
	}
	roomWithTurn := &room{gameState: state, players: []*roomPlayer{{player: statePlayer}}}
	if !roomWithTurn.isCurrentTurn("state-player") {
		t.Fatal("matching turn isCurrentTurn() = false; want true")
	}
	if roomWithTurn.isCurrentTurn("other") {
		t.Fatal("different player isCurrentTurn() = true; want false")
	}

	sessions := map[string]*playerSession{
		"connected": {conn: func() *websocket.Conn { conn, _, cleanup := newSocketPair(t); t.Cleanup(cleanup); return conn }()},
	}
	roomForRecipients := &room{
		code:      "ROOM",
		gameState: state,
		players: []*roomPlayer{
			nil,
			{player: statePlayer, sessionID: "connected", connected: true},
			{player: newPlayerWithID("offline"), sessionID: "offline", connected: false},
			{player: newPlayerWithID("missing-session"), sessionID: "missing", connected: true},
		},
	}
	recipients, err := roomForRecipients.gameStateRecipients(sessions, roomSnapshot{Code: "ROOM"})
	if err != nil {
		t.Fatalf("gameStateRecipients() error = %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("len(recipients) = %d; want 1", len(recipients))
	}

	roomForRecipients.players[1].player = newPlayerWithID("not-in-state")
	if _, err := roomForRecipients.gameStateRecipients(sessions, roomSnapshot{Code: "ROOM"}); err == nil || err.Error() != "game state snapshot failed" {
		t.Fatalf("gameStateRecipients(snapshot failure) error = %v; want game state snapshot failed", err)
	}
}

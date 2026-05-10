package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/gorilla/websocket"
)

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

	startedRoom, startRecipients, err := lobby.startGame(hostEvent.SessionID, 0)
	if err != nil {
		t.Fatalf("startGame() error = %v", err)
	}
	if startedRoom.Phase != "in_progress" {
		t.Fatalf("startedRoom.Phase = %q; want in_progress", startedRoom.Phase)
	}
	if len(startRecipients) != 4 {
		t.Fatalf("len(startRecipients) = %d; want 4", len(startRecipients))
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
	for i := 0; i < b.N; i++ {
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

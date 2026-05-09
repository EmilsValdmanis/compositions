package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/backend/internal/game"
	"github.com/gorilla/websocket"
)

func TestWebSocketLobbyFlowCreateJoinDisconnectReconnectAndStart(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	hostConnected := mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)
	if hostRoom.Phase != "lobby" {
		t.Fatalf("hostRoom.Phase = %q; want lobby", hostRoom.Phase)
	}
	if hostRoom.Code == "" {
		t.Fatal("hostRoom.Code = empty; want room code")
	}
	if len(hostRoom.Players) != 1 {
		t.Fatalf("len(hostRoom.Players) = %d; want 1", len(hostRoom.Players))
	}
	if !hostRoom.Players[0].IsHost {
		t.Fatal("hostRoom.Players[0].IsHost = false; want true")
	}
	if hostRoom.HostPlayerID != hostConnected.PlayerID {
		t.Fatalf("hostRoom.HostPlayerID = %q; want %q", hostRoom.HostPlayerID, hostConnected.PlayerID)
	}

	guestConn := mustDialWS(t, httpServer.URL)
	guestConnected := mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	guestRoom := mustReadRoomState(t, guestConn)
	hostRoom = mustReadRoomState(t, hostConn)
	for _, room := range []roomSnapshot{guestRoom, hostRoom} {
		if len(room.Players) != 2 {
			t.Fatalf("len(room.Players) = %d; want 2", len(room.Players))
		}
		if room.Players[1].PlayerID != guestConnected.PlayerID {
			t.Fatalf("room.Players[1].PlayerID = %q; want %q", room.Players[1].PlayerID, guestConnected.PlayerID)
		}
		if room.Players[1].Name != "Guest" {
			t.Fatalf("room.Players[1].Name = %q; want Guest", room.Players[1].Name)
		}
	}

	guestConn.Close()
	hostRoom = mustReadRoomState(t, hostConn)
	if hostRoom.Players[1].Connected {
		t.Fatal("hostRoom.Players[1].Connected = true; want false after disconnect")
	}

	mustSendEnvelope(t, hostConn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, hostConn, "all players must be connected")

	reconnectedGuestConn := mustDialWS(t, httpServer.URL)
	reconnected := mustConnectSession(t, reconnectedGuestConn, guestConnected.SessionID)
	if reconnected.PlayerID != guestConnected.PlayerID {
		t.Fatalf("reconnected.PlayerID = %q; want %q", reconnected.PlayerID, guestConnected.PlayerID)
	}
	reconnectedRoom := mustReadRoomState(t, reconnectedGuestConn)
	hostRoom = mustReadRoomState(t, hostConn)
	for _, room := range []roomSnapshot{reconnectedRoom, hostRoom} {
		if !room.Players[1].Connected {
			t.Fatal("guest connected flag = false; want true after reconnect")
		}
	}

	mustSendEnvelope(t, hostConn, "start_game", startGameRequest{DealerIndex: 1})
	hostStarted := mustReadRoomState(t, hostConn)
	guestStarted := mustReadRoomState(t, reconnectedGuestConn)
	for _, room := range []roomSnapshot{hostStarted, guestStarted} {
		if room.Phase != "in_progress" {
			t.Fatalf("room.Phase = %q; want in_progress", room.Phase)
		}
		if room.DealerIndex != 1 {
			t.Fatalf("room.DealerIndex = %d; want 1", room.DealerIndex)
		}
	}
	if err := reconnectedGuestConn.Close(); err != nil {
		t.Fatalf("reconnectedGuestConn.Close() error = %v", err)
	}
}

func TestLobbyServerCoverage(t *testing.T) {
	lobby := newLobbyServer()
	if _, err := lobby.requireSession("missing"); err == nil {
		t.Fatal("requireSession(missing) error = nil; want error")
	}

	state := game.NewGameState()
	if got := state.Phase(); got != game.PhaseLobby {
		t.Fatalf("state.Phase() = %v; want %v", got, game.PhaseLobby)
	}
	if got := state.DealerIndex(); got != 0 {
		t.Fatalf("state.DealerIndex() = %d; want 0", got)
	}

	hostConn, hostPeer, closeHostPair := newSocketPair(t)
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

	guestConn, guestPeer, closeGuestPair := newSocketPair(t)
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

	thirdConn, thirdPeer, closeThirdPair := newSocketPair(t)
	defer closeThirdPair()
	thirdEvent, _, _, err := lobby.connect("", thirdConn)
	if err != nil {
		t.Fatalf("third connect(new) error = %v", err)
	}
	fourthConn, fourthPeer, closeFourthPair := newSocketPair(t)
	defer closeFourthPair()
	fourthEvent, _, _, err := lobby.connect("", fourthConn)
	if err != nil {
		t.Fatalf("fourth connect(new) error = %v", err)
	}
	fifthConn, fifthPeer, closeFifthPair := newSocketPair(t)
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

	notHostConn, notHostPeer, closeNotHostPair := newSocketPair(t)
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

	server := newWSServer()
	errConn, errPeer, closeErrPair := newSocketPair(t)
	defer closeErrPair()
	server.writeError(errConn, errors.New("boom"))
	gotErrorEnvelope := mustReadEnvelopeFromConn(t, errPeer)
	if gotErrorEnvelope.Type != "error" {
		t.Fatalf("gotErrorEnvelope.Type = %q; want error", gotErrorEnvelope.Type)
	}
	var gotError errorEvent
	if err := json.Unmarshal(gotErrorEnvelope.Data, &gotError); err != nil {
		t.Fatalf("json.Unmarshal(error event) error = %v", err)
	}
	if gotError.Message != "boom" {
		t.Fatalf("gotError.Message = %q; want boom", gotError.Message)
	}

	roomConnA, roomPeerA, closeRoomPairA := newSocketPair(t)
	defer closeRoomPairA()
	roomConnB, roomPeerB, closeRoomPairB := newSocketPair(t)
	defer closeRoomPairB()
	server.broadcastRoomState(hostRoom, []*websocket.Conn{nil, roomConnA, roomConnB})
	for _, peer := range []*websocket.Conn{roomPeerA, roomPeerB} {
		envelope := mustReadEnvelopeFromConn(t, peer)
		if envelope.Type != "room_state" {
			t.Fatalf("envelope.Type = %q; want room_state", envelope.Type)
		}
	}

	if err := decodePayload(nil, &struct{}{}); err == nil {
		t.Fatal("decodePayload(nil) error = nil; want error")
	}
	if err := decodePayload(json.RawMessage(`{`), &struct{}{}); err == nil {
		t.Fatal("decodePayload(invalid) error = nil; want error")
	}
	var payload struct{ Name string }
	if err := decodePayload(json.RawMessage(`{"name":"ok"}`), &payload); err != nil {
		t.Fatalf("decodePayload(valid) error = %v", err)
	}
	if payload.Name != "ok" {
		t.Fatalf("payload.Name = %q; want ok", payload.Name)
	}

	envelope := mustMarshalRawMessage(struct{ Name string `json:"name"` }{Name: "x"})
	if string(envelope) != `{"name":"x"}` {
		t.Fatalf("mustMarshalRawMessage() = %s; want {\"name\":\"x\"}", string(envelope))
	}

	writeConn, writePeer, closeWritePair := newSocketPair(t)
	defer closeWritePair()
	if err := writeEvent(writeConn, "connected", connectedEvent{SessionID: "s", PlayerID: "p"}); err != nil {
		t.Fatalf("writeEvent() error = %v", err)
	}
	gotEnvelope := mustReadEnvelopeFromConn(t, writePeer)
	if gotEnvelope.Type != "connected" {
		t.Fatalf("gotEnvelope.Type = %q; want connected", gotEnvelope.Type)
	}
	_ = writeConn.Close()
	if err := writeEvent(writeConn, "connected", connectedEvent{SessionID: "s", PlayerID: "p"}); err == nil {
		t.Fatal("writeEvent(closed conn) error = nil; want error")
	}
	server.writeError(writeConn, errors.New("ignored"))
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		return errSocketClosed
	}
	server.writeError(writeConn, errors.New("ignored closed"))
	server.broadcastRoomState(hostRoom, []*websocket.Conn{writeConn})
	lobby.broadcastDisconnect(hostRoom, []*websocket.Conn{writeConn})
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		return errors.New("emit boom")
	}
	server.broadcastRoomState(hostRoom, []*websocket.Conn{writeConn})
	lobby.broadcastDisconnect(hostRoom, []*websocket.Conn{nil, writeConn})

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

	connSet := []*websocket.Conn{hostConn, guestConn, thirdConn}
	filtered := otherConnections(connSet, guestConn)
	if len(filtered) != 2 || filtered[0] != hostConn || filtered[1] != thirdConn {
		t.Fatalf("otherConnections() = %v; want [%p %p]", filtered, hostConn, thirdConn)
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
	originalMakeGameState := makeGameState
	originalAddPlayer := addPlayerToGameState

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
	addPlayerToGameState = originalAddPlayer
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

	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	defer func() { makeGameState = originalMakeGameState }()
	defer func() { addPlayerToGameState = originalAddPlayer }()
	calledAddrs := make([]string, 0, 2)
	listenAndServe = func(addr string, handler http.Handler) error {
		calledAddrs = append(calledAddrs, addr)
		if addr != ":0" && addr != ":8080" {
			t.Fatalf("listenAndServe addr = %q; want :0 or :8080", addr)
		}
		if handler == nil {
			t.Fatal("listenAndServe handler = nil; want handler")
		}
		return errors.New("stop")
	}
	if err := runServer(":0"); err == nil || err.Error() != "stop" {
		t.Fatalf("runServer() error = %v; want stop", err)
	}
	if len(calledAddrs) == 0 || calledAddrs[0] != ":0" {
		t.Fatal("runServer() did not call listenAndServe")
	}
	fatalCalled := false
	fatalOnRunError = func(v ...any) {
		fatalCalled = true
		if len(v) != 1 {
			t.Fatalf("fatalOnRunError args = %v; want single arg", v)
		}
		err, ok := v[0].(error)
		if !ok || err == nil || err.Error() != "stop" {
			t.Fatalf("fatalOnRunError arg = %v; want stop error", v[0])
		}
	}
	main()
	if !fatalCalled {
		t.Fatal("main() did not call fatalOnRunError")
	}
	if len(calledAddrs) != 2 || calledAddrs[1] != ":8080" {
		t.Fatalf("calledAddrs = %v; want [:0 :8080]", calledAddrs)
	}
	_ = guestPeer
	_ = fourthPeer
	_ = fifthPeer
	_ = hostPeer
	_ = notHostPeer
	_ = thirdPeer
}

func TestHandleWSUpgradeFailure(t *testing.T) {
	server := newWSServer()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)

	server.handleWS(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("recorder.Code = %d; want %d", recorder.Code, http.StatusBadRequest)
	}
	body, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if len(body) == 0 {
		t.Fatal("handleWS response body = empty; want upgrade error body")
	}
}

func TestHandleConnectionErrorsAndDisconnectBroadcasts(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = writeEvent
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()
	makeGameState = game.NewGameState
	originalAddPlayer := addPlayerToGameState
	defer func() { addPlayerToGameState = originalAddPlayer }()
	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return state.AddPlayer(player)
	}

	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	hostConnected := mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)

	guestConn := mustDialWS(t, httpServer.URL)
	mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, hostConn)

	rawConn := mustDialWS(t, httpServer.URL)
	if err := rawConn.WriteJSON(wsEnvelope{Type: "create_room", Data: mustMarshalRawMessage(createRoomRequest{Name: "NoConnect"})}); err != nil {
		t.Fatalf("WriteJSON(create_room before connect) error = %v", err)
	}
	mustReadError(t, rawConn, "connect first")
	if err := rawConn.WriteJSON(wsEnvelope{Type: "join_room", Data: mustMarshalRawMessage(joinRoomRequest{RoomCode: hostRoom.Code, Name: "NoConnect"})}); err != nil {
		t.Fatalf("WriteJSON(join_room before connect) error = %v", err)
	}
	mustReadError(t, rawConn, "connect first")
	if err := rawConn.WriteJSON(wsEnvelope{Type: "start_game", Data: mustMarshalRawMessage(startGameRequest{DealerIndex: 0})}); err != nil {
		t.Fatalf("WriteJSON(start_game before connect) error = %v", err)
	}
	mustReadError(t, rawConn, "connect first")
	mustConnectSession(t, rawConn, "")
	if err := rawConn.WriteMessage(websocket.TextMessage, []byte("{")); err != nil {
		t.Fatalf("WriteMessage(invalid json) error = %v", err)
	}
	if _, _, err := rawConn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage() after invalid json error = nil; want close error")
	}

	rawConn = mustDialWS(t, httpServer.URL)
	mustConnectSession(t, rawConn, "")
	if err := rawConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"create_room","data":{`)); err != nil {
		t.Fatalf("WriteMessage(create_room invalid data) error = %v", err)
	}
	if _, _, err := rawConn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage() after create_room invalid data error = nil; want close error")
	}

	rawConn = mustDialWS(t, httpServer.URL)
	mustConnectSession(t, rawConn, "")
	if err := rawConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"join_room","data":{`)); err != nil {
		t.Fatalf("WriteMessage(join_room invalid data) error = %v", err)
	}
	if _, _, err := rawConn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage() after join_room invalid data error = nil; want close error")
	}

	rawConn = mustDialWS(t, httpServer.URL)
	mustConnectSession(t, rawConn, "")
	if err := rawConn.WriteMessage(websocket.TextMessage, []byte(`{"type":"start_game","data":{`)); err != nil {
		t.Fatalf("WriteMessage(start_game invalid data) error = %v", err)
	}
	if _, _, err := rawConn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage() after start_game invalid data error = nil; want close error")
	}

	rawConn = mustDialWS(t, httpServer.URL)
	mustConnectSession(t, rawConn, "")
	if err := rawConn.WriteJSON(wsEnvelope{Type: "connect"}); err != nil {
		t.Fatalf("WriteJSON(missing data) error = %v", err)
	}
	mustReadError(t, rawConn, "missing data")

	rawConn = mustDialWS(t, httpServer.URL)
	mustConnectSession(t, rawConn, "")
	if err := rawConn.WriteJSON(wsEnvelope{Type: "mystery", Data: mustMarshalRawMessage(struct{}{})}); err != nil {
		t.Fatalf("WriteJSON(unknown type) error = %v", err)
	}
	mustReadError(t, rawConn, "unknown message type")

	mustSendEnvelope(t, rawConn, "connect", connectRequest{SessionID: hostConnected.SessionID})
	connectedEnvelope := mustReadEnvelopeFromConn(t, rawConn)
	if connectedEnvelope.Type != "connected" {
		t.Fatalf("connectedEnvelope.Type = %q; want connected", connectedEnvelope.Type)
	}
	_ = mustReadRoomState(t, rawConn)
	_ = mustReadRoomState(t, guestConn)

	if err := rawConn.Close(); err != nil {
		t.Fatalf("rawConn.Close() error = %v", err)
	}
	updatedGuestRoom := mustReadRoomState(t, guestConn)
	if updatedGuestRoom.Players[0].Connected {
		t.Fatal("updatedGuestRoom.Players[0].Connected = true; want false")
	}
	if err := guestConn.Close(); err != nil {
		t.Fatalf("guestConn.Close() error = %v", err)
	}
	if err := hostConn.Close(); err != nil {
		t.Fatalf("hostConn.Close() error = %v", err)
	}

	server.lobby = newLobbyServer()
	httpServer2 := httptest.NewServer(server.routes())
	defer httpServer2.Close()
	soloConn := mustDialWS(t, httpServer2.URL)
	mustConnectSession(t, soloConn, "")
	mustSendEnvelope(t, soloConn, "create_room", createRoomRequest{Name: "Solo"})
	soloRoom := mustReadRoomState(t, soloConn)
	mustSendEnvelope(t, soloConn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, soloConn, "need at least 2 players to start")
	soloHostSession := server.lobby.sessions[soloRoom.Players[0].SessionID]
	delete(server.lobby.rooms, soloRoom.Code)
	mustSendEnvelope(t, soloConn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, soloConn, "room not found")
	soloHostSession.roomCode = ""
	mustSendEnvelope(t, soloConn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, soloConn, "join a room first")
}

type rawInvalidMessage struct{}

func (rawInvalidMessage) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal boom")
}

func TestMustMarshalRawMessagePanicsOnMarshalError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustMarshalRawMessage() did not panic")
		}
	}()

	_ = mustMarshalRawMessage(rawInvalidMessage{})
}

func TestHandleConnectionReturnsWhenInitialConnectedWriteFails(t *testing.T) {
	server := newWSServer()
	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		server.handleConnection(serverConn)
		close(done)
	}()

	if err := clientConn.WriteJSON(wsEnvelope{Type: "connect", Data: mustMarshalRawMessage(connectRequest{})}); err != nil {
		t.Fatalf("WriteJSON(connect) error = %v", err)
	}
	if err := clientConn.Close(); err != nil {
		t.Fatalf("clientConn.Close() error = %v", err)
	}
	<-done
}

func TestHandleConnectionReturnsWhenRoomStateWriteFailsAfterReconnect(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = writeEvent
	originalAddPlayer := addPlayerToGameState
	defer func() { addPlayerToGameState = originalAddPlayer }()
	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return state.AddPlayer(player)
	}

	server := newWSServer()
	lobby := server.lobby

	hostConn, _, closeHostPair := newSocketPair(t)
	defer closeHostPair()
	hostEvent, _, _, err := lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("lobby.connect(host) error = %v", err)
	}
	roomState, _, err := lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("lobby.createRoom() error = %v", err)
	}

	reconnectServerConn, reconnectClientConn, closeReconnectPair := newSocketPair(t)
	defer closeReconnectPair()
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		if conn == reconnectServerConn && messageType == "room_state" {
			return errors.New("forced room_state failure")
		}
		return writeEvent(conn, messageType, data)
	}
	session := lobby.sessions[hostEvent.SessionID]
	session.conn = nil
	roomPlayer := lobby.rooms[roomState.Code].playerByID(hostEvent.PlayerID)
	roomPlayer.connected = false

	done := make(chan struct{})
	go func() {
		server.handleConnection(reconnectServerConn)
		close(done)
	}()

	if err := reconnectClientConn.WriteJSON(wsEnvelope{Type: "connect", Data: mustMarshalRawMessage(connectRequest{SessionID: hostEvent.SessionID})}); err != nil {
		t.Fatalf("WriteJSON(reconnect) error = %v", err)
	}
	envelope := mustReadEnvelopeFromConn(t, reconnectClientConn)
	if envelope.Type != "connected" {
		t.Fatalf("envelope.Type = %q; want connected", envelope.Type)
	}
	if err := reconnectClientConn.Close(); err != nil {
		t.Fatalf("reconnectClientConn.Close() error = %v", err)
	}
	<-done
}

func TestHandleConnectionOperationErrors(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = writeEvent
	originalAddPlayer := addPlayerToGameState
	defer func() { addPlayerToGameState = originalAddPlayer }()
	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return state.AddPlayer(player)
	}

	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	conn := mustDialWS(t, httpServer.URL)
	mustSendEnvelope(t, conn, "connect", connectRequest{SessionID: "missing"})
	mustReadError(t, conn, "session not found")

	mustConnectSession(t, conn, "")
	if err := conn.WriteJSON(wsEnvelope{Type: "create_room"}); err != nil {
		t.Fatalf("WriteJSON(create_room missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
	mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Host"})
	room := mustReadRoomState(t, conn)
	mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Again"})
	mustReadError(t, conn, "already in a room")
	if err := conn.WriteJSON(wsEnvelope{Type: "join_room"}); err != nil {
		t.Fatalf("WriteJSON(join_room missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
	mustSendEnvelope(t, conn, "join_room", joinRoomRequest{RoomCode: room.Code, Name: "Again"})
	mustReadError(t, conn, "already in a room")
	if err := conn.WriteJSON(wsEnvelope{Type: "start_game"}); err != nil {
		t.Fatalf("WriteJSON(start_game missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
	mustSendEnvelope(t, conn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, conn, "need at least 2 players to start")

	guestConn := mustDialWS(t, httpServer.URL)
	mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: room.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, conn)
	mustSendEnvelope(t, conn, "start_game", startGameRequest{DealerIndex: 99})
	mustReadError(t, conn, game.ErrInvalidDealer.Error())
	_ = conn.Close()
	_ = guestConn.Close()
}

func TestCreateRoomAddPlayerErrorWithFreshSession(t *testing.T) {
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
}

func TestWriteEventReturnsSocketClosedForCloseSent(t *testing.T) {
	conn, _, cleanup := newSocketPair(t)
	defer cleanup()
	if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("WriteControl(CloseMessage) error = %v", err)
	}
	if err := writeEvent(conn, "connected", connectedEvent{SessionID: "s", PlayerID: "p"}); !errors.Is(err, errSocketClosed) {
		t.Fatalf("writeEvent() error = %v; want errSocketClosed", err)
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

func newSocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)
	serverErrCh := make(chan error, 1)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErrCh <- err
			return
		}
		serverConnCh <- conn
	}))

	clientConn := mustDialWS(t, httpServer.URL)
	var serverConn *websocket.Conn
	select {
	case serverConn = <-serverConnCh:
	case err := <-serverErrCh:
		t.Fatalf("upgrade server connection error = %v", err)
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = serverConn.Close()
			_ = clientConn.Close()
			httpServer.Close()
		})
	}

	return serverConn, clientConn, cleanup
}

func mustDialWS(t *testing.T, baseURL string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial(%q) error = %v", url, err)
	}
	return conn
}

func mustConnectSession(t *testing.T, conn *websocket.Conn, sessionID string) connectedEvent {
	t.Helper()

	mustSendEnvelope(t, conn, "connect", connectRequest{SessionID: sessionID})
	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "connected" {
		t.Fatalf("connect response type = %q; want connected", envelope.Type)
	}
	var event connectedEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(connected) error = %v", err)
	}
	if event.SessionID == "" {
		t.Fatal("connected.SessionID = empty; want session ID")
	}
	if event.PlayerID == "" {
		t.Fatal("connected.PlayerID = empty; want player ID")
	}
	return event
}

func mustSendEnvelope(t *testing.T, conn *websocket.Conn, messageType string, data any) {
	t.Helper()

	if err := conn.WriteJSON(wsEnvelope{Type: messageType, Data: mustMarshalRawMessage(data)}); err != nil {
		t.Fatalf("WriteJSON(%q) error = %v", messageType, err)
	}
}

func mustReadEnvelopeFromConn(t *testing.T, conn *websocket.Conn) wsEnvelope {
	t.Helper()

	var envelope wsEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	return envelope
}

func mustReadRoomState(t *testing.T, conn *websocket.Conn) roomSnapshot {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "room_state" {
		t.Fatalf("room_state response type = %q; want room_state", envelope.Type)
	}
	var event roomStateEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(room_state) error = %v", err)
	}
	return event.Room
}

func mustReadError(t *testing.T, conn *websocket.Conn, want string) {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "error" {
		t.Fatalf("error response type = %q; want error", envelope.Type)
	}
	var event errorEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	if event.Message != want {
		t.Fatalf("error message = %q; want %q", event.Message, want)
	}
}

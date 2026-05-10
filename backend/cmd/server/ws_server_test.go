package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/gorilla/websocket"
)

type staticSessionVerifier struct {
	usersByToken map[string]authenticatedUser
}

func (v staticSessionVerifier) VerifySession(_ context.Context, bearerToken string) (authenticatedUser, error) {
	if user, ok := v.usersByToken[bearerToken]; ok {
		return user, nil
	}
	return authenticatedUser{}, errAuthenticationRequired
}

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

	_ = guestConn.Close()
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

func TestAuthenticatedWebSocketRequiresVerifiedUser(t *testing.T) {
	server := newWSServerWithVerifier(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"host-token":  {ID: "host-user", Name: "Host Account", Email: "host@example.com"},
		"guest-token": {ID: "guest-user", Name: "Guest Account", Email: "guest@example.com"},
	}})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	unauthenticatedConn := mustDialWS(t, httpServer.URL)
	defer unauthenticatedConn.Close()
	mustSendEnvelope(t, unauthenticatedConn, "connect", connectRequest{})
	mustReadError(t, unauthenticatedConn, errAuthenticationRequired.Error())

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	hostConnected := mustConnectAuthenticatedSession(t, hostConn, "", "host-token")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Spoofed Host"})
	hostRoom := mustReadRoomState(t, hostConn)
	if got := hostRoom.Players[0].Name; got != "Host Account" {
		t.Fatalf("hostRoom.Players[0].Name = %q; want Host Account", got)
	}

	guestConn := mustDialWS(t, httpServer.URL)
	defer guestConn.Close()
	guestConnected := mustConnectAuthenticatedSession(t, guestConn, "", "guest-token")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Spoofed Guest"})
	guestRoom := mustReadRoomState(t, guestConn)
	hostRoom = mustReadRoomState(t, hostConn)
	for _, room := range []roomSnapshot{guestRoom, hostRoom} {
		if room.Players[1].PlayerID != guestConnected.PlayerID {
			t.Fatalf("room.Players[1].PlayerID = %q; want %q", room.Players[1].PlayerID, guestConnected.PlayerID)
		}
		if room.Players[1].Name != "Guest Account" {
			t.Fatalf("room.Players[1].Name = %q; want Guest Account", room.Players[1].Name)
		}
	}

	hijackConn := mustDialWS(t, httpServer.URL)
	defer hijackConn.Close()
	mustSendEnvelope(t, hijackConn, "connect", connectRequest{SessionID: hostConnected.SessionID, AuthToken: "guest-token"})
	mustReadError(t, hijackConn, "session belongs to a different user")
}

func TestAuthenticatedWebSocketRejectsSecondLiveSocketForSameUser(t *testing.T) {
	server := newWSServerWithVerifier(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"same-user-token": {ID: "same-user", Name: "Same Account", Email: "same@example.com"},
	}})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	firstConn := mustDialWS(t, httpServer.URL)
	defer firstConn.Close()
	firstConnected := mustConnectAuthenticatedSession(t, firstConn, "", "same-user-token")

	secondConn := mustDialWS(t, httpServer.URL)
	defer secondConn.Close()
	mustSendEnvelope(t, secondConn, "connect", connectRequest{AuthToken: "same-user-token"})
	mustReadError(t, secondConn, "session already connected")
	if len(server.lobby.sessions) != 1 {
		t.Fatalf("len(server.lobby.sessions) = %d; want 1", len(server.lobby.sessions))
	}

	mustSendEnvelope(t, firstConn, "create_room", createRoomRequest{Name: "Should Work"})
	room := mustReadRoomState(t, firstConn)
	if room.HostPlayerID != firstConnected.PlayerID {
		t.Fatalf("room.HostPlayerID = %q; want %q", room.HostPlayerID, firstConnected.PlayerID)
	}
}

func TestAuthenticatedWebSocketReusesSessionAfterDisconnectForSameUser(t *testing.T) {
	server := newWSServerWithVerifier(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"same-user-token": {ID: "same-user", Name: "Same Account", Email: "same@example.com"},
	}})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	firstConn := mustDialWS(t, httpServer.URL)
	firstConnected := mustConnectAuthenticatedSession(t, firstConn, "", "same-user-token")
	if err := firstConn.Close(); err != nil {
		t.Fatalf("firstConn.Close() error = %v", err)
	}

	secondConn := mustDialWS(t, httpServer.URL)
	defer secondConn.Close()
	secondConnected := mustConnectAuthenticatedSession(t, secondConn, "", "same-user-token")

	if secondConnected.SessionID != firstConnected.SessionID {
		t.Fatalf("secondConnected.SessionID = %q; want %q", secondConnected.SessionID, firstConnected.SessionID)
	}
	if secondConnected.PlayerID != firstConnected.PlayerID {
		t.Fatalf("secondConnected.PlayerID = %q; want %q", secondConnected.PlayerID, firstConnected.PlayerID)
	}
}

func TestAuthenticatedWebSocketRejectsUnexpectedOrigin(t *testing.T) {
	server := newWSServerWithAllowedOrigin(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"host-token": {ID: "host-user", Name: "Host Account", Email: "host@example.com"},
	}}, "http://frontend.test")
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"http://evil.test"}})
	if err == nil {
		_ = conn.Close()
		t.Fatal("Dial() error = nil; want websocket upgrade rejection")
	}
	if resp == nil {
		t.Fatal("Dial() response = nil; want forbidden response")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("Dial() status = %d; want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestAuthenticatedWebSocketAcceptsConfiguredOrigin(t *testing.T) {
	server := newWSServerWithAllowedOrigin(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"host-token": {ID: "host-user", Name: "Host Account", Email: "host@example.com"},
	}}, "http://frontend.test")
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"http://frontend.test"}})
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	mustConnectAuthenticatedSession(t, conn, "", "host-token")
}

func TestWebSocketOriginHelpers(t *testing.T) {
	t.Run("configured server allows empty origin for non-browser clients", func(t *testing.T) {
		server := newWSServerWithAllowedOrigin(nil, "http://frontend.test/")
		if !server.isAllowedOrigin("") {
			t.Fatal("isAllowedOrigin(empty) = false; want true")
		}
		if !server.isAllowedOrigin("http://frontend.test/") {
			t.Fatal("isAllowedOrigin(configured origin) = false; want true")
		}
	})

	t.Run("invalid base url yields empty origin", func(t *testing.T) {
		if got := originFromBaseURL("://bad"); got != "" {
			t.Fatalf("originFromBaseURL(invalid) = %q; want empty", got)
		}
		if got := originFromBaseURL("frontend.test"); got != "" {
			t.Fatalf("originFromBaseURL(missing scheme) = %q; want empty", got)
		}
	})
}

func TestInactiveSocketCannotMutateSessionState(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	conn := mustDialWS(t, httpServer.URL)
	defer conn.Close()
	connected := mustConnectSession(t, conn, "")

	server.lobby.sessions[connected.SessionID].conn = nil

	mustSendEnvelope(t, conn, "leave_room", leaveRoomRequest{})
	mustReadError(t, conn, "session not active on this connection")
}

func TestWebSocketLeaveRoomFlow(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)

	guestConn := mustDialWS(t, httpServer.URL)
	defer guestConn.Close()
	mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, hostConn)

	mustSendEnvelope(t, guestConn, "leave_room", leaveRoomRequest{})
	left := mustReadLeftRoom(t, guestConn)
	if left.RoomCode != hostRoom.Code {
		t.Fatalf("left.RoomCode = %q; want %q", left.RoomCode, hostRoom.Code)
	}

	updatedHostRoom := mustReadRoomState(t, hostConn)
	if len(updatedHostRoom.Players) != 1 {
		t.Fatalf("len(updatedHostRoom.Players) = %d; want 1", len(updatedHostRoom.Players))
	}
	if updatedHostRoom.Players[0].Name != "Host" {
		t.Fatalf("updatedHostRoom.Players[0].Name = %q; want Host", updatedHostRoom.Players[0].Name)
	}

	mustSendEnvelope(t, guestConn, "leave_room", leaveRoomRequest{})
	mustReadError(t, guestConn, "join a room first")
}

func TestSecondLiveWebSocketConnectionForSessionIsRejected(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	hostConnected := mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)

	replacementConn := mustDialWS(t, httpServer.URL)
	defer replacementConn.Close()
	mustSendEnvelope(t, replacementConn, "connect", connectRequest{SessionID: hostConnected.SessionID})
	mustReadError(t, replacementConn, "session already connected")

	mustSendEnvelope(t, hostConn, "leave_room", leaveRoomRequest{})
	left := mustReadLeftRoom(t, hostConn)
	if left.RoomCode != hostRoom.Code {
		t.Fatalf("left.RoomCode = %q; want %q", left.RoomCode, hostRoom.Code)
	}
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

func TestHandleHealth(t *testing.T) {
	server := newWSServer()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	server.handleHealth(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("recorder.Code = %d; want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q; want *", got)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q; want application/json", got)
	}
	var response healthResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("json.NewDecoder().Decode() error = %v", err)
	}
	if response.Status != "ok" {
		t.Fatalf("response.Status = %q; want ok", response.Status)
	}
}

func TestHandleHealthWriteError(t *testing.T) {
	server := newWSServer()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)

	server.handleHealth(failingResponseWriter{}, request)
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
	guestConnected := mustConnectSession(t, guestConn, "")
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
	if err := rawConn.WriteJSON(wsEnvelope{Type: "leave_room", Data: mustMarshalRawMessage(leaveRoomRequest{})}); err != nil {
		t.Fatalf("WriteJSON(leave_room before connect) error = %v", err)
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
	if connectedEnvelope.Type != "error" {
		t.Fatalf("connectedEnvelope.Type = %q; want error", connectedEnvelope.Type)
	}
	var reconnectError errorEvent
	if err := json.Unmarshal(connectedEnvelope.Data, &reconnectError); err != nil {
		t.Fatalf("json.Unmarshal(error event) error = %v", err)
	}
	if reconnectError.Message != "session already connected" {
		t.Fatalf("reconnectError.Message = %q; want session already connected", reconnectError.Message)
	}

	mustSendEnvelope(t, hostConn, "leave_room", leaveRoomRequest{})
	_ = mustReadLeftRoom(t, hostConn)
	updatedGuestRoom := mustReadRoomState(t, guestConn)
	if len(updatedGuestRoom.Players) != 1 {
		t.Fatalf("len(updatedGuestRoom.Players) = %d; want 1", len(updatedGuestRoom.Players))
	}
	if updatedGuestRoom.Players[0].PlayerID != guestConnected.PlayerID {
		t.Fatalf("updatedGuestRoom.Players[0].PlayerID = %q; want %q", updatedGuestRoom.Players[0].PlayerID, guestConnected.PlayerID)
	}

	if err := rawConn.Close(); err != nil {
		t.Fatalf("rawConn.Close() error = %v", err)
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
	mustReadError(t, soloConn, "join a room first")
	if soloHostSession.roomCode != "" {
		t.Fatalf("soloHostSession.roomCode = %q; want empty", soloHostSession.roomCode)
	}
	soloHostSession.roomCode = ""
	mustSendEnvelope(t, soloConn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, soloConn, "join a room first")
}

func TestHandleConnectionReturnsWhenInitialConnectedWriteFails(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		if messageType == "connected" {
			return errors.New("forced connected failure")
		}
		return writeEvent(conn, messageType, data)
	}

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

func TestHandleConnectionReturnsWhenLeftRoomWriteFails(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = writeEvent
	originalAddPlayer := addPlayerToGameState
	defer func() { addPlayerToGameState = originalAddPlayer }()
	addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
		return state.AddPlayer(player)
	}

	server := newWSServer()
	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		if conn == serverConn && messageType == "left_room" {
			return errors.New("forced left_room failure")
		}
		return writeEvent(conn, messageType, data)
	}

	done := make(chan struct{})
	go func() {
		server.handleConnection(serverConn)
		close(done)
	}()

	if err := clientConn.WriteJSON(wsEnvelope{Type: "connect", Data: mustMarshalRawMessage(connectRequest{})}); err != nil {
		t.Fatalf("WriteJSON(connect) error = %v", err)
	}
	if envelope := mustReadEnvelopeFromConn(t, clientConn); envelope.Type != "connected" {
		t.Fatalf("envelope.Type = %q; want connected", envelope.Type)
	}
	if err := clientConn.WriteJSON(wsEnvelope{Type: "create_room", Data: mustMarshalRawMessage(createRoomRequest{Name: "Host"})}); err != nil {
		t.Fatalf("WriteJSON(create_room) error = %v", err)
	}
	if envelope := mustReadEnvelopeFromConn(t, clientConn); envelope.Type != "room_state" {
		t.Fatalf("envelope.Type = %q; want room_state", envelope.Type)
	}
	if err := clientConn.WriteJSON(wsEnvelope{Type: "leave_room", Data: mustMarshalRawMessage(leaveRoomRequest{})}); err != nil {
		t.Fatalf("WriteJSON(leave_room) error = %v", err)
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
	if err := conn.Close(); err != nil {
		t.Fatalf("conn.Close() after failed connect error = %v", err)
	}

	conn = mustDialWS(t, httpServer.URL)
	mustConnectSession(t, conn, "")
	if err := conn.WriteJSON(wsEnvelope{Type: "leave_room"}); err != nil {
		t.Fatalf("WriteJSON(leave_room missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
	mustSendEnvelope(t, conn, "leave_room", leaveRoomRequest{})
	mustReadError(t, conn, "join a room first")
	if err := conn.WriteJSON(wsEnvelope{Type: "create_room"}); err != nil {
		t.Fatalf("WriteJSON(create_room missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
	mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Host"})
	room := mustReadRoomState(t, conn)
	mustSendEnvelope(t, conn, "leave_room", leaveRoomRequest{})
	_ = mustReadLeftRoom(t, conn)
	mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Host"})
	room = mustReadRoomState(t, conn)
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
	mustSendEnvelope(t, guestConn, "leave_room", leaveRoomRequest{})
	_ = mustReadLeftRoom(t, guestConn)
	room = mustReadRoomState(t, conn)
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: room.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, conn)
	mustSendEnvelope(t, conn, "start_game", startGameRequest{DealerIndex: 99})
	mustReadError(t, conn, game.ErrInvalidDealer.Error())
	mustSendEnvelope(t, guestConn, "leave_room", leaveRoomRequest{})
	_ = mustReadLeftRoom(t, guestConn)
	_ = mustReadRoomState(t, conn)
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: room.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, conn)
	mustSendEnvelope(t, conn, "start_game", startGameRequest{DealerIndex: 0})
	_ = mustReadRoomState(t, conn)
	_ = mustReadRoomState(t, guestConn)
	mustSendEnvelope(t, guestConn, "leave_room", leaveRoomRequest{})
	mustReadError(t, guestConn, "can only leave in lobby")
	_ = conn.Close()
	_ = guestConn.Close()
}

func TestWriteErrorAndBroadcastRoomState(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()

	server := newWSServer()
	hostRoom := roomSnapshot{Code: "ROOM", Phase: "lobby"}

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

	writeConn, _, closeWritePair := newSocketPair(t)
	defer closeWritePair()
	server.writeError(writeConn, errors.New("ignored"))
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		return errors.New("emit boom")
	}
	server.writeError(writeConn, errors.New("write error branch"))
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		return errSocketClosed
	}
	server.writeError(writeConn, errors.New("ignored closed"))
	server.broadcastRoomState(hostRoom, []*websocket.Conn{writeConn})
	emitEvent = func(conn *websocket.Conn, messageType string, data any) error {
		return errors.New("emit boom")
	}
	server.broadcastRoomState(hostRoom, []*websocket.Conn{writeConn})
}

func TestDecodePayload(t *testing.T) {
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
}

func TestMustMarshalRawMessage(t *testing.T) {
	envelope := mustMarshalRawMessage(struct {
		Name string `json:"name"`
	}{Name: "x"})
	if string(envelope) != `{"name":"x"}` {
		t.Fatalf("mustMarshalRawMessage() = %s; want {\"name\":\"x\"}", string(envelope))
	}
}

func TestMustMarshalRawMessagePanicsOnMarshalError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("mustMarshalRawMessage() did not panic")
		}
	}()

	_ = mustMarshalRawMessage(rawInvalidMessage{})
}

func TestWriteEvent(t *testing.T) {
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

func TestOtherConnections(t *testing.T) {
	hostConn, _, closeHost := newSocketPair(t)
	defer closeHost()
	guestConn, _, closeGuest := newSocketPair(t)
	defer closeGuest()
	thirdConn, _, closeThird := newSocketPair(t)
	defer closeThird()

	filtered := otherConnections([]*websocket.Conn{hostConn, guestConn, thirdConn}, guestConn)
	if len(filtered) != 2 || filtered[0] != hostConn || filtered[1] != thirdConn {
		t.Fatalf("otherConnections() = %v; want [%p %p]", filtered, hostConn, thirdConn)
	}
}

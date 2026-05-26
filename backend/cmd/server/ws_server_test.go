package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/EmilsValdmanis/compositions/internal/game"
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
	hostPendingStart := mustReadRoomState(t, hostConn)
	guestPendingStart := mustReadRoomState(t, reconnectedGuestConn)
	for _, room := range []roomSnapshot{hostPendingStart, guestPendingStart} {
		if room.Phase != "lobby" {
			t.Fatalf("room.Phase = %q; want lobby while waiting for deal choice", room.Phase)
		}
		if room.PendingDealChoice == nil {
			t.Fatal("room.PendingDealChoice = nil; want pending dealing choice")
		}
		if room.PendingDealChoice.ChooserPlayerID != hostConnected.PlayerID {
			t.Fatalf("room.PendingDealChoice.ChooserPlayerID = %q; want %q", room.PendingDealChoice.ChooserPlayerID, hostConnected.PlayerID)
		}
	}
	mustSendEnvelope(t, hostConn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	hostStarted := mustReadRoomState(t, hostConn)
	guestStarted := mustReadRoomState(t, reconnectedGuestConn)
	hostInitialGame := mustReadGameState(t, hostConn, hostStarted.Code)
	guestInitialGame := mustReadGameState(t, reconnectedGuestConn, guestStarted.Code)
	for _, room := range []roomSnapshot{hostStarted, guestStarted} {
		if room.Phase != "in_progress" {
			t.Fatalf("room.Phase = %q; want in_progress", room.Phase)
		}
		if room.DealerIndex != 1 {
			t.Fatalf("room.DealerIndex = %d; want 1", room.DealerIndex)
		}
	}
	if hostInitialGame.Game.Turn.PlayerID != hostConnected.PlayerID || guestInitialGame.Game.Turn.PlayerID != hostConnected.PlayerID {
		t.Fatalf("initial turn player = host:%q guest:%q; want %q", hostInitialGame.Game.Turn.PlayerID, guestInitialGame.Game.Turn.PlayerID, hostConnected.PlayerID)
	}
	if err := hostConn.Close(); err != nil {
		t.Fatalf("hostConn.Close() error = %v", err)
	}
	hostDisconnectedRoom := mustReadRoomState(t, reconnectedGuestConn)
	if hostDisconnectedRoom.Players[0].Connected {
		t.Fatal("host connected flag = true; want false after refresh disconnect")
	}

	refreshedHostConn := mustDialWS(t, httpServer.URL)
	defer refreshedHostConn.Close()
	refreshedHost := mustConnectSession(t, refreshedHostConn, hostConnected.SessionID)
	if refreshedHost.SessionID != hostConnected.SessionID || refreshedHost.PlayerID != hostConnected.PlayerID {
		t.Fatalf("refreshedHost = %#v; want same host session/player", refreshedHost)
	}
	refreshedHostRoom := mustReadRoomState(t, refreshedHostConn)
	refreshedHostGame := mustReadGameState(t, refreshedHostConn, refreshedHostRoom.Code)
	if refreshedHostRoom.Phase != "in_progress" {
		t.Fatalf("refreshedHostRoom.Phase = %q; want in_progress", refreshedHostRoom.Phase)
	}
	if len(refreshedHostGame.Game.Hand) != game.InitialHandSize {
		t.Fatalf("refreshed host hand = %d; want %d", len(refreshedHostGame.Game.Hand), game.InitialHandSize)
	}
	if refreshedHostGame.Game.Turn.PlayerID != hostConnected.PlayerID {
		t.Fatalf("refreshed host turn player = %q; want %q", refreshedHostGame.Game.Turn.PlayerID, hostConnected.PlayerID)
	}
	_ = mustReadRoomState(t, reconnectedGuestConn)
	if err := reconnectedGuestConn.Close(); err != nil {
		t.Fatalf("reconnectedGuestConn.Close() error = %v", err)
	}
}

func TestAuthenticatedWebSocketRequiresVerifiedUser(t *testing.T) {
	server := newWSServerWithAuth(&authHandler{store: &stubAuthStore{}, now: time.Now})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	unauthenticatedConn := mustDialWS(t, httpServer.URL)
	defer unauthenticatedConn.Close()
	mustSendEnvelope(t, unauthenticatedConn, "connect", connectRequest{})
	mustReadError(t, unauthenticatedConn, errAuthenticationRequired.Error())

	hostStore := &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "host-user", Name: "Host Account", Email: "host@example.com", ImageURL: "https://cdn.example.com/host.png"}}
	server.auth = &authHandler{store: hostStore, now: time.Now}
	hostConn := mustDialWSWithCookie(t, httpServer.URL, "host-token")
	defer hostConn.Close()
	hostConnected := mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Spoofed Host"})
	hostRoom := mustReadRoomState(t, hostConn)
	if got := hostRoom.Players[0].Name; got != "Host Account" {
		t.Fatalf("hostRoom.Players[0].Name = %q; want Host Account", got)
	}
	if got := hostRoom.Players[0].ImageURL; got != "https://cdn.example.com/host.png" {
		t.Fatalf("hostRoom.Players[0].ImageURL = %q; want https://cdn.example.com/host.png", got)
	}

	guestStore := &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "guest-user", Name: "Guest Account", Email: "guest@example.com", ImageURL: "https://cdn.example.com/guest.png"}}
	server.auth = &authHandler{store: guestStore, now: time.Now}
	guestConn := mustDialWSWithCookie(t, httpServer.URL, "guest-token")
	defer guestConn.Close()
	guestConnected := mustConnectSession(t, guestConn, "")
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
		if room.Players[1].ImageURL != "https://cdn.example.com/guest.png" {
			t.Fatalf("room.Players[1].ImageURL = %q; want https://cdn.example.com/guest.png", room.Players[1].ImageURL)
		}
	}

	mustSendEnvelope(t, hostConn, "start_game", startGameRequest{DealerIndex: 1})
	pendingHostRoom := mustReadRoomState(t, hostConn)
	pendingGuestRoom := mustReadRoomState(t, guestConn)
	for _, room := range []roomSnapshot{pendingHostRoom, pendingGuestRoom} {
		if room.PendingDealChoice == nil || room.PendingDealChoice.ChooserPlayerID != hostConnected.PlayerID {
			t.Fatalf("pending room = %#v; want host as pending chooser", room)
		}
	}
	mustSendEnvelope(t, hostConn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	startedHostRoom := mustReadRoomState(t, hostConn)
	startedGuestRoom := mustReadRoomState(t, guestConn)
	_ = mustReadGameState(t, hostConn, startedHostRoom.Code)
	_ = mustReadGameState(t, guestConn, startedGuestRoom.Code)
	if err := hostConn.Close(); err != nil {
		t.Fatalf("hostConn.Close() error = %v", err)
	}
	_ = mustReadRoomState(t, guestConn)

	server.auth = &authHandler{store: hostStore, now: time.Now}
	refreshedHostConn := mustDialWSWithCookie(t, httpServer.URL, "host-token")
	defer refreshedHostConn.Close()
	refreshedHost := mustConnectSession(t, refreshedHostConn, "")
	if refreshedHost.SessionID != hostConnected.SessionID || refreshedHost.PlayerID != hostConnected.PlayerID {
		t.Fatalf("refreshedHost = %#v; want same authenticated host session/player", refreshedHost)
	}
	refreshedHostRoom := mustReadRoomState(t, refreshedHostConn)
	refreshedHostGame := mustReadGameState(t, refreshedHostConn, refreshedHostRoom.Code)
	if refreshedHostRoom.Phase != "in_progress" {
		t.Fatalf("refreshedHostRoom.Phase = %q; want in_progress", refreshedHostRoom.Phase)
	}
	if len(refreshedHostGame.Game.Hand) != game.InitialHandSize {
		t.Fatalf("refreshed authenticated host hand = %d; want %d", len(refreshedHostGame.Game.Hand), game.InitialHandSize)
	}
	_ = mustReadRoomState(t, guestConn)

	server.auth = &authHandler{store: guestStore, now: time.Now}
	hijackConn := mustDialWSWithCookie(t, httpServer.URL, "guest-token")
	defer hijackConn.Close()
	mustSendEnvelope(t, hijackConn, "connect", connectRequest{SessionID: hostConnected.SessionID})
	mustReadError(t, hijackConn, "session belongs to a different user")
}

func TestAuthenticatedWebSocketReplacesSecondLiveSocketForSameUser(t *testing.T) {
	store := &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "same-user", Name: "Same Account", Email: "same@example.com"}}
	server := newWSServerWithAuth(&authHandler{store: store, now: time.Now})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	firstConn := mustDialWSWithCookie(t, httpServer.URL, "same-user-token")
	defer firstConn.Close()
	firstConnected := mustConnectSession(t, firstConn, "")

	secondConn := mustDialWSWithCookie(t, httpServer.URL, "same-user-token")
	defer secondConn.Close()
	mustSendEnvelope(t, secondConn, "connect", connectRequest{})
	secondConnected := mustReadConnectedEvent(t, secondConn)
	if secondConnected.SessionID != firstConnected.SessionID {
		t.Fatalf("secondConnected.SessionID = %q; want %q", secondConnected.SessionID, firstConnected.SessionID)
	}
	if secondConnected.PlayerID != firstConnected.PlayerID {
		t.Fatalf("secondConnected.PlayerID = %q; want %q", secondConnected.PlayerID, firstConnected.PlayerID)
	}
	if len(server.lobby.sessions) != 1 {
		t.Fatalf("len(server.lobby.sessions) = %d; want 1", len(server.lobby.sessions))
	}

	mustSendEnvelope(t, firstConn, "create_room", createRoomRequest{Name: "Should Work"})
	mustReadError(t, firstConn, "session not active on this connection")
	mustSendEnvelope(t, secondConn, "create_room", createRoomRequest{Name: "Should Work"})
	room := mustReadRoomState(t, secondConn)
	if room.HostPlayerID != firstConnected.PlayerID {
		t.Fatalf("room.HostPlayerID = %q; want %q", room.HostPlayerID, firstConnected.PlayerID)
	}
}

func TestAuthenticatedWebSocketReusesSessionAfterDisconnectForSameUser(t *testing.T) {
	store := &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "same-user", Name: "Same Account", Email: "same@example.com"}}
	server := newWSServerWithAuth(&authHandler{store: store, now: time.Now})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	firstConn := mustDialWSWithCookie(t, httpServer.URL, "same-user-token")
	firstConnected := mustConnectSession(t, firstConn, "")
	if err := firstConn.Close(); err != nil {
		t.Fatalf("firstConn.Close() error = %v", err)
	}

	secondConn := mustDialWSWithCookie(t, httpServer.URL, "same-user-token")
	defer secondConn.Close()
	secondConnected := mustConnectSession(t, secondConn, "")

	if secondConnected.SessionID != firstConnected.SessionID {
		t.Fatalf("secondConnected.SessionID = %q; want %q", secondConnected.SessionID, firstConnected.SessionID)
	}
	if secondConnected.PlayerID != firstConnected.PlayerID {
		t.Fatalf("secondConnected.PlayerID = %q; want %q", secondConnected.PlayerID, firstConnected.PlayerID)
	}
}

func TestHandleConnectResumeGameStateErrorPaths(t *testing.T) {
	t.Run("snapshot failure", func(t *testing.T) {
		originalEmit := emitEvent
		defer func() { emitEvent = originalEmit }()
		emitEvent = func(_ *websocket.Conn, _ string, _ any) error {
			return nil
		}

		server := newWSServer()
		gameState := game.NewGameState()
		if err := gameState.AddPlayer(newPlayerWithID("other-a")); err != nil {
			t.Fatalf("AddPlayer(other-a) error = %v", err)
		}
		if err := gameState.AddPlayer(newPlayerWithID("other-b")); err != nil {
			t.Fatalf("AddPlayer(other-b) error = %v", err)
		}
		if err := gameState.StartGame(0, 1, game.DealRoundRobin, nil, 0); err != nil {
			t.Fatalf("StartGame() error = %v", err)
		}

		sessionID := "session-with-missing-game-player"
		playerID := "room-player-only"
		server.lobby.sessions[sessionID] = &playerSession{
			sessionID: sessionID,
			playerID:  playerID,
			roomCode:  "ROOM",
		}
		server.lobby.rooms["ROOM"] = &room{
			code:      "ROOM",
			gameState: gameState,
			hostID:    playerID,
			players: []*roomPlayer{{
				player:    newPlayerWithID(playerID),
				sessionID: sessionID,
				seat:      0,
				host:      true,
			}},
		}

		_, shouldClose := server.handleConnect(nil, httptest.NewRequest(http.MethodGet, "/ws", nil), wsEnvelope{Data: mustMarshalRawMessage(connectRequest{SessionID: sessionID})})
		if !shouldClose {
			t.Fatal("handleConnect() shouldClose = false; want true after game state snapshot failure")
		}
	})

	t.Run("game state write failure", func(t *testing.T) {
		originalEmit := emitEvent
		defer func() { emitEvent = originalEmit }()
		emitEvent = func(_ *websocket.Conn, messageType string, _ any) error {
			if messageType == "game_state" {
				return errors.New("write game state boom")
			}
			return nil
		}

		server := newWSServer()
		hostEvent, _, _, err := server.lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect(host) error = %v", err)
		}
		hostRoom, _, err := server.lobby.createRoom(hostEvent.SessionID, "Host")
		if err != nil {
			t.Fatalf("createRoom() error = %v", err)
		}
		guestEvent, _, _, err := server.lobby.connect("", nil)
		if err != nil {
			t.Fatalf("connect(guest) error = %v", err)
		}
		if _, _, err := server.lobby.joinRoom(guestEvent.SessionID, hostRoom.Code, "Guest"); err != nil {
			t.Fatalf("joinRoom() error = %v", err)
		}
		if _, _, err := server.lobby.startGame(hostEvent.SessionID, 0); err != nil {
			t.Fatalf("startGame() error = %v", err)
		}
		if _, _, err := server.lobby.chooseDealing(guestEvent.SessionID, "round_robin"); err != nil {
			t.Fatalf("chooseDealing() error = %v", err)
		}
		server.lobby.sessions[hostEvent.SessionID].conn = nil

		_, shouldClose := server.handleConnect(nil, httptest.NewRequest(http.MethodGet, "/ws", nil), wsEnvelope{Data: mustMarshalRawMessage(connectRequest{SessionID: hostEvent.SessionID})})
		if !shouldClose {
			t.Fatal("handleConnect() shouldClose = false; want true after game state write failure")
		}
	})
}

func TestAuthenticatedWebSocketRejectsUnexpectedOrigin(t *testing.T) {
	server := newWSServerWithAllowedOrigin(&authHandler{store: &stubAuthStore{}, now: time.Now}, "http://frontend.test")
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
	server := newWSServerWithAllowedOrigin(&authHandler{store: &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "host-user", Name: "Host Account", Email: "host@example.com"}}, now: time.Now}, "http://frontend.test")
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"http://frontend.test"}, "Cookie": {authCookieName + "=host-token"}})
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	mustConnectSession(t, conn, "")
}

func TestCloneIndexMap(t *testing.T) {
	t.Run("returns nil for empty input", func(t *testing.T) {
		if cloned := cloneIndexMap(nil); cloned != nil {
			t.Fatalf("cloneIndexMap(nil) = %#v; want nil", cloned)
		}
		if cloned := cloneIndexMap(map[string]int{}); cloned != nil {
			t.Fatalf("cloneIndexMap(empty) = %#v; want nil", cloned)
		}
	})

	t.Run("clones non-empty map", func(t *testing.T) {
		source := map[string]int{"ace-1": 0}
		cloned := cloneIndexMap(source)
		if cloned["ace-1"] != 0 {
			t.Fatalf("cloneIndexMap(source) = %#v; want preserved values", cloned)
		}
		cloned["ace-1"] = 2
		if source["ace-1"] != 0 {
			t.Fatal("cloneIndexMap(source) reused source map")
		}
	})
}

func TestAuthenticatedWebSocketRejectsMissingOriginWhenConfigured(t *testing.T) {
	server := newWSServerWithAllowedOrigin(&authHandler{store: &stubAuthStore{}, now: time.Now}, "http://frontend.test")
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
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

func TestWebSocketOriginHelpers(t *testing.T) {
	t.Run("configured server requires matching browser origin", func(t *testing.T) {
		server := newWSServerWithAllowedOrigin(nil, "http://frontend.test/")
		if server.isAllowedOrigin("") {
			t.Fatal("isAllowedOrigin(empty) = true; want false")
		}
		if !server.isAllowedOrigin("http://frontend.test/") {
			t.Fatal("isAllowedOrigin(configured origin) = false; want true")
		}
	})

	t.Run("test constructors stay permissive without configured origin", func(t *testing.T) {
		server := newWSServerWithAllowedOrigin(nil, "")
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

func TestNewConfiguredWSServerRejectsInvalidOriginConfig(t *testing.T) {
	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "frontend.test")

	server, err := newConfiguredWSServer()
	if err == nil || err.Error() != "FRONTEND_URL must be a valid absolute URL" {
		t.Fatalf("newConfiguredWSServer() error = %v; want FRONTEND_URL must be a valid absolute URL", err)
	}
	if server != nil {
		t.Fatalf("server = %#v; want nil", server)
	}
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

func TestSecondLiveWebSocketConnectionForSessionReplacesActiveSocket(t *testing.T) {
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
	replacementConnected := mustReadConnectedEvent(t, replacementConn)
	if replacementConnected.SessionID != hostConnected.SessionID {
		t.Fatalf("replacementConnected.SessionID = %q; want %q", replacementConnected.SessionID, hostConnected.SessionID)
	}
	if replacementConnected.PlayerID != hostConnected.PlayerID {
		t.Fatalf("replacementConnected.PlayerID = %q; want %q", replacementConnected.PlayerID, hostConnected.PlayerID)
	}
	replacementRoom := mustReadRoomState(t, replacementConn)
	if replacementRoom.Code != hostRoom.Code {
		t.Fatalf("replacementRoom.Code = %q; want %q", replacementRoom.Code, hostRoom.Code)
	}

	mustSendEnvelope(t, hostConn, "leave_room", leaveRoomRequest{})
	mustReadError(t, hostConn, "session not active on this connection")
	mustSendEnvelope(t, replacementConn, "leave_room", leaveRoomRequest{})
	left := mustReadLeftRoom(t, replacementConn)
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
	reconnected := mustReadConnectedEvent(t, rawConn)
	if reconnected.SessionID != hostConnected.SessionID {
		t.Fatalf("reconnected.SessionID = %q; want %q", reconnected.SessionID, hostConnected.SessionID)
	}
	reconnectedRoom := mustReadRoomState(t, rawConn)
	if reconnectedRoom.Code != hostRoom.Code {
		t.Fatalf("reconnectedRoom.Code = %q; want %q", reconnectedRoom.Code, hostRoom.Code)
	}
	guestRoomAfterReconnect := mustReadRoomState(t, guestConn)
	if len(guestRoomAfterReconnect.Players) != 2 {
		t.Fatalf("len(guestRoomAfterReconnect.Players) = %d; want 2", len(guestRoomAfterReconnect.Players))
	}
	mustSendEnvelope(t, hostConn, "leave_room", leaveRoomRequest{})
	mustReadError(t, hostConn, "session not active on this connection")

	mustSendEnvelope(t, rawConn, "leave_room", leaveRoomRequest{})
	_ = mustReadLeftRoom(t, rawConn)
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
	soloConnected := mustConnectSession(t, soloConn, "")
	mustSendEnvelope(t, soloConn, "create_room", createRoomRequest{Name: "Solo"})
	soloRoom := mustReadRoomState(t, soloConn)
	mustSendEnvelope(t, soloConn, "start_game", startGameRequest{DealerIndex: 0})
	mustReadError(t, soloConn, "need at least 2 players to start")
	soloHostSession := server.lobby.sessions[soloConnected.SessionID]
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

func TestRoomStateOmitsPlayerSessionIDs(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)
	if hostRoom.Players[0].SessionID != "" {
		t.Fatalf("hostRoom.Players[0].SessionID = %q; want empty", hostRoom.Players[0].SessionID)
	}

	guestConn := mustDialWS(t, httpServer.URL)
	defer guestConn.Close()
	mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	guestRoom := mustReadRoomState(t, guestConn)
	hostRoom = mustReadRoomState(t, hostConn)

	for _, room := range []roomSnapshot{hostRoom, guestRoom} {
		for i, player := range room.Players {
			if player.SessionID != "" {
				t.Fatalf("room.Players[%d].SessionID = %q; want empty", i, player.SessionID)
			}
		}
	}
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
		server.handleConnection(serverConn, httptest.NewRequest(http.MethodGet, "/ws", nil))
		close(done)
	}()

	if err := clientConn.WriteJSON(wsEnvelope{Type: "connect", Data: mustMarshalRawMessage(connectRequest{})}); err != nil {
		t.Fatalf("WriteJSON(connect) error = %v", err)
	}
	<-done
}

func TestHandleConnectionSendsHeartbeatPing(t *testing.T) {
	server := newWSServer()
	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	originalPingInterval := defaultWSPingInterval
	originalReadTimeout := defaultWSReadTimeout
	defer func() {
		defaultWSPingInterval = originalPingInterval
		defaultWSReadTimeout = originalReadTimeout
	}()
	defaultWSPingInterval = 10 * time.Millisecond
	defaultWSReadTimeout = 50 * time.Millisecond

	done := make(chan struct{})
	go func() {
		server.handleConnection(serverConn, httptest.NewRequest(http.MethodGet, "/ws", nil))
		close(done)
	}()

	pingReceived := make(chan struct{}, 1)
	clientConn.SetPingHandler(func(appData string) error {
		select {
		case pingReceived <- struct{}{}:
		default:
		}
		return clientConn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})
	if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}
	readDone := make(chan error, 1)
	go func() {
		for {
			if _, _, err := clientConn.ReadMessage(); err != nil {
				readDone <- err
				return
			}
		}
	}()

	select {
	case <-pingReceived:
	case err := <-readDone:
		t.Fatalf("ReadMessage() error before ping = %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for heartbeat ping")
	}

	if err := clientConn.WriteControl(websocket.PongMessage, []byte("pong"), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("WriteControl(PongMessage) error = %v", err)
	}

	if err := serverConn.Close(); err != nil {
		t.Fatalf("serverConn.Close() error = %v", err)
	}
	if err := clientConn.Close(); err != nil {
		t.Fatalf("clientConn.Close() error = %v", err)
	}
	<-done
}

func TestHandleConnectionProcessesPongFrames(t *testing.T) {
	server := newWSServer()
	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		server.handleConnection(serverConn, httptest.NewRequest(http.MethodGet, "/ws", nil))
		close(done)
	}()

	if err := clientConn.WriteControl(websocket.PongMessage, []byte("pong"), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("WriteControl(PongMessage) error = %v", err)
	}
	if err := clientConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second)); err != nil {
		t.Fatalf("WriteControl(CloseMessage) error = %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handleConnection to exit after pong")
	}
}

func TestSetWSReadDeadline(t *testing.T) {
	serverConn, _, cleanup := newSocketPair(t)
	defer cleanup()

	if err := setWSReadDeadline(serverConn); err != nil {
		t.Fatalf("setWSReadDeadline() error = %v", err)
	}
}

func TestHandleConnectionReturnsWhenHeartbeatPingWriteFails(t *testing.T) {
	server := newWSServer()
	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	originalPingInterval := defaultWSPingInterval
	originalReadTimeout := defaultWSReadTimeout
	originalWriteControl := writeControl
	defer func() {
		defaultWSPingInterval = originalPingInterval
		defaultWSReadTimeout = originalReadTimeout
		writeControl = originalWriteControl
	}()
	defaultWSPingInterval = 10 * time.Millisecond
	defaultWSReadTimeout = 5 * time.Second
	writeControl = func(conn *websocket.Conn, messageType int, data []byte, deadline time.Time) error {
		return errors.New("forced ping failure")
	}

	if err := clientConn.Close(); err != nil {
		t.Fatalf("clientConn.Close() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		server.handleConnection(serverConn, httptest.NewRequest(http.MethodGet, "/ws", nil))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for heartbeat ping failure to stop connection")
	}
}

func TestRunHeartbeatPingLoopReturnsOnPingDone(t *testing.T) {
	conn, _, cleanup := newSocketPair(t)
	defer cleanup()

	pingDone := make(chan struct{})
	ticks := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		runHeartbeatPingLoop(conn, pingDone, ticks)
		close(done)
	}()

	close(pingDone)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ping loop to stop on pingDone")
	}
}

func TestRunHeartbeatPingLoopReturnsOnWriteFailure(t *testing.T) {
	originalWriteControl := writeControl
	defer func() { writeControl = originalWriteControl }()
	writeControl = func(conn *websocket.Conn, messageType int, data []byte, deadline time.Time) error {
		return errors.New("forced ping failure")
	}

	pingDone := make(chan struct{})
	ticks := make(chan time.Time, 1)
	done := make(chan struct{})
	go func() {
		runHeartbeatPingLoop(nil, pingDone, ticks)
		close(done)
	}()

	ticks <- time.Now()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ping loop to stop on write failure")
	}
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
		server.handleConnection(reconnectServerConn, httptest.NewRequest(http.MethodGet, "/ws", nil))
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
		server.handleConnection(serverConn, httptest.NewRequest(http.MethodGet, "/ws", nil))
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
	if err := conn.WriteJSON(wsEnvelope{Type: "choose_dealing"}); err != nil {
		t.Fatalf("WriteJSON(choose_dealing missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
	mustSendEnvelope(t, conn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	mustReadError(t, conn, "no dealing choice is pending")
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
	pendingRoom := mustReadRoomState(t, conn)
	guestPendingRoom := mustReadRoomState(t, guestConn)
	if pendingRoom.PendingDealChoice == nil || guestPendingRoom.PendingDealChoice == nil {
		t.Fatalf("pending deal choice = host:%#v guest:%#v; want pending state", pendingRoom.PendingDealChoice, guestPendingRoom.PendingDealChoice)
	}
	mustSendEnvelope(t, conn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	mustReadError(t, conn, "only the deal chooser can choose dealing type")
	mustSendEnvelope(t, guestConn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	startedRoom := mustReadRoomState(t, conn)
	guestStartedRoom := mustReadRoomState(t, guestConn)
	_ = mustReadGameState(t, conn, startedRoom.Code)
	_ = mustReadGameState(t, guestConn, guestStartedRoom.Code)
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
	server.broadcastGameState([]gameStateRecipient{{conn: nil}, {conn: writeConn, event: gameStateEvent{Room: hostRoom}}})
	server.broadcastActionResult(actionResultEvent{Action: "draw", PlayerID: "p1", OK: true}, []*websocket.Conn{nil, writeConn})
	server.broadcastActionSuccess(actionResultEvent{Action: "draw", PlayerID: "p1", OK: true}, hostRoom, []gameStateRecipient{{conn: nil}, {conn: writeConn, event: gameStateEvent{Room: hostRoom}}})
}

func TestBroadcastActionSuccessResetsRoomAfterGameOver(t *testing.T) {
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()

	server := newWSServer()
	hostConn, hostPeer, closeHostPair := newSocketPair(t)
	defer closeHostPair()

	hostEvent, _, _, err := server.lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(host) error = %v", err)
	}
	hostRoom, _, err := server.lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	room := server.lobby.rooms[hostRoom.Code]
	room.gameState = game.NewGameState()
	if err := room.gameState.AddPlayer(newPlayerWithID(hostEvent.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(host) error = %v", err)
	}
	setGameStatePhaseForTest(t, room.gameState, game.PhaseGameOver)

	state, ok := room.gameState.SnapshotForPlayer(hostEvent.PlayerID)
	if !ok {
		t.Fatal("SnapshotForPlayer() = false; want true")
	}
	server.broadcastActionSuccess(
		actionResultEvent{Action: "discard", PlayerID: hostEvent.PlayerID, OK: true},
		roomSnapshot{Code: hostRoom.Code, Phase: "game_over", HostPlayerID: hostEvent.PlayerID},
		[]gameStateRecipient{{
			conn:  hostConn,
			event: gameStateEvent{Room: roomSnapshot{Code: hostRoom.Code, Phase: "game_over", HostPlayerID: hostEvent.PlayerID}, Game: state},
		}},
	)

	if envelope := mustReadEnvelopeFromConn(t, hostPeer); envelope.Type != "action_result" {
		t.Fatalf("envelope.Type = %q; want action_result", envelope.Type)
	}
	if envelope := mustReadEnvelopeFromConn(t, hostPeer); envelope.Type != "room_state" {
		t.Fatalf("envelope.Type = %q; want room_state", envelope.Type)
	}
	if envelope := mustReadEnvelopeFromConn(t, hostPeer); envelope.Type != "game_state" {
		t.Fatalf("envelope.Type = %q; want game_state", envelope.Type)
	}
	resetRoom := mustReadRoomState(t, hostPeer)
	if resetRoom.Phase != "lobby" {
		t.Fatalf("resetRoom.Phase = %q; want lobby", resetRoom.Phase)
	}
}

func TestHandleStartNextRound(t *testing.T) {
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()
	makeGameState = game.NewGameState

	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	hostConnected := mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)

	guestConn := mustDialWS(t, httpServer.URL)
	defer guestConn.Close()
	guestConnected := mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, hostConn)

	room := server.lobby.rooms[hostRoom.Code]
	room.gameState = game.NewGameState()
	if err := room.gameState.AddPlayer(newPlayerWithID(hostConnected.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(host) error = %v", err)
	}
	if err := room.gameState.AddPlayer(newPlayerWithID(guestConnected.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(guest) error = %v", err)
	}
	setGameStatePhaseForTest(t, room.gameState, game.PhaseRoundOver)

	mustSendEnvelope(t, guestConn, "start_next_round", startNextRoundRequest{})
	mustReadError(t, guestConn, "only the host can start the next round")

	mustSendEnvelope(t, hostConn, "start_next_round", startNextRoundRequest{})
	hostStartedRoom := mustReadRoomState(t, hostConn)
	guestStartedRoom := mustReadRoomState(t, guestConn)
	hostGame := mustReadGameState(t, hostConn, hostStartedRoom.Code)
	guestGame := mustReadGameState(t, guestConn, guestStartedRoom.Code)
	if hostStartedRoom.Phase != "in_progress" || guestStartedRoom.Phase != "in_progress" {
		t.Fatalf("started phases = %q/%q; want in_progress", hostStartedRoom.Phase, guestStartedRoom.Phase)
	}
	if hostGame.Game.Round != 2 || guestGame.Game.Round != 2 {
		t.Fatalf("game rounds = %d/%d; want 2", hostGame.Game.Round, guestGame.Game.Round)
	}
}

func TestHandleStartNextRoundErrors(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	conn := mustDialWS(t, httpServer.URL)
	defer conn.Close()
	mustConnectSession(t, conn, "")
	if err := conn.WriteJSON(wsEnvelope{Type: "start_next_round"}); err != nil {
		t.Fatalf("WriteJSON(start_next_round missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
}

func TestBroadcastActionSuccessGameOverResetFailure(t *testing.T) {
	originalMakeGameState := makeGameState
	defer func() { makeGameState = originalMakeGameState }()

	server := newWSServer()
	hostConn, hostPeer, closeHostPair := newSocketPair(t)
	defer closeHostPair()

	hostEvent, _, _, err := server.lobby.connect("", hostConn)
	if err != nil {
		t.Fatalf("connect(host) error = %v", err)
	}
	hostRoom, _, err := server.lobby.createRoom(hostEvent.SessionID, "Host")
	if err != nil {
		t.Fatalf("createRoom() error = %v", err)
	}
	room := server.lobby.rooms[hostRoom.Code]
	room.gameState = game.NewGameState()
	if err := room.gameState.AddPlayer(newPlayerWithID(hostEvent.PlayerID)); err != nil {
		t.Fatalf("AddPlayer(host) error = %v", err)
	}
	setGameStatePhaseForTest(t, room.gameState, game.PhaseGameOver)
	makeGameState = func() *game.GameState { return nil }

	state, ok := room.gameState.SnapshotForPlayer(hostEvent.PlayerID)
	if !ok {
		t.Fatal("SnapshotForPlayer() = false; want true")
	}
	server.broadcastActionSuccess(
		actionResultEvent{Action: "discard", PlayerID: hostEvent.PlayerID, OK: true},
		roomSnapshot{Code: hostRoom.Code, Phase: "game_over", HostPlayerID: hostEvent.PlayerID},
		[]gameStateRecipient{{
			conn:  hostConn,
			event: gameStateEvent{Room: roomSnapshot{Code: hostRoom.Code, Phase: "game_over", HostPlayerID: hostEvent.PlayerID}, Game: state},
		}},
	)

	if envelope := mustReadEnvelopeFromConn(t, hostPeer); envelope.Type != "action_result" {
		t.Fatalf("envelope.Type = %q; want action_result", envelope.Type)
	}
	if envelope := mustReadEnvelopeFromConn(t, hostPeer); envelope.Type != "room_state" {
		t.Fatalf("envelope.Type = %q; want room_state", envelope.Type)
	}
	if envelope := mustReadEnvelopeFromConn(t, hostPeer); envelope.Type != "game_state" {
		t.Fatalf("envelope.Type = %q; want game_state", envelope.Type)
	}
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

func TestWebSocketActiveGameTurnFlowBroadcastsAndInvalidActions(t *testing.T) {
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
	guestConnected := mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, hostConn)

	mustSendEnvelope(t, hostConn, "start_game", startGameRequest{DealerIndex: 0})
	pendingHostRoom := mustReadRoomState(t, hostConn)
	pendingGuestRoom := mustReadRoomState(t, guestConn)
	if pendingHostRoom.PendingDealChoice == nil || pendingGuestRoom.PendingDealChoice == nil {
		t.Fatalf("pending deal choice = host:%#v guest:%#v; want pending state", pendingHostRoom.PendingDealChoice, pendingGuestRoom.PendingDealChoice)
	}
	mustSendEnvelope(t, guestConn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	hostStartedRoom := mustReadRoomState(t, hostConn)
	guestStartedRoom := mustReadRoomState(t, guestConn)
	hostInitialGame := mustReadGameState(t, hostConn, hostStartedRoom.Code)
	guestInitialGame := mustReadGameState(t, guestConn, guestStartedRoom.Code)
	if hostInitialGame.Game.Turn.PlayerID != guestConnected.PlayerID || guestInitialGame.Game.Turn.PlayerID != guestConnected.PlayerID {
		t.Fatalf("initial turn player = host:%q guest:%q; want %q", hostInitialGame.Game.Turn.PlayerID, guestInitialGame.Game.Turn.PlayerID, guestConnected.PlayerID)
	}
	if len(hostInitialGame.Game.Hand) != game.InitialHandSize || len(guestInitialGame.Game.Hand) != game.InitialHandSize {
		t.Fatalf("initial hand sizes = host:%d guest:%d; want %d", len(hostInitialGame.Game.Hand), len(guestInitialGame.Game.Hand), game.InitialHandSize)
	}

	mustSendEnvelope(t, hostConn, "draw", drawRequest{Source: "deck"})
	mustReadError(t, hostConn, "not your turn")
	mustSendEnvelope(t, guestConn, "draw", drawRequest{Source: "sideways"})
	mustReadError(t, guestConn, "unknown draw source")

	mustSendEnvelope(t, guestConn, "draw", drawRequest{Source: "discard"})
	guestDrawState := mustReadActionBroadcast(t, guestConn, "draw", guestConnected.PlayerID)
	hostDrawState := mustReadActionBroadcast(t, hostConn, "draw", guestConnected.PlayerID)
	if len(guestDrawState.Game.Hand) != game.InitialHandSize+1 {
		t.Fatalf("guest hand after draw = %d; want %d", len(guestDrawState.Game.Hand), game.InitialHandSize+1)
	}
	if len(hostDrawState.Game.Hand) != game.InitialHandSize {
		t.Fatalf("host hand after guest draw = %d; want %d", len(hostDrawState.Game.Hand), game.InitialHandSize)
	}
	if !guestDrawState.Game.Turn.MustUseDiscardDraw {
		t.Fatal("guest draw state MustUseDiscardDraw = false; want true")
	}
	if hostDrawState.Game.TurnActivity == nil {
		t.Fatal("host draw state TurnActivity = nil; want spectator turn activity")
	}
	if hostDrawState.Game.TurnActivity.PlayerID != guestConnected.PlayerID {
		t.Fatalf("host draw TurnActivity.PlayerID = %q; want %q", hostDrawState.Game.TurnActivity.PlayerID, guestConnected.PlayerID)
	}
	if len(hostDrawState.Game.TurnActivity.DraftCompositions) != 0 {
		t.Fatalf("len(host draw TurnActivity.DraftCompositions) = %d; want 0", len(hostDrawState.Game.TurnActivity.DraftCompositions))
	}
	if len(hostDrawState.Game.ActiveCompositions) != len(hostInitialGame.Game.ActiveCompositions) {
		t.Fatalf("host spectator table len after draw = %d; want %d baseline compositions", len(hostDrawState.Game.ActiveCompositions), len(hostInitialGame.Game.ActiveCompositions))
	}

	mustSendEnvelope(t, guestConn, "play", playRequest{Compositions: []compositionRequest{{Cards: []cardRequest{}}}})
	mustReadError(t, guestConn, game.ErrInvalidComposition.Error())
	mustSendEnvelope(t, guestConn, "play", playRequest{Compositions: []compositionRequest{
		{Cards: []cardRequest{cardReq(game.King, game.Hearts), cardReq(game.King, game.Diamonds), cardReq(game.King, game.Clubs)}},
		{Cards: []cardRequest{cardReq(game.Ace, game.Spades), cardReq(game.Two, game.Spades), cardReq(game.Three, game.Spades), cardReq(game.Four, game.Spades)}},
		{Cards: []cardRequest{cardReq(game.Five, game.Hearts), jokerReq(), cardReq(game.Seven, game.Hearts)}},
	}})
	guestPlayState := mustReadActionBroadcast(t, guestConn, "play", guestConnected.PlayerID)
	hostPlayState := mustReadActionBroadcast(t, hostConn, "play", guestConnected.PlayerID)
	if len(guestPlayState.Game.ActiveCompositions) != 3 {
		t.Fatalf("active compositions after play = %d; want 3", len(guestPlayState.Game.ActiveCompositions))
	}
	if !guestPlayState.Game.Players[1].HasOpened {
		t.Fatalf("guest HasOpened = false; want true")
	}
	if hostPlayState.Game.TurnActivity == nil {
		t.Fatal("host play state TurnActivity = nil; want spectator turn activity")
	}
	if len(hostPlayState.Game.TurnActivity.DraftCompositions) != 3 {
		t.Fatalf("len(host play drafts) = %d; want 3", len(hostPlayState.Game.TurnActivity.DraftCompositions))
	}
	if len(hostPlayState.Game.TurnActivity.CompositionActivities) != 3 {
		t.Fatalf("len(host play composition activities) = %d; want 3", len(hostPlayState.Game.TurnActivity.CompositionActivities))
	}
	if len(hostPlayState.Game.ActiveCompositions) != 3 {
		t.Fatalf("host play state should show current table, got %d compositions", len(hostPlayState.Game.ActiveCompositions))
	}
	if len(hostPlayState.Game.TurnActivity.BaselineCompositions) != 0 {
		t.Fatalf("host play baseline table len = %d; want 0 baseline compositions", len(hostPlayState.Game.TurnActivity.BaselineCompositions))
	}

	mustSendEnvelope(t, guestConn, "discard", discardRequest{CardIndex: -1})
	mustReadError(t, guestConn, game.ErrRemovingCard.Error())
	mustSendEnvelope(t, guestConn, "discard", discardRequest{CardIndex: 2})
	guestDiscardState := mustReadActionBroadcast(t, guestConn, "discard", guestConnected.PlayerID)
	hostAfterDiscardState := mustReadActionBroadcast(t, hostConn, "discard", guestConnected.PlayerID)
	if guestDiscardState.Game.Turn.PlayerIndex != 0 {
		t.Fatalf("turn player after guest discard = %d; want 0", guestDiscardState.Game.Turn.PlayerIndex)
	}
	if hostAfterDiscardState.Game.TurnActivity != nil {
		t.Fatalf("host TurnActivity after discard = %#v; want cleared turn activity", hostAfterDiscardState.Game.TurnActivity)
	}

	mustSendEnvelope(t, hostConn, "draw", drawRequest{Source: "deck"})
	_ = mustReadActionBroadcast(t, hostConn, "draw", hostDrawState.Room.HostPlayerID)
	_ = mustReadActionBroadcast(t, guestConn, "draw", hostDrawState.Room.HostPlayerID)
	mustSendEnvelope(t, hostConn, "discard", discardRequest{CardIndex: 9})
	_ = mustReadActionBroadcast(t, hostConn, "discard", hostDrawState.Room.HostPlayerID)
	_ = mustReadActionBroadcast(t, guestConn, "discard", hostDrawState.Room.HostPlayerID)

	mustSendEnvelope(t, guestConn, "draw", drawRequest{Source: "deck"})
	_ = mustReadActionBroadcast(t, guestConn, "draw", guestConnected.PlayerID)
	_ = mustReadActionBroadcast(t, hostConn, "draw", guestConnected.PlayerID)
	mustSendEnvelope(t, guestConn, "play", playRequest{
		Additions: []compositionAdditionRequest{{
			CompositionIndex: 1,
			Cards:            []cardRequest{cardReq(game.Five, game.Spades)},
		}},
		Reclaims: []reclaimRequest{{
			CompositionIndex: 2,
			JokerIndex:       1,
			ReplacementCard:  cardReq(game.Six, game.Hearts),
		}},
	})
	addState := mustReadActionBroadcast(t, guestConn, "play", guestConnected.PlayerID)
	hostAddState := mustReadActionBroadcast(t, hostConn, "play", guestConnected.PlayerID)
	if addState.Game.ActiveCompositions[2].Cards[1].Rank != game.Six {
		t.Fatalf("reclaimed composition card = %#v; want six of hearts", addState.Game.ActiveCompositions[2].Cards[1])
	}
	if len(addState.Game.ActiveCompositions[1].Cards) != 5 {
		t.Fatalf("spade run length after add = %d; want 5", len(addState.Game.ActiveCompositions[1].Cards))
	}
	if hostAddState.Game.TurnActivity == nil {
		t.Fatal("host add state TurnActivity = nil; want spectator turn activity")
	}
	if len(hostAddState.Game.TurnActivity.DraftCompositions) != 1 {
		t.Fatalf("len(host add drafts) = %d; want 1 staged addition draft", len(hostAddState.Game.TurnActivity.DraftCompositions))
	}
	if hostAddState.Game.TurnActivity.DraftCompositions[0].TableIndex == nil || *hostAddState.Game.TurnActivity.DraftCompositions[0].TableIndex != 1 {
		t.Fatalf("host add draft table index = %#v; want 1", hostAddState.Game.TurnActivity.DraftCompositions[0].TableIndex)
	}
	if len(hostAddState.Game.TurnActivity.CompositionActivities) != 2 {
		t.Fatalf("len(host add activities) = %d; want 2", len(hostAddState.Game.TurnActivity.CompositionActivities))
	}
}

func TestWebSocketActionDecodeAndConversionErrors(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	rawConn := mustDialWS(t, httpServer.URL)
	defer rawConn.Close()
	for _, messageType := range []string{"draw", "play", "discard"} {
		mustSendEnvelope(t, rawConn, messageType, struct{}{})
		mustReadError(t, rawConn, "connect first")
	}

	conn := mustDialWS(t, httpServer.URL)
	defer conn.Close()
	mustConnectSession(t, conn, "")
	for _, messageType := range []string{"draw", "play", "discard"} {
		if err := conn.WriteJSON(wsEnvelope{Type: messageType}); err != nil {
			t.Fatalf("WriteJSON(%q missing data) error = %v", messageType, err)
		}
		mustReadError(t, conn, "missing data")
	}
	mustSendEnvelope(t, conn, "play", playRequest{Compositions: []compositionRequest{{Cards: []cardRequest{cardReq(game.King, game.Hearts), cardReq(game.King, game.Diamonds), cardReq(game.King, game.Clubs)}}}})
	mustReadError(t, conn, "join a room first")
	mustSendEnvelope(t, conn, "play", playRequest{Compositions: []compositionRequest{{Cards: []cardRequest{{Rank: 99, Suit: int(game.Hearts)}}}}})
	mustReadError(t, conn, "invalid card rank")
	mustSendEnvelope(t, conn, "play", playRequest{Additions: []compositionAdditionRequest{{CompositionIndex: 0, Cards: []cardRequest{{Rank: int(game.Ace), Suit: 99}}}}})
	mustReadError(t, conn, "invalid card suit")
	mustSendEnvelope(t, conn, "play", playRequest{Reclaims: []reclaimRequest{{ReplacementCard: cardRequest{Rank: 99, Suit: int(game.Clubs)}}}})
	mustReadError(t, conn, "invalid card rank")
	if _, err := compositionsFromRequest([]compositionRequest{{Cards: []cardRequest{{Rank: 99, Suit: int(game.Hearts)}}}}); err == nil || err.Error() != "invalid card rank" {
		t.Fatalf("compositionsFromRequest(invalid card) error = %v; want invalid card rank", err)
	}
	if _, err := compositionsFromRequest([]compositionRequest{{Cards: []cardRequest{cardReq(game.King, game.Hearts), cardReq(game.King, game.Hearts), cardReq(game.King, game.Clubs)}}}); !errors.Is(err, game.ErrInvalidComposition) {
		t.Fatalf("compositionsFromRequest(invalid set) error = %v; want ErrInvalidComposition", err)
	}
	if comps, err := compositionsFromRequest([]compositionRequest{{Cards: []cardRequest{cardReq(game.Five, game.Hearts), cardReq(game.Six, game.Hearts), jokerReq()}}}); err != nil || len(comps) != 1 {
		t.Fatalf("compositionsFromRequest(valid run) = %v, %v; want one comp", comps, err)
	}
	if comps, err := compositionsFromRequest([]compositionRequest{{Cards: []cardRequest{jokerReq(), cardReq(game.Nine, game.Hearts), cardReq(game.Eight, game.Hearts)}}}); err != nil || len(comps) != 1 {
		t.Fatalf("compositionsFromRequest(unordered joker run) = %v, %v; want one comp", comps, err)
	}
	if comps, err := compositionsFromRequest([]compositionRequest{{Cards: []cardRequest{jokerReq(), jokerReq(), jokerReq()}}}); err != nil || len(comps) != 1 || comps[0].Points() != 30 {
		t.Fatalf("compositionsFromRequest(ambiguous jokers) = %v, %v; want one inferred set", comps, err)
	}
	if _, err := additionsFromRequest([]compositionAdditionRequest{{CompositionIndex: 0, Cards: []cardRequest{{Rank: 99, Suit: int(game.Hearts)}}}}); err == nil || err.Error() != "invalid card rank" {
		t.Fatalf("additionsFromRequest(invalid rank) error = %v; want invalid card rank", err)
	}
	if additions, err := additionsFromRequest([]compositionAdditionRequest{{CompositionIndex: 2, Cards: []cardRequest{cardReq(game.Eight, game.Hearts)}}}); err != nil || len(additions) != 1 {
		t.Fatalf("additionsFromRequest(valid) = %v, %v; want one addition", additions, err)
	}
	if _, err := reclaimFromRequest(reclaimRequest{ReplacementCard: cardRequest{Rank: int(game.Ace), Suit: 99}}); err == nil || err.Error() != "invalid card suit" {
		t.Fatalf("reclaimFromRequest(invalid suit) error = %v; want invalid card suit", err)
	}
	if reclaim, err := reclaimFromRequest(reclaimRequest{CompositionIndex: 1, JokerIndex: 2, ReplacementCard: cardReq(game.Ten, game.Clubs)}); err != nil || reclaim.CompositionIndex != 1 || reclaim.JokerIndex != 2 {
		t.Fatalf("reclaimFromRequest(valid) = %#v, %v; want reclaim", reclaim, err)
	}
	if _, err := reclaimsFromRequest([]reclaimRequest{{ReplacementCard: cardRequest{Rank: int(game.Ace), Suit: 99}}}); err == nil || err.Error() != "invalid card suit" {
		t.Fatalf("reclaimsFromRequest(invalid suit) error = %v; want invalid card suit", err)
	}
	if reclaims, err := reclaimsFromRequest([]reclaimRequest{{CompositionIndex: 1, JokerIndex: 2, ReplacementCard: cardReq(game.Ten, game.Clubs)}}); err != nil || len(reclaims) != 1 || reclaims[0].CompositionIndex != 1 || reclaims[0].JokerIndex != 2 {
		t.Fatalf("reclaimsFromRequest(valid) = %#v, %v; want one reclaim", reclaims, err)
	}
	if card, err := cardFromRequest(jokerReq()); err != nil || !card.IsJoker() {
		t.Fatalf("cardFromRequest(joker) = %#v, %v; want joker", card, err)
	}
}

func TestWebSocketDraftUpdateBroadcastsTurnActivity(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	hostConn := mustDialWS(t, httpServer.URL)
	defer hostConn.Close()
	hostConnected := mustConnectSession(t, hostConn, "")
	mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
	hostRoom := mustReadRoomState(t, hostConn)

	guestConn := mustDialWS(t, httpServer.URL)
	defer guestConn.Close()
	guestConnected := mustConnectSession(t, guestConn, "")
	mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: hostRoom.Code, Name: "Guest"})
	_ = mustReadRoomState(t, guestConn)
	_ = mustReadRoomState(t, hostConn)

	mustSendEnvelope(t, hostConn, "start_game", startGameRequest{DealerIndex: 1})
	_ = mustReadRoomState(t, hostConn)
	_ = mustReadRoomState(t, guestConn)
	mustSendEnvelope(t, hostConn, "choose_dealing", chooseDealingRequest{DealType: "round_robin"})
	hostStartedRoom := mustReadRoomState(t, hostConn)
	guestStartedRoom := mustReadRoomState(t, guestConn)
	_ = mustReadGameState(t, hostConn, hostStartedRoom.Code)
	guestInitialGame := mustReadGameState(t, guestConn, guestStartedRoom.Code)
	if guestInitialGame.Game.Turn.PlayerID != hostConnected.PlayerID {
		t.Fatalf("turn player = %q; want host %q", guestInitialGame.Game.Turn.PlayerID, hostConnected.PlayerID)
	}

	mustSendEnvelope(t, hostConn, "draw", drawRequest{Source: "deck"})
	_ = mustReadActionBroadcast(t, hostConn, "draw", hostConnected.PlayerID)
	guestDrawState := mustReadActionBroadcast(t, guestConn, "draw", hostConnected.PlayerID)
	if guestDrawState.Game.TurnActivity == nil {
		t.Fatal("guest draw TurnActivity = nil; want turn context")
	}

	insertIndex := 0
	mustSendEnvelope(t, hostConn, "draft_update", draftUpdateRequest{Compositions: []draftCompositionRequest{{
		TableIndex:        nil,
		InsertIndex:       &insertIndex,
		CardInsertIndices: map[string]int{"king-hearts": 0},
		ReclaimTargets:    map[string]int{"king-diamonds": 1},
		Cards:             []cardRequest{cardReq(game.King, game.Hearts), cardReq(game.King, game.Diamonds), cardReq(game.King, game.Clubs)},
	}}})
	updatedHostRoom := mustReadRoomState(t, hostConn)
	if updatedHostRoom.Code != hostRoom.Code {
		t.Fatalf("updated host room code = %q; want %q", updatedHostRoom.Code, hostRoom.Code)
	}
	updatedHostGame := mustReadGameState(t, hostConn, hostRoom.Code)
	updatedGuestRoom := mustReadRoomState(t, guestConn)
	if updatedGuestRoom.Code != hostRoom.Code {
		t.Fatalf("updated guest room code = %q; want %q", updatedGuestRoom.Code, hostRoom.Code)
	}
	updatedGuestGame := mustReadGameState(t, guestConn, hostRoom.Code)
	if updatedHostGame.Game.TurnActivity != nil {
		t.Fatalf("active player TurnActivity = %#v; want nil", updatedHostGame.Game.TurnActivity)
	}
	if updatedGuestGame.Game.TurnActivity == nil {
		t.Fatal("guest TurnActivity = nil; want draft activity")
	}
	if updatedGuestGame.Game.TurnActivity.PlayerID != hostConnected.PlayerID {
		t.Fatalf("guest TurnActivity.PlayerID = %q; want %q", updatedGuestGame.Game.TurnActivity.PlayerID, hostConnected.PlayerID)
	}
	if len(updatedGuestGame.Game.TurnActivity.DraftCompositions) != 1 {
		t.Fatalf("len(guest draft compositions) = %d; want 1", len(updatedGuestGame.Game.TurnActivity.DraftCompositions))
	}
	if len(updatedGuestGame.Game.TurnActivity.BaselineCompositions) != len(guestDrawState.Game.ActiveCompositions) {
		t.Fatalf("guest baseline table len = %d; want %d", len(updatedGuestGame.Game.TurnActivity.BaselineCompositions), len(guestDrawState.Game.ActiveCompositions))
	}
	if len(updatedGuestGame.Game.ActiveCompositions) != len(guestInitialGame.Game.ActiveCompositions) {
		t.Fatalf("guest current table len = %d; want %d", len(updatedGuestGame.Game.ActiveCompositions), len(guestInitialGame.Game.ActiveCompositions))
	}
	if updatedGuestGame.Game.TurnActivity.DraftCompositions[0].TableIndex != nil {
		t.Fatalf("draft composition tableIndex = %#v; want nil new composition", updatedGuestGame.Game.TurnActivity.DraftCompositions[0].TableIndex)
	}
	if len(updatedGuestGame.Game.TurnActivity.DraftCompositions[0].Cards) != 3 {
		t.Fatalf("draft composition cards = %d; want 3", len(updatedGuestGame.Game.TurnActivity.DraftCompositions[0].Cards))
	}
	if updatedGuestGame.Game.TurnActivity.DraftCompositions[0].CardInsertIndices["king-hearts"] != 0 {
		t.Fatalf("draft composition cardInsertIndices = %#v; want preserved metadata", updatedGuestGame.Game.TurnActivity.DraftCompositions[0].CardInsertIndices)
	}
	if updatedGuestGame.Game.TurnActivity.DraftCompositions[0].ReclaimTargets["king-diamonds"] != 1 {
		t.Fatalf("draft composition reclaimTargets = %#v; want preserved metadata", updatedGuestGame.Game.TurnActivity.DraftCompositions[0].ReclaimTargets)
	}

	mustSendEnvelope(t, guestConn, "draft_update", draftUpdateRequest{})
	mustReadError(t, guestConn, "not your turn")
	mustSendEnvelope(t, hostConn, "draft_update", draftUpdateRequest{Compositions: []draftCompositionRequest{{Cards: []cardRequest{{Rank: 99, Suit: int(game.Hearts)}}}}})
	mustReadError(t, hostConn, "invalid card rank")
	_ = guestConnected
}

func TestWebSocketDraftUpdateRequiresValidSessionPayload(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	conn := mustDialWS(t, httpServer.URL)
	defer conn.Close()
	mustConnectSession(t, conn, "")
	if err := conn.WriteJSON(wsEnvelope{Type: "draft_update"}); err != nil {
		t.Fatalf("WriteJSON(draft_update missing data) error = %v", err)
	}
	mustReadError(t, conn, "missing data")
}

func mustReadActionBroadcast(t *testing.T, conn *websocket.Conn, action, playerID string) gameStateEvent {
	t.Helper()

	actionEnvelope := mustReadEnvelopeFromConn(t, conn)
	if actionEnvelope.Type != "action_result" {
		t.Fatalf("action result type = %q; want action_result", actionEnvelope.Type)
	}
	var result actionResultEvent
	if err := json.Unmarshal(actionEnvelope.Data, &result); err != nil {
		t.Fatalf("json.Unmarshal(action_result) error = %v", err)
	}
	if result.Action != action || result.PlayerID != playerID || !result.OK {
		t.Fatalf("action result = %#v; want %s by %s ok", result, action, playerID)
	}

	room := mustReadRoomState(t, conn)
	gameEnvelope := mustReadEnvelopeFromConn(t, conn)
	if gameEnvelope.Type != "game_state" {
		t.Fatalf("game state type = %q; want game_state", gameEnvelope.Type)
	}
	var event gameStateEvent
	if err := json.Unmarshal(gameEnvelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(game_state) error = %v", err)
	}
	if event.Room.Code != room.Code {
		t.Fatalf("game_state room code = %q; want %q", event.Room.Code, room.Code)
	}
	return event
}

func mustReadGameState(t *testing.T, conn *websocket.Conn, roomCode string) gameStateEvent {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "game_state" {
		t.Fatalf("game state type = %q; want game_state", envelope.Type)
	}
	var event gameStateEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(game_state) error = %v", err)
	}
	if event.Room.Code != roomCode {
		t.Fatalf("game_state room code = %q; want %q", event.Room.Code, roomCode)
	}
	return event
}

func cardReq(rank game.Rank, suit game.Suit) cardRequest {
	return cardRequest{Rank: int(rank), Suit: int(suit)}
}

func jokerReq() cardRequest {
	return cardRequest{IsJoker: true}
}

func roundRobinDeckForServerTest(firstPlayerHand, dealerHand []game.Card, discard game.Card, drawCards ...game.Card) []game.Card {
	deck := make([]game.Card, 0, len(firstPlayerHand)+len(dealerHand)+1+len(drawCards))
	for i := range firstPlayerHand {
		deck = append(deck, firstPlayerHand[i], dealerHand[i])
	}
	deck = append(deck, discard)
	deck = append(deck, drawCards...)
	return deck
}

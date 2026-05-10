package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

func TestRunServerAndMain(t *testing.T) {
	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	t.Setenv("BETTER_AUTH_URL", "http://frontend.test")

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
}

func TestRunServerReturnsEnvErrorBeforeListen(t *testing.T) {
	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	t.Setenv("BETTER_AUTH_URL", "")

	listenCalled := false
	listenAndServe = func(addr string, handler http.Handler) error {
		listenCalled = true
		return nil
	}

	err := runServer(":0")
	if err == nil || err.Error() != "BETTER_AUTH_URL is required" {
		t.Fatalf("runServer() error = %v; want BETTER_AUTH_URL is required", err)
	}
	if listenCalled {
		t.Fatal("listenAndServe() was called; want startup to fail before listen")
	}
}

type rawInvalidMessage struct{}

func (rawInvalidMessage) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal boom")
}

type failingResponseWriter struct{}

func (failingResponseWriter) Header() http.Header {
	return http.Header{}
}

func (failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write boom")
}

func (failingResponseWriter) WriteHeader(statusCode int) {}

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
	return mustReadConnectedEvent(t, conn)
}

func mustConnectAuthenticatedSession(t *testing.T, conn *websocket.Conn, sessionID, authToken string) connectedEvent {
	t.Helper()

	mustSendEnvelope(t, conn, "connect", connectRequest{SessionID: sessionID, AuthToken: authToken})
	return mustReadConnectedEvent(t, conn)
}

func mustReadConnectedEvent(t *testing.T, conn *websocket.Conn) connectedEvent {
	t.Helper()

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

func mustReadLeftRoom(t *testing.T, conn *websocket.Conn) leftRoomEvent {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "left_room" {
		t.Fatalf("left_room response type = %q; want left_room", envelope.Type)
	}
	var event leftRoomEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(left_room) error = %v", err)
	}
	return event
}

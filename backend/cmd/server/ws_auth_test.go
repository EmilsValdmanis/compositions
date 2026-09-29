package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/gorilla/websocket"
)

type socketSessionStore struct {
	noopUserStore
	mu        sync.Mutex
	records   map[string]database.SessionUserRecord
	lookupErr error
}

func (s *socketSessionStore) GetSessionUserByToken(_ context.Context, token string, _ time.Time) (database.SessionUserRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lookupErr != nil {
		return database.SessionUserRecord{}, s.lookupErr
	}
	record, ok := s.records[token]
	if !ok {
		return database.SessionUserRecord{}, database.ErrSessionNotFound
	}
	return record, nil
}

func (s *socketSessionStore) DeleteSession(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, token)
	return nil
}

func requireAuthSocketClosed(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	for {
		var envelope wsEnvelope
		err := conn.ReadJSON(&envelope)
		if err == nil {
			continue
		}
		if !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
			t.Fatalf("socket close = %v; want authentication policy close", err)
		}
		return
	}
}

func TestLogoutClosesEverySocketForTokenAndPreservesOtherSessions(t *testing.T) {
	store := &socketSessionStore{records: map[string]database.SessionUserRecord{
		"first": {ID: "user-1", Name: "First"}, "other": {ID: "user-2", Name: "Other"},
	}}
	auth := &authHandler{store: store, now: time.Now}
	server := newWSServerWithAuth(auth)
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()
	first := mustDialWSWithCookie(t, httpServer.URL, "first")
	defer first.Close()
	mustConnectSession(t, first, "")
	second := mustDialWSWithCookie(t, httpServer.URL, "first")
	defer second.Close()
	mustConnectSession(t, second, "")
	other := mustDialWSWithCookie(t, httpServer.URL, "other")
	defer other.Close()
	mustConnectSession(t, other, "")
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "first"})
	response := httptest.NewRecorder()
	auth.handleLogout(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("logout = %d", response.Code)
	}
	requireAuthSocketClosed(t, first)
	requireAuthSocketClosed(t, second)
	mustSendEnvelope(t, other, "create_room", createRoomRequest{})
	mustReadRoomState(t, other)
	// Repeating logout for an already removed token is harmless.
	auth.handleLogout(httptest.NewRecorder(), request)
	t.Log("logout: both tabs closed with code 1008; unrelated login remains usable")
}

func TestRevokedSocketCannotRunCommand(t *testing.T) {
	store := &socketSessionStore{records: map[string]database.SessionUserRecord{"token": {ID: "user-1", Name: "First"}}}
	server := newWSServerWithAuth(&authHandler{store: store, now: time.Now})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()
	conn := mustDialWSWithCookie(t, httpServer.URL, "token")
	defer conn.Close()
	mustConnectSession(t, conn, "")
	_ = store.DeleteSession(context.Background(), "token")
	mustSendEnvelope(t, conn, "create_room", createRoomRequest{})
	requireAuthSocketClosed(t, conn)
	server.lobby.mu.Lock()
	defer server.lobby.mu.Unlock()
	if len(server.lobby.rooms) != 0 {
		t.Fatal("revoked socket created a room")
	}
	t.Log("revoked token: next command rejected, socket closed, zero rooms created")
}

func TestHeartbeatRevalidatesIdleAuthSockets(t *testing.T) {
	originalInterval := defaultWSPingInterval
	defaultWSPingInterval = 5 * time.Millisecond
	defer func() { defaultWSPingInterval = originalInterval }()
	for _, failure := range []string{"revoked", "expired", "identity changed", "store unavailable"} {
		t.Run(failure, func(t *testing.T) {
			store := &socketSessionStore{records: map[string]database.SessionUserRecord{"token": {ID: "user-1", Name: "First"}}}
			server := newWSServerWithAuth(&authHandler{store: store, now: time.Now})
			httpServer := httptest.NewServer(server.routes())
			defer httpServer.Close()
			conn := mustDialWSWithCookie(t, httpServer.URL, "token")
			defer conn.Close()
			mustConnectSession(t, conn, "")
			store.mu.Lock()
			switch failure {
			case "revoked":
				delete(store.records, "token")
			case "expired":
				store.records["token"] = database.SessionUserRecord{ID: "user-1", ExpiresAt: time.Now().Add(-time.Minute)}
			case "identity changed":
				store.records["token"] = database.SessionUserRecord{ID: "different-user"}
			case "store unavailable":
				store.lookupErr = errors.New("database unavailable")
			}
			store.mu.Unlock()
			requireAuthSocketClosed(t, conn)
			t.Logf("idle %s session closed without sending a command", failure)
		})
	}
}

func TestAuthenticationExpiryAndMissingRegistration(t *testing.T) {
	store := &socketSessionStore{records: map[string]database.SessionUserRecord{"token": {ID: "user-1", ExpiresAt: time.Now().Add(-time.Minute)}}}
	server := newWSServerWithAuth(&authHandler{store: store, now: time.Now})
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
	if _, err := server.registerAuthSocket(nil, request); !errors.Is(err, errAuthenticationRequired) {
		t.Fatalf("expired authentication = %v", err)
	}
	if !server.revalidateIdleAuthSocket(nil) {
		t.Fatal("heartbeat rejected an in-progress handshake")
	}
	if server.revalidateAuthSocket(nil) {
		t.Fatal("unregistered authenticated socket was accepted")
	}
	// Expiry captured at connect also applies if storage later extends the record.
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()
	store.mu.Lock()
	store.records["token"] = database.SessionUserRecord{ID: "user-1", Name: "First", ExpiresAt: time.Now().Add(time.Hour)}
	store.mu.Unlock()
	conn := mustDialWSWithCookie(t, httpServer.URL, "token")
	defer conn.Close()
	mustConnectSession(t, conn, "")
	server.authSocketsMu.Lock()
	for socket, session := range server.authSockets {
		session.expiresAt = time.Now().Add(-time.Minute)
		server.authSockets[socket] = session
	}
	server.authSocketsMu.Unlock()
	mustSendEnvelope(t, conn, "create_room", createRoomRequest{})
	requireAuthSocketClosed(t, conn)
}

type pausedSocketSessionStore struct {
	socketSessionStore
	started chan struct{}
	release chan struct{}
	deleted chan struct{}
}

func (s *pausedSocketSessionStore) GetSessionUserByToken(ctx context.Context, token string, now time.Time) (database.SessionUserRecord, error) {
	record, err := s.socketSessionStore.GetSessionUserByToken(ctx, token, now)
	close(s.started)
	<-s.release
	return record, err
}

func (s *pausedSocketSessionStore) DeleteSession(ctx context.Context, token string) error {
	err := s.socketSessionStore.DeleteSession(ctx, token)
	close(s.deleted)
	return err
}

func TestLogoutDuringSocketRegistrationCannotMissNewConnection(t *testing.T) {
	store := &pausedSocketSessionStore{
		socketSessionStore: socketSessionStore{records: map[string]database.SessionUserRecord{"token": {ID: "user-1", Name: "First"}}},
		started:            make(chan struct{}), release: make(chan struct{}), deleted: make(chan struct{}),
	}
	auth := &authHandler{store: store, now: time.Now}
	server := newWSServerWithAuth(auth)
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()
	conn := mustDialWSWithCookie(t, httpServer.URL, "token")
	defer conn.Close()
	mustSendEnvelope(t, conn, "connect", connectRequest{})
	<-store.started // Authentication read succeeded, but registration has not finished.
	logoutDone := make(chan struct{})
	go func() {
		request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "token"})
		auth.handleLogout(httptest.NewRecorder(), request)
		close(logoutDone)
	}()
	<-store.deleted
	close(store.release)
	<-logoutDone
	requireAuthSocketClosed(t, conn)
	deadline := time.Now().Add(time.Second)
	for {
		server.authSocketsMu.Lock()
		registered := len(server.authSockets)
		server.authSocketsMu.Unlock()
		if registered == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket registration was not cleaned up")
		}
		time.Sleep(time.Millisecond)
	}
	server.lobby.mu.Lock()
	for _, session := range server.lobby.sessions {
		if session.conn != nil {
			t.Error("closed connection left stale lobby presence")
		}
	}
	server.lobby.mu.Unlock()
	t.Log("logout raced with a successful auth read; newly registering socket still closed")
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type recordingUserStore struct {
	users []database.UserRecord
	err   error
}

func (s *recordingUserStore) UpsertUser(_ context.Context, user database.UserRecord) (database.UserRecord, error) {
	if s.err != nil {
		return database.UserRecord{}, s.err
	}
	s.users = append(s.users, user)
	return user, nil
}

func (s *recordingUserStore) Close() error { return nil }

func (s *recordingUserStore) CreateSession(context.Context, database.SessionRecord) error { return nil }

func (s *recordingUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (s *recordingUserStore) DeleteSession(context.Context, string) error { return nil }

func (s *recordingUserStore) UpdateOnboardingVersion(context.Context, string, int) error { return nil }

func (s *recordingUserStore) SaveLobbyState(context.Context, json.RawMessage) error { return nil }

func (s *recordingUserStore) LoadLobbyState(context.Context) (json.RawMessage, error) {
	return nil, database.ErrLobbyStateNotFound
}

func (s *recordingUserStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return report, nil
}

type recordingSessionStore struct {
	user authenticatedUser
}

func (s *recordingSessionStore) UpsertUser(_ context.Context, user database.UserRecord) (database.UserRecord, error) {
	return user, nil
}

func (s *recordingSessionStore) CreateSession(context.Context, database.SessionRecord) error {
	return nil
}

func (s *recordingSessionStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{
		ID:       s.user.ID,
		Name:     s.user.Name,
		Email:    s.user.Email,
		ImageURL: s.user.Image,
	}, nil
}

func (s *recordingSessionStore) DeleteSession(context.Context, string) error { return nil }

func (s *recordingSessionStore) UpdateOnboardingVersion(context.Context, string, int) error {
	return nil
}

func (s *recordingSessionStore) SaveLobbyState(context.Context, json.RawMessage) error {
	return nil
}

func (s *recordingSessionStore) LoadLobbyState(context.Context) (json.RawMessage, error) {
	return nil, database.ErrLobbyStateNotFound
}

func (s *recordingSessionStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return report, nil
}

func (s *recordingSessionStore) Close() error { return nil }

func TestHandleConnectPersistsAuthenticatedUser(t *testing.T) {
	store := &recordingUserStore{}
	server := newWSServerWithDependencies(&authHandler{store: store, now: time.Now}, store, "")

	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
	server.auth = &authHandler{store: &recordingSessionStore{user: authenticatedUser{ID: "user-1", Name: "Player One", Email: "player@example.com", Image: "https://cdn.example.com/player.png"}}, now: time.Now}

	nextSessionID, shouldClose := server.handleConnect(serverConn, request, wsEnvelope{
		Type: "connect",
		Data: mustMarshalRawMessage(connectRequest{}),
	})

	if shouldClose {
		t.Fatal("handleConnect() requested close; want open connection")
	}
	if nextSessionID == "" {
		t.Fatal("handleConnect() sessionID = empty; want session ID")
	}
	if len(store.users) != 1 {
		t.Fatalf("persisted users = %d; want 1", len(store.users))
	}
	if store.users[0].ID != "user-1" {
		t.Fatalf("persisted user id = %q; want user-1", store.users[0].ID)
	}

	connected := mustReadConnectedEvent(t, clientConn)
	if connected.SessionID != nextSessionID {
		t.Fatalf("connected sessionID = %q; want %q", connected.SessionID, nextSessionID)
	}
}

func TestHandleConnectClosesWhenUserPersistenceFails(t *testing.T) {
	store := &recordingUserStore{err: errors.New("db unavailable")}
	server := newWSServerWithDependencies(&authHandler{store: &recordingSessionStore{user: authenticatedUser{ID: "user-1", Name: "Player One", Email: "player@example.com"}}, now: time.Now}, store, "")

	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})

	nextSessionID, shouldClose := server.handleConnect(serverConn, request, wsEnvelope{
		Type: "connect",
		Data: mustMarshalRawMessage(connectRequest{}),
	})

	if !shouldClose {
		t.Fatal("handleConnect() shouldClose = false; want true")
	}
	if nextSessionID != "" {
		t.Fatalf("handleConnect() sessionID = %q; want empty", nextSessionID)
	}

	mustReadError(t, clientConn, "internal server error")
}

func newWSServer() *wsServer {
	return newWSServerWithAuth(nil)
}

func newWSServerWithAuth(auth *authHandler) *wsServer {
	return newWSServerWithAllowedOrigin(auth, "")
}

func newWSServerWithAllowedOrigin(auth *authHandler, allowedOrigin string) *wsServer {
	return newWSServerWithDependencies(auth, noopUserStore{}, allowedOrigin)
}

func newLobbyServer() *lobbyServer {
	return newLobbyServerWithStore(noopUserStore{})
}

type noopUserStore struct{}

func (noopUserStore) UpsertUser(_ context.Context, user database.UserRecord) (database.UserRecord, error) {
	return user, nil
}

func (noopUserStore) CreateSession(context.Context, database.SessionRecord) error { return nil }

func (noopUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (noopUserStore) DeleteSession(context.Context, string) error { return database.ErrSessionNotFound }

func (noopUserStore) UpdateOnboardingVersion(context.Context, string, int) error { return nil }

func (noopUserStore) SaveLobbyState(context.Context, json.RawMessage) error { return nil }

func (noopUserStore) LoadLobbyState(context.Context) (json.RawMessage, error) {
	return nil, database.ErrLobbyStateNotFound
}

func (noopUserStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return report, nil
}

func (noopUserStore) Close() error { return nil }

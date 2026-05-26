package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type recordingUserStore struct {
	users []authenticatedUser
	err   error
}

func (s *recordingUserStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	if s.err != nil {
		return authenticatedUser{}, s.err
	}
	s.users = append(s.users, user)
	return user, nil
}

func (s *recordingUserStore) Close() error { return nil }

func (s *recordingUserStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (s *recordingUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (s *recordingUserStore) DeleteSession(context.Context, string) error { return nil }

type recordingSessionStore struct {
	user authenticatedUser
}

func (s *recordingSessionStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	return user, nil
}

func (s *recordingSessionStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (s *recordingSessionStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{
		ID:       s.user.ID,
		Name:     s.user.Name,
		Email:    s.user.Email,
		ImageURL: s.user.Image,
	}, nil
}

func (s *recordingSessionStore) DeleteSession(context.Context, string) error { return nil }

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

	mustReadError(t, clientConn, "save user: db unavailable")
}

func TestUserStoreSessionHelpers(t *testing.T) {
	t.Run("noop user store methods", func(t *testing.T) {
		store := noopUserStore{}
		if err := store.CreateSession(context.Background(), authSessionRecord{}); err != nil {
			t.Fatalf("noop CreateSession() error = %v", err)
		}
		if _, err := store.GetSessionUserByToken(context.Background(), "token", time.Now()); !errors.Is(err, database.ErrSessionNotFound) {
			t.Fatalf("noop GetSessionUserByToken() error = %v; want ErrSessionNotFound", err)
		}
		if err := store.DeleteSession(context.Background(), "token"); !errors.Is(err, database.ErrSessionNotFound) {
			t.Fatalf("noop DeleteSession() error = %v; want ErrSessionNotFound", err)
		}
	})

	t.Run("postgres session wrappers", func(t *testing.T) {
		var nilStore *postgresUserStore
		if err := nilStore.CreateSession(context.Background(), authSessionRecord{}); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("CreateSession(nil) error = %v; want user store is not configured", err)
		}
		if _, err := nilStore.GetSessionUserByToken(context.Background(), "token", time.Now()); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("GetSessionUserByToken(nil) error = %v; want user store is not configured", err)
		}
		if err := nilStore.DeleteSession(context.Background(), "token"); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("DeleteSession(nil) error = %v; want user store is not configured", err)
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		if err := store.CreateSession(context.Background(), authSessionRecord{Token: " token ", UserID: " user-1 ", ExpiresAt: time.Now()}); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("CreateSession(store) error = %v; want user store is not configured", err)
		}
		if _, err := store.GetSessionUserByToken(context.Background(), " token ", time.Now()); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("GetSessionUserByToken(store) error = %v; want user store is not configured", err)
		}
		if err := store.DeleteSession(context.Background(), " token "); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("DeleteSession(store) error = %v; want user store is not configured", err)
		}
	})
}

package main

import (
	"context"
	"errors"
	"testing"
)

type recordingUserStore struct {
	users []authenticatedUser
	err   error
}

func (s *recordingUserStore) UpsertUser(_ context.Context, user authenticatedUser) error {
	if s.err != nil {
		return s.err
	}
	s.users = append(s.users, user)
	return nil
}

func (s *recordingUserStore) Close() error { return nil }

func TestHandleConnectPersistsAuthenticatedUser(t *testing.T) {
	store := &recordingUserStore{}
	server := newWSServerWithDependencies(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"player-token": {ID: "user-1", Name: "Player One", Email: "player@example.com", Image: "https://cdn.example.com/player.png"},
	}}, store, "")

	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	nextSessionID, shouldClose := server.handleConnect(serverConn, wsEnvelope{
		Type: "connect",
		Data: mustMarshalRawMessage(connectRequest{AuthToken: "player-token"}),
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
	server := newWSServerWithDependencies(staticSessionVerifier{usersByToken: map[string]authenticatedUser{
		"player-token": {ID: "user-1", Name: "Player One", Email: "player@example.com"},
	}}, &recordingUserStore{err: errors.New("db unavailable")}, "")

	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()

	nextSessionID, shouldClose := server.handleConnect(serverConn, wsEnvelope{
		Type: "connect",
		Data: mustMarshalRawMessage(connectRequest{AuthToken: "player-token"}),
	})

	if !shouldClose {
		t.Fatal("handleConnect() shouldClose = false; want true")
	}
	if nextSessionID != "" {
		t.Fatalf("handleConnect() sessionID = %q; want empty", nextSessionID)
	}

	mustReadError(t, clientConn, "save user: db unavailable")
}

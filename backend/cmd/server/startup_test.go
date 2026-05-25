package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type closingUserStore struct {
	err    error
	closed bool
}

func (s *closingUserStore) UpsertUser(_ context.Context, _ authenticatedUser) error {
	return nil
}

func (s *closingUserStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (s *closingUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (s *closingUserStore) DeleteSession(context.Context, string) error { return nil }

func (s *closingUserStore) Close() error {
	s.closed = true
	return s.err
}

func TestNewWSServerWithDependenciesUsesNoopStoreWhenNil(t *testing.T) {
	server := newWSServerWithDependencies(nil, nil, "http://frontend.test/")
	if server.userStore == nil {
		t.Fatal("server.userStore = nil; want noop user store")
	}
	if server.allowedOrigin != "http://frontend.test" {
		t.Fatalf("server.allowedOrigin = %q; want http://frontend.test", server.allowedOrigin)
	}
}

func TestNewConfiguredWSServerPropagatesStoreError(t *testing.T) {
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()

	t.Setenv("BASE_URL", "http://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	openConfiguredUserStore = func() (userStore, error) {
		return nil, errors.New("open store boom")
	}

	server, err := newConfiguredWSServer()
	if err == nil || err.Error() != "open store boom" {
		t.Fatalf("newConfiguredWSServer() error = %v; want open store boom", err)
	}
	if server != nil {
		t.Fatalf("server = %#v; want nil", server)
	}
}

func TestWSServerClose(t *testing.T) {
	var nilServer *wsServer
	if err := nilServer.Close(); err != nil {
		t.Fatalf("nilServer.Close() error = %v", err)
	}

	server := &wsServer{}
	if err := server.Close(); err != nil {
		t.Fatalf("server.Close() error = %v", err)
	}

	store := &closingUserStore{err: errors.New("close boom")}
	server = &wsServer{userStore: store}
	if err := server.Close(); err == nil || err.Error() != "close boom" {
		t.Fatalf("server.Close() error = %v; want close boom", err)
	}
	if !store.closed {
		t.Fatal("store.closed = false; want true")
	}
}

func TestRunServerWarnsWhenCloseFails(t *testing.T) {
	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	t.Setenv("BASE_URL", "http://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	store := &closingUserStore{err: errors.New("close boom")}
	openConfiguredUserStore = func() (userStore, error) { return store, nil }

	var output bytes.Buffer
	configureLogger(&output)

	listenAndServe = func(addr string, handler http.Handler) error {
		if addr != ":0" {
			t.Fatalf("listenAndServe addr = %q; want :0", addr)
		}
		return errors.New("stop")
	}

	err := runServer(":0")
	if err == nil || err.Error() != "stop" {
		t.Fatalf("runServer() error = %v; want stop", err)
	}
	if !store.closed {
		t.Fatal("store.closed = false; want true")
	}
	if got := output.String(); !strings.Contains(got, "close user store failed") {
		t.Fatalf("logger output = %q; want close user store failed", got)
	}
}

func TestNewConfiguredWSServerClosesStoreOnAuthError(t *testing.T) {
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()

	t.Setenv("BASE_URL", "http://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	store := &closingUserStore{}
	openConfiguredUserStore = func() (userStore, error) {
		return store, nil
	}

	server, err := newConfiguredWSServer()
	if err == nil || err.Error() != "GOOGLE_CLIENT_ID is required" {
		t.Fatalf("newConfiguredWSServer() error = %v; want GOOGLE_CLIENT_ID is required", err)
	}
	if server != nil {
		t.Fatalf("server = %#v; want nil", server)
	}
	if !store.closed {
		t.Fatal("store.closed = false; want true")
	}
}

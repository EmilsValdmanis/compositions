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
	err     error
	loadErr error
	closed  bool
}

type cleaningUserStore struct {
	closingUserStore
	cleanupCount int64
	cleanupErr   error
	called       chan struct{}
}

func (s *cleaningUserStore) DeleteExpiredSessions(context.Context, time.Time) (int64, error) {
	if s.called != nil {
		select {
		case s.called <- struct{}{}:
		default:
		}
	}
	return s.cleanupCount, s.cleanupErr
}

func TestSuperviseHTTPServerShutdownBranches(t *testing.T) {
	t.Run("serve returns", func(t *testing.T) {
		want := errors.New("serve failed")
		if err := superviseHTTPServer(context.Background(), func() error { return want }, func(context.Context) error { return nil }); !errors.Is(err, want) {
			t.Fatalf("superviseHTTPServer() error = %v", err)
		}
	})
	t.Run("graceful shutdown", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := superviseHTTPServer(ctx, func() error { return http.ErrServerClosed }, func(context.Context) error { return nil }); err != nil {
			t.Fatalf("superviseHTTPServer() error = %v", err)
		}
	})
	t.Run("shutdown error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		want := errors.New("shutdown failed")
		if err := superviseHTTPServer(ctx, func() error { return http.ErrServerClosed }, func(context.Context) error { return want }); !errors.Is(err, want) {
			t.Fatalf("superviseHTTPServer() error = %v", err)
		}
	})
	t.Run("serve error after shutdown", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		want := errors.New("late serve failure")
		if err := superviseHTTPServer(ctx, func() error { return want }, func(context.Context) error { return nil }); !errors.Is(err, want) {
			t.Fatalf("superviseHTTPServer() error = %v", err)
		}
	})
}

func TestSessionCleanupLifecycle(t *testing.T) {
	server := &wsServer{userStore: noopUserStore{}}
	if count, err := server.cleanupExpiredSessions(context.Background(), time.Now()); err != nil || count != 0 {
		t.Fatalf("cleanupExpiredSessions(no cleaner) = %d, %v", count, err)
	}
	stop := server.startMaintenance(context.Background())
	stop()

	store := &cleaningUserStore{cleanupCount: 2, called: make(chan struct{}, 1)}
	server.userStore = store
	if count, err := server.cleanupExpiredSessions(context.Background(), time.Now()); err != nil || count != 2 {
		t.Fatalf("cleanupExpiredSessions() = %d, %v", count, err)
	}
	select {
	case <-store.called:
	default:
	}
	originalInterval := sessionCleanupInterval
	sessionCleanupInterval = time.Millisecond
	defer func() { sessionCleanupInterval = originalInterval }()
	stop = server.startMaintenance(context.Background())
	select {
	case <-store.called:
	case <-time.After(time.Second):
		t.Fatal("maintenance cleanup did not run")
	}
	stop()

	store = &cleaningUserStore{cleanupErr: errors.New("cleanup failed"), called: make(chan struct{}, 1)}
	server.userStore = store
	stop = server.startMaintenance(context.Background())
	select {
	case <-store.called:
	case <-time.After(time.Second):
		t.Fatal("failing maintenance cleanup did not run")
	}
	stop()
}

func TestNewConfiguredWSServerClosesStoreOnCleanupError(t *testing.T) {
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()
	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "https://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	store := &cleaningUserStore{cleanupErr: errors.New("cleanup boom")}
	openConfiguredUserStore = func() (userStore, error) { return store, nil }

	server, err := newConfiguredWSServer()
	if err == nil || err.Error() != "clean expired sessions: cleanup boom" {
		t.Fatalf("newConfiguredWSServer() = %#v, %v; want cleanup error", server, err)
	}
	if !store.closed {
		t.Fatal("store was not closed")
	}
}

func (s *closingUserStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	return user, nil
}

func (s *closingUserStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (s *closingUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (s *closingUserStore) DeleteSession(context.Context, string) error { return nil }

func (s *closingUserStore) UpdateOnboardingVersion(context.Context, string, int) error { return nil }

func (s *closingUserStore) SaveLobbyState(context.Context, persistedLobbyState) error { return nil }

func (s *closingUserStore) LoadLobbyState(context.Context) (persistedLobbyState, error) {
	if s.loadErr != nil {
		return persistedLobbyState{}, s.loadErr
	}
	return persistedLobbyState{}, nil
}

func (s *closingUserStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return report, nil
}

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

	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "https://frontend.test")
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

	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "https://frontend.test")
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

	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "https://frontend.test")
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

func TestNewConfiguredWSServerClosesStoreOnRestoreError(t *testing.T) {
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()

	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "https://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	store := &closingUserStore{loadErr: errors.New("restore boom")}
	openConfiguredUserStore = func() (userStore, error) {
		return store, nil
	}

	server, err := newConfiguredWSServer()
	if err == nil || err.Error() != "restore lobby state: restore boom" {
		t.Fatalf("newConfiguredWSServer() error = %v; want restore lobby state: restore boom", err)
	}
	if server != nil {
		t.Fatalf("server = %#v; want nil", server)
	}
	if !store.closed {
		t.Fatal("store.closed = false; want true")
	}
}

func TestNewConfiguredWSServerUsesFrontendOrigin(t *testing.T) {
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()

	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("FRONTEND_URL", "https://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	store := &closingUserStore{}
	openConfiguredUserStore = func() (userStore, error) {
		return store, nil
	}

	server, err := newConfiguredWSServer()
	if err != nil {
		t.Fatalf("newConfiguredWSServer() error = %v", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("server.Close() error = %v", err)
		}
	}()

	if server.allowedOrigin != "https://frontend.test" {
		t.Fatalf("server.allowedOrigin = %q; want https://frontend.test", server.allowedOrigin)
	}
}

func TestDefaultListenAndServeReturnsInvalidAddressError(t *testing.T) {
	err := listenAndServe("invalid-address\x00", http.NewServeMux())
	if err == nil {
		t.Fatal("listenAndServe(invalid address) error = nil; want error")
	}
}

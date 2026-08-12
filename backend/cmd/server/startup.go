package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
)

var sessionCleanupInterval = time.Hour

var listenAndServe = func(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr: addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return superviseHTTPServer(ctx, server.ListenAndServe, server.Shutdown)
}

func superviseHTTPServer(ctx context.Context, serve func() error, shutdown func(context.Context) error) error {
	errCh := make(chan error, 1)
	go func() { errCh <- serve() }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newConfiguredWSServer() (*wsServer, error) {
	if _, err := baseURLFromEnv(); err != nil {
		return nil, err
	}
	_, frontendOrigin, err := frontendConfigFromEnv()
	if err != nil {
		return nil, err
	}
	store, err := openConfiguredUserStore()
	if err != nil {
		return nil, err
	}
	auth, err := newConfiguredAuthHandler(store)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	server := newWSServerWithDependencies(auth, store, frontendOrigin)
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	if err := server.lobby.restorePersistedState(ctx); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("restore lobby state: %w", err)
	}
	if _, err := server.cleanupExpiredSessions(ctx, time.Now().UTC()); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("clean expired sessions: %w", err)
	}
	return server, nil
}

type expiredSessionCleaner interface {
	DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error)
}

func (s *wsServer) cleanupExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	cleaner, ok := s.userStore.(expiredSessionCleaner)
	if !ok {
		return 0, nil
	}
	return cleaner.DeleteExpiredSessions(ctx, before)
}

func (s *wsServer) startMaintenance(parent context.Context) context.CancelFunc {
	ctx, cancel := context.WithCancel(parent)
	if _, ok := s.userStore.(expiredSessionCleaner); !ok {
		return cancel
	}
	interval := sessionCleanupInterval
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				cleanupCtx, cleanupCancel := context.WithTimeout(ctx, defaultUserStoreTimeout)
				if count, err := s.cleanupExpiredSessions(cleanupCtx, now.UTC()); err != nil {
					slog.Warn("clean expired sessions failed", "error", err)
				} else if count > 0 {
					slog.Info("expired sessions cleaned", "count", count)
				}
				cleanupCancel()
			case <-ctx.Done():
				return
			}
		}
	}()
	return cancel
}

func runServer(addr string) error {
	server, err := newConfiguredWSServer()
	if err != nil {
		return err
	}
	defer func() {
		if err := server.Close(); err != nil {
			slog.Warn("close user store failed", "error", err)
		}
	}()
	stopMaintenance := server.startMaintenance(context.Background())
	defer stopMaintenance()
	slog.Info("server starting", "addr", addr, "allowedOrigin", server.allowedOrigin)
	handler := server.routes()
	if sentryEnabled() {
		handler = sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle(handler)
	}
	return listenAndServe(addr, handler)
}

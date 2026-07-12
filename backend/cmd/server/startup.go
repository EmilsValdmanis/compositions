package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
)

var listenAndServe = func(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr: addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
	}
	return server.ListenAndServe()
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
	return server, nil
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
	slog.Info("server starting", "addr", addr, "allowedOrigin", server.allowedOrigin)
	handler := server.routes()
	if sentryEnabled() {
		handler = sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle(handler)
	}
	return listenAndServe(addr, handler)
}

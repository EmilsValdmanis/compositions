package main

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

var listenAndServe = func(addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return server.ListenAndServe()
}
var fatalOnRunError = log.Fatal
var sentryInit = sentry.Init
var sentryFlush = sentry.Flush

const sentryFlushTimeout = 2 * time.Second

func main() {
	if err := configureObservability(os.Stdout); err != nil {
		reportFatal(err)
		return
	}
	defer sentryFlush(sentryFlushTimeout)

	if err := runServer(":8080"); err != nil {
		reportFatal(err)
	}
}

func configureObservability(output io.Writer) error {
	if sentryEnabled() {
		if err := sentryInit(newSentryClientOptions()); err != nil {
			return err
		}
	}

	configureLogger(output)
	return nil
}

func newSentryClientOptions() sentry.ClientOptions {
	return sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		Environment:      os.Getenv("SENTRY_ENVIRONMENT"),
		SendDefaultPII:   true,
		AttachStacktrace: true,
		EnableTracing:    true,
		TracesSampler:    sentry.TracesSampler(traceSampleRate),
		EnableLogs:       true,
	}
}

func sentryEnabled() bool {
	return strings.EqualFold(os.Getenv("SENTRY_ENVIRONMENT"), "production") && strings.TrimSpace(os.Getenv("SENTRY_DSN")) != ""
}

func traceSampleRate(ctx sentry.SamplingContext) float64 {
	if ctx.Span != nil && strings.HasSuffix(ctx.Span.Name, "/health") {
		return 0
	}

	return 0.2
}

func configureLogger(output io.Writer) {
	handlers := []slog.Handler{slog.NewTextHandler(output, nil)}
	if sentry.CurrentHub().Client() != nil {
		handlers = append(handlers, sentryslog.Option{
			LogLevel:  []slog.Level{slog.LevelInfo, slog.LevelWarn, slog.LevelError},
			AddSource: true,
		}.NewSentryHandler(context.Background()))
	}

	slog.SetDefault(slog.New(multiHandler{handlers: handlers}))
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

func (s *wsServer) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if handleCORSPreflight(w, r) {
		return
	}
	if s == nil || s.auth == nil {
		http.Error(w, "auth is not configured", http.StatusInternalServerError)
		return
	}

	switch r.URL.Path {
	case "/auth/google":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.auth.handleGoogleSignIn(w, r)
	case "/auth/google/callback":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.auth.handleGoogleCallback(w, r)
	case "/auth/session":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.auth.handleSession(w, r)
	case "/auth/logout":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.auth.handleLogout(w, r)
	default:
		http.NotFound(w, r)
	}
}

func reportFatal(err error) {
	if err == nil {
		return
	}

	sentry.CaptureException(err)
	sentryFlush(sentryFlushTimeout)
	fatalOnRunError(err)
}

type multiHandler struct {
	handlers []slog.Handler
}

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (h multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var handleErr error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil && handleErr == nil {
			handleErr = err
		}
	}

	return handleErr
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithAttrs(attrs))
	}

	return multiHandler{handlers: next}
}

func (h multiHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, 0, len(h.handlers))
	for _, handler := range h.handlers {
		next = append(next, handler.WithGroup(name))
	}

	return multiHandler{handlers: next}
}

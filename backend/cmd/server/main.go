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
	sentryslog "github.com/getsentry/sentry-go/slog"
)

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
	off := &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff}
	return sentry.ClientOptions{
		Dsn:         os.Getenv("SENTRY_DSN"),
		Environment: os.Getenv("SENTRY_ENVIRONMENT"),
		DataCollection: &sentry.DataCollection{
			UserInfo:    sentry.Set(false),
			Cookies:     off,
			HTTPHeaders: &sentry.HeaderCollectionConfig{Request: off, Response: off},
			HTTPBodies:  []sentry.BodyType{},
			QueryParams: off,
		},
		AttachStacktrace: true,
		EnableTracing:    true,
		TracesSampler:    sentry.TracesSampler(traceSampleRate),
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
			LogLevel:  []slog.Level{slog.LevelWarn, slog.LevelError},
			AddSource: true,
		}.NewSentryHandler(context.Background()))
	}

	slog.SetDefault(slog.New(multiHandler{handlers: handlers}))
}

func (s *wsServer) handleSessionRoutes(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if handleCORSPreflight(w, r) {
		return
	}
	if s == nil || s.auth == nil {
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "auth is not configured")
		return
	}
	if r.Method == http.MethodPost && !sameOriginRequest(r, s.auth.config.frontendOrigin) {
		writeHTTPError(w, http.StatusForbidden, "invalid_origin", "request origin is not allowed")
		return
	}

	switch r.URL.Path {
	case "/auth/google":
		if r.Method != http.MethodGet {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.auth.handleGoogleSignIn(w, r)
	case "/auth/google/callback":
		if r.Method != http.MethodGet {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.auth.handleGoogleCallback(w, r)
	case "/auth/session":
		if r.Method != http.MethodGet {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.auth.handleSession(w, r)
	case "/auth/onboarding/complete":
		if r.Method != http.MethodPost {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.auth.handleCompleteOnboarding(w, r)
	case "/auth/logout":
		if r.Method != http.MethodPost {
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.auth.handleLogout(w, r)
	default:
		writeHTTPError(w, http.StatusNotFound, "not_found", "not found")
	}
}

func sameOriginRequest(r *http.Request, allowedOrigin string) bool {
	if r == nil {
		return false
	}
	origin := normalizeOrigin(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	return allowedOrigin != "" && origin == normalizeOrigin(allowedOrigin)
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

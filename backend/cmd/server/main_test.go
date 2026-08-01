package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
)

func TestRunServerAndMain(t *testing.T) {
	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()
	originalInit := sentryInit
	defer func() { sentryInit = originalInit }()
	originalFlush := sentryFlush
	defer func() { sentryFlush = originalFlush }()
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)
	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("DATABASE_URL", "postgres://unused")
	t.Setenv("SENTRY_ENVIRONMENT", "development")
	sentryFlush = func(timeout time.Duration) bool { return true }
	sentryInit = func(options sentry.ClientOptions) error { return nil }
	openConfiguredUserStore = func() (userStore, error) { return noopUserStore{}, nil }

	calledAddrs := make([]string, 0, 2)
	listenAndServe = func(addr string, handler http.Handler) error {
		calledAddrs = append(calledAddrs, addr)
		if addr != ":0" && addr != ":8080" {
			t.Fatalf("listenAndServe addr = %q; want :0 or :8080", addr)
		}
		if handler == nil {
			t.Fatal("listenAndServe handler = nil; want handler")
		}
		return errors.New("stop")
	}

	if err := runServer(":0"); err == nil || err.Error() != "stop" {
		t.Fatalf("runServer() error = %v; want stop", err)
	}
	if len(calledAddrs) == 0 || calledAddrs[0] != ":0" {
		t.Fatal("runServer() did not call listenAndServe")
	}

	fatalCalled := false
	fatalOnRunError = func(v ...any) {
		fatalCalled = true
		if len(v) != 1 {
			t.Fatalf("fatalOnRunError args = %v; want single arg", v)
		}
		err, ok := v[0].(error)
		if !ok || err == nil || err.Error() != "stop" {
			t.Fatalf("fatalOnRunError arg = %v; want stop error", v[0])
		}
	}

	main()
	if !fatalCalled {
		t.Fatal("main() did not call fatalOnRunError")
	}
	if len(calledAddrs) != 2 || calledAddrs[1] != ":8080" {
		t.Fatalf("calledAddrs = %v; want [:0 :8080]", calledAddrs)
	}
}

func TestWriteHTTPErrorIncludesCodeAndMessage(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeHTTPError(recorder, http.StatusBadRequest, "invalid_data", "invalid data")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q; want application/json", got)
	}

	var response httpErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != "invalid_data" || response.Message != "invalid data" {
		t.Fatalf("response = %#v; want code and message", response)
	}
}

func TestConfigureLoggerWritesToProvidedOutput(t *testing.T) {
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	var output bytes.Buffer
	configureLogger(&output)

	slog.Info("logger configured", "stream", "stdout")

	logged := output.String()
	if !strings.Contains(logged, "level=INFO") {
		t.Fatalf("configured logger output = %q; want INFO level entry", logged)
	}
	if !strings.Contains(logged, "msg=\"logger configured\"") {
		t.Fatalf("configured logger output = %q; want message entry", logged)
	}
	if !strings.Contains(logged, "stream=stdout") {
		t.Fatalf("configured logger output = %q; want custom field", logged)
	}
}

func TestConfigureLoggerWithSentryClient(t *testing.T) {
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)
	originalInit := sentryInit
	defer func() { sentryInit = originalInit }()
	t.Setenv("SENTRY_ENVIRONMENT", "production")
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")

	sentryInit = sentry.Init
	if err := sentryInit(newSentryClientOptions()); err != nil {
		t.Fatalf("sentryInit() error = %v", err)
	}

	var output bytes.Buffer
	configureLogger(&output)

	logger := slog.Default()
	multi, ok := logger.Handler().(multiHandler)
	if !ok {
		t.Fatalf("logger handler type = %T; want multiHandler", logger.Handler())
	}
	if len(multi.handlers) != 2 {
		t.Fatalf("len(multi.handlers) = %d; want 2", len(multi.handlers))
	}
}

func TestRunServerReturnsEnvErrorBeforeListen(t *testing.T) {
	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	t.Setenv("BASE_URL", "")
	t.Setenv("FRONTEND_URL", "http://frontend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")

	listenCalled := false
	listenAndServe = func(addr string, handler http.Handler) error {
		listenCalled = true
		return nil
	}

	err := runServer(":0")
	if err == nil || err.Error() != "BASE_URL is required" {
		t.Fatalf("runServer() error = %v; want BASE_URL is required", err)
	}
	if listenCalled {
		t.Fatal("listenAndServe() was called; want startup to fail before listen")
	}
}

func TestTraceSampleRate(t *testing.T) {
	if got := traceSampleRate(sentry.SamplingContext{}); got != 0.2 {
		t.Fatalf("traceSampleRate() = %v; want 0.2", got)
	}

	ctx := sentry.SamplingContext{Span: &sentry.Span{Name: "GET /health"}}
	if got := traceSampleRate(ctx); got != 0 {
		t.Fatalf("traceSampleRate() = %v; want 0 for health checks", got)
	}
}

func TestNewSentryClientOptions(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	t.Setenv("SENTRY_ENVIRONMENT", "production")

	options := newSentryClientOptions()
	if options.Dsn != "https://examplePublicKey@o0.ingest.sentry.io/0" {
		t.Fatalf("newSentryClientOptions().Dsn = %q; want configured DSN", options.Dsn)
	}
	if options.Environment != "production" {
		t.Fatalf("newSentryClientOptions().Environment = %q; want production", options.Environment)
	}
	if options.Release != "" {
		t.Fatalf("newSentryClientOptions().Release = %q; want empty", options.Release)
	}
	if !options.EnableTracing {
		t.Fatal("newSentryClientOptions().EnableTracing = false; want true")
	}
	if options.DisableLogs {
		t.Fatal("newSentryClientOptions().DisableLogs = true; want false")
	}
}

func TestSentryEnabled(t *testing.T) {
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	t.Setenv("SENTRY_ENVIRONMENT", "production")
	if !sentryEnabled() {
		t.Fatal("sentryEnabled() = false; want true in production with DSN")
	}

	t.Setenv("SENTRY_ENVIRONMENT", "development")
	if sentryEnabled() {
		t.Fatal("sentryEnabled() = true; want false in development")
	}

	t.Setenv("SENTRY_ENVIRONMENT", "production")
	t.Setenv("SENTRY_DSN", "   ")
	if sentryEnabled() {
		t.Fatal("sentryEnabled() = true; want false without DSN")
	}
}

func TestConfigureObservabilityReturnsInitError(t *testing.T) {
	originalInit := sentryInit
	defer func() { sentryInit = originalInit }()
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	t.Setenv("SENTRY_ENVIRONMENT", "production")
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	sentryInit = func(options sentry.ClientOptions) error {
		return errors.New("init boom")
	}

	if err := configureObservability(&bytes.Buffer{}); err == nil || err.Error() != "init boom" {
		t.Fatalf("configureObservability() error = %v; want init boom", err)
	}
}

func TestMainReportsObservabilityInitError(t *testing.T) {
	originalInit := sentryInit
	defer func() { sentryInit = originalInit }()
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	originalFlush := sentryFlush
	defer func() { sentryFlush = originalFlush }()
	t.Setenv("SENTRY_ENVIRONMENT", "production")
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")

	sentryInit = func(options sentry.ClientOptions) error { return errors.New("init boom") }
	sentryFlush = func(timeout time.Duration) bool { return true }

	fatalCalled := false
	fatalOnRunError = func(v ...any) {
		fatalCalled = true
		if len(v) != 1 {
			t.Fatalf("fatalOnRunError args = %v; want single arg", v)
		}
		err, ok := v[0].(error)
		if !ok || err == nil || err.Error() != "init boom" {
			t.Fatalf("fatalOnRunError arg = %v; want init boom", v[0])
		}
	}

	main()

	if !fatalCalled {
		t.Fatal("main() did not call fatalOnRunError for observability init error")
	}
}

func TestReportFatalIgnoresNil(t *testing.T) {
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	originalFlush := sentryFlush
	defer func() { sentryFlush = originalFlush }()

	flushCalled := false
	sentryFlush = func(timeout time.Duration) bool {
		flushCalled = true
		return true
	}

	fatalCalled := false
	fatalOnRunError = func(v ...any) {
		fatalCalled = true
	}

	reportFatal(nil)

	if flushCalled {
		t.Fatal("reportFatal(nil) flushed Sentry; want no flush")
	}
	if fatalCalled {
		t.Fatal("reportFatal(nil) called fatalOnRunError; want no fatal")
	}
}

type stubHandler struct {
	enabled bool
	err     error
	attrs   []slog.Attr
	groups  []string
	records []slog.Record
}

func (h *stubHandler) Enabled(context.Context, slog.Level) bool { return h.enabled }

func (h *stubHandler) Handle(_ context.Context, record slog.Record) error {
	h.records = append(h.records, record)
	return h.err
}

func (h *stubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *stubHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func TestMultiHandler(t *testing.T) {
	disabled := &stubHandler{}
	enabled := &stubHandler{enabled: true}
	combined := multiHandler{handlers: []slog.Handler{disabled, enabled}}

	if !combined.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Enabled() = false; want true when one handler is enabled")
	}

	onlyDisabled := multiHandler{handlers: []slog.Handler{disabled}}
	if onlyDisabled.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Enabled() = true; want false when no handlers are enabled")
	}

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "message", 0)
	errHandler := &stubHandler{enabled: true, err: errors.New("handle boom")}
	okHandler := &stubHandler{enabled: true}
	handling := multiHandler{handlers: []slog.Handler{disabled, errHandler, okHandler}}
	if err := handling.Handle(context.Background(), record); err == nil || err.Error() != "handle boom" {
		t.Fatalf("Handle() error = %v; want handle boom", err)
	}
	if len(disabled.records) != 0 {
		t.Fatalf("disabled handler saw %d records; want 0", len(disabled.records))
	}
	if len(errHandler.records) != 1 {
		t.Fatalf("errHandler saw %d records; want 1", len(errHandler.records))
	}
	if len(okHandler.records) != 1 {
		t.Fatalf("okHandler saw %d records; want 1", len(okHandler.records))
	}

	withAttrs := handling.WithAttrs([]slog.Attr{slog.String("key", "value")})
	withAttrsCombined, ok := withAttrs.(multiHandler)
	if !ok {
		t.Fatalf("WithAttrs() type = %T; want multiHandler", withAttrs)
	}
	for i, handler := range withAttrsCombined.handlers {
		stub, ok := handler.(*stubHandler)
		if !ok {
			t.Fatalf("WithAttrs() handler %d type = %T; want *stubHandler", i, handler)
		}
		if len(stub.attrs) != 1 {
			t.Fatalf("WithAttrs() handler %d attrs = %d; want 1", i, len(stub.attrs))
		}
	}

	withGroup := handling.WithGroup("group")
	withGroupCombined, ok := withGroup.(multiHandler)
	if !ok {
		t.Fatalf("WithGroup() type = %T; want multiHandler", withGroup)
	}
	for i, handler := range withGroupCombined.handlers {
		stub, ok := handler.(*stubHandler)
		if !ok {
			t.Fatalf("WithGroup() handler %d type = %T; want *stubHandler", i, handler)
		}
		if len(stub.groups) != 1 || stub.groups[0] != "group" {
			t.Fatalf("WithGroup() handler %d groups = %v; want [group]", i, stub.groups)
		}
	}
}

func TestRunServerWrapsHandlerWhenSentryEnabled(t *testing.T) {
	originalListen := listenAndServe
	defer func() { listenAndServe = originalListen }()
	originalOpenConfiguredUserStore := openConfiguredUserStore
	defer func() { openConfiguredUserStore = originalOpenConfiguredUserStore }()
	t.Setenv("BASE_URL", "https://backend.test")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "client-secret")
	t.Setenv("DATABASE_URL", "postgres://unused")
	t.Setenv("SENTRY_ENVIRONMENT", "production")
	t.Setenv("SENTRY_DSN", "https://examplePublicKey@o0.ingest.sentry.io/0")
	openConfiguredUserStore = func() (userStore, error) { return noopUserStore{}, nil }

	listenAndServe = func(addr string, handler http.Handler) error {
		if addr != ":0" {
			t.Fatalf("listenAndServe addr = %q; want :0", addr)
		}

		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("wrapped handler status = %d; want %d", response.Code, http.StatusOK)
		}
		if sentry.GetHubFromContext(request.Context()) != nil {
			t.Fatal("request context unexpectedly mutated before handler execution")
		}
		return errors.New("stop")
	}

	if err := runServer(":0"); err == nil || err.Error() != "stop" {
		t.Fatalf("runServer() error = %v; want stop", err)
	}
}

func TestReportFatalFlushesBeforeExit(t *testing.T) {
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	originalFlush := sentryFlush
	defer func() { sentryFlush = originalFlush }()

	flushCalled := false
	sentryFlush = func(timeout time.Duration) bool {
		flushCalled = true
		if timeout != sentryFlushTimeout {
			t.Fatalf("sentryFlush timeout = %v; want %v", timeout, sentryFlushTimeout)
		}
		return true
	}

	fatalCalled := false
	fatalOnRunError = func(v ...any) {
		fatalCalled = true
		if len(v) != 1 {
			t.Fatalf("fatalOnRunError args = %v; want single arg", v)
		}
	}

	reportFatal(errors.New("boom"))

	if !flushCalled {
		t.Fatal("reportFatal() did not flush Sentry")
	}
	if !fatalCalled {
		t.Fatal("reportFatal() did not call fatalOnRunError")
	}
}

type rawInvalidMessage struct{}

func (rawInvalidMessage) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal boom")
}

type failingResponseWriter struct{}

func (failingResponseWriter) Header() http.Header {
	return http.Header{}
}

func (failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write boom")
}

func (failingResponseWriter) WriteHeader(statusCode int) {}

func newSocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)
	serverErrCh := make(chan error, 1)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErrCh <- err
			return
		}
		serverConnCh <- conn
	}))

	clientConn := mustDialWS(t, httpServer.URL)
	var serverConn *websocket.Conn
	select {
	case serverConn = <-serverConnCh:
	case err := <-serverErrCh:
		t.Fatalf("upgrade server connection error = %v", err)
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = serverConn.Close()
			_ = clientConn.Close()
			httpServer.Close()
		})
	}

	return serverConn, clientConn, cleanup
}

func mustDialWS(t *testing.T, baseURL string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Dial(%q) error = %v", url, err)
	}
	return conn
}

func mustDialWSWithCookie(t *testing.T, baseURL, sessionToken string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
	headers := http.Header{}
	if strings.TrimSpace(sessionToken) != "" {
		headers.Set("Cookie", authCookieName+"="+sessionToken)
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		t.Fatalf("Dial(%q) error = %v", url, err)
	}
	return conn
}

func mustConnectSession(t *testing.T, conn *websocket.Conn, sessionID string) connectedEvent {
	t.Helper()

	mustSendEnvelope(t, conn, "connect", connectRequest{SessionID: sessionID})
	return mustReadConnectedEvent(t, conn)
}

func TestHandleSessionRoutes(t *testing.T) {
	t.Setenv("FRONTEND_URL", "http://frontend.test")
	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)
	handler := &authHandler{
		config: authConfig{oauthConfig: &oauth2.Config{Endpoint: oauth2.Endpoint{AuthURL: "https://oauth.test/auth"}}},
		store:  &stubAuthStore{},
		now:    func() time.Time { return now },
		state:  func() (string, error) { return "state-token", nil },
	}
	server := &wsServer{auth: handler}

	t.Run("preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/auth/session", nil)
		request.Header.Set("Origin", "http://frontend.test")
		response := httptest.NewRecorder()
		server.handleSessionRoutes(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("preflight status = %d; want 204", response.Code)
		}
	})

	t.Run("nil auth", func(t *testing.T) {
		response := httptest.NewRecorder()
		(&wsServer{}).handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/session", nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("nil auth status = %d; want 500", response.Code)
		}
	})

	t.Run("google routes", func(t *testing.T) {
		response := httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/google", nil))
		if response.Code != http.StatusFound {
			t.Fatalf("google sign in status = %d; want 302", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodPost, "/auth/google", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("google sign in wrong method status = %d; want 405", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/google/callback", nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("google callback status = %d; want 400", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodPost, "/auth/google/callback", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("google callback wrong method status = %d; want 405", response.Code)
		}
	})

	t.Run("session and logout routes", func(t *testing.T) {
		response := httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/session", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("session status = %d; want 200", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodPost, "/auth/session", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("session wrong method status = %d; want 405", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodPost, "/auth/onboarding/complete", nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("onboarding completion status = %d; want 401", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/onboarding/complete", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("onboarding completion wrong method status = %d; want 405", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("logout status = %d; want 204", response.Code)
		}

		response = httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/logout", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("logout wrong method status = %d; want 405", response.Code)
		}
	})

	t.Run("default route", func(t *testing.T) {
		response := httptest.NewRecorder()
		server.handleSessionRoutes(response, httptest.NewRequest(http.MethodGet, "/auth/unknown", nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("default route status = %d; want 404", response.Code)
		}
	})
}

func mustReadConnectedEvent(t *testing.T, conn *websocket.Conn) connectedEvent {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "connected" {
		t.Fatalf("connect response type = %q; want connected", envelope.Type)
	}
	var event connectedEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(connected) error = %v", err)
	}
	if event.SessionID == "" {
		t.Fatal("connected.SessionID = empty; want session ID")
	}
	if event.PlayerID == "" {
		t.Fatal("connected.PlayerID = empty; want player ID")
	}
	return event
}

func mustSendEnvelope(t *testing.T, conn *websocket.Conn, messageType string, data any) {
	t.Helper()

	if err := conn.WriteJSON(wsEnvelope{Type: messageType, Data: mustMarshalRawMessage(data)}); err != nil {
		t.Fatalf("WriteJSON(%q) error = %v", messageType, err)
	}
}

func mustReadEnvelopeFromConn(t *testing.T, conn *websocket.Conn) wsEnvelope {
	t.Helper()

	var envelope wsEnvelope
	if err := conn.ReadJSON(&envelope); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	return envelope
}

func mustReadRoomState(t *testing.T, conn *websocket.Conn) roomSnapshot {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "room_state" {
		t.Fatalf("room_state response type = %q; want room_state", envelope.Type)
	}
	var event roomStateEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(room_state) error = %v", err)
	}
	return event.Room
}

func mustReadError(t *testing.T, conn *websocket.Conn, want string) {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "error" {
		t.Fatalf("error response type = %q; want error", envelope.Type)
	}
	var event errorEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	wantCode := clientErrorCode(errors.New(want))
	if event.Code != wantCode {
		t.Fatalf("error code = %q; want %q", event.Code, wantCode)
	}
	if event.Message != want {
		t.Fatalf("error message = %q; want %q", event.Message, want)
	}
}

func mustReadActionError(t *testing.T, conn *websocket.Conn, action, want string) {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "error" {
		t.Fatalf("error response type = %q; want error", envelope.Type)
	}
	var event errorEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(error) error = %v", err)
	}
	if event.Action != action {
		t.Fatalf("error action = %q; want %q", event.Action, action)
	}
	wantCode := clientErrorCode(errors.New(want))
	if event.Code != wantCode {
		t.Fatalf("error code = %q; want %q", event.Code, wantCode)
	}
	if event.Message != want {
		t.Fatalf("error message = %q; want %q", event.Message, want)
	}
}

func mustReadLeftRoom(t *testing.T, conn *websocket.Conn) leftRoomEvent {
	t.Helper()

	envelope := mustReadEnvelopeFromConn(t, conn)
	if envelope.Type != "left_room" {
		t.Fatalf("left_room response type = %q; want left_room", envelope.Type)
	}
	var event leftRoomEvent
	if err := json.Unmarshal(envelope.Data, &event); err != nil {
		t.Fatalf("json.Unmarshal(left_room) error = %v", err)
	}
	return event
}

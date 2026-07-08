package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

func TestWebSocketConnectionAttemptsAreRateLimited(t *testing.T) {
	server := newWSServer()
	server.rateLimits = newWSRateLimiters(wsRateLimitConfig{
		ConnectionRate:  rate.Every(time.Hour),
		ConnectionBurst: 1,
		CreateRoomRate:  rate.Inf,
		CreateRoomBurst: 1,
		JoinRoomRate:    rate.Inf,
		JoinRoomBurst:   1,
		MessageRate:     rate.Inf,
		MessageBurst:    1,
		VisitorTTL:      time.Hour,
	})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	firstConn := mustDialWS(t, httpServer.URL)
	defer firstConn.Close()

	url := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	secondConn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if err == nil {
		_ = secondConn.Close()
		t.Fatal("second Dial() error = nil; want rate limit rejection")
	}
	if resp == nil {
		t.Fatal("second Dial() response = nil; want 429 response")
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("second Dial() status = %d; want %d", resp.StatusCode, http.StatusTooManyRequests)
	}
}

func TestWebSocketCreateAndJoinAttemptsAreRateLimited(t *testing.T) {
	t.Run("create room", func(t *testing.T) {
		server := newWSServer()
		server.rateLimits = newWSRateLimiters(wsRateLimitConfig{
			ConnectionRate:  rate.Inf,
			ConnectionBurst: 1,
			CreateRoomRate:  rate.Every(time.Hour),
			CreateRoomBurst: 1,
			JoinRoomRate:    rate.Inf,
			JoinRoomBurst:   1,
			MessageRate:     rate.Inf,
			MessageBurst:    1,
			VisitorTTL:      time.Hour,
		})
		httpServer := httptest.NewServer(server.routes())
		defer httpServer.Close()

		conn := mustDialWS(t, httpServer.URL)
		defer conn.Close()
		mustConnectSession(t, conn, "")

		mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Host"})
		_ = mustReadRoomState(t, conn)
		mustSendEnvelope(t, conn, "leave_room", leaveRoomRequest{})
		_ = mustReadLeftRoom(t, conn)

		mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Again"})
		mustReadError(t, conn, "rate limit exceeded")
	})

	t.Run("join room", func(t *testing.T) {
		server := newWSServer()
		server.rateLimits = newWSRateLimiters(wsRateLimitConfig{
			ConnectionRate:  rate.Inf,
			ConnectionBurst: 1,
			CreateRoomRate:  rate.Inf,
			CreateRoomBurst: 1,
			JoinRoomRate:    rate.Every(time.Hour),
			JoinRoomBurst:   1,
			MessageRate:     rate.Inf,
			MessageBurst:    1,
			VisitorTTL:      time.Hour,
		})
		httpServer := httptest.NewServer(server.routes())
		defer httpServer.Close()

		hostConn := mustDialWS(t, httpServer.URL)
		defer hostConn.Close()
		mustConnectSession(t, hostConn, "")
		mustSendEnvelope(t, hostConn, "create_room", createRoomRequest{Name: "Host"})
		room := mustReadRoomState(t, hostConn)

		guestConn := mustDialWS(t, httpServer.URL)
		defer guestConn.Close()
		mustConnectSession(t, guestConn, "")
		mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: room.Code, Name: "Guest"})
		_ = mustReadRoomState(t, guestConn)
		_ = mustReadRoomState(t, hostConn)
		mustSendEnvelope(t, guestConn, "leave_room", leaveRoomRequest{})
		_ = mustReadLeftRoom(t, guestConn)
		_ = mustReadRoomState(t, hostConn)

		mustSendEnvelope(t, guestConn, "join_room", joinRoomRequest{RoomCode: room.Code, Name: "Guest"})
		mustReadError(t, guestConn, "rate limit exceeded")
	})
}

func TestWebSocketMessageThroughputIsRateLimited(t *testing.T) {
	server := newWSServer()
	server.rateLimits = newWSRateLimiters(wsRateLimitConfig{
		ConnectionRate:  rate.Inf,
		ConnectionBurst: 1,
		CreateRoomRate:  rate.Inf,
		CreateRoomBurst: 1,
		JoinRoomRate:    rate.Inf,
		JoinRoomBurst:   1,
		MessageRate:     rate.Every(time.Hour),
		MessageBurst:    1,
		VisitorTTL:      time.Hour,
	})
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()

	conn := mustDialWS(t, httpServer.URL)
	defer conn.Close()

	mustSendEnvelope(t, conn, "connect", connectRequest{})
	connected := mustReadConnectedEvent(t, conn)
	if connected.SessionID == "" {
		t.Fatal("connected.SessionID = empty; want session")
	}

	mustSendEnvelope(t, conn, "create_room", createRoomRequest{Name: "Flood"})
	mustReadError(t, conn, "rate limit exceeded")

	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage() after message rate limit error = nil; want closed socket")
	}
}

func TestWSRateLimiterHelpers(t *testing.T) {
	defaults := defaultWSRateLimitConfig()
	normalized := normalizeWSRateLimitConfig(wsRateLimitConfig{})
	if normalized != defaults {
		t.Fatalf("normalizeWSRateLimitConfig(empty) = %#v; want defaults %#v", normalized, defaults)
	}

	var nilLimiters *wsRateLimiters
	if !nilLimiters.allowConnectionAttempt(nil) {
		t.Fatal("nil allowConnectionAttempt() = false; want true")
	}
	if !nilLimiters.allowCreateRoom("") {
		t.Fatal("nil allowCreateRoom() = false; want true")
	}
	if !nilLimiters.allowJoinRoom("") {
		t.Fatal("nil allowJoinRoom() = false; want true")
	}
	if !nilLimiters.newMessageLimiter().Allow() {
		t.Fatal("nil newMessageLimiter().Allow() = false; want true")
	}

	now := time.Date(2026, time.May, 25, 12, 0, 0, 0, time.UTC)
	limiters := newWSRateLimiters(wsRateLimitConfig{
		ConnectionRate:  rate.Inf,
		ConnectionBurst: 1,
		CreateRoomRate:  rate.Inf,
		CreateRoomBurst: 1,
		JoinRoomRate:    rate.Inf,
		JoinRoomBurst:   1,
		MessageRate:     rate.Inf,
		MessageBurst:    1,
		VisitorTTL:      time.Minute,
	})
	limiters.now = func() time.Time { return now }
	if !limiters.allow(limiters.connectionAttempts, "", rate.Inf, 1) {
		t.Fatal("allow(empty key) = false; want true")
	}
	if _, ok := limiters.connectionAttempts["unknown"]; !ok {
		t.Fatal("connectionAttempts[unknown] missing after empty key allow")
	}
	limiters.createRoomAttempts["stale"] = &trackedLimiter{
		limiter:  rate.NewLimiter(rate.Inf, 1),
		lastSeen: now.Add(-time.Minute),
	}
	limiters.joinRoomAttempts["fresh"] = &trackedLimiter{
		limiter:  rate.NewLimiter(rate.Inf, 1),
		lastSeen: now,
	}
	limiters.lastCleanup = now.Add(-time.Minute)
	limiters.cleanupLocked(now)
	if _, ok := limiters.createRoomAttempts["stale"]; ok {
		t.Fatal("stale create room limiter still exists; want deleted")
	}
	if _, ok := limiters.joinRoomAttempts["fresh"]; !ok {
		t.Fatal("fresh join room limiter missing; want retained")
	}
}

func TestClientIPFromRequestBranches(t *testing.T) {
	if got := clientIPFromRequest(nil); got != "" {
		t.Fatalf("clientIPFromRequest(nil) = %q; want empty", got)
	}

	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("X-Forwarded-For", " 203.0.113.10, 203.0.113.11 ")
	if got := clientIPFromRequest(request); got != "203.0.113.10" {
		t.Fatalf("clientIPFromRequest(XFF) = %q; want 203.0.113.10", got)
	}

	request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.Header.Set("X-Real-IP", "2001:db8::1")
	if got := clientIPFromRequest(request); got != "2001:db8::1" {
		t.Fatalf("clientIPFromRequest(X-Real-IP) = %q; want 2001:db8::1", got)
	}

	request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.RemoteAddr = net.JoinHostPort("203.0.113.12", "4567")
	if got := clientIPFromRequest(request); got != "203.0.113.12" {
		t.Fatalf("clientIPFromRequest(host:port ip) = %q; want 203.0.113.12", got)
	}

	request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.RemoteAddr = net.JoinHostPort("client.local", "4567")
	if got := clientIPFromRequest(request); got != "client.local" {
		t.Fatalf("clientIPFromRequest(host:port host) = %q; want client.local", got)
	}

	request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.RemoteAddr = "203.0.113.13"
	if got := clientIPFromRequest(request); got != "203.0.113.13" {
		t.Fatalf("clientIPFromRequest(remote ip) = %q; want 203.0.113.13", got)
	}

	request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.RemoteAddr = " client.local "
	if got := clientIPFromRequest(request); got != "client.local" {
		t.Fatalf("clientIPFromRequest(remote host) = %q; want client.local", got)
	}
}

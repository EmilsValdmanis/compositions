package main

import (
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

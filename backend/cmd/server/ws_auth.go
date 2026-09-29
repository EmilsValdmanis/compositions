package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type socketAuthSession struct {
	auth      *authHandler
	token     string
	userID    string
	expiresAt time.Time
}

// Serialize registration with logout notification. If logout deletes the
// session during validation, it will close the newly registered socket too.
func (s *wsServer) registerAuthSocket(conn *websocket.Conn, request *http.Request) (authSession, error) {
	s.authSocketsMu.Lock()
	defer s.authSocketsMu.Unlock()
	ctx, cancel := context.WithTimeout(request.Context(), defaultUserStoreTimeout)
	defer cancel()
	session, err := s.auth.sessionFromRequest(request.WithContext(ctx))
	if err != nil {
		return authSession{}, err
	}
	s.authSockets[conn] = socketAuthSession{
		auth: s.auth, token: session.token, userID: session.user.ID, expiresAt: session.expiresAt,
	}
	return session, nil
}

func (s *wsServer) forgetAuthSocket(conn *websocket.Conn) {
	s.authSocketsMu.Lock()
	delete(s.authSockets, conn)
	s.authSocketsMu.Unlock()
}

// The connect timeout governs sockets still completing their handshake.
func (s *wsServer) revalidateIdleAuthSocket(conn *websocket.Conn) bool {
	s.authSocketsMu.Lock()
	_, registered := s.authSockets[conn]
	s.authSocketsMu.Unlock()
	return !registered || s.revalidateAuthSocket(conn)
}

// Check every command and heartbeat, including an idle connection. Store
// errors fail closed; a socket never keeps authenticated access during an outage.
func (s *wsServer) revalidateAuthSocket(conn *websocket.Conn) bool {
	s.authSocketsMu.Lock()
	session, registered := s.authSockets[conn]
	s.authSocketsMu.Unlock()
	if !registered {
		return s.auth == nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	now := session.auth.now()
	record, err := session.auth.store.GetSessionUserByToken(ctx, session.token, now)
	if err == nil && record.ID == session.userID &&
		(session.expiresAt.IsZero() || now.Before(session.expiresAt)) &&
		(record.ExpiresAt.IsZero() || now.Before(record.ExpiresAt)) {
		return true
	}
	closeUnauthenticatedSocket(conn)
	return false
}

func (s *wsServer) revokeAuthSockets(token string) {
	s.authSocketsMu.Lock()
	var connections []*websocket.Conn
	for conn, session := range s.authSockets {
		if session.token == token {
			connections = append(connections, conn)
		}
	}
	s.authSocketsMu.Unlock()
	for _, conn := range connections {
		closeUnauthenticatedSocket(conn)
	}
}

func closeUnauthenticatedSocket(conn *websocket.Conn) {
	_ = writeControl(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication required"), time.Now().Add(defaultWSWriteTimeout))
	_ = conn.Close()
}

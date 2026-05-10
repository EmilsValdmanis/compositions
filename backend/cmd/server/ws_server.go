package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

var errSocketClosed = errors.New("socket closed")
var emitEvent = writeEvent

type wsEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type connectRequest struct {
	SessionID string `json:"sessionId"`
	AuthToken string `json:"authToken"`
}

type createRoomRequest struct {
	Name string `json:"name"`
}

type joinRoomRequest struct {
	RoomCode string `json:"roomCode"`
	Name     string `json:"name"`
}

type startGameRequest struct {
	DealerIndex int `json:"dealerIndex"`
}

type leaveRoomRequest struct{}

type connectedEvent struct {
	SessionID string `json:"sessionId"`
	PlayerID  string `json:"playerId"`
}

type errorEvent struct {
	Message string `json:"message"`
}

type roomStateEvent struct {
	Room roomSnapshot `json:"room"`
}

type leftRoomEvent struct {
	RoomCode string `json:"roomCode,omitempty"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type roomSnapshot struct {
	Code         string           `json:"code"`
	Phase        string           `json:"phase"`
	HostPlayerID string           `json:"hostPlayerId"`
	DealerIndex  int              `json:"dealerIndex,omitempty"`
	Players      []playerSnapshot `json:"players"`
}

type playerSnapshot struct {
	PlayerID     string `json:"playerId"`
	SessionID    string `json:"sessionId"`
	Name         string `json:"name"`
	ImageURL     string `json:"imageUrl,omitempty"`
	Connected    bool   `json:"connected"`
	Seat         int    `json:"seat"`
	IsHost       bool   `json:"isHost"`
	CanReconnect bool   `json:"canReconnect"`
}

type wsServer struct {
	lobby    *lobbyServer
	verifier sessionVerifier
	allowedOrigin string
	upgrader websocket.Upgrader
}

func newWSServer() *wsServer {
	return newWSServerWithVerifier(nil)
}

func newWSServerWithVerifier(verifier sessionVerifier) *wsServer {
	return newWSServerWithAllowedOrigin(verifier, "")
}

func newWSServerWithAllowedOrigin(verifier sessionVerifier, allowedOrigin string) *wsServer {
	server := &wsServer{
		lobby:         newLobbyServer(),
		verifier:      verifier,
		allowedOrigin: normalizeOrigin(allowedOrigin),
	}
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return server.isAllowedOrigin(r.Header.Get("Origin"))
		},
	}
	return server
}

func newConfiguredWSServer() (*wsServer, error) {
	baseURL, err := betterAuthBaseURLFromEnv()
	if err != nil {
		return nil, err
	}

	verifier := newBetterAuthSessionVerifier(baseURL, nil)
	return newWSServerWithAllowedOrigin(verifier, originFromBaseURL(baseURL)), nil
}

func (s *wsServer) isAllowedOrigin(origin string) bool {
	if s == nil || s.allowedOrigin == "" {
		return true
	}

	cleanOrigin := normalizeOrigin(origin)
	if cleanOrigin == "" {
		return true
	}

	return cleanOrigin == s.allowedOrigin
}

func normalizeOrigin(origin string) string {
	return strings.TrimRight(strings.TrimSpace(origin), "/")
}

func originFromBaseURL(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return normalizeOrigin(parsed.Scheme + "://" + parsed.Host)
}

func (s *wsServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ws", s.handleWS)
	return mux
}

func (s *wsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		log.Printf("write health response: %v", err)
	}
}

func (s *wsServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade websocket: %v", err)
		return
	}

	s.handleConnection(conn)
}

func (s *wsServer) handleConnection(conn *websocket.Conn) {
	defer conn.Close()

	sessionID := ""
	for {
		var envelope wsEnvelope
		if err := conn.ReadJSON(&envelope); err != nil {
			if sessionID != "" {
				s.lobby.disconnect(sessionID, conn)
			}
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("read websocket message: %v", err)
			}
			return
		}

		switch envelope.Type {
		case "connect":
			nextSessionID, shouldClose := s.handleConnect(conn, envelope)
			if shouldClose {
				return
			}
			sessionID = nextSessionID
		case "create_room":
			s.handleCreateRoom(conn, sessionID, envelope)
		case "join_room":
			s.handleJoinRoom(conn, sessionID, envelope)
		case "start_game":
			s.handleStartGame(conn, sessionID, envelope)
		case "leave_room":
			if s.handleLeaveRoom(conn, sessionID, envelope) {
				return
			}
		default:
			s.writeError(conn, errors.New("unknown message type"))
		}
	}
}

func (s *wsServer) handleConnect(conn *websocket.Conn, envelope wsEnvelope) (string, bool) {
	var req connectRequest
	if err := decodePayload(envelope.Data, &req); err != nil {
		s.writeError(conn, err)
		return "", false
	}

	user := authenticatedUser{}
	if s.verifier != nil {
		verifiedUser, err := s.verifier.VerifySession(context.Background(), req.AuthToken)
		if err != nil {
			s.writeError(conn, err)
			return "", true
		}
		user = verifiedUser
	}

	event, roomState, recipients, err := s.lobby.connectWithUser(req.SessionID, user, conn)
	if err != nil {
		s.writeError(conn, err)
		return "", true
	}
	if err := emitEvent(conn, "connected", event); err != nil {
		return "", true
	}
	if roomState != nil {
		if err := emitEvent(conn, "room_state", roomStateEvent{Room: *roomState}); err != nil {
			return "", true
		}
		s.broadcastRoomState(*roomState, otherConnections(recipients, conn))
	}

	return event.SessionID, false
}

func (s *wsServer) handleCreateRoom(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[createRoomRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, err := s.lobby.createRoom(sessionID, req.Name)
	if err != nil {
		s.writeError(conn, err)
		return
	}
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleJoinRoom(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[joinRoomRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, err := s.lobby.joinRoom(sessionID, req.RoomCode, req.Name)
	if err != nil {
		s.writeError(conn, err)
		return
	}
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleStartGame(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[startGameRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, err := s.lobby.startGame(sessionID, req.DealerIndex)
	if err != nil {
		s.writeError(conn, err)
		return
	}
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleLeaveRoom(conn *websocket.Conn, sessionID string, envelope wsEnvelope) bool {
	if _, ok := decodeSessionRequest[leaveRoomRequest](s, conn, sessionID, envelope); !ok {
		return false
	}

	roomState, recipients, roomCode, err := s.lobby.leaveRoom(sessionID)
	if err != nil {
		s.writeError(conn, err)
		return false
	}
	if err := emitEvent(conn, "left_room", leftRoomEvent{RoomCode: roomCode}); err != nil {
		return true
	}
	if roomState != nil {
		s.broadcastRoomState(*roomState, recipients)
	}
	return false
}

func (s *wsServer) writeError(conn *websocket.Conn, err error) {
	if writeErr := emitEvent(conn, "error", errorEvent{Message: err.Error()}); writeErr != nil && !errors.Is(writeErr, errSocketClosed) {
		log.Printf("write websocket error event: %v", writeErr)
	}
}

func (s *wsServer) broadcastRoomState(roomState roomSnapshot, recipients []*websocket.Conn) {
	for _, conn := range recipients {
		if conn == nil {
			continue
		}
		if err := emitEvent(conn, "room_state", roomStateEvent{Room: roomState}); err != nil && !errors.Is(err, errSocketClosed) {
			log.Printf("broadcast room_state: %v", err)
		}
	}
}

func decodeSessionRequest[T any](s *wsServer, conn *websocket.Conn, sessionID string, envelope wsEnvelope) (T, bool) {
	var req T
	if err := requireConnectedSession(sessionID); err != nil {
		s.writeError(conn, err)
		return req, false
	}
	if err := s.lobby.requireActiveSessionConnection(sessionID, conn); err != nil {
		s.writeError(conn, err)
		return req, false
	}
	if err := decodePayload(envelope.Data, &req); err != nil {
		s.writeError(conn, err)
		return req, false
	}
	return req, true
}

func requireConnectedSession(sessionID string) error {
	if sessionID == "" {
		return errors.New("connect first")
	}
	return nil
}

func decodePayload(data json.RawMessage, target any) error {
	if len(data) == 0 {
		return errors.New("missing data")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return errors.New("invalid data")
	}
	return nil
}

func writeEvent(conn *websocket.Conn, messageType string, data any) error {
	if err := conn.WriteJSON(wsEnvelope{Type: messageType, Data: mustMarshalRawMessage(data)}); err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) || errors.Is(err, websocket.ErrCloseSent) {
			return errSocketClosed
		}
		return err
	}
	return nil
}

func mustMarshalRawMessage(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func otherConnections(conns []*websocket.Conn, exclude *websocket.Conn) []*websocket.Conn {
	filtered := make([]*websocket.Conn, 0, len(conns))
	for _, conn := range conns {
		if conn == exclude {
			continue
		}
		filtered = append(filtered, conn)
	}
	return filtered
}

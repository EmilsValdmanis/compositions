package main

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	minPlayersToStart = 2
	maxPlayersPerRoom = 4
	roomCodeLength    = 6
)

var errSocketClosed = errors.New("socket closed")

type wsEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type connectRequest struct {
	SessionID string `json:"sessionId"`
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
	Connected    bool   `json:"connected"`
	Seat         int    `json:"seat"`
	IsHost       bool   `json:"isHost"`
	CanReconnect bool   `json:"canReconnect"`
}

type playerSession struct {
	sessionID string
	playerID  string
	conn      *websocket.Conn
	roomCode  string
}

type roomPlayer struct {
	player    *game.Player
	name      string
	sessionID string
	connected bool
	seat      int
	host      bool
}

type room struct {
	code      string
	gameState *game.GameState
	players   []*roomPlayer
	hostID    string
}

type lobbyServer struct {
	mu       sync.Mutex
	rng      *rand.Rand
	sessions map[string]*playerSession
	rooms    map[string]*room
}

type wsServer struct {
	lobby    *lobbyServer
	upgrader websocket.Upgrader
}

var listenAndServe = http.ListenAndServe
var fatalOnRunError = log.Fatal
var emitEvent = writeEvent
var makeGameState = game.NewGameState
var addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
	return state.AddPlayer(player)
}

func newLobbyServer() *lobbyServer {
	return &lobbyServer{
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		sessions: make(map[string]*playerSession),
		rooms:    make(map[string]*room),
	}
}

func newWSServer() *wsServer {
	return &wsServer{
		lobby: newLobbyServer(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
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
			var req connectRequest
			if err := decodePayload(envelope.Data, &req); err != nil {
				s.writeError(conn, err)
				continue
			}

			event, roomState, recipients, err := s.lobby.connect(req.SessionID, conn)
			if err != nil {
				s.writeError(conn, err)
				continue
			}
			sessionID = event.SessionID
			if err := emitEvent(conn, "connected", event); err != nil {
				return
			}
			if roomState != nil {
				if err := emitEvent(conn, "room_state", roomStateEvent{Room: *roomState}); err != nil {
					return
				}
				s.broadcastRoomState(*roomState, otherConnections(recipients, conn))
			}
		case "create_room":
			if sessionID == "" {
				s.writeError(conn, errors.New("connect first"))
				continue
			}

			var req createRoomRequest
			if err := decodePayload(envelope.Data, &req); err != nil {
				s.writeError(conn, err)
				continue
			}

			roomState, recipients, err := s.lobby.createRoom(sessionID, req.Name)
			if err != nil {
				s.writeError(conn, err)
				continue
			}
			s.broadcastRoomState(roomState, recipients)
		case "join_room":
			if sessionID == "" {
				s.writeError(conn, errors.New("connect first"))
				continue
			}

			var req joinRoomRequest
			if err := decodePayload(envelope.Data, &req); err != nil {
				s.writeError(conn, err)
				continue
			}

			roomState, recipients, err := s.lobby.joinRoom(sessionID, req.RoomCode, req.Name)
			if err != nil {
				s.writeError(conn, err)
				continue
			}
			s.broadcastRoomState(roomState, recipients)
		case "start_game":
			if sessionID == "" {
				s.writeError(conn, errors.New("connect first"))
				continue
			}

			var req startGameRequest
			if err := decodePayload(envelope.Data, &req); err != nil {
				s.writeError(conn, err)
				continue
			}

			roomState, recipients, err := s.lobby.startGame(sessionID, req.DealerIndex)
			if err != nil {
				s.writeError(conn, err)
				continue
			}
			s.broadcastRoomState(roomState, recipients)
		default:
			s.writeError(conn, errors.New("unknown message type"))
		}
	}
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

func (l *lobbyServer) connect(existingSessionID string, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if existingSessionID != "" {
		session, ok := l.sessions[existingSessionID]
		if !ok {
			return connectedEvent{}, nil, nil, errors.New("session not found")
		}

		session.conn = conn
		var roomState *roomSnapshot
		var recipients []*websocket.Conn
		if session.roomCode != "" {
			if room := l.rooms[session.roomCode]; room != nil {
				if player := room.playerByID(session.playerID); player != nil {
					player.connected = true
					player.sessionID = session.sessionID
					snapshot := room.snapshot()
					roomState = &snapshot
					recipients = room.connectedConns(l.sessions)
				}
			}
		}

		return connectedEvent{SessionID: session.sessionID, PlayerID: session.playerID}, roomState, recipients, nil
	}

	sessionID := uuid.NewString()
	playerID := newPlayer().ID
	l.sessions[sessionID] = &playerSession{
		sessionID: sessionID,
		playerID:  playerID,
		conn:      conn,
	}

	return connectedEvent{SessionID: sessionID, PlayerID: playerID}, nil, nil, nil
}

func (l *lobbyServer) createRoom(sessionID, name string) (roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	if session.roomCode != "" {
		return roomSnapshot{}, nil, errors.New("already in a room")
	}
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return roomSnapshot{}, nil, errors.New("name is required")
	}

	player := &roomPlayer{
		player:    newPlayerWithID(session.playerID),
		name:      cleanName,
		sessionID: session.sessionID,
		connected: true,
		seat:      0,
		host:      true,
	}
	room := &room{
		code:      l.generateRoomCode(),
		gameState: makeGameState(),
		players:   []*roomPlayer{player},
		hostID:    session.playerID,
	}
	if err := addPlayerToGameState(room.gameState, player.player); err != nil {
		return roomSnapshot{}, nil, err
	}

	l.rooms[room.code] = room
	session.roomCode = room.code

	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) joinRoom(sessionID, roomCode, name string) (roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	if session.roomCode != "" {
		return roomSnapshot{}, nil, errors.New("already in a room")
	}

	roomCode = strings.ToUpper(strings.TrimSpace(roomCode))
	room := l.rooms[roomCode]
	if room == nil {
		return roomSnapshot{}, nil, errors.New("room not found")
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby {
		return roomSnapshot{}, nil, errors.New("game already started")
	}
	if len(room.players) >= maxPlayersPerRoom {
		return roomSnapshot{}, nil, errors.New("room is full")
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return roomSnapshot{}, nil, errors.New("name is required")
	}

	player := &roomPlayer{
		player:    newPlayerWithID(session.playerID),
		name:      cleanName,
		sessionID: session.sessionID,
		connected: true,
		seat:      len(room.players),
	}
	if err := addPlayerToGameState(room.gameState, player.player); err != nil {
		return roomSnapshot{}, nil, err
	}

	room.players = append(room.players, player)
	session.roomCode = room.code

	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) startGame(sessionID string, dealerIndex int) (roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	if session.roomCode == "" {
		return roomSnapshot{}, nil, errors.New("join a room first")
	}

	room := l.rooms[session.roomCode]
	if room == nil {
		return roomSnapshot{}, nil, errors.New("room not found")
	}
	if room.hostID != session.playerID {
		return roomSnapshot{}, nil, errors.New("only the host can start the game")
	}
	if len(room.players) < minPlayersToStart {
		return roomSnapshot{}, nil, errors.New("need at least 2 players to start")
	}
	if !room.allPlayersConnected() {
		return roomSnapshot{}, nil, errors.New("all players must be connected")
	}

	chooserIndex := (dealerIndex - 1 + len(room.players)) % len(room.players)
	if err := room.gameState.StartGame(dealerIndex, chooserIndex, game.DealRoundRobin, nil, 0); err != nil {
		return roomSnapshot{}, nil, err
	}

	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) disconnect(sessionID string, conn *websocket.Conn) {
	var roomState roomSnapshot
	var recipients []*websocket.Conn
	shouldBroadcast := false

	l.mu.Lock()

	session := l.sessions[sessionID]
	if session == nil || session.conn != conn {
		l.mu.Unlock()
		return
	}

	session.conn = nil
	if session.roomCode == "" {
		l.mu.Unlock()
		return
	}

	room := l.rooms[session.roomCode]
	if room == nil {
		l.mu.Unlock()
		return
	}

	player := room.playerByID(session.playerID)
	if player == nil {
		l.mu.Unlock()
		return
	}
	player.connected = false

	roomState = room.snapshot()
	recipients = room.connectedConns(l.sessions)
	shouldBroadcast = len(recipients) > 0
	l.mu.Unlock()

	if shouldBroadcast {
		l.broadcastDisconnect(roomState, recipients)
	}
}

func (l *lobbyServer) broadcastDisconnect(roomState roomSnapshot, recipients []*websocket.Conn) {
	for _, conn := range recipients {
		if conn == nil {
			continue
		}
		_ = emitEvent(conn, "room_state", roomStateEvent{Room: roomState})
	}
}

func (l *lobbyServer) requireSession(sessionID string) (*playerSession, error) {
	session := l.sessions[sessionID]
	if session == nil {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (l *lobbyServer) generateRoomCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for {
		code := strings.Builder{}
		code.Grow(roomCodeLength)
		for range roomCodeLength {
			code.WriteByte(alphabet[l.rng.Intn(len(alphabet))])
		}
		roomCode := code.String()
		if _, exists := l.rooms[roomCode]; !exists {
			return roomCode
		}
	}
}

func (r *room) snapshot() roomSnapshot {
	players := make([]playerSnapshot, 0, len(r.players))
	for _, player := range r.players {
		if player == nil {
			continue
		}
		players = append(players, playerSnapshot{
			PlayerID:     player.player.ID,
			SessionID:    player.sessionID,
			Name:         player.name,
			Connected:    player.connected,
			Seat:         player.seat,
			IsHost:       player.host,
			CanReconnect: true,
		})
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Seat < players[j].Seat
	})

	snapshot := roomSnapshot{
		Code:         r.code,
		Phase:        phaseName(r.gameStatePhase()),
		HostPlayerID: r.hostID,
		Players:      players,
	}
	if r.gameStatePhase() != game.PhaseLobby {
		snapshot.DealerIndex = r.gameStateDealerIndex()
	}
	return snapshot
}

func (r *room) connectedConns(sessions map[string]*playerSession) []*websocket.Conn {
	recipients := make([]*websocket.Conn, 0, len(r.players))
	for _, player := range r.players {
		if player == nil || !player.connected {
			continue
		}
		session := sessions[player.sessionID]
		if session == nil || session.conn == nil {
			continue
		}
		recipients = append(recipients, session.conn)
	}
	return recipients
}

func (r *room) playerByID(playerID string) *roomPlayer {
	for _, player := range r.players {
		if player != nil && player.player.ID == playerID {
			return player
		}
	}
	return nil
}

func (r *room) allPlayersConnected() bool {
	for _, player := range r.players {
		if player == nil || !player.connected {
			return false
		}
	}
	return true
}

func (r *room) gameStatePhase() game.GamePhase {
	if r == nil || r.gameState == nil {
		return game.PhaseLobby
	}
	return r.gameState.Phase()
}

func (r *room) gameStateDealerIndex() int {
	if r == nil || r.gameState == nil {
		return 0
	}
	return r.gameState.DealerIndex()
}

func phaseName(phase game.GamePhase) string {
	switch phase {
	case game.PhaseLobby:
		return "lobby"
	case game.PhaseInProgress:
		return "in_progress"
	case game.PhaseRoundOver:
		return "round_over"
	case game.PhaseGameOver:
		return "game_over"
	default:
		return "unknown"
	}
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

func newPlayer() *game.Player {
	return game.NewPlayer()
}

func newPlayerWithID(id string) *game.Player {
	player := game.NewPlayer()
	player.ID = id
	return player
}

func main() {
	if err := runServer(":8080"); err != nil {
		fatalOnRunError(err)
	}
}

func runServer(addr string) error {
	server := newWSServer()
	log.Printf("server running on %s", addr)
	return listenAndServe(addr, server.routes())
}

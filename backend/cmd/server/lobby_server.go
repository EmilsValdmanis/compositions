package main

import (
	"errors"
	"math/rand"
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

func (l *lobbyServer) connect(existingSessionID string, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if existingSessionID != "" {
		return l.connectExistingSession(existingSessionID, conn)
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

func (l *lobbyServer) connectExistingSession(existingSessionID string, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	session, ok := l.sessions[existingSessionID]
	if !ok {
		return connectedEvent{}, nil, nil, errors.New("session not found")
	}

	session.conn = conn
	var roomState *roomSnapshot
	var recipients []*websocket.Conn
	if room := l.sessionRoom(session); room != nil {
		player := room.playerByID(session.playerID)
		player.connected = true
		player.sessionID = session.sessionID
		snapshot := room.snapshot()
		roomState = &snapshot
		recipients = room.connectedConns(l.sessions)
	}

	return connectedEvent{SessionID: session.sessionID, PlayerID: session.playerID}, roomState, recipients, nil
}

func (l *lobbyServer) createRoom(sessionID, name string) (roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	if l.sessionRoom(session) != nil {
		return roomSnapshot{}, nil, errors.New("already in a room")
	}
	cleanName, err := normalizePlayerName(name)
	if err != nil {
		return roomSnapshot{}, nil, err
	}

	gameState := makeGameState()
	if gameState == nil {
		return roomSnapshot{}, nil, errors.New("game state not initialized")
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
		gameState: gameState,
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
	if l.sessionRoom(session) != nil {
		return roomSnapshot{}, nil, errors.New("already in a room")
	}

	room := l.rooms[normalizeRoomCode(roomCode)]
	if room == nil {
		return roomSnapshot{}, nil, errors.New("room not found")
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby {
		return roomSnapshot{}, nil, errors.New("game already started")
	}
	if len(room.players) >= maxPlayersPerRoom {
		return roomSnapshot{}, nil, errors.New("room is full")
	}

	cleanName, err := normalizePlayerName(name)
	if err != nil {
		return roomSnapshot{}, nil, err
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
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, errors.New("join a room first")
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

func (l *lobbyServer) leaveRoom(sessionID string) (*roomSnapshot, []*websocket.Conn, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return nil, nil, "", err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return nil, nil, "", errors.New("join a room first")
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby {
		return nil, nil, "", errors.New("can only leave in lobby")
	}

	updatedPlayers := make([]*roomPlayer, 0, len(room.players))
	for _, player := range room.players {
		if player == nil {
			continue
		}
		if player.player.ID == session.playerID {
			continue
		}
		updatedPlayers = append(updatedPlayers, player)
	}

	roomCode := room.code
	if len(updatedPlayers) == 0 {
		delete(l.rooms, roomCode)
		session.roomCode = ""
		return nil, nil, roomCode, nil
	}

	nextGameState := makeGameState()
	if nextGameState == nil {
		return nil, nil, "", errors.New("game state not initialized")
	}
	for _, player := range updatedPlayers {
		if err := addPlayerToGameState(nextGameState, player.player); err != nil {
			return nil, nil, "", err
		}
	}

	nextHostID := room.hostID
	if nextHostID == session.playerID || room.playerByID(nextHostID) == nil {
		nextHostID = updatedPlayers[0].player.ID
	}
	for i, player := range updatedPlayers {
		player.seat = i
		player.host = player.player.ID == nextHostID
	}

	room.players = updatedPlayers
	room.hostID = nextHostID
	room.gameState = nextGameState
	session.roomCode = ""

	snapshot := room.snapshot()
	return &snapshot, room.connectedConns(l.sessions), roomCode, nil
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

	room := l.sessionRoom(session)
	if room == nil {
		l.mu.Unlock()
		return
	}

	player := room.playerByID(session.playerID)
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

func (l *lobbyServer) requireActiveSessionConnection(sessionID string, conn *websocket.Conn) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return err
	}
	if session.conn != conn {
		return errors.New("session not active on this connection")
	}

	return nil
}

func (l *lobbyServer) sessionRoom(session *playerSession) *room {
	if session == nil || session.roomCode == "" {
		return nil
	}

	room := l.rooms[session.roomCode]
	if room == nil || room.playerByID(session.playerID) == nil {
		session.roomCode = ""
		return nil
	}

	return room
}

func (l *lobbyServer) generateRoomCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for {
		var code strings.Builder
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

func normalizePlayerName(name string) (string, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return "", errors.New("name is required")
	}
	return cleanName, nil
}

func normalizeRoomCode(roomCode string) string {
	return strings.ToUpper(strings.TrimSpace(roomCode))
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

	phase := r.gameStatePhase()
	snapshot := roomSnapshot{
		Code:         r.code,
		Phase:        phaseName(phase),
		HostPlayerID: r.hostID,
		Players:      players,
	}
	if phase != game.PhaseLobby {
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

func newPlayer() *game.Player {
	return game.NewPlayer()
}

func newPlayerWithID(id string) *game.Player {
	player := game.NewPlayer()
	player.ID = id
	return player
}

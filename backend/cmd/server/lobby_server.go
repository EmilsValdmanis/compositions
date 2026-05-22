package main

import (
	"errors"
	"log/slog"
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
	sessionID     string
	playerID      string
	conn          *websocket.Conn
	roomCode      string
	authUserID    string
	displayName   string
	imageURL      string
	authenticated bool
}

type roomPlayer struct {
	player    *game.Player
	name      string
	imageURL  string
	sessionID string
	connected bool
	seat      int
	host      bool
}

type room struct {
	code              string
	gameState         *game.GameState
	players           []*roomPlayer
	hostID            string
	pendingDealChoice *pendingDealChoice
	turnBaseline      *game.GameSnapshot
	turnActivity      *game.TurnActivitySnapshot
}

type pendingDealChoice struct {
	dealerIndex  int
	chooserIndex int
}

type gameStateRecipient struct {
	conn  *websocket.Conn
	event gameStateEvent
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
	return l.connectWithUser(existingSessionID, authenticatedUser{}, conn)
}

func (l *lobbyServer) connectWithUser(existingSessionID string, user authenticatedUser, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if existingSessionID != "" {
		return l.connectExistingSessionWithUser(existingSessionID, user, conn)
	}
	if user.isAuthenticated() {
		if existingSession := l.sessionByAuthUserID(user.ID); existingSession != nil {
			return l.connectExistingSessionWithUser(existingSession.sessionID, user, conn)
		}
	}

	sessionID := uuid.NewString()
	playerID := newPlayer().ID
	l.sessions[sessionID] = &playerSession{
		sessionID:     sessionID,
		playerID:      playerID,
		conn:          conn,
		authUserID:    user.ID,
		displayName:   user.displayName(),
		imageURL:      user.Image,
		authenticated: user.isAuthenticated(),
	}

	slog.Info("session created",
		"sessionID", sessionID,
		"playerID", playerID,
		"authenticated", user.isAuthenticated(),
		"displayName", user.displayName(),
	)
	return connectedEvent{SessionID: sessionID, PlayerID: playerID}, nil, nil, nil
}

func (l *lobbyServer) connectExistingSessionWithUser(existingSessionID string, user authenticatedUser, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	session, ok := l.sessions[existingSessionID]
	if !ok {
		return connectedEvent{}, nil, nil, errors.New("session not found")
	}
	if session.authenticated != user.isAuthenticated() {
		slog.Warn("session auth mismatch", "sessionID", existingSessionID, "sessionAuthenticated", session.authenticated)
		return connectedEvent{}, nil, nil, errAuthenticationRequired
	}
	if session.authenticated && session.authUserID != user.ID {
		slog.Warn("session user mismatch", "sessionID", existingSessionID, "sessionUserID", session.authUserID, "requestUserID", user.ID)
		return connectedEvent{}, nil, nil, errors.New("session belongs to a different user")
	}
	if session.conn != nil && session.conn != conn {
		slog.Warn("session connection replaced", "sessionID", existingSessionID)
	}
	if session.authenticated {
		session.displayName = user.displayName()
		session.imageURL = user.Image
	}

	session.conn = conn
	var roomState *roomSnapshot
	var recipients []*websocket.Conn
	if room := l.sessionRoom(session); room != nil {
		player := room.playerByID(session.playerID)
		player.connected = true
		player.sessionID = session.sessionID
		if session.authenticated {
			player.name = session.displayName
			player.imageURL = session.imageURL
		}
		snapshot := room.snapshot()
		roomState = &snapshot
		recipients = room.connectedConns(l.sessions)
		slog.Info("session reconnected to room",
			"sessionID", session.sessionID,
			"playerID", session.playerID,
			"roomCode", room.code,
		)
	} else {
		slog.Info("session reconnected",
			"sessionID", session.sessionID,
			"playerID", session.playerID,
		)
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
	cleanName, err := session.playerName(name)
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
		imageURL:  session.imageURL,
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
	room.clearTurnTracking()
	if err := addPlayerToGameState(room.gameState, player.player); err != nil {
		return roomSnapshot{}, nil, err
	}

	l.rooms[room.code] = room
	session.roomCode = room.code

	slog.Info("room created", "roomCode", room.code, "sessionID", session.sessionID, "playerID", session.playerID)
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
	if room.pendingDealChoice != nil {
		return roomSnapshot{}, nil, errors.New("game is already starting")
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby {
		return roomSnapshot{}, nil, errors.New("game already started")
	}
	if len(room.players) >= maxPlayersPerRoom {
		return roomSnapshot{}, nil, errors.New("room is full")
	}

	cleanName, err := session.playerName(name)
	if err != nil {
		return roomSnapshot{}, nil, err
	}

	player := &roomPlayer{
		player:    newPlayerWithID(session.playerID),
		name:      cleanName,
		imageURL:  session.imageURL,
		sessionID: session.sessionID,
		connected: true,
		seat:      len(room.players),
	}
	if err := addPlayerToGameState(room.gameState, player.player); err != nil {
		return roomSnapshot{}, nil, err
	}

	room.players = append(room.players, player)
	session.roomCode = room.code

	slog.Info("player joined room", "roomCode", room.code, "sessionID", session.sessionID, "playerID", session.playerID, "playerName", cleanName)
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
	if room.pendingDealChoice != nil {
		return roomSnapshot{}, nil, errors.New("dealing choice already pending")
	}
	if dealerIndex < 0 || dealerIndex >= len(room.players) {
		return roomSnapshot{}, nil, game.ErrInvalidDealer
	}

	chooserIndex := (dealerIndex - 1 + len(room.players)) % len(room.players)
	room.pendingDealChoice = &pendingDealChoice{dealerIndex: dealerIndex, chooserIndex: chooserIndex}

	slog.Info("game start requested", "roomCode", room.code, "sessionID", session.sessionID, "dealerIndex", dealerIndex, "chooserIndex", chooserIndex, "playerCount", len(room.players))
	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) chooseDealing(sessionID, dealType string) (roomSnapshot, []gameStateRecipient, error) {
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
	if room.pendingDealChoice == nil {
		return roomSnapshot{}, nil, errors.New("no dealing choice is pending")
	}
	chooser := room.players[room.pendingDealChoice.chooserIndex]
	if chooser == nil || chooser.player.ID != session.playerID {
		return roomSnapshot{}, nil, errors.New("only the deal chooser can choose dealing type")
	}

	switch normalizeDealType(dealType) {
	case "round_robin":
		if err := room.gameState.StartGame(
			room.pendingDealChoice.dealerIndex,
			room.pendingDealChoice.chooserIndex,
			game.DealRoundRobin,
			nil,
			0,
		); err != nil {
			return roomSnapshot{}, nil, err
		}
	case "tap":
		return roomSnapshot{}, nil, errors.New("tap dealing is not available yet")
	default:
		return roomSnapshot{}, nil, game.ErrInvalidDealingType
	}

	room.pendingDealChoice = nil
	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	return roomState, recipients, nil
}

func (l *lobbyServer) startNextRound(sessionID string) (roomSnapshot, []gameStateRecipient, error) {
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
		return roomSnapshot{}, nil, errors.New("only the host can start the next round")
	}
	if !room.allPlayersConnected() {
		return roomSnapshot{}, nil, errors.New("all players must be connected")
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, errors.New("game state not initialized")
	}
	if err := room.gameState.StartNextRound(game.DealRoundRobin, nil, 0); err != nil {
		return roomSnapshot{}, nil, err
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	return roomState, recipients, nil
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
		slog.Info("room disbanded", "roomCode", roomCode, "sessionID", session.sessionID)
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
	room.clearTurnTracking()

	slog.Info("player left room",
		"roomCode", roomCode,
		"sessionID", session.sessionID,
		"remainingPlayers", len(updatedPlayers),
		"newHostID", nextHostID,
	)
	snapshot := room.snapshot()
	return &snapshot, room.connectedConns(l.sessions), roomCode, nil
}

func (l *lobbyServer) draw(sessionID, source string) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "draw", func(state *game.GameState) error {
		switch source {
		case "deck":
			return state.DrawFromDeck()
		case "discard":
			return state.DrawFromDiscard()
		default:
			return errors.New("unknown draw source")
		}
	}, func(room *room, session *playerSession) error {
		room.resetTurnTracking(session.playerID)
		return nil
	})
}

func (l *lobbyServer) play(sessionID string, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "play", func(state *game.GameState) error {
		return state.PlayTable(comps, additions, reclaims...)
	}, func(room *room, session *playerSession) error {
		room.applySubmittedTurnActivity(session.playerID, comps, additions, reclaims)
		return nil
	})
}

func (l *lobbyServer) discard(sessionID string, cardIndex int) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "discard", func(state *game.GameState) error {
		return state.DiscardFromHand(cardIndex)
	}, func(room *room, _ *playerSession) error {
		room.clearTurnTracking()
		return nil
	})
}

func (l *lobbyServer) updateDraftActivity(sessionID string, drafts []game.DraftCompositionSnapshot) (roomSnapshot, []gameStateRecipient, error) {
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
	if room.gameState == nil {
		return roomSnapshot{}, nil, errors.New("game state not initialized")
	}
	if room.gameState.Phase() != game.PhaseInProgress {
		return roomSnapshot{}, nil, game.ErrGameNotInProgress
	}
	if !room.isCurrentTurn(session.playerID) {
		return roomSnapshot{}, nil, errors.New("not your turn")
	}
	if room.turnActivity == nil || room.turnActivity.PlayerID != session.playerID {
		room.resetTurnTracking(session.playerID)
	}
	if room.turnActivity != nil {
		room.turnActivity.DraftCompositions = cloneDraftCompositionSnapshots(drafts)
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	return roomState, recipients, nil
}

func (l *lobbyServer) applyGameAction(sessionID, action string, mutate func(*game.GameState) error, afterMutate func(*room, *playerSession) error) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("join a room first")
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("game state not initialized")
	}
	if room.gameState.Phase() != game.PhaseInProgress {
		return roomSnapshot{}, nil, actionResultEvent{}, game.ErrGameNotInProgress
	}
	if !room.isCurrentTurn(session.playerID) {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("not your turn")
	}
	if err := mutate(room.gameState); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	if afterMutate != nil {
		if err := afterMutate(room, session); err != nil {
			return roomSnapshot{}, nil, actionResultEvent{}, err
		}
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: action, PlayerID: session.playerID, OK: true}
	return roomState, recipients, result, nil
}

func (l *lobbyServer) resetRoomAfterGameOver(roomCode string) (*roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	room := l.rooms[normalizeRoomCode(roomCode)]
	if room == nil {
		return nil, nil, errors.New("room not found")
	}
	if room.gameState == nil {
		return nil, nil, errors.New("game state not initialized")
	}
	if room.gameStatePhase() != game.PhaseGameOver {
		return nil, nil, errors.New("game is not over")
	}
	if err := room.resetForLobby(); err != nil {
		return nil, nil, err
	}

	snapshot := room.snapshot()
	return &snapshot, room.connectedConns(l.sessions), nil
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
		slog.Info("client disconnected", "sessionID", sessionID)
		return
	}

	room := l.sessionRoom(session)
	if room == nil {
		l.mu.Unlock()
		slog.Info("client disconnected", "sessionID", sessionID)
		return
	}

	player := room.playerByID(session.playerID)
	player.connected = false

	roomState = room.snapshot()
	recipients = room.connectedConns(l.sessions)
	shouldBroadcast = len(recipients) > 0
	l.mu.Unlock()

	slog.Info("client disconnected from room", "sessionID", sessionID, "roomCode", room.code)
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

func (l *lobbyServer) sessionByAuthUserID(authUserID string) *playerSession {
	for _, session := range l.sessions {
		if session != nil && session.authenticated && session.authUserID == authUserID {
			return session
		}
	}
	return nil
}

func (s *playerSession) playerName(fallback string) (string, error) {
	if s != nil && s.authenticated {
		return normalizePlayerName(s.displayName)
	}
	return normalizePlayerName(fallback)
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

func (l *lobbyServer) gameStateForSession(sessionID string, roomState roomSnapshot) (*gameStateEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return nil, err
	}
	room := l.sessionRoom(session)
	if room == nil || room.gameState == nil || room.gameStatePhase() == game.PhaseLobby {
		return nil, nil
	}

	state, ok := room.gameState.SnapshotForPlayer(session.playerID)
	if !ok {
		return nil, errors.New("game state snapshot failed")
	}

	event := gameStateEvent{Room: roomState, Game: state}
	return &event, nil
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

func normalizeDealType(dealType string) string {
	return strings.ToLower(strings.TrimSpace(dealType))
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
			ImageURL:     player.imageURL,
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
	if r.pendingDealChoice != nil {
		snapshot.PendingDealChoice = &pendingDealChoiceSnapshot{
			DealerIndex:     r.pendingDealChoice.dealerIndex,
			ChooserIndex:    r.pendingDealChoice.chooserIndex,
			ChooserPlayerID: r.players[r.pendingDealChoice.chooserIndex].player.ID,
		}
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

func (r *room) resetForLobby() error {
	nextGameState := makeGameState()
	if nextGameState == nil {
		return errors.New("game state not initialized")
	}

	for _, player := range r.players {
		if player == nil {
			continue
		}

		player.player = newPlayerWithID(player.player.ID)
		if err := addPlayerToGameState(nextGameState, player.player); err != nil {
			return err
		}
	}

	r.gameState = nextGameState
	r.pendingDealChoice = nil
	r.clearTurnTracking()
	return nil
}

func (r *room) gameStateRecipients(sessions map[string]*playerSession, roomState roomSnapshot) ([]gameStateRecipient, error) {
	recipients := make([]gameStateRecipient, 0, len(r.players))
	for _, player := range r.players {
		if player == nil || !player.connected {
			continue
		}
		session := sessions[player.sessionID]
		if session == nil || session.conn == nil {
			continue
		}
		state, ok := r.gameState.SnapshotForPlayer(player.player.ID)
		if !ok {
			return nil, errors.New("game state snapshot failed")
		}
		if baseline, activity := r.turnContextForPlayer(player.player.ID); baseline != nil {
			state.TurnActivity = activity
		}
		recipients = append(recipients, gameStateRecipient{
			conn:  session.conn,
			event: gameStateEvent{Room: roomState, Game: state},
		})
	}
	return recipients, nil
}

func (r *room) resetTurnTracking(playerID string) {
	if r == nil || r.gameState == nil {
		return
	}
	baseline, ok := r.gameState.SnapshotForPlayer(playerID)
	if !ok {
		return
	}
	r.turnBaseline = &baseline
	r.turnActivity = &game.TurnActivitySnapshot{
		PlayerID:   playerID,
		Round:      r.gameState.RoundNumber(),
		TurnNumber: r.gameState.TurnNumber(),
	}
}

func (r *room) clearTurnTracking() {
	if r == nil {
		return
	}
	r.turnBaseline = nil
	r.turnActivity = nil
}

func (r *room) turnContextForPlayer(playerID string) (*game.GameSnapshot, *game.TurnActivitySnapshot) {
	if r == nil || r.turnBaseline == nil || r.turnActivity == nil {
		return nil, nil
	}
	if r.turnActivity.PlayerID == playerID {
		return nil, nil
	}
	activity := *r.turnActivity
	activity.BaselineCompositions = cloneCompositionSnapshots(r.turnBaseline.ActiveCompositions)
	activity.DraftCompositions = cloneDraftCompositionSnapshots(r.turnActivity.DraftCompositions)
	activity.CompositionActivities = cloneCompositionActivitySnapshots(r.turnActivity.CompositionActivities)
	baseline := *r.turnBaseline
	baseline.ActiveCompositions = cloneCompositionSnapshots(r.turnBaseline.ActiveCompositions)
	return &baseline, &activity
}

func (r *room) applySubmittedTurnActivity(playerID string, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) {
	if r == nil || r.turnActivity == nil || r.turnBaseline == nil {
		return
	}
	if r.turnActivity.PlayerID != playerID {
		return
	}
	r.turnActivity.DraftCompositions = buildDraftSnapshotsFromSubmitted(comps, additions)
	mergeCompositionActivities(r.turnActivity, buildCompositionActivities(playerID, comps, additions, reclaims, r.gameState.ActiveCompositions()))
}

func buildDraftSnapshotsFromSubmitted(comps []*game.Composition, additions []game.CompositionAddition) []game.DraftCompositionSnapshot {
	drafts := make([]game.DraftCompositionSnapshot, 0, len(comps)+len(additions))
	for _, comp := range comps {
		if comp == nil {
			continue
		}
		snapshot := comp.Snapshot()
		cards := make([]game.CardSnapshot, len(snapshot.Cards))
		copy(cards, snapshot.Cards)
		drafts = append(drafts, game.DraftCompositionSnapshot{Cards: cards})
	}
	for _, addition := range additions {
		cards := make([]game.CardSnapshot, 0, len(addition.Cards))
		for _, card := range addition.Cards {
			cards = append(cards, card.Snapshot())
		}
		index := addition.CompositionIndex
		drafts = append(drafts, game.DraftCompositionSnapshot{TableIndex: &index, InsertIndex: addition.InsertIndex, Cards: cards})
	}
	return drafts
}

func mergeCompositionActivities(target *game.TurnActivitySnapshot, updates []game.CompositionActivitySnapshot) {
	if target == nil || len(updates) == 0 {
		return
	}
	merged := cloneCompositionActivitySnapshots(target.CompositionActivities)
	byIndex := make(map[int]int, len(merged))
	for i := range merged {
		byIndex[merged[i].TableIndex] = i
	}
	for _, update := range updates {
		if existingIndex, ok := byIndex[update.TableIndex]; ok {
			existing := &merged[existingIndex]
			if existing.Kind == "" {
				existing.Kind = update.Kind
			}
			if existing.PlayerID == "" {
				existing.PlayerID = update.PlayerID
			}
			if len(update.CardActivities) > 0 {
				if existing.CardActivities == nil {
					existing.CardActivities = map[int]game.CardActivitySnapshot{}
				}
				for index, activity := range update.CardActivities {
					existing.CardActivities[index] = activity
				}
			}
			continue
		}
		merged = append(merged, update)
		byIndex[update.TableIndex] = len(merged) - 1
	}
	target.CompositionActivities = merged
}

func buildCompositionActivities(playerID string, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim, active []*game.Composition) []game.CompositionActivitySnapshot {
	activities := make([]game.CompositionActivitySnapshot, 0, len(comps)+len(additions)+len(reclaims))
	newCount := len(active) - len(comps)
	if newCount < 0 {
		newCount = 0
	}
	for offset, comp := range comps {
		if comp == nil {
			continue
		}
		compSnapshot := comp.Snapshot()
		tableIndex := newCount + offset
		cardActivities := make(map[int]game.CardActivitySnapshot, len(compSnapshot.Cards))
		for index := range compSnapshot.Cards {
			cardActivities[index] = game.CardActivitySnapshot{Kind: "new", PlayerID: playerID}
		}
		activities = append(activities, game.CompositionActivitySnapshot{
			TableIndex:     tableIndex,
			Kind:           "new_composition",
			PlayerID:       playerID,
			CardActivities: cardActivities,
		})
	}

	activityByIndex := make(map[int]int)
	for i := range activities {
		activityByIndex[activities[i].TableIndex] = i
	}
	for _, addition := range additions {
		activityIndex, ok := activityByIndex[addition.CompositionIndex]
		if !ok {
			activities = append(activities, game.CompositionActivitySnapshot{TableIndex: addition.CompositionIndex})
			activityIndex = len(activities) - 1
			activityByIndex[addition.CompositionIndex] = activityIndex
		}
		activity := &activities[activityIndex]
		if activity.CardActivities == nil {
			activity.CardActivities = map[int]game.CardActivitySnapshot{}
		}
		startIndex := 0
		if addition.CompositionIndex >= 0 && addition.CompositionIndex < len(active) && active[addition.CompositionIndex] != nil {
			startIndex = len(active[addition.CompositionIndex].Snapshot().Cards) - len(addition.Cards)
			if startIndex < 0 {
				startIndex = 0
			}
		}
		for offset := range addition.Cards {
			activity.CardActivities[startIndex+offset] = game.CardActivitySnapshot{Kind: "addition", PlayerID: playerID}
		}
	}
	for _, reclaim := range reclaims {
		activityIndex, ok := activityByIndex[reclaim.CompositionIndex]
		if !ok {
			activities = append(activities, game.CompositionActivitySnapshot{TableIndex: reclaim.CompositionIndex})
			activityIndex = len(activities) - 1
			activityByIndex[reclaim.CompositionIndex] = activityIndex
		}
		activity := &activities[activityIndex]
		if activity.CardActivities == nil {
			activity.CardActivities = map[int]game.CardActivitySnapshot{}
		}
		activity.CardActivities[reclaim.JokerIndex] = game.CardActivitySnapshot{Kind: "joker_reclaim", PlayerID: playerID}
	}
	return activities
}

func cloneCompositionSnapshots(source []game.CompositionSnapshot) []game.CompositionSnapshot {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]game.CompositionSnapshot, 0, len(source))
	for _, comp := range source {
		next := comp
		next.Cards = append([]game.CardSnapshot(nil), comp.Cards...)
		if len(comp.JokerRepresentations) > 0 {
			next.JokerRepresentations = make(map[int][]game.CardSnapshot, len(comp.JokerRepresentations))
			for index, cards := range comp.JokerRepresentations {
				next.JokerRepresentations[index] = append([]game.CardSnapshot(nil), cards...)
			}
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneDraftCompositionSnapshots(source []game.DraftCompositionSnapshot) []game.DraftCompositionSnapshot {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]game.DraftCompositionSnapshot, 0, len(source))
	for _, draft := range source {
		next := draft
		next.Cards = append([]game.CardSnapshot(nil), draft.Cards...)
		if draft.TableIndex != nil {
			index := *draft.TableIndex
			next.TableIndex = &index
		}
		if draft.InsertIndex != nil {
			index := *draft.InsertIndex
			next.InsertIndex = &index
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneCompositionActivitySnapshots(source []game.CompositionActivitySnapshot) []game.CompositionActivitySnapshot {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]game.CompositionActivitySnapshot, 0, len(source))
	for _, activity := range source {
		next := activity
		if len(activity.CardActivities) > 0 {
			next.CardActivities = make(map[int]game.CardActivitySnapshot, len(activity.CardActivities))
			for index, cardActivity := range activity.CardActivities {
				next.CardActivities[index] = cardActivity
			}
		}
		cloned = append(cloned, next)
	}
	return cloned
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

func (r *room) isCurrentTurn(playerID string) bool {
	if r == nil || r.gameState == nil {
		return false
	}
	playerIndex := r.gameState.CurrentPlayerIndex()
	if playerIndex < 0 || playerIndex >= len(r.players) || r.players[playerIndex] == nil {
		return false
	}
	return r.players[playerIndex].player.ID == playerID
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

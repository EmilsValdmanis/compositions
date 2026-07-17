package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	minPlayersToStart   = 2
	maxPlayersPerRoom   = 4
	roomCodeLength      = 6
	playerEmoteTTL      = 4 * time.Second
	endProposalTTL      = 90 * time.Second
	endProposalCooldown = 30 * time.Second
	statisticsIdleLimit = 15 * time.Minute
	maxIssueLength      = 500
)

var allowedPlayerEmotes = map[string]struct{}{
	"👋":  {},
	"👍":  {},
	"😂":  {},
	"😅":  {},
	"🤔":  {},
	"😮":  {},
	"😡":  {},
	"👀":  {},
	"😭":  {},
	"🔥":  {},
	"❤️": {},
	"🎉":  {},
}

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
	player      *game.Player
	authUserID  string
	name        string
	imageURL    string
	sessionID   string
	connected   bool
	seat        int
	host        bool
	activeEmote *playerEmote
	forfeited   bool
}

type playerEmote struct {
	id        string
	emoji     string
	expiresAt time.Time
}

type room struct {
	code                     string
	gameState                *game.GameState
	players                  []*roomPlayer
	hostID                   string
	pendingDealChoice        *pendingDealChoice
	turnBaseline             *game.GameSnapshot
	turnActivity             *game.TurnActivitySnapshot
	endProposal              *endGameProposal
	conclusion               *gameConclusion
	statisticsGameID         string
	statisticsStartedAt      time.Time
	statisticsPlaytime       time.Duration
	statisticsActiveSince    time.Time
	statisticsSaved          bool
	statisticsDirty          bool
	endProposalCooldownUntil time.Time
}

type endGameProposal struct {
	id                string
	kind              string
	proposerPlayerID  string
	description       string
	reportID          string
	eligiblePlayerIDs []string
	agreedPlayerIDs   map[string]bool
	createdAt         time.Time
	expiresAt         time.Time
}

type gameConclusion struct {
	kind           string
	winnerPlayerID string
	reportID       string
}

type pendingDealChoice struct {
	dealerIndex  int
	chooserIndex int
	gameMode     game.GameMode
}

type dealingChoiceOptions struct {
	order   []int
	cutSize *int
}

const persistedLobbyStateVersion = 1

type persistedLobbyState struct {
	Version  int                      `json:"version"`
	Sessions []persistedPlayerSession `json:"sessions"`
	Rooms    []persistedRoom          `json:"rooms"`
}

type persistedPlayerSession struct {
	SessionID     string `json:"sessionId"`
	PlayerID      string `json:"playerId"`
	RoomCode      string `json:"roomCode,omitempty"`
	AuthUserID    string `json:"authUserId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	ImageURL      string `json:"imageUrl,omitempty"`
	Authenticated bool   `json:"authenticated"`
}

type persistedRoom struct {
	Code                     string                      `json:"code"`
	GameState                game.PersistenceSnapshot    `json:"gameState"`
	Players                  []persistedRoomPlayer       `json:"players"`
	HostID                   string                      `json:"hostId"`
	PendingDealChoice        *persistedPendingDealChoice `json:"pendingDealChoice,omitempty"`
	TurnBaseline             *game.GameSnapshot          `json:"turnBaseline,omitempty"`
	TurnActivity             *game.TurnActivitySnapshot  `json:"turnActivity,omitempty"`
	EndProposal              *persistedEndGameProposal   `json:"endProposal,omitempty"`
	Conclusion               *persistedGameConclusion    `json:"conclusion,omitempty"`
	StatisticsGameID         string                      `json:"statisticsGameId,omitempty"`
	StatisticsStartedAt      time.Time                   `json:"statisticsStartedAt,omitempty"`
	StatisticsPlaytime       time.Duration               `json:"statisticsPlaytime,omitempty"`
	StatisticsActiveSince    time.Time                   `json:"statisticsActiveSince,omitempty"`
	StatisticsSaved          bool                        `json:"statisticsSaved,omitempty"`
	StatisticsDirty          bool                        `json:"statisticsDirty,omitempty"`
	EndProposalCooldownUntil time.Time                   `json:"endProposalCooldownUntil,omitempty"`
}

type persistedPendingDealChoice struct {
	DealerIndex  int           `json:"dealerIndex"`
	ChooserIndex int           `json:"chooserIndex"`
	GameMode     game.GameMode `json:"gameMode,omitempty"`
}

type persistedRoomPlayer struct {
	PlayerID  string `json:"playerId"`
	Name      string `json:"name"`
	ImageURL  string `json:"imageUrl,omitempty"`
	SessionID string `json:"sessionId"`
	Connected bool   `json:"connected"`
	Seat      int    `json:"seat"`
	Host      bool   `json:"host"`
	Forfeited bool   `json:"forfeited,omitempty"`
}

type persistedEndGameProposal struct {
	ID                string    `json:"id"`
	Kind              string    `json:"kind"`
	ProposerPlayerID  string    `json:"proposerPlayerId"`
	Description       string    `json:"description,omitempty"`
	ReportID          string    `json:"reportId,omitempty"`
	EligiblePlayerIDs []string  `json:"eligiblePlayerIds"`
	AgreedPlayerIDs   []string  `json:"agreedPlayerIds"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
}

type persistedGameConclusion struct {
	Kind           string `json:"kind"`
	WinnerPlayerID string `json:"winnerPlayerId,omitempty"`
	ReportID       string `json:"reportId,omitempty"`
}

type gameStateRecipient struct {
	conn  *websocket.Conn
	event gameStateEvent
}

type lobbyServer struct {
	mu       sync.Mutex
	rng      *rand.Rand
	store    userStore
	sessions map[string]*playerSession
	rooms    map[string]*room
}

var makeGameState = game.NewGameState
var addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
	return state.AddPlayer(player)
}

func newLobbyServer() *lobbyServer {
	return newLobbyServerWithStore(noopUserStore{})
}

func newLobbyServerWithStore(store userStore) *lobbyServer {
	if store == nil {
		store = noopUserStore{}
	}
	return &lobbyServer{
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		store:    store,
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

	if user.isAuthenticated() {
		if existingSessionID != "" {
			if session := l.sessions[existingSessionID]; session != nil && session.authenticated && session.authUserID == user.ID {
				return l.connectExistingSessionWithUser(existingSessionID, user, conn)
			}
			slog.Warn("ignoring session id that does not belong to authenticated user", "sessionID", existingSessionID, "userID", user.ID)
			existingSessionID = ""
		}
		if existingSession := l.sessionByAuthUserID(user.ID); existingSession != nil {
			return l.connectExistingSessionWithUser(existingSession.sessionID, user, conn)
		}
	}
	if existingSessionID != "" {
		return l.connectExistingSessionWithUser(existingSessionID, user, conn)
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
	l.persistLocked("session created")
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

	l.persistLocked("session reconnected")
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
		player:     newPlayerWithID(session.playerID),
		authUserID: session.authUserID,
		name:       cleanName,
		imageURL:   session.imageURL,
		sessionID:  session.sessionID,
		connected:  true,
		seat:       0,
		host:       true,
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
	l.persistLocked("room created")
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
		player:     newPlayerWithID(session.playerID),
		authUserID: session.authUserID,
		name:       cleanName,
		imageURL:   session.imageURL,
		sessionID:  session.sessionID,
		connected:  true,
		seat:       len(room.players),
	}
	if err := addPlayerToGameState(room.gameState, player.player); err != nil {
		return roomSnapshot{}, nil, err
	}

	room.players = append(room.players, player)
	session.roomCode = room.code

	slog.Info("player joined room", "roomCode", room.code, "sessionID", session.sessionID, "playerID", session.playerID, "playerName", cleanName)
	l.persistLocked("player joined room")
	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) startGame(sessionID string, dealerIndex int, requestedModes ...game.GameMode) (roomSnapshot, []*websocket.Conn, error) {
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
	gameMode := game.GameModeFull
	if len(requestedModes) > 0 {
		gameMode = requestedModes[0]
	}
	if !gameMode.Valid() {
		return roomSnapshot{}, nil, game.ErrInvalidGameMode
	}

	chooserIndex := (dealerIndex - 1 + len(room.players)) % len(room.players)
	room.pendingDealChoice = &pendingDealChoice{dealerIndex: dealerIndex, chooserIndex: chooserIndex, gameMode: gameMode}

	slog.Info("game start requested", "roomCode", room.code, "sessionID", session.sessionID, "dealerIndex", dealerIndex, "chooserIndex", chooserIndex, "playerCount", len(room.players), "gameMode", gameMode)
	l.persistLocked("game start requested")
	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) chooseDealing(sessionID, dealType string, options ...dealingChoiceOptions) (roomSnapshot, []gameStateRecipient, error) {
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
	if room.gameState == nil {
		return roomSnapshot{}, nil, errors.New("game state not initialized")
	}

	choiceOptions := dealingChoiceOptions{}
	if len(options) > 0 {
		choiceOptions = options[0]
	}
	if choiceOptions.cutSize == nil {
		return roomSnapshot{}, nil, errors.New("cut size is required")
	}

	cutSize := *choiceOptions.cutSize

	startingGame := room.gameStatePhase() == game.PhaseLobby
	startRound := func(dt game.DealTypes, order []int) error {
		switch room.gameStatePhase() {
		case game.PhaseLobby:
			gameMode := room.pendingDealChoice.gameMode
			if gameMode == "" {
				gameMode = game.GameModeFull
			}
			return room.gameState.StartGameWithMode(
				room.pendingDealChoice.dealerIndex,
				room.pendingDealChoice.chooserIndex,
				dt,
				order,
				cutSize,
				gameMode,
			)
		case game.PhaseRoundOver:
			return room.gameState.StartNextRound(dt, order, cutSize)
		default:
			return errors.New("game is not waiting for a dealing choice")
		}
	}

	switch normalizeDealType(dealType) {
	case "round_robin":
		if err := startRound(game.DealRoundRobin, nil); err != nil {
			return roomSnapshot{}, nil, err
		}
	case "tap":
		if err := startRound(game.DealInBlocks, choiceOptions.order); err != nil {
			return roomSnapshot{}, nil, err
		}
	default:
		return roomSnapshot{}, nil, game.ErrInvalidDealingType
	}

	room.pendingDealChoice = nil
	if startingGame {
		room.statisticsGameID = uuid.NewString()
		room.statisticsStartedAt = time.Now().UTC()
		room.statisticsPlaytime = 0
		room.statisticsActiveSince = time.Time{}
		room.statisticsSaved = false
		room.statisticsDirty = true
	}
	if room.gameStatePhase() == game.PhaseRoundOver || room.gameStatePhase() == game.PhaseGameOver {
		room.statisticsDirty = true
	}
	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	l.persistLocked("dealing chosen")
	return roomState, recipients, nil
}

func (l *lobbyServer) startNextRound(sessionID string) (roomSnapshot, []*websocket.Conn, error) {
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
	if room.pendingDealChoice != nil {
		return roomSnapshot{}, nil, errors.New("dealing choice already pending")
	}
	if room.gameStatePhase() != game.PhaseRoundOver {
		return roomSnapshot{}, nil, game.ErrCannotStartNextRound
	}

	dealerIndex, chooserIndex, err := room.gameState.NextRoundDealerAndChooser()
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	room.pendingDealChoice = &pendingDealChoice{dealerIndex: dealerIndex, chooserIndex: chooserIndex}

	roomState := room.snapshot()
	l.persistLocked("next round requested")
	return roomState, room.connectedConns(l.sessions), nil
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
		l.persistLocked("room disbanded")
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
	l.persistLocked("player left room")
	return &snapshot, room.connectedConns(l.sessions), roomCode, nil
}

func (l *lobbyServer) forfeitGame(sessionID string) (roomSnapshot, []gameStateRecipient, actionResultEvent, string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", errors.New("join a room first")
	}
	player := room.playerByID(session.playerID)
	if player == nil || player.forfeited {
		return roomSnapshot{}, nil, actionResultEvent{}, "", errors.New("player is not active")
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", errors.New("game state not initialized")
	}

	winnerPlayerID, err := room.gameState.ForfeitPlayer(session.playerID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", err
	}
	player.forfeited = true
	player.connected = false
	player.activeEmote = nil
	session.roomCode = ""
	room.pendingDealChoice = nil
	room.clearTurnTracking()
	room.statisticsDirty = true

	if room.hostID == session.playerID {
		room.transferHostToFirstActive()
	}
	room.removePlayerFromProposal(session.playerID)
	if winnerPlayerID != "" {
		room.conclusion = &gameConclusion{kind: "forfeit", winnerPlayerID: winnerPlayerID}
		room.endProposal = nil
	} else if room.endProposal != nil && room.endProposal.unanimouslyApproved() {
		proposal := room.endProposal
		// ForfeitPlayer succeeded and left multiple active players, so the game
		// is guaranteed to still be in a phase EndWithoutWinner accepts.
		_ = room.gameState.EndWithoutWinner()
		room.conclusion = &gameConclusion{kind: proposal.kind, reportID: proposal.reportID}
		room.endProposal = nil
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", err
	}
	result := actionResultEvent{Action: "forfeit_game", PlayerID: session.playerID, OK: true}
	l.persistLocked("player forfeited game")
	return roomState, recipients, result, room.code, nil
}

func (l *lobbyServer) requestEndGame(sessionID, kind string) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.createEndProposal(sessionID, kind, "", "")
}

func (l *lobbyServer) reportIssue(sessionID, description string, requestAbort bool) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
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
	if room.gameState == nil || (room.gameStatePhase() != game.PhaseInProgress && room.gameStatePhase() != game.PhaseRoundOver) {
		return roomSnapshot{}, nil, actionResultEvent{}, game.ErrGameNotInProgress
	}
	if player := room.playerByID(session.playerID); player == nil || player.forfeited {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("player is not active")
	}
	if requestAbort {
		if room.activeEndProposal(time.Now()) != nil {
			return roomSnapshot{}, nil, actionResultEvent{}, errors.New("an end-game request is already active")
		}
		if time.Now().Before(room.endProposalCooldownUntil) {
			return roomSnapshot{}, nil, actionResultEvent{}, errors.New("wait before starting another end-game request")
		}
	}

	cleanDescription, err := normalizeIssueDescription(description)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	gameStateJSON, _ := json.Marshal(room.gameState.PersistenceSnapshot())
	reportID := uuid.NewString()
	reportCreatedAt := time.Now().UTC()
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	savedReport, err := l.store.CreateGameBugReport(ctx, database.GameBugReportRecord{
		ID:               reportID,
		RoomCode:         room.code,
		ReporterPlayerID: session.playerID,
		ReporterUserID:   session.authUserID,
		Description:      cleanDescription,
		GameState:        gameStateJSON,
		Round:            room.gameState.RoundNumber(),
		Turn:             room.gameState.TurnNumber(),
		RequestedAbort:   requestAbort,
		CreatedAt:        reportCreatedAt,
	})
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, fmt.Errorf("save bug report: %w", err)
	}
	slog.Warn("game issue reported", "roomCode", room.code, "reportID", savedReport.ID, "playerID", session.playerID, "description", cleanDescription)

	if requestAbort {
		room.endProposal = room.newEndProposal("technical_abort", session.playerID, cleanDescription, savedReport.ID)
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: "report_issue", PlayerID: session.playerID, OK: true}
	l.persistLocked("game issue reported")
	return roomState, recipients, result, nil
}

func (l *lobbyServer) createEndProposal(sessionID, kind, description, reportID string) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
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
	if room.gameState == nil || (room.gameStatePhase() != game.PhaseInProgress && room.gameStatePhase() != game.PhaseRoundOver) {
		return roomSnapshot{}, nil, actionResultEvent{}, game.ErrGameNotInProgress
	}
	if kind != "mutual_end" {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("unknown end-game request type")
	}
	if player := room.playerByID(session.playerID); player == nil || player.forfeited {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("player is not active")
	}
	if room.activeEndProposal(time.Now()) != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("an end-game request is already active")
	}
	if time.Now().Before(room.endProposalCooldownUntil) {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("wait before starting another end-game request")
	}

	room.endProposal = room.newEndProposal(kind, session.playerID, description, reportID)
	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: "request_end_game", PlayerID: session.playerID, OK: true}
	l.persistLocked("end game requested")
	return roomState, recipients, result, nil
}

func (l *lobbyServer) voteEndGame(sessionID, proposalID string, approve bool) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
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
	proposal := room.activeEndProposal(time.Now())
	if proposal == nil || proposal.id != strings.TrimSpace(proposalID) {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("end-game request not found or expired")
	}
	if !slices.Contains(proposal.eligiblePlayerIDs, session.playerID) {
		return roomSnapshot{}, nil, actionResultEvent{}, errors.New("player is not eligible to vote")
	}

	if !approve {
		room.endProposal = nil
		room.endProposalCooldownUntil = time.Now().Add(endProposalCooldown)
	} else {
		proposal.agreedPlayerIDs[session.playerID] = true
		if proposal.unanimouslyApproved() {
			if err := room.gameState.EndWithoutWinner(); err != nil {
				return roomSnapshot{}, nil, actionResultEvent{}, err
			}
			room.conclusion = &gameConclusion{kind: proposal.kind, reportID: proposal.reportID}
			room.endProposal = nil
			room.pendingDealChoice = nil
			room.clearTurnTracking()
			room.statisticsDirty = true
		}
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: "vote_end_game", PlayerID: session.playerID, OK: true}
	l.persistLocked("end game vote recorded")
	return roomState, recipients, result, nil
}

func (l *lobbyServer) sendEmote(sessionID, emoji string) (roomSnapshot, []*websocket.Conn, error) {
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
	if _, ok := allowedPlayerEmotes[emoji]; !ok {
		return roomSnapshot{}, nil, errors.New("unknown emote")
	}

	player := room.playerByID(session.playerID)
	player.activeEmote = &playerEmote{
		id:        uuid.NewString(),
		emoji:     emoji,
		expiresAt: time.Now().Add(playerEmoteTTL),
	}

	return room.snapshot(), room.connectedConns(l.sessions), nil
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
		if room.turnActivity != nil {
			room.turnActivity.DrawSource = source
		}
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

func (l *lobbyServer) playAndDiscard(sessionID string, cardIndex int, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.playAndDiscardMatching(sessionID, cardIndex, nil, comps, additions, reclaims)
}

func (l *lobbyServer) playAndDiscardMatching(sessionID string, cardIndex int, expectedCard *game.Card, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "play_and_discard", func(state *game.GameState) error {
		return state.PlayTableAndDiscardMatching(cardIndex, expectedCard, comps, additions, reclaims...)
	}, func(room *room, _ *playerSession) error {
		room.clearTurnTracking()
		return nil
	})
}

func (l *lobbyServer) discard(sessionID string, cardIndex int) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.discardMatching(sessionID, cardIndex, nil)
}

func (l *lobbyServer) discardMatching(sessionID string, cardIndex int, expectedCard *game.Card) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "discard", func(state *game.GameState) error {
		return state.DiscardFromHandMatching(cardIndex, expectedCard)
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
	l.persistLocked("draft activity updated")
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
	if room.gameStatePhase() == game.PhaseRoundOver || room.gameStatePhase() == game.PhaseGameOver {
		room.statisticsDirty = true
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: action, PlayerID: session.playerID, OK: true}
	l.persistLocked("game action applied")
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
	l.persistLocked("room reset after game over")
	return &snapshot, room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) disconnect(sessionID string, conn *websocket.Conn) {
	l.disconnectWithEmitter(sessionID, conn, emitEvent)
}

func (l *lobbyServer) disconnectWithEmitter(sessionID string, conn *websocket.Conn, emitter eventEmitter) {
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
	l.persistLocked("session disconnected")

	roomState = room.snapshot()
	recipients = room.connectedConns(l.sessions)
	shouldBroadcast = len(recipients) > 0
	l.mu.Unlock()

	slog.Info("client disconnected from room", "sessionID", sessionID, "roomCode", room.code)
	if shouldBroadcast {
		l.broadcastDisconnectWithEmitter(roomState, recipients, emitter)
	}
}

func (l *lobbyServer) broadcastDisconnect(roomState roomSnapshot, recipients []*websocket.Conn) {
	l.broadcastDisconnectWithEmitter(roomState, recipients, emitEvent)
}

type eventEmitter func(conn *websocket.Conn, messageType string, data any) error

func (l *lobbyServer) broadcastDisconnectWithEmitter(roomState roomSnapshot, recipients []*websocket.Conn, emitter eventEmitter) {
	for _, conn := range recipients {
		if conn == nil {
			continue
		}
		_ = emitter(conn, "room_state", roomStateEvent{Room: roomState})
	}
}

func (l *lobbyServer) restorePersistedState(ctx context.Context) error {
	if l == nil || l.store == nil {
		return nil
	}

	state, err := l.store.LoadLobbyState(ctx)
	if err != nil {
		return err
	}
	if state.Version == 0 && len(state.Rooms) == 0 && len(state.Sessions) == 0 {
		return nil
	}
	if state.Version != persistedLobbyStateVersion {
		return errors.New("unsupported lobby persistence snapshot version")
	}

	sessions := make(map[string]*playerSession, len(state.Sessions))
	for _, persistedSession := range state.Sessions {
		if persistedSession.SessionID == "" || persistedSession.PlayerID == "" {
			return errors.New("persisted session requires session and player ids")
		}
		if persistedSession.Authenticated && persistedSession.AuthUserID == "" {
			return errors.New("persisted authenticated session requires auth user id")
		}
		sessions[persistedSession.SessionID] = &playerSession{
			sessionID:     persistedSession.SessionID,
			playerID:      persistedSession.PlayerID,
			roomCode:      normalizeRoomCode(persistedSession.RoomCode),
			authUserID:    persistedSession.AuthUserID,
			displayName:   persistedSession.DisplayName,
			imageURL:      persistedSession.ImageURL,
			authenticated: persistedSession.Authenticated,
		}
	}

	rooms := make(map[string]*room, len(state.Rooms))
	for _, persistedRoom := range state.Rooms {
		roomCode := normalizeRoomCode(persistedRoom.Code)
		if roomCode == "" {
			return errors.New("persisted room requires code")
		}
		gameState, err := game.RestoreGameState(persistedRoom.GameState)
		if err != nil {
			return err
		}
		restoredRoom := &room{
			code:                     roomCode,
			gameState:                gameState,
			players:                  make([]*roomPlayer, 0, len(persistedRoom.Players)),
			hostID:                   persistedRoom.HostID,
			turnBaseline:             persistedRoom.TurnBaseline,
			turnActivity:             persistedRoom.TurnActivity,
			endProposalCooldownUntil: persistedRoom.EndProposalCooldownUntil,
			statisticsGameID:         persistedRoom.StatisticsGameID,
			statisticsStartedAt:      persistedRoom.StatisticsStartedAt,
			statisticsPlaytime:       persistedRoom.StatisticsPlaytime,
			statisticsSaved:          persistedRoom.StatisticsSaved,
			statisticsDirty:          persistedRoom.StatisticsDirty,
		}
		if persistedRoom.Conclusion != nil {
			restoredRoom.conclusion = &gameConclusion{
				kind:           persistedRoom.Conclusion.Kind,
				winnerPlayerID: persistedRoom.Conclusion.WinnerPlayerID,
				reportID:       persistedRoom.Conclusion.ReportID,
			}
		}
		if persistedRoom.EndProposal != nil {
			agreedPlayerIDs := make(map[string]bool, len(persistedRoom.EndProposal.AgreedPlayerIDs))
			for _, playerID := range persistedRoom.EndProposal.AgreedPlayerIDs {
				agreedPlayerIDs[playerID] = true
			}
			restoredRoom.endProposal = &endGameProposal{
				id:                persistedRoom.EndProposal.ID,
				kind:              persistedRoom.EndProposal.Kind,
				proposerPlayerID:  persistedRoom.EndProposal.ProposerPlayerID,
				description:       persistedRoom.EndProposal.Description,
				reportID:          persistedRoom.EndProposal.ReportID,
				eligiblePlayerIDs: append([]string(nil), persistedRoom.EndProposal.EligiblePlayerIDs...),
				agreedPlayerIDs:   agreedPlayerIDs,
				createdAt:         persistedRoom.EndProposal.CreatedAt,
				expiresAt:         persistedRoom.EndProposal.ExpiresAt,
			}
		}
		if persistedRoom.PendingDealChoice != nil {
			gameMode := persistedRoom.PendingDealChoice.GameMode
			if gameMode == "" {
				gameMode = game.GameModeFull
			}
			if !gameMode.Valid() {
				return errors.New("persisted deal choice has invalid game mode")
			}
			restoredRoom.pendingDealChoice = &pendingDealChoice{
				dealerIndex:  persistedRoom.PendingDealChoice.DealerIndex,
				chooserIndex: persistedRoom.PendingDealChoice.ChooserIndex,
				gameMode:     gameMode,
			}
		}
		for _, persistedPlayer := range persistedRoom.Players {
			if persistedPlayer.PlayerID == "" || persistedPlayer.SessionID == "" {
				return errors.New("persisted room player requires player and session ids")
			}
			if _, ok := sessions[persistedPlayer.SessionID]; !ok {
				return errors.New("persisted room player references missing session")
			}
			restoredRoom.players = append(restoredRoom.players, &roomPlayer{
				player:     newPlayerWithID(persistedPlayer.PlayerID),
				authUserID: sessions[persistedPlayer.SessionID].authUserID,
				name:       persistedPlayer.Name,
				imageURL:   persistedPlayer.ImageURL,
				sessionID:  persistedPlayer.SessionID,
				connected:  false,
				seat:       persistedPlayer.Seat,
				host:       persistedPlayer.Host,
				forfeited:  persistedPlayer.Forfeited,
			})
			if !persistedPlayer.Forfeited {
				sessions[persistedPlayer.SessionID].roomCode = roomCode
			}
		}
		if restoredRoom.hostID == "" && len(restoredRoom.players) > 0 {
			restoredRoom.hostID = restoredRoom.players[0].player.ID
			restoredRoom.players[0].host = true
		}
		if restoredRoom.pendingDealChoice != nil {
			if restoredRoom.pendingDealChoice.dealerIndex < 0 ||
				restoredRoom.pendingDealChoice.dealerIndex >= len(restoredRoom.players) ||
				restoredRoom.pendingDealChoice.chooserIndex < 0 ||
				restoredRoom.pendingDealChoice.chooserIndex >= len(restoredRoom.players) {
				return errors.New("invalid persisted deal choice")
			}
		}
		rooms[roomCode] = restoredRoom
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.sessions = sessions
	l.rooms = rooms

	slog.Info("lobby state restored", "rooms", len(rooms), "sessions", len(sessions))
	return nil
}

func (l *lobbyServer) persistLocked(reason string) {
	if l == nil || l.store == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	now := time.Now().UTC()
	for _, room := range l.rooms {
		room.updateStatisticsPlaytime(now)
	}
	// Persist the authoritative game state with its dirty checkpoint marker
	// before writing derived statistics. If this write fails, a statistics row
	// must not get ahead of the state we would restore after a restart.
	if err := l.store.SaveLobbyState(ctx, l.persistenceSnapshotLocked()); err != nil {
		slog.Error("persist lobby state failed", "reason", reason, "error", err)
		return
	}
	if !l.saveStatisticsLocked(ctx) {
		return
	}
	// Statistics are idempotent. Saving the cleared dirty/finalized marker in a
	// second write means a failure here merely causes a safe retry on restart.
	if err := l.store.SaveLobbyState(ctx, l.persistenceSnapshotLocked()); err != nil {
		slog.Error("persist statistics marker failed", "reason", reason, "error", err)
	}
}

func (l *lobbyServer) saveStatisticsLocked(ctx context.Context) bool {
	store, ok := l.store.(gameStatisticsStore)
	if !ok {
		return false
	}
	changed := false
	for _, room := range l.rooms {
		if room == nil || room.statisticsSaved || room.statisticsGameID == "" {
			continue
		}
		phase := room.gameStatePhase()
		completed := room.gameState.CompletedPlayerStatistics()
		results := completed
		if len(results) == 0 {
			results = room.gameState.PlayerStatistics()
		}
		players := make([]database.CompletedGamePlayerRecord, 0, len(results))
		for _, result := range results {
			roomPlayer := room.playerByID(result.PlayerID)
			if roomPlayer == nil {
				continue
			}
			session := l.sessions[roomPlayer.sessionID]
			if session == nil || !session.authenticated || session.authUserID == "" {
				continue
			}
			s := result.Statistics
			players = append(players, database.CompletedGamePlayerRecord{
				UserID: session.authUserID, Placement: result.Placement,
				Won: result.Winner, Forfeited: result.Forfeited, TotalPoints: result.TotalPoints,
				RoundsPlayed: s.RoundsPlayed, RoundsWon: s.RoundsWon, SameSuitWins: s.SameSuitWins, SixPairsWins: s.SixPairsWins,
				TurnsTaken: s.TurnsTaken, CardsDrawnFromDeck: s.CardsDrawnFromDeck, CardsDrawnFromDiscard: s.CardsDrawnFromDiscard,
				CardsDiscarded: s.CardsDiscarded, CardsPlayed: s.CardsPlayed, CompositionsCreated: s.CompositionsCreated,
				SetsCreated: s.SetsCreated, RunsCreated: s.RunsCreated, AdditionsDone: s.AdditionsDone,
				CompositionsCompleted: s.CompositionsCompleted, SetsCompleted: s.SetsCompleted, RunsCompleted: s.RunsCompleted,
				JokersPlayed: s.JokersPlayed, JokersReclaimed: s.JokersReclaimed, CardsRemaining: s.CardsRemaining,
				HandPoints: s.HandPoints, PenaltyPoints: s.PenaltyPoints, PointsInflicted: s.PointsInflicted,
				LargestRoundPenalty: s.LargestRoundPenalty, LargestRoundPointsInflicted: s.LargestRoundPointsInflicted,
				MostCardsRemaining: s.MostCardsRemaining, RoundsOpened: s.RoundsOpened, FastestOpeningTurn: s.FastestOpeningTurn,
				StartingRoundWinStreak: s.StartingRoundWinStreak, EndingRoundWinStreak: s.CurrentRoundWinStreak,
				LongestRoundWinStreak: s.LongestRoundWinStreak,
			})
		}
		if len(players) == 0 {
			room.statisticsDirty = false
			if phase == game.PhaseGameOver {
				room.statisticsSaved = true
			}
			changed = true
			continue
		}
		startedAt := room.statisticsStartedAt
		if startedAt.IsZero() {
			startedAt = time.Now().UTC()
		}
		checkpoint := database.GameCheckpointRecord{
			ID: room.statisticsGameID, RoomCode: room.code, RoundsPlayed: room.gameState.RoundNumber(),
			GameMode: string(room.gameState.GameMode()), Ranked: room.gameState.GameMode() == game.GameModeFull,
			PlayerCount: len(results), StartedAt: startedAt,
			PlaytimeSeconds: int64(room.statisticsPlaytime / time.Second), Players: players,
		}
		var err error
		switch {
		case phase == game.PhaseGameOver && len(completed) > 0:
			kind := "normal"
			if room.conclusion != nil && room.conclusion.kind == "forfeit" {
				kind = "forfeit"
			}
			err = store.SaveCompletedGame(ctx, database.CompletedGameRecord{
				ID: checkpoint.ID, RoomCode: checkpoint.RoomCode, CompletionKind: kind,
				GameMode: checkpoint.GameMode, Ranked: checkpoint.Ranked,
				RoundsPlayed: checkpoint.RoundsPlayed, PlayerCount: checkpoint.PlayerCount,
				StartedAt: checkpoint.StartedAt, CompletedAt: time.Now().UTC(),
				PlaytimeSeconds: checkpoint.PlaytimeSeconds, Players: players,
			})
		case phase == game.PhaseGameOver:
			status := "abandoned"
			if room.conclusion != nil && (room.conclusion.kind == "mutual_end" || room.conclusion.kind == "technical_abort") {
				status = room.conclusion.kind
			}
			err = store.SaveUnrankedGame(ctx, checkpoint, status, time.Now().UTC())
		case room.statisticsDirty:
			err = store.SaveGameCheckpoint(ctx, checkpoint)
		default:
			continue
		}
		if err != nil {
			slog.Error("persist game statistics failed", "roomCode", room.code, "gameID", room.statisticsGameID, "error", err)
			continue
		}
		room.statisticsDirty = false
		if phase == game.PhaseGameOver {
			room.statisticsSaved = true
		}
		changed = true
	}
	return changed
}

func (l *lobbyServer) persistenceSnapshotLocked() persistedLobbyState {
	state := persistedLobbyState{
		Version:  persistedLobbyStateVersion,
		Sessions: make([]persistedPlayerSession, 0, len(l.sessions)),
		Rooms:    make([]persistedRoom, 0, len(l.rooms)),
	}

	for _, session := range l.sessions {
		if session == nil {
			continue
		}
		state.Sessions = append(state.Sessions, persistedPlayerSession{
			SessionID:     session.sessionID,
			PlayerID:      session.playerID,
			RoomCode:      session.roomCode,
			AuthUserID:    session.authUserID,
			DisplayName:   session.displayName,
			ImageURL:      session.imageURL,
			Authenticated: session.authenticated,
		})
	}
	sort.Slice(state.Sessions, func(i, j int) bool {
		return state.Sessions[i].SessionID < state.Sessions[j].SessionID
	})

	for _, room := range l.rooms {
		if room == nil || room.gameState == nil {
			continue
		}
		persistedRoom := persistedRoom{
			Code:                     room.code,
			GameState:                room.gameState.PersistenceSnapshot(),
			Players:                  make([]persistedRoomPlayer, 0, len(room.players)),
			HostID:                   room.hostID,
			TurnBaseline:             cloneGameSnapshot(room.turnBaseline),
			TurnActivity:             cloneTurnActivitySnapshot(room.turnActivity),
			EndProposalCooldownUntil: room.endProposalCooldownUntil,
			StatisticsGameID:         room.statisticsGameID,
			StatisticsStartedAt:      room.statisticsStartedAt,
			StatisticsPlaytime:       room.statisticsPlaytime,
			StatisticsActiveSince:    room.statisticsActiveSince,
			StatisticsSaved:          room.statisticsSaved,
			StatisticsDirty:          room.statisticsDirty,
		}
		if room.endProposal != nil {
			agreedPlayerIDs := make([]string, 0, len(room.endProposal.agreedPlayerIDs))
			for _, playerID := range room.endProposal.eligiblePlayerIDs {
				if room.endProposal.agreedPlayerIDs[playerID] {
					agreedPlayerIDs = append(agreedPlayerIDs, playerID)
				}
			}
			persistedRoom.EndProposal = &persistedEndGameProposal{
				ID:                room.endProposal.id,
				Kind:              room.endProposal.kind,
				ProposerPlayerID:  room.endProposal.proposerPlayerID,
				Description:       room.endProposal.description,
				ReportID:          room.endProposal.reportID,
				EligiblePlayerIDs: append([]string(nil), room.endProposal.eligiblePlayerIDs...),
				AgreedPlayerIDs:   agreedPlayerIDs,
				CreatedAt:         room.endProposal.createdAt,
				ExpiresAt:         room.endProposal.expiresAt,
			}
		}
		if room.conclusion != nil {
			persistedRoom.Conclusion = &persistedGameConclusion{
				Kind:           room.conclusion.kind,
				WinnerPlayerID: room.conclusion.winnerPlayerID,
				ReportID:       room.conclusion.reportID,
			}
		}
		if room.pendingDealChoice != nil {
			gameMode := room.pendingDealChoice.gameMode
			if gameMode == "" {
				gameMode = game.GameModeFull
			}
			persistedRoom.PendingDealChoice = &persistedPendingDealChoice{
				DealerIndex:  room.pendingDealChoice.dealerIndex,
				ChooserIndex: room.pendingDealChoice.chooserIndex,
				GameMode:     gameMode,
			}
		}
		for _, player := range room.players {
			if player == nil || player.player == nil {
				continue
			}
			persistedRoom.Players = append(persistedRoom.Players, persistedRoomPlayer{
				PlayerID:  player.player.ID,
				Name:      player.name,
				ImageURL:  player.imageURL,
				SessionID: player.sessionID,
				Connected: player.connected,
				Seat:      player.seat,
				Host:      player.host,
				Forfeited: player.forfeited,
			})
		}
		sort.Slice(persistedRoom.Players, func(i, j int) bool {
			return persistedRoom.Players[i].Seat < persistedRoom.Players[j].Seat
		})
		state.Rooms = append(state.Rooms, persistedRoom)
	}
	sort.Slice(state.Rooms, func(i, j int) bool {
		return state.Rooms[i].Code < state.Rooms[j].Code
	})

	return state
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

func normalizeIssueDescription(description string) (string, error) {
	cleanDescription := strings.TrimSpace(description)
	if cleanDescription == "" {
		return "", errors.New("problem description is required")
	}
	if len([]rune(cleanDescription)) > maxIssueLength {
		return "", fmt.Errorf("problem description must be %d characters or fewer", maxIssueLength)
	}
	return cleanDescription, nil
}

func (r *room) activeEndProposal(now time.Time) *endGameProposal {
	if r == nil || r.endProposal == nil {
		return nil
	}
	if !r.endProposal.expiresAt.After(now) {
		r.endProposal = nil
		r.endProposalCooldownUntil = now.Add(endProposalCooldown)
		return nil
	}
	return r.endProposal
}

func (r *room) newEndProposal(kind, proposerPlayerID, description, reportID string) *endGameProposal {
	eligiblePlayerIDs := make([]string, 0, len(r.players))
	for _, player := range r.players {
		if player != nil && !player.forfeited {
			eligiblePlayerIDs = append(eligiblePlayerIDs, player.player.ID)
		}
	}
	now := time.Now()
	return &endGameProposal{
		id:                uuid.NewString(),
		kind:              kind,
		proposerPlayerID:  proposerPlayerID,
		description:       description,
		reportID:          reportID,
		eligiblePlayerIDs: eligiblePlayerIDs,
		agreedPlayerIDs:   map[string]bool{proposerPlayerID: true},
		createdAt:         now,
		expiresAt:         now.Add(endProposalTTL),
	}
}

func (p *endGameProposal) unanimouslyApproved() bool {
	if p == nil || len(p.eligiblePlayerIDs) == 0 {
		return false
	}
	for _, playerID := range p.eligiblePlayerIDs {
		if !p.agreedPlayerIDs[playerID] {
			return false
		}
	}
	return true
}

func (r *room) removePlayerFromProposal(playerID string) {
	if r == nil || r.endProposal == nil {
		return
	}
	nextEligible := make([]string, 0, len(r.endProposal.eligiblePlayerIDs))
	for _, eligiblePlayerID := range r.endProposal.eligiblePlayerIDs {
		if eligiblePlayerID != playerID {
			nextEligible = append(nextEligible, eligiblePlayerID)
		}
	}
	r.endProposal.eligiblePlayerIDs = nextEligible
	delete(r.endProposal.agreedPlayerIDs, playerID)
	if r.endProposal.proposerPlayerID == playerID || len(nextEligible) < 2 {
		r.endProposal = nil
	}
}

func (r *room) transferHostToFirstActive() {
	if r == nil {
		return
	}
	r.hostID = ""
	for _, player := range r.players {
		if player != nil && !player.forfeited {
			r.hostID = player.player.ID
			break
		}
	}
	for _, player := range r.players {
		if player != nil {
			player.host = player.player.ID == r.hostID
		}
	}
}

func (r *room) snapshot() roomSnapshot {
	players := make([]playerSnapshot, 0, len(r.players))
	now := time.Now()
	for _, player := range r.players {
		if player == nil {
			continue
		}
		snapshot := playerSnapshot{
			PlayerID:     player.player.ID,
			UserID:       player.authUserID,
			Name:         player.name,
			ImageURL:     player.imageURL,
			Connected:    player.connected,
			Seat:         player.seat,
			IsHost:       player.host,
			CanReconnect: true,
			Forfeited:    player.forfeited,
		}
		if player.forfeited {
			snapshot.Connected = false
			snapshot.CanReconnect = false
		}
		if player.activeEmote != nil {
			if player.activeEmote.expiresAt.After(now) {
				snapshot.ActiveEmote = &playerEmoteSnapshot{
					ID:        player.activeEmote.id,
					Emoji:     player.activeEmote.emoji,
					ExpiresAt: player.activeEmote.expiresAt,
				}
			} else {
				player.activeEmote = nil
			}
		}
		players = append(players, snapshot)
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Seat < players[j].Seat
	})

	phase := r.gameStatePhase()
	snapshot := roomSnapshot{
		Code:         r.code,
		Phase:        phaseName(phase),
		HostPlayerID: r.hostID,
		GameMode:     r.gameState.GameMode(),
		Players:      players,
	}
	if proposal := r.activeEndProposal(now); proposal != nil {
		agreedPlayerIDs := make([]string, 0, len(proposal.agreedPlayerIDs))
		for _, playerID := range proposal.eligiblePlayerIDs {
			if proposal.agreedPlayerIDs[playerID] {
				agreedPlayerIDs = append(agreedPlayerIDs, playerID)
			}
		}
		snapshot.EndProposal = &endGameProposalSnapshot{
			ID:                proposal.id,
			Kind:              proposal.kind,
			ProposerPlayerID:  proposal.proposerPlayerID,
			Description:       proposal.description,
			EligiblePlayerIDs: append([]string(nil), proposal.eligiblePlayerIDs...),
			AgreedPlayerIDs:   agreedPlayerIDs,
			ExpiresAt:         proposal.expiresAt,
		}
	}
	if r.conclusion != nil {
		snapshot.Conclusion = &gameConclusionSnapshot{
			Kind:           r.conclusion.kind,
			WinnerPlayerID: r.conclusion.winnerPlayerID,
			ReportID:       r.conclusion.reportID,
		}
	}
	if r.pendingDealChoice != nil {
		gameMode := r.pendingDealChoice.gameMode
		if gameMode == "" {
			gameMode = game.GameModeFull
		}
		snapshot.GameMode = gameMode
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

	activePlayers := make([]*roomPlayer, 0, len(r.players))
	for _, player := range r.players {
		if player == nil || player.forfeited {
			continue
		}

		player.player = newPlayerWithID(player.player.ID)
		player.seat = len(activePlayers)
		player.forfeited = false
		if err := addPlayerToGameState(nextGameState, player.player); err != nil {
			return err
		}
		activePlayers = append(activePlayers, player)
	}

	r.players = activePlayers
	if r.playerByID(r.hostID) == nil && len(activePlayers) > 0 {
		r.hostID = activePlayers[0].player.ID
	}
	for _, player := range activePlayers {
		player.host = player.player.ID == r.hostID
	}
	r.gameState = nextGameState
	r.pendingDealChoice = nil
	r.endProposal = nil
	r.endProposalCooldownUntil = time.Time{}
	r.conclusion = nil
	r.statisticsGameID = ""
	r.statisticsStartedAt = time.Time{}
	r.statisticsPlaytime = 0
	r.statisticsActiveSince = time.Time{}
	r.statisticsSaved = false
	r.statisticsDirty = false
	r.clearTurnTracking()
	return nil
}

func (r *room) gameStateRecipients(sessions map[string]*playerSession, roomState roomSnapshot) ([]gameStateRecipient, error) {
	recipients := make([]gameStateRecipient, 0, len(r.players))
	for _, player := range r.players {
		if player == nil || player.forfeited || !player.connected {
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
					existing.CardActivities = make(map[int]game.CardActivitySnapshot, len(update.CardActivities))
				}
				maps.Copy(existing.CardActivities, update.CardActivities)
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
	newCount = max(newCount, 0)
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
		if addition.InsertIndex != nil {
			startIndex = *addition.InsertIndex
		}
		if startIndex < 0 {
			startIndex = 0
		}
		if addition.CompositionIndex >= 0 && addition.CompositionIndex < len(active) && active[addition.CompositionIndex] != nil {
			maxStartIndex := len(active[addition.CompositionIndex].Snapshot().Cards)
			if startIndex > maxStartIndex {
				startIndex = maxStartIndex
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
		if len(draft.CardPlacements) > 0 {
			next.CardPlacements = make([]game.DraftCardPlacementSnapshot, len(draft.CardPlacements))
			for index, placement := range draft.CardPlacements {
				next.CardPlacements[index] = game.DraftCardPlacementSnapshot{
					InsertIndex:       cloneIntPointer(placement.InsertIndex),
					ReclaimJokerIndex: cloneIntPointer(placement.ReclaimJokerIndex),
				}
			}
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
			next.CardActivities = maps.Clone(activity.CardActivities)
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneGameSnapshot(source *game.GameSnapshot) *game.GameSnapshot {
	if source == nil {
		return nil
	}
	cloned := *source
	cloned.Players = append([]game.PlayerStateSnapshot(nil), source.Players...)
	cloned.Hand = append([]game.CardSnapshot(nil), source.Hand...)
	cloned.DiscardPile = append([]game.CardSnapshot(nil), source.DiscardPile...)
	cloned.ActiveCompositions = cloneCompositionSnapshots(source.ActiveCompositions)
	cloned.TurnActivity = cloneTurnActivitySnapshot(source.TurnActivity)
	return &cloned
}

func cloneTurnActivitySnapshot(source *game.TurnActivitySnapshot) *game.TurnActivitySnapshot {
	if source == nil {
		return nil
	}
	cloned := *source
	cloned.BaselineCompositions = cloneCompositionSnapshots(source.BaselineCompositions)
	cloned.DraftCompositions = cloneDraftCompositionSnapshots(source.DraftCompositions)
	cloned.CompositionActivities = cloneCompositionActivitySnapshots(source.CompositionActivities)
	return &cloned
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
		if player == nil {
			return false
		}
		if player.forfeited {
			continue
		}
		if !player.connected {
			return false
		}
	}
	return true
}

// updateStatisticsPlaytime settles the current active interval, then starts a
// new one only while a resumable game has every active player online. Restored
// rooms deliberately do not restore statisticsActiveSince because all
// connections start offline after a server restart.
func (r *room) updateStatisticsPlaytime(now time.Time) {
	if r == nil {
		return
	}
	if !r.statisticsActiveSince.IsZero() {
		if elapsed := now.Sub(r.statisticsActiveSince); elapsed > 0 {
			r.statisticsPlaytime += min(elapsed, statisticsIdleLimit)
		}
		r.statisticsActiveSince = time.Time{}
	}
	phase := r.gameStatePhase()
	if r.statisticsGameID != "" && phase != game.PhaseLobby && phase != game.PhaseGameOver && r.allPlayersConnected() {
		r.statisticsActiveSince = now
	}
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

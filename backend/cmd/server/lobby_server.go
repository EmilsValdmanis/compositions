package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	issueReportCooldown = 30 * time.Second
	statisticsIdleLimit = 15 * time.Minute
)

var (
	errAlreadyInRoom        = clientError{"already_in_room", "already in a room"}
	errJoinRoomFirst        = clientError{"join_room_first", "join a room first"}
	errRoomNotFound         = clientError{"room_not_found", "room not found"}
	errRoomFull             = clientError{"room_full", "room is full"}
	errGameAlreadyStarted   = clientError{"game_already_started", "game already started"}
	errGameNotInitialized   = clientError{"game_not_initialized", "game state not initialized"}
	errPlayersDisconnected  = clientError{"players_disconnected", "all players must be connected"}
	errDealChoicePending    = clientError{"deal_choice_pending", "dealing choice already pending"}
	errPlayerInactive       = clientError{"player_inactive", "player is not active"}
	errEndProposalActive    = clientError{"end_request_active", "an end-game request is already active"}
	errEndProposalCooldown  = clientError{"end_request_cooldown", "wait before starting another end-game request"}
	errNotYourTurn          = clientError{"not_your_turn", "not your turn"}
	errLobbySessionNotFound = clientError{"session_not_found", "session not found"}
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
	issueReportCooldowns     map[string]time.Time
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

type gameStateRecipient struct {
	conn  *websocket.Conn
	event gameStateEvent
}

type lobbyServer struct {
	commandMu            sync.Mutex
	mutationBefore       *lobbyMutationState
	mu                   sync.Mutex
	pendingStatistics    map[string]pendingGameStatistics
	rng                  *rand.Rand
	store                userStore
	sessions             map[string]*playerSession
	rooms                map[string]*room
	spectatorsByRoom     map[string]map[*websocket.Conn]struct{}
	spectatingRoomByConn map[*websocket.Conn]string
}

var makeGameState = game.NewGameState
var addPlayerToGameState = func(state *game.GameState, player *game.Player) error {
	return state.AddPlayer(player)
}

func newLobbyServerWithStore(store userStore) *lobbyServer {
	return &lobbyServer{
		rng:                  rand.New(rand.NewSource(time.Now().UnixNano())),
		store:                store,
		sessions:             make(map[string]*playerSession),
		rooms:                make(map[string]*room),
		spectatorsByRoom:     make(map[string]map[*websocket.Conn]struct{}),
		spectatingRoomByConn: make(map[*websocket.Conn]string),
	}
}

func (l *lobbyServer) connect(existingSessionID string, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	return l.connectWithUser(existingSessionID, authenticatedUser{}, conn)
}

func (l *lobbyServer) connectWithUser(existingSessionID string, user authenticatedUser, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	unlock := l.lockMutation()
	defer unlock()

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
	if err := l.persistLocked("session created"); err != nil {
		return connectedEvent{}, nil, nil, err
	}
	return connectedEvent{SessionID: sessionID, PlayerID: playerID}, nil, nil, nil
}

func (l *lobbyServer) connectExistingSessionWithUser(existingSessionID string, user authenticatedUser, conn *websocket.Conn) (connectedEvent, *roomSnapshot, []*websocket.Conn, error) {
	session, ok := l.sessions[existingSessionID]
	if !ok {
		return connectedEvent{}, nil, nil, errLobbySessionNotFound
	}
	if session.authenticated != user.isAuthenticated() {
		slog.Warn("session auth mismatch", "sessionID", existingSessionID, "sessionAuthenticated", session.authenticated)
		return connectedEvent{}, nil, nil, errAuthenticationRequired
	}
	if session.authenticated && session.authUserID != user.ID {
		slog.Warn("session user mismatch", "sessionID", existingSessionID, "sessionUserID", session.authUserID, "requestUserID", user.ID)
		return connectedEvent{}, nil, nil, clientError{"session_user_mismatch", "session belongs to a different user"}
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

	if err := l.persistLocked("session reconnected"); err != nil {
		return connectedEvent{}, nil, nil, err
	}
	return connectedEvent{SessionID: session.sessionID, PlayerID: session.playerID}, roomState, recipients, nil
}

func (l *lobbyServer) createRoom(sessionID, name string) (roomSnapshot, []*websocket.Conn, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	if l.sessionRoom(session) != nil {
		return roomSnapshot{}, nil, errAlreadyInRoom
	}
	cleanName, err := session.playerName(name)
	if err != nil {
		return roomSnapshot{}, nil, err
	}

	gameState := makeGameState()
	if gameState == nil {
		return roomSnapshot{}, nil, errGameNotInitialized
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
	if err := l.persistLocked("room created"); err != nil {
		return roomSnapshot{}, nil, err
	}
	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) joinRoom(sessionID, roomCode, name string) (roomSnapshot, []*websocket.Conn, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	if l.sessionRoom(session) != nil {
		return roomSnapshot{}, nil, errAlreadyInRoom
	}

	room := l.rooms[normalizeRoomCode(roomCode)]
	if room == nil {
		return roomSnapshot{}, nil, errRoomNotFound
	}
	if room.pendingDealChoice != nil {
		return roomSnapshot{}, nil, clientError{"game_starting", "game is already starting"}
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby {
		return roomSnapshot{}, nil, errGameAlreadyStarted
	}
	if len(room.players) >= maxPlayersPerRoom {
		return roomSnapshot{}, nil, errRoomFull
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
	if err := l.persistLocked("player joined room"); err != nil {
		return roomSnapshot{}, nil, err
	}
	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) startGame(sessionID string, dealerIndex int, requestedModes ...game.GameMode) (roomSnapshot, []*websocket.Conn, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, errJoinRoomFirst
	}
	if room.hostID != session.playerID {
		return roomSnapshot{}, nil, clientError{"host_only_start", "only the host can start the game"}
	}
	if len(room.players) < minPlayersToStart {
		return roomSnapshot{}, nil, game.ErrNotEnoughPlayers
	}
	if !room.allPlayersConnected() {
		return roomSnapshot{}, nil, errPlayersDisconnected
	}
	if room.pendingDealChoice != nil {
		return roomSnapshot{}, nil, errDealChoicePending
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
	if err := l.persistLocked("game start requested"); err != nil {
		return roomSnapshot{}, nil, err
	}
	return room.snapshot(), room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) chooseDealing(sessionID, dealType string, options ...dealingChoiceOptions) (roomSnapshot, []gameStateRecipient, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, errJoinRoomFirst
	}
	if room.pendingDealChoice == nil {
		return roomSnapshot{}, nil, clientError{"no_deal_choice", "no dealing choice is pending"}
	}
	chooser := room.players[room.pendingDealChoice.chooserIndex]
	if chooser == nil || chooser.player.ID != session.playerID {
		return roomSnapshot{}, nil, clientError{"deal_chooser_only", "only the deal chooser can choose dealing type"}
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, errGameNotInitialized
	}

	choiceOptions := dealingChoiceOptions{}
	if len(options) > 0 {
		choiceOptions = options[0]
	}
	if choiceOptions.cutSize == nil {
		return roomSnapshot{}, nil, clientError{"cut_size_required", "cut size is required"}
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
			return clientError{"no_deal_choice", "game is not waiting for a dealing choice"}
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
	if err := l.persistLocked("dealing chosen"); err != nil {
		return roomSnapshot{}, nil, err
	}
	return roomState, recipients, nil
}

func (l *lobbyServer) startNextRound(sessionID string) (roomSnapshot, []*websocket.Conn, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, errJoinRoomFirst
	}
	if room.hostID != session.playerID {
		return roomSnapshot{}, nil, clientError{"host_only_next_round", "only the host can start the next round"}
	}
	if !room.allPlayersConnected() {
		return roomSnapshot{}, nil, errPlayersDisconnected
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, errGameNotInitialized
	}
	if room.pendingDealChoice != nil {
		return roomSnapshot{}, nil, errDealChoicePending
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
	if err := l.persistLocked("next round requested"); err != nil {
		return roomSnapshot{}, nil, err
	}
	return roomState, room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) leaveRoom(sessionID string) (*roomSnapshot, []*websocket.Conn, string, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return nil, nil, "", err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return nil, nil, "", errJoinRoomFirst
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby {
		return nil, nil, "", clientError{"lobby_only_leave", "can only leave in lobby"}
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
		if err := l.persistLocked("room disbanded"); err != nil {
			return nil, nil, "", err
		}
		return nil, nil, roomCode, nil
	}

	nextGameState := makeGameState()
	if nextGameState == nil {
		return nil, nil, "", errGameNotInitialized
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
	if err := l.persistLocked("player left room"); err != nil {
		return nil, nil, "", err
	}
	return &snapshot, room.connectedConns(l.sessions), roomCode, nil
}

func (l *lobbyServer) forfeitGame(sessionID string) (roomSnapshot, []gameStateRecipient, actionResultEvent, string, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", errJoinRoomFirst
	}
	player := room.playerByID(session.playerID)
	if player == nil || player.forfeited {
		return roomSnapshot{}, nil, actionResultEvent{}, "", errPlayerInactive
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", errGameNotInitialized
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
	if err := l.persistLocked("player forfeited game"); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, "", err
	}
	return roomState, recipients, result, room.code, nil
}

func (l *lobbyServer) requestEndGame(sessionID, kind string) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.createEndProposal(sessionID, kind, "", "")
}

func (l *lobbyServer) reportIssue(sessionID, description string, requestAbort bool) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errJoinRoomFirst
	}
	if room.gameState == nil || (room.gameStatePhase() != game.PhaseInProgress && room.gameStatePhase() != game.PhaseRoundOver) {
		return roomSnapshot{}, nil, actionResultEvent{}, game.ErrGameNotInProgress
	}
	if player := room.playerByID(session.playerID); player == nil || player.forfeited {
		return roomSnapshot{}, nil, actionResultEvent{}, errPlayerInactive
	}
	now := time.Now()
	if now.Before(room.issueReportCooldowns[session.playerID]) {
		return roomSnapshot{}, nil, actionResultEvent{}, errRateLimitExceeded
	}
	if requestAbort {
		if room.activeEndProposal(time.Now()) != nil {
			return roomSnapshot{}, nil, actionResultEvent{}, errEndProposalActive
		}
		if time.Now().Before(room.endProposalCooldownUntil) {
			return roomSnapshot{}, nil, actionResultEvent{}, errEndProposalCooldown
		}
	}

	cleanDescription, err := normalizeIssueDescription(description)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	gameStateJSON, _ := json.Marshal(room.gameState.PersistenceSnapshot())
	reportID := uuid.NewString()
	reportCreatedAt := now.UTC()
	roomCode := room.code
	playerID := session.playerID
	userID := session.authUserID
	round := room.gameState.RoundNumber()
	turn := room.gameState.TurnNumber()
	if room.issueReportCooldowns == nil {
		room.issueReportCooldowns = make(map[string]time.Time)
	}
	room.issueReportCooldowns[playerID] = now.Add(issueReportCooldown)
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	l.mu.Unlock()
	savedReport, err := l.store.CreateGameBugReport(ctx, database.GameBugReportRecord{
		ID:               reportID,
		RoomCode:         roomCode,
		ReporterPlayerID: playerID,
		ReporterUserID:   userID,
		Description:      cleanDescription,
		GameState:        gameStateJSON,
		Round:            round,
		Turn:             turn,
		RequestedAbort:   requestAbort,
		CreatedAt:        reportCreatedAt,
	})
	cancel()
	l.mu.Lock()
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, fmt.Errorf("save bug report: %w", err)
	}
	slog.Warn("game issue reported", "roomCode", roomCode, "reportID", savedReport.ID, "playerID", playerID)

	if requestAbort {
		room.endProposal = room.newEndProposal("technical_abort", playerID, cleanDescription, savedReport.ID)
	}

	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: "report_issue", PlayerID: playerID, OK: true}
	if err := l.persistLocked("game issue reported"); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	return roomState, recipients, result, nil
}

func (l *lobbyServer) createEndProposal(sessionID, kind, description, reportID string) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errJoinRoomFirst
	}
	if room.gameState == nil || (room.gameStatePhase() != game.PhaseInProgress && room.gameStatePhase() != game.PhaseRoundOver) {
		return roomSnapshot{}, nil, actionResultEvent{}, game.ErrGameNotInProgress
	}
	if kind != "mutual_end" {
		return roomSnapshot{}, nil, actionResultEvent{}, clientError{"unknown_end_request", "unknown end-game request type"}
	}
	if player := room.playerByID(session.playerID); player == nil || player.forfeited {
		return roomSnapshot{}, nil, actionResultEvent{}, errPlayerInactive
	}
	if room.activeEndProposal(time.Now()) != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errEndProposalActive
	}
	if time.Now().Before(room.endProposalCooldownUntil) {
		return roomSnapshot{}, nil, actionResultEvent{}, errEndProposalCooldown
	}

	room.endProposal = room.newEndProposal(kind, session.playerID, description, reportID)
	roomState := room.snapshot()
	recipients, err := room.gameStateRecipients(l.sessions, roomState)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	result := actionResultEvent{Action: "request_end_game", PlayerID: session.playerID, OK: true}
	if err := l.persistLocked("end game requested"); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	return roomState, recipients, result, nil
}

func (l *lobbyServer) voteEndGame(sessionID, proposalID string, approve bool) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errJoinRoomFirst
	}
	proposal := room.activeEndProposal(time.Now())
	if proposal == nil || proposal.id != strings.TrimSpace(proposalID) {
		return roomSnapshot{}, nil, actionResultEvent{}, clientError{"end_request_not_found", "end-game request not found or expired"}
	}
	if !slices.Contains(proposal.eligiblePlayerIDs, session.playerID) {
		return roomSnapshot{}, nil, actionResultEvent{}, clientError{"vote_ineligible", "player is not eligible to vote"}
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
	if err := l.persistLocked("end game vote recorded"); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	return roomState, recipients, result, nil
}

func (l *lobbyServer) sendEmote(sessionID, emoji string) (roomSnapshot, []*websocket.Conn, error) {
	l.commandMu.Lock()
	defer l.commandMu.Unlock()
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, errJoinRoomFirst
	}
	if _, ok := allowedPlayerEmotes[emoji]; !ok {
		return roomSnapshot{}, nil, clientError{"unknown_emote", "unknown emote"}
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
			return clientError{"unknown_draw_source", "unknown draw source"}
		}
	}, func(room *room, session *playerSession) {
		room.resetTurnTracking(session.playerID)
		if room.turnActivity != nil {
			room.turnActivity.DrawSource = source
		}
	})
}

func (l *lobbyServer) play(sessionID string, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "play", func(state *game.GameState) error {
		return state.PlayTable(comps, additions, reclaims...)
	}, func(room *room, session *playerSession) {
		room.applySubmittedTurnActivity(session.playerID, comps, additions, reclaims)
	})
}

func (l *lobbyServer) playAndDiscard(sessionID string, cardIndex int, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.playAndDiscardMatching(sessionID, cardIndex, nil, comps, additions, reclaims)
}

func (l *lobbyServer) playAndDiscardMatching(sessionID string, cardIndex int, expectedCard *game.Card, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "play_and_discard", func(state *game.GameState) error {
		return state.PlayTableAndDiscardMatching(cardIndex, expectedCard, comps, additions, reclaims...)
	}, func(room *room, _ *playerSession) {
		room.clearTurnTracking()
	})
}

func (l *lobbyServer) discard(sessionID string, cardIndex int) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.discardMatching(sessionID, cardIndex, nil)
}

func (l *lobbyServer) discardMatching(sessionID string, cardIndex int, expectedCard *game.Card) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	return l.applyGameAction(sessionID, "discard", func(state *game.GameState) error {
		return state.DiscardFromHandMatching(cardIndex, expectedCard)
	}, func(room *room, _ *playerSession) {
		room.clearTurnTracking()
	})
}

func (l *lobbyServer) updateDraftActivity(sessionID string, drafts []game.DraftCompositionSnapshot) (roomSnapshot, []gameStateRecipient, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, errJoinRoomFirst
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, errGameNotInitialized
	}
	if room.gameState.Phase() != game.PhaseInProgress {
		return roomSnapshot{}, nil, game.ErrGameNotInProgress
	}
	if !room.isCurrentTurn(session.playerID) {
		return roomSnapshot{}, nil, errNotYourTurn
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
	if err := l.persistLocked("draft activity updated"); err != nil {
		return roomSnapshot{}, nil, err
	}
	return roomState, recipients, nil
}

func (l *lobbyServer) applyGameAction(sessionID, action string, mutate func(*game.GameState) error, afterMutate func(*room, *playerSession)) (roomSnapshot, []gameStateRecipient, actionResultEvent, error) {
	unlock := l.lockMutation()
	defer unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errJoinRoomFirst
	}
	if room.gameState == nil {
		return roomSnapshot{}, nil, actionResultEvent{}, errGameNotInitialized
	}
	if room.gameState.Phase() != game.PhaseInProgress {
		return roomSnapshot{}, nil, actionResultEvent{}, game.ErrGameNotInProgress
	}
	if !room.isCurrentTurn(session.playerID) {
		return roomSnapshot{}, nil, actionResultEvent{}, errNotYourTurn
	}
	if err := mutate(room.gameState); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	if afterMutate != nil {
		afterMutate(room, session)
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
	if err := l.persistLocked("game action applied"); err != nil {
		return roomSnapshot{}, nil, actionResultEvent{}, err
	}
	return roomState, recipients, result, nil
}

func (l *lobbyServer) resetRoomAfterGameOver(roomCode string) (*roomSnapshot, []*websocket.Conn, error) {
	unlock := l.lockMutation()
	defer unlock()

	room := l.rooms[normalizeRoomCode(roomCode)]
	if room == nil {
		return nil, nil, errRoomNotFound
	}
	if room.gameState == nil {
		return nil, nil, errGameNotInitialized
	}
	if room.gameStatePhase() != game.PhaseGameOver {
		return nil, nil, errors.New("game is not over")
	}
	if err := room.resetForLobby(); err != nil {
		return nil, nil, err
	}

	snapshot := room.snapshot()
	if err := l.persistLocked("room reset after game over"); err != nil {
		return nil, nil, err
	}
	return &snapshot, room.connectedConns(l.sessions), nil
}

func (l *lobbyServer) disconnect(sessionID string, conn *websocket.Conn) {
	l.disconnectWithEmitter(sessionID, conn, emitEvent)
}

func (l *lobbyServer) disconnectWithEmitter(sessionID string, conn *websocket.Conn, emitter eventEmitter) {
	l.commandMu.Lock()
	defer l.commandMu.Unlock()
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
	if err := l.persistLocked("session disconnected"); err != nil {
		slog.Error("persist disconnected session failed", "sessionID", sessionID, "error", err)
	}

	roomState = room.snapshot()
	roomState.SpectatorCount = len(l.spectatorsByRoom[room.code])
	recipients = append(room.connectedConns(l.sessions), l.spectatorConnectionsLocked(room.code)...)
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

func (l *lobbyServer) requireSession(sessionID string) (*playerSession, error) {
	session := l.sessions[sessionID]
	if session == nil {
		return nil, errLobbySessionNotFound
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
		return clientError{"session_not_found", "session not active on this connection"}
	}

	return nil
}

func (l *lobbyServer) authenticatedUserID(sessionID string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return "", err
	}
	if !session.authenticated || session.authUserID == "" {
		return "", errAuthenticationRequired
	}
	return session.authUserID, nil
}

func (l *lobbyServer) joinableRoomCode(sessionID string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return "", err
	}
	room := l.sessionRoom(session)
	if room == nil {
		return "", errJoinRoomFirst
	}
	if room.gameState == nil || room.gameStatePhase() != game.PhaseLobby || room.pendingDealChoice != nil {
		return "", errGameAlreadyStarted
	}
	if len(room.players) >= maxPlayersPerRoom {
		return "", errRoomFull
	}
	return room.code, nil
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

	event, err := room.gameStateForPlayer(session.playerID, roomState)
	if err != nil {
		return nil, err
	}
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
		return "", clientError{"name_required", "name is required"}
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
		return "", clientError{"problem_required", "problem description is required"}
	}
	if len([]rune(cleanDescription)) > database.GameBugReportDescriptionMaxLength {
		return "", clientError{"problem_too_long", fmt.Sprintf("problem description must be %d characters or fewer", database.GameBugReportDescriptionMaxLength)}
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
		return errGameNotInitialized
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
		event, err := r.gameStateForPlayer(player.player.ID, roomState)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, gameStateRecipient{conn: session.conn, event: event})
	}
	return recipients, nil
}

func (r *room) gameStateForPlayer(playerID string, roomState roomSnapshot) (gameStateEvent, error) {
	state, ok := r.gameState.SnapshotForPlayer(playerID)
	if !ok {
		return gameStateEvent{}, errors.New("game state snapshot failed")
	}
	state.TurnActivity = r.turnActivityForPlayer(playerID)
	return gameStateEvent{Room: roomState, Game: state}, nil
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

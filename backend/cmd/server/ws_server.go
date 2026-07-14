package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/gorilla/websocket"
)

var errSocketClosed = errors.New("socket closed")
var emitEvent = writeEvent
var writeControl = func(conn *websocket.Conn, messageType int, data []byte, deadline time.Time) error {
	return conn.WriteControl(messageType, data, deadline)
}

var (
	wsDataWriteLocks      sync.Map
	defaultWSReadLimit    int64 = 64 * 1024
	defaultWSReadTimeout        = 75 * time.Second
	defaultWSPingInterval       = 25 * time.Second
	defaultWSWriteTimeout       = 10 * time.Second
)

func setWSReadDeadline(conn *websocket.Conn) error {
	return conn.SetReadDeadline(time.Now().Add(defaultWSReadTimeout))
}

func runHeartbeatPingLoop(conn *websocket.Conn, pingDone <-chan struct{}, ticks <-chan time.Time) {
	for {
		select {
		case <-ticks:
			if err := writeControl(conn, websocket.PingMessage, nil, time.Now().Add(defaultWSWriteTimeout)); err != nil {
				return
			}
		case <-pingDone:
			return
		}
	}
}

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

type chooseDealingRequest struct {
	DealType string `json:"dealType"`
	Order    []int  `json:"order,omitempty"`
	CutSize  *int   `json:"cutSize,omitempty"`
}

type startNextRoundRequest struct{}

type leaveRoomRequest struct{}

type forfeitGameRequest struct{}

type requestEndGameRequest struct {
	Kind string `json:"kind"`
}

type voteEndGameRequest struct {
	ProposalID string `json:"proposalId"`
	Approve    bool   `json:"approve"`
}

type reportIssueRequest struct {
	Description  string `json:"description"`
	RequestAbort bool   `json:"requestAbort"`
}

type sendEmoteRequest struct {
	Emoji string `json:"emoji"`
}

type drawRequest struct {
	Source string `json:"source"`
}

type cardRequest struct {
	Rank    int  `json:"rank,omitempty"`
	Suit    int  `json:"suit,omitempty"`
	IsJoker bool `json:"isJoker,omitempty"`
}

type compositionRequest struct {
	Cards []cardRequest `json:"cards"`
}

type playRequest struct {
	Compositions []compositionRequest         `json:"compositions"`
	Additions    []compositionAdditionRequest `json:"additions,omitempty"`
	Reclaims     []reclaimRequest             `json:"reclaims,omitempty"`
}

type draftUpdateRequest struct {
	Compositions []draftCompositionRequest `json:"compositions,omitempty"`
}

type draftCardPlacementRequest struct {
	InsertIndex       *int `json:"insertIndex,omitempty"`
	ReclaimJokerIndex *int `json:"reclaimJokerIndex,omitempty"`
}

type draftCompositionRequest struct {
	TableIndex     *int                        `json:"tableIndex,omitempty"`
	InsertIndex    *int                        `json:"insertIndex,omitempty"`
	CardPlacements []draftCardPlacementRequest `json:"cardPlacements,omitempty"`
	Cards          []cardRequest               `json:"cards"`
}

type compositionAdditionRequest struct {
	CompositionIndex int           `json:"compositionIndex"`
	InsertIndex      *int          `json:"insertIndex,omitempty"`
	Cards            []cardRequest `json:"cards"`
}

type reclaimRequest struct {
	CompositionIndex int         `json:"compositionIndex"`
	JokerIndex       int         `json:"jokerIndex"`
	ReplacementCard  cardRequest `json:"replacementCard"`
}

type discardRequest struct {
	CardIndex int          `json:"cardIndex"`
	Card      *cardRequest `json:"card,omitempty"`
}

type playAndDiscardRequest struct {
	playRequest
	CardIndex int          `json:"cardIndex"`
	Card      *cardRequest `json:"card,omitempty"`
}

type connectedEvent struct {
	SessionID string `json:"sessionId"`
	PlayerID  string `json:"playerId"`
}

type errorEvent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type roomStateEvent struct {
	Room roomSnapshot `json:"room"`
}

type gameStateEvent struct {
	Room roomSnapshot      `json:"room"`
	Game game.GameSnapshot `json:"game"`
}

type actionResultEvent struct {
	Action   string `json:"action"`
	PlayerID string `json:"playerId"`
	OK       bool   `json:"ok"`
}

type leftRoomEvent struct {
	RoomCode string `json:"roomCode,omitempty"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type roomSnapshot struct {
	Code              string                     `json:"code"`
	Phase             string                     `json:"phase"`
	HostPlayerID      string                     `json:"hostPlayerId"`
	DealerIndex       int                        `json:"dealerIndex,omitempty"`
	PendingDealChoice *pendingDealChoiceSnapshot `json:"pendingDealChoice,omitempty"`
	Players           []playerSnapshot           `json:"players"`
	EndProposal       *endGameProposalSnapshot   `json:"endProposal,omitempty"`
	Conclusion        *gameConclusionSnapshot    `json:"conclusion,omitempty"`
}

type pendingDealChoiceSnapshot struct {
	DealerIndex     int    `json:"dealerIndex"`
	ChooserIndex    int    `json:"chooserIndex"`
	ChooserPlayerID string `json:"chooserPlayerId"`
}

type playerSnapshot struct {
	PlayerID     string               `json:"playerId"`
	UserID       string               `json:"userId,omitempty"`
	SessionID    string               `json:"sessionId,omitempty"`
	Name         string               `json:"name"`
	ImageURL     string               `json:"imageUrl,omitempty"`
	Connected    bool                 `json:"connected"`
	Seat         int                  `json:"seat"`
	IsHost       bool                 `json:"isHost"`
	CanReconnect bool                 `json:"canReconnect"`
	ActiveEmote  *playerEmoteSnapshot `json:"activeEmote,omitempty"`
	Forfeited    bool                 `json:"forfeited,omitempty"`
}

type endGameProposalSnapshot struct {
	ID                string    `json:"id"`
	Kind              string    `json:"kind"`
	ProposerPlayerID  string    `json:"proposerPlayerId"`
	Description       string    `json:"description,omitempty"`
	EligiblePlayerIDs []string  `json:"eligiblePlayerIds"`
	AgreedPlayerIDs   []string  `json:"agreedPlayerIds"`
	ExpiresAt         time.Time `json:"expiresAt"`
}

type gameConclusionSnapshot struct {
	Kind           string `json:"kind"`
	WinnerPlayerID string `json:"winnerPlayerId,omitempty"`
	ReportID       string `json:"reportId,omitempty"`
}

type playerEmoteSnapshot struct {
	ID        string    `json:"id"`
	Emoji     string    `json:"emoji"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type wsServer struct {
	lobby         *lobbyServer
	auth          *authHandler
	userStore     userStore
	allowedOrigin string
	rateLimits    *wsRateLimiters
	upgrader      websocket.Upgrader
}

func newWSServer() *wsServer {
	return newWSServerWithAuth(nil)
}

func newWSServerWithAuth(auth *authHandler) *wsServer {
	return newWSServerWithAllowedOrigin(auth, "")
}

func newWSServerWithAllowedOrigin(auth *authHandler, allowedOrigin string) *wsServer {
	return newWSServerWithDependencies(auth, noopUserStore{}, allowedOrigin)
}

func newWSServerWithDependencies(auth *authHandler, store userStore, allowedOrigin string) *wsServer {
	if store == nil {
		store = noopUserStore{}
	}

	server := &wsServer{
		lobby:         newLobbyServerWithStore(store),
		auth:          auth,
		userStore:     store,
		allowedOrigin: normalizeOrigin(allowedOrigin),
		rateLimits:    newWSRateLimiters(defaultWSRateLimitConfig()),
	}
	server.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return server.isAllowedOrigin(r.Header.Get("Origin"))
		},
	}
	return server
}

func (s *wsServer) Close() error {
	if s == nil || s.userStore == nil {
		return nil
	}

	return s.userStore.Close()
}

func (s *wsServer) isAllowedOrigin(origin string) bool {
	if s == nil || s.allowedOrigin == "" {
		return true
	}

	cleanOrigin := normalizeOrigin(origin)
	if cleanOrigin == "" {
		return false
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
	mux.HandleFunc("/auth/", s.handleSessionRoutes)
	mux.HandleFunc("/api/players/", s.handlePlayerProfile)
	mux.HandleFunc("/ws", s.handleWS)
	return mux
}

func (s *wsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		slog.Error("write health response", "error", err)
	}
}

func (s *wsServer) handleWS(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimits.allowConnectionAttempt(r) {
		writeHTTPError(w, http.StatusTooManyRequests, clientErrorCode(errRateLimitExceeded), errRateLimitExceeded.Error())
		slog.Warn("websocket connection rate limited", "remote", r.RemoteAddr)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("websocket upgrade failed", "error", err, "remote", r.RemoteAddr)
		return
	}

	slog.Debug("websocket connection established", "remote", conn.RemoteAddr().String())
	s.handleConnection(conn, r)
}

func (s *wsServer) handleConnection(conn *websocket.Conn, request *http.Request) {
	connectionEmitter := emitEvent
	defer wsDataWriteLocks.Delete(conn)
	defer conn.Close()
	conn.SetReadLimit(defaultWSReadLimit)
	_ = setWSReadDeadline(conn)
	conn.SetPongHandler(func(string) error {
		return setWSReadDeadline(conn)
	})

	pingDone := make(chan struct{})
	pingExited := make(chan struct{})
	pingInterval := defaultWSPingInterval
	defer func() {
		close(pingDone)
		<-pingExited
	}()
	go func() {
		defer close(pingExited)
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		runHeartbeatPingLoop(conn, pingDone, ticker.C)
	}()

	messageLimiter := s.rateLimits.newMessageLimiter()
	sessionID := ""
	for {
		var envelope wsEnvelope
		if err := conn.ReadJSON(&envelope); err != nil {
			if sessionID != "" {
				s.lobby.disconnectWithEmitter(sessionID, conn, connectionEmitter)
			}
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Debug("websocket read error", "sessionID", sessionID, "error", err)
			}
			return
		}
		if !messageLimiter.Allow() {
			slog.Warn("websocket message rate limited", "sessionID", sessionID, "remote", conn.RemoteAddr().String())
			s.writeError(conn, errRateLimitExceeded)
			return
		}
		_ = setWSReadDeadline(conn)

		slog.Debug("websocket message received", "sessionID", sessionID, "type", envelope.Type)

		switch envelope.Type {
		case "connect":
			nextSessionID, shouldClose := s.handleConnect(conn, request, envelope)
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
		case "choose_dealing":
			s.handleChooseDealing(conn, sessionID, envelope)
		case "start_next_round":
			s.handleStartNextRound(conn, sessionID, envelope)
		case "leave_room":
			if s.handleLeaveRoom(conn, sessionID, envelope) {
				return
			}
		case "forfeit_game":
			s.handleForfeitGame(conn, sessionID, envelope)
		case "request_end_game":
			s.handleRequestEndGame(conn, sessionID, envelope)
		case "vote_end_game":
			s.handleVoteEndGame(conn, sessionID, envelope)
		case "report_issue":
			s.handleReportIssue(conn, sessionID, envelope)
		case "send_emote":
			s.handleSendEmote(conn, sessionID, envelope)
		case "draw":
			s.handleDraw(conn, sessionID, envelope)
		case "play":
			s.handlePlay(conn, sessionID, envelope)
		case "play_and_discard":
			s.handlePlayAndDiscard(conn, sessionID, envelope)
		case "draft_update":
			s.handleDraftUpdate(conn, sessionID, envelope)
		case "discard":
			s.handleDiscard(conn, sessionID, envelope)
		default:
			s.writeError(conn, errors.New("unknown message type"))
		}
	}
}

func (s *wsServer) handleConnect(conn *websocket.Conn, request *http.Request, envelope wsEnvelope) (string, bool) {
	var req connectRequest
	if err := decodePayload(envelope.Data, &req); err != nil {
		slog.Warn("connect: invalid payload", "error", err)
		s.writeError(conn, err)
		return "", false
	}

	user := authenticatedUser{}
	if s.auth != nil {
		session, err := s.auth.sessionFromRequest(request)
		if err != nil {
			slog.Warn("connect: session verification failed", "sessionID", req.SessionID, "error", err)
			s.writeError(conn, err)
			return "", true
		}
		user = session.user
	}
	if err := s.persistAuthenticatedUser(user); err != nil {
		slog.Error("connect: persist user failed", "sessionID", req.SessionID, "userID", user.ID, "error", err)
		s.writeError(conn, err)
		return "", true
	}

	event, roomState, recipients, err := s.lobby.connectWithUser(req.SessionID, user, conn)
	if err != nil {
		slog.Warn("connect: lobby connect failed", "sessionID", req.SessionID, "error", err)
		s.writeError(conn, err)
		return "", true
	}
	if err := emitEvent(conn, "connected", event); err != nil {
		return "", true
	}

	slog.Info("client connected", "sessionID", event.SessionID, "playerID", event.PlayerID, "authenticated", user.isAuthenticated())

	if roomState != nil {
		if err := emitEvent(conn, "room_state", roomStateEvent{Room: *roomState}); err != nil {
			return "", true
		}
		gameState, err := s.lobby.gameStateForSession(event.SessionID, *roomState)
		if err != nil {
			slog.Warn("connect: game state resume failed", "sessionID", event.SessionID, "error", err)
			s.writeError(conn, err)
			return "", true
		}
		if gameState != nil {
			if err := emitEvent(conn, "game_state", *gameState); err != nil {
				return "", true
			}
		}
		s.broadcastRoomState(*roomState, otherConnections(recipients, conn))
	}

	return event.SessionID, false
}

func (s *wsServer) persistAuthenticatedUser(user authenticatedUser) error {
	if s == nil || !user.isAuthenticated() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()

	if _, err := s.userStore.UpsertUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}

	return nil
}

func (s *wsServer) handleCreateRoom(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[createRoomRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	if !s.rateLimits.allowCreateRoom(sessionID) {
		s.writeError(conn, errRateLimitExceeded)
		return
	}

	roomState, recipients, err := s.lobby.createRoom(sessionID, req.Name)
	if err != nil {
		slog.Warn("create room failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	slog.Info("room created", "roomCode", roomState.Code, "sessionID", sessionID)
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleJoinRoom(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[joinRoomRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	if !s.rateLimits.allowJoinRoom(sessionID) {
		s.writeError(conn, errRateLimitExceeded)
		return
	}

	roomState, recipients, err := s.lobby.joinRoom(sessionID, req.RoomCode, req.Name)
	if err != nil {
		slog.Warn("join room failed", "sessionID", sessionID, "roomCode", req.RoomCode, "error", err)
		s.writeError(conn, err)
		return
	}
	slog.Info("player joined room", "sessionID", sessionID, "roomCode", roomState.Code)
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleStartGame(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[startGameRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, err := s.lobby.startGame(sessionID, req.DealerIndex)
	if err != nil {
		slog.Warn("start game failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	slog.Info("game start requested", "roomCode", roomState.Code, "sessionID", sessionID, "dealerIndex", req.DealerIndex)
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleChooseDealing(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[chooseDealingRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, err := s.lobby.chooseDealing(sessionID, req.DealType, dealingChoiceOptions{
		order:   req.Order,
		cutSize: req.CutSize,
	})
	if err != nil {
		slog.Warn("choose dealing failed", "sessionID", sessionID, "dealType", req.DealType, "error", err)
		s.writeError(conn, err)
		return
	}
	slog.Info("game started", "roomCode", roomState.Code, "sessionID", sessionID, "dealType", req.DealType)
	conns := gameRecipientConns(recipients)
	s.broadcastRoomState(roomState, conns)
	s.broadcastGameState(recipients)
}

func (s *wsServer) handleStartNextRound(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	if _, ok := decodeSessionRequest[startNextRoundRequest](s, conn, sessionID, envelope); !ok {
		return
	}

	roomState, recipients, err := s.lobby.startNextRound(sessionID)
	if err != nil {
		slog.Warn("start next round failed", "sessionID", sessionID, "error", err)
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
		slog.Warn("leave room failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return false
	}
	if err := emitEvent(conn, "left_room", leftRoomEvent{RoomCode: roomCode}); err != nil {
		slog.Warn("write left_room event failed", "sessionID", sessionID, "roomCode", roomCode, "error", err)
		return true
	}
	slog.Info("player left room", "sessionID", sessionID, "roomCode", roomCode)
	if roomState != nil {
		s.broadcastRoomState(*roomState, recipients)
	}
	return false
}

func (s *wsServer) handleForfeitGame(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	if _, ok := decodeSessionRequest[forfeitGameRequest](s, conn, sessionID, envelope); !ok {
		return
	}

	roomState, recipients, result, roomCode, err := s.lobby.forfeitGame(sessionID)
	if err != nil {
		slog.Warn("forfeit game failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	logEmitFailure(conn, "action_result", result, "write forfeit result failed", "sessionID", sessionID)
	s.broadcastActionSuccess(result, roomState, recipients)
	logEmitFailure(conn, "left_room", leftRoomEvent{RoomCode: roomCode}, "write forfeit left-room event failed", "sessionID", sessionID)
}

func (s *wsServer) handleRequestEndGame(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[requestEndGameRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	roomState, recipients, result, err := s.lobby.requestEndGame(sessionID, req.Kind)
	if err != nil {
		slog.Warn("request end game failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) handleVoteEndGame(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[voteEndGameRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	roomState, recipients, result, err := s.lobby.voteEndGame(sessionID, req.ProposalID, req.Approve)
	if err != nil {
		slog.Warn("vote end game failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) handleReportIssue(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[reportIssueRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	roomState, recipients, result, err := s.lobby.reportIssue(sessionID, req.Description, req.RequestAbort)
	if err != nil {
		slog.Warn("report issue failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) handleSendEmote(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[sendEmoteRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, err := s.lobby.sendEmote(sessionID, req.Emoji)
	if err != nil {
		slog.Warn("send emote failed", "sessionID", sessionID, "emoji", req.Emoji, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastRoomState(roomState, recipients)
}

func (s *wsServer) handleDraw(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[drawRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	roomState, recipients, result, err := s.lobby.draw(sessionID, req.Source)
	if err != nil {
		slog.Warn("draw failed", "sessionID", sessionID, "source", req.Source, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) handlePlay(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[playRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	comps, err := compositionsFromRequest(req.Compositions)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	additions, err := additionsFromRequest(req.Additions)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	reclaims, err := reclaimsFromRequest(req.Reclaims)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	roomState, recipients, result, err := s.lobby.play(sessionID, comps, additions, reclaims)
	if err != nil {
		slog.Warn("play failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) handlePlayAndDiscard(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[playAndDiscardRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	comps, err := compositionsFromRequest(req.Compositions)
	if err != nil {
		s.writeError(conn, err)
		return
	}
	additions, err := additionsFromRequest(req.Additions)
	if err != nil {
		s.writeError(conn, err)
		return
	}
	reclaims, err := reclaimsFromRequest(req.Reclaims)
	if err != nil {
		s.writeError(conn, err)
		return
	}
	expectedCard, err := discardCardFromRequest(req.Card)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	roomState, recipients, result, err := s.lobby.playAndDiscardMatching(
		sessionID,
		req.CardIndex,
		expectedCard,
		comps,
		additions,
		reclaims,
	)
	if err != nil {
		slog.Warn("play and discard failed", "sessionID", sessionID, "cardIndex", req.CardIndex, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) handleDraftUpdate(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[draftUpdateRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	drafts, err := draftCompositionsFromRequest(req.Compositions)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	roomState, recipients, err := s.lobby.updateDraftActivity(sessionID, drafts)
	if err != nil {
		slog.Warn("draft update failed", "sessionID", sessionID, "error", err)
		s.writeError(conn, err)
		return
	}
	conns := gameRecipientConns(recipients)
	s.broadcastRoomState(roomState, conns)
	s.broadcastGameState(recipients)
}

func (s *wsServer) handleDiscard(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, ok := decodeSessionRequest[discardRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}

	expectedCard, err := discardCardFromRequest(req.Card)
	if err != nil {
		s.writeError(conn, err)
		return
	}

	roomState, recipients, result, err := s.lobby.discardMatching(sessionID, req.CardIndex, expectedCard)
	if err != nil {
		slog.Warn("discard failed", "sessionID", sessionID, "cardIndex", req.CardIndex, "error", err)
		s.writeError(conn, err)
		return
	}
	s.broadcastActionSuccess(result, roomState, recipients)
}

func (s *wsServer) writeError(conn *websocket.Conn, err error) {
	logEmitFailure(conn, "error", errorEvent{Code: clientErrorCode(err), Message: clientErrorMessage(err)}, "write websocket error event failed")
}

func (s *wsServer) broadcastRoomState(roomState roomSnapshot, recipients []*websocket.Conn) {
	for _, conn := range recipients {
		logEmitFailure(conn, "room_state", roomStateEvent{Room: roomState}, "broadcast room_state failed", "roomCode", roomState.Code)
	}
}

func (s *wsServer) broadcastGameState(recipients []gameStateRecipient) {
	for _, recipient := range recipients {
		logEmitFailure(recipient.conn, "game_state", recipient.event, "broadcast game_state failed", "roomCode", recipient.event.Room.Code)
	}
}

func (s *wsServer) broadcastActionResult(result actionResultEvent, recipients []*websocket.Conn) {
	for _, conn := range recipients {
		logEmitFailure(conn, "action_result", result, "broadcast action_result failed", "action", result.Action, "playerID", result.PlayerID)
	}
}

func (s *wsServer) broadcastActionSuccess(result actionResultEvent, roomState roomSnapshot, recipients []gameStateRecipient) {
	conns := gameRecipientConns(recipients)
	s.broadcastActionResult(result, conns)
	s.broadcastRoomState(roomState, conns)
	s.broadcastGameState(recipients)

	if roomState.Phase == "game_over" {
		if resetRoomState, resetRecipients, err := s.lobby.resetRoomAfterGameOver(roomState.Code); err != nil {
			slog.Error("reset room after game over failed", "roomCode", roomState.Code, "error", err)
		} else if resetRoomState != nil {
			s.broadcastRoomState(*resetRoomState, resetRecipients)
		}
	}
}

func gameRecipientConns(recipients []gameStateRecipient) []*websocket.Conn {
	conns := make([]*websocket.Conn, 0, len(recipients))
	for _, recipient := range recipients {
		conns = append(conns, recipient.conn)
	}
	return conns
}

func logEmitFailure(conn *websocket.Conn, messageType string, data any, logMessage string, attrs ...any) {
	if conn == nil {
		return
	}
	if err := emitEvent(conn, messageType, data); err != nil && !errors.Is(err, errSocketClosed) {
		slog.Error(logMessage, append(attrs, "error", err)...)
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
	lock, _ := wsDataWriteLocks.LoadOrStore(conn, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	_ = conn.SetWriteDeadline(time.Now().Add(defaultWSWriteTimeout))
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

func compositionsFromRequest(requests []compositionRequest) ([]*game.Composition, error) {
	comps := make([]*game.Composition, 0, len(requests))
	for _, req := range requests {
		cards, err := cardsFromRequest(req.Cards)
		if err != nil {
			return nil, err
		}

		comp, ok := inferComposition(cards)
		if !ok {
			return nil, game.ErrInvalidComposition
		}
		comps = append(comps, comp)
	}
	return comps, nil
}

func inferComposition(cards []game.Card) (*game.Composition, bool) {
	if comp, ok := game.NewSet(cards); ok {
		return comp, true
	}

	return game.NewUnorderedRun(cards)
}

func additionsFromRequest(requests []compositionAdditionRequest) ([]game.CompositionAddition, error) {
	additions := make([]game.CompositionAddition, 0, len(requests))
	for _, req := range requests {
		cards, err := cardsFromRequest(req.Cards)
		if err != nil {
			return nil, err
		}
		additions = append(additions, game.CompositionAddition{CompositionIndex: req.CompositionIndex, InsertIndex: req.InsertIndex, Cards: cards})
	}
	return additions, nil
}

func reclaimFromRequest(req reclaimRequest) (game.JokerReclaim, error) {
	replacement, err := cardFromRequest(req.ReplacementCard)
	if err != nil {
		return game.JokerReclaim{}, err
	}
	return game.JokerReclaim{CompositionIndex: req.CompositionIndex, JokerIndex: req.JokerIndex, ReplacementCard: replacement}, nil
}

func reclaimsFromRequest(requests []reclaimRequest) ([]game.JokerReclaim, error) {
	reclaims := make([]game.JokerReclaim, 0, len(requests))
	for _, req := range requests {
		reclaim, err := reclaimFromRequest(req)
		if err != nil {
			return nil, err
		}
		reclaims = append(reclaims, reclaim)
	}
	return reclaims, nil
}

func draftCompositionsFromRequest(requests []draftCompositionRequest) ([]game.DraftCompositionSnapshot, error) {
	drafts := make([]game.DraftCompositionSnapshot, 0, len(requests))
	for _, req := range requests {
		cards, err := cardsFromRequest(req.Cards)
		if err != nil {
			return nil, err
		}
		if len(req.CardPlacements) > 0 && len(req.CardPlacements) != len(cards) {
			return nil, errors.New("draft card placements must match cards")
		}
		cardPlacements := make([]game.DraftCardPlacementSnapshot, 0, len(req.CardPlacements))
		for _, placement := range req.CardPlacements {
			if placement.InsertIndex != nil && *placement.InsertIndex < 0 {
				return nil, errors.New("invalid draft card insert index")
			}
			if placement.ReclaimJokerIndex != nil && *placement.ReclaimJokerIndex < 0 {
				return nil, errors.New("invalid draft reclaim joker index")
			}
			cardPlacements = append(cardPlacements, game.DraftCardPlacementSnapshot{
				InsertIndex:       cloneIntPointer(placement.InsertIndex),
				ReclaimJokerIndex: cloneIntPointer(placement.ReclaimJokerIndex),
			})
		}
		snapshots := make([]game.CardSnapshot, 0, len(cards))
		for _, card := range cards {
			snapshots = append(snapshots, card.Snapshot())
		}
		drafts = append(drafts, game.DraftCompositionSnapshot{
			TableIndex:     req.TableIndex,
			InsertIndex:    req.InsertIndex,
			CardPlacements: cardPlacements,
			Cards:          snapshots,
		})
	}
	return drafts, nil
}

func cloneIntPointer(source *int) *int {
	if source == nil {
		return nil
	}
	cloned := *source
	return &cloned
}

func cardsFromRequest(requests []cardRequest) ([]game.Card, error) {
	cards := make([]game.Card, 0, len(requests))
	for _, req := range requests {
		card, err := cardFromRequest(req)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func cardFromRequest(req cardRequest) (game.Card, error) {
	if req.IsJoker {
		return game.NewJoker(), nil
	}
	rank := game.Rank(req.Rank)
	if rank < game.Ace || rank > game.King {
		return game.Card{}, errors.New("invalid card rank")
	}
	suit := game.Suit(req.Suit)
	if suit < game.Hearts || suit > game.Spades {
		return game.Card{}, errors.New("invalid card suit")
	}
	return game.NewCard(rank, suit), nil
}

func discardCardFromRequest(req *cardRequest) (*game.Card, error) {
	if req == nil {
		return nil, errors.New("discard card is required")
	}
	card, err := cardFromRequest(*req)
	if err != nil {
		return nil, err
	}
	return &card, nil
}

package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/gorilla/websocket"
)

const gameInviteTTL = 10 * time.Minute

type socialStore interface {
	ListSocialSnapshot(ctx context.Context, userID string) (database.SocialSnapshotRecord, error)
	SendFriendRequest(ctx context.Context, senderID, recipientID string) (database.FriendRequestRecord, error)
	RespondFriendRequest(ctx context.Context, recipientID, requestID string, accept bool) (string, error)
	RemoveFriend(ctx context.Context, userID, friendID string) error
	SendGameInvite(ctx context.Context, senderID, recipientID, roomCode string, expiresAt time.Time) (database.GameInviteRecord, error)
	GetGameInvite(ctx context.Context, recipientID, inviteID string) (database.GameInviteRecord, error)
	DeleteGameInvite(ctx context.Context, recipientID, inviteID string) (string, error)
}

type batchSocialStore interface {
	ListSocialSnapshots(ctx context.Context, userIDs []string) (map[string]database.SocialSnapshotRecord, error)
}

type socialUserSnapshot struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	ImageURL   string              `json:"imageUrl,omitempty"`
	Online     bool                `json:"online"`
	ActiveGame *activeGameSnapshot `json:"activeGame,omitempty"`
}

type activeGameSnapshot struct {
	StartedAt time.Time `json:"startedAt"`
}

type friendRequestSnapshot struct {
	ID        string             `json:"id"`
	User      socialUserSnapshot `json:"user"`
	CreatedAt time.Time          `json:"createdAt"`
}

type gameInviteSnapshot struct {
	ID        string             `json:"id"`
	User      socialUserSnapshot `json:"user"`
	RoomCode  string             `json:"roomCode"`
	CreatedAt time.Time          `json:"createdAt"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

type socialStateEvent struct {
	UserID                       string                  `json:"userId"`
	Friends                      []socialUserSnapshot    `json:"friends"`
	IncomingFriendRequests       []friendRequestSnapshot `json:"incomingFriendRequests"`
	OutgoingFriendRequestUserIDs []string                `json:"outgoingFriendRequestUserIds"`
	GameInvites                  []gameInviteSnapshot    `json:"gameInvites"`
}

type sendFriendRequestRequest struct {
	UserID string `json:"userId"`
}

type respondFriendRequestRequest struct {
	RequestID string `json:"requestId"`
	Accept    bool   `json:"accept"`
}

type removeFriendRequest struct {
	UserID string `json:"userId"`
}

type sendGameInviteRequest struct {
	UserID string `json:"userId"`
}

type respondGameInviteRequest struct {
	InviteID string `json:"inviteId"`
	Accept   bool   `json:"accept"`
}

type spectateGameRequest struct {
	UserID string `json:"userId"`
}

type stopSpectatingRequest struct{}

type socialPresence struct {
	mu          sync.RWMutex
	connections map[string]map[*websocket.Conn]struct{}
	usersByConn map[*websocket.Conn]string
}

func newSocialPresence() *socialPresence {
	return &socialPresence{
		connections: make(map[string]map[*websocket.Conn]struct{}),
		usersByConn: make(map[*websocket.Conn]string),
	}
}

func (p *socialPresence) add(userID string, conn *websocket.Conn) {
	if p == nil || userID == "" || conn == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if previousUserID := p.usersByConn[conn]; previousUserID != "" && previousUserID != userID {
		delete(p.connections[previousUserID], conn)
	}
	if p.connections[userID] == nil {
		p.connections[userID] = make(map[*websocket.Conn]struct{})
	}
	p.connections[userID][conn] = struct{}{}
	p.usersByConn[conn] = userID
}

func (p *socialPresence) remove(conn *websocket.Conn) (string, bool) {
	if p == nil || conn == nil {
		return "", false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	userID := p.usersByConn[conn]
	if userID == "" {
		return "", false
	}
	delete(p.usersByConn, conn)
	delete(p.connections[userID], conn)
	if len(p.connections[userID]) > 0 {
		return userID, false
	}
	delete(p.connections, userID)
	return userID, true
}

func (p *socialPresence) isOnline(userID string) bool {
	if p == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections[userID]) > 0
}

func (p *socialPresence) userConnections(userID string) []*websocket.Conn {
	if p == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	connections := make([]*websocket.Conn, 0, len(p.connections[userID]))
	for conn := range p.connections[userID] {
		connections = append(connections, conn)
	}
	return connections
}

func (s *wsServer) socialConnected(userID string, conn *websocket.Conn) {
	if s == nil || s.socialStore == nil || userID == "" {
		return
	}
	s.socialPresence.add(userID, conn)
	record, err := s.loadAndEmitSocialState(userID)
	if err != nil {
		slog.Warn("load social state after connect failed", "userID", userID, "error", err)
		return
	}
	friendIDs := make([]string, 0, len(record.Friends))
	for _, friend := range record.Friends {
		friendIDs = append(friendIDs, friend.ID)
	}
	s.refreshSocialUsers(friendIDs...)
}

func (s *wsServer) socialDisconnected(conn *websocket.Conn) {
	if s == nil || s.socialStore == nil {
		return
	}
	userID, wentOffline := s.socialPresence.remove(conn)
	if !wentOffline {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	record, err := s.socialStore.ListSocialSnapshot(ctx, userID)
	if err != nil {
		slog.Warn("load friends after disconnect failed", "userID", userID, "error", err)
		return
	}
	friendIDs := make([]string, 0, len(record.Friends))
	for _, friend := range record.Friends {
		friendIDs = append(friendIDs, friend.ID)
	}
	s.refreshSocialUsers(friendIDs...)
}

func (s *wsServer) refreshSocialUsers(userIDs ...string) {
	seen := make(map[string]struct{}, len(userIDs))
	uniqueUserIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		uniqueUserIDs = append(uniqueUserIDs, userID)
	}
	if len(uniqueUserIDs) == 0 {
		return
	}
	if store, ok := s.socialStore.(batchSocialStore); ok {
		ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
		records, err := store.ListSocialSnapshots(ctx, uniqueUserIDs)
		cancel()
		if err != nil {
			slog.Warn("refresh social states failed", "userCount", len(uniqueUserIDs), "error", err)
			return
		}
		for _, userID := range uniqueUserIDs {
			s.emitSocialState(userID, records[userID])
		}
		return
	}
	for _, userID := range uniqueUserIDs {
		if _, err := s.loadAndEmitSocialState(userID); err != nil {
			slog.Warn("refresh social state failed", "userID", userID, "error", err)
		}
	}
}

func (s *wsServer) loadAndEmitSocialState(userID string) (database.SocialSnapshotRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	record, err := s.socialStore.ListSocialSnapshot(ctx, userID)
	if err != nil {
		return database.SocialSnapshotRecord{}, err
	}
	event := s.socialEventFromRecord(userID, record)
	s.emitSocialStateEvent(userID, event)
	return record, nil
}

func (s *wsServer) emitSocialState(userID string, record database.SocialSnapshotRecord) {
	s.emitSocialStateEvent(userID, s.socialEventFromRecord(userID, record))
}

func (s *wsServer) emitSocialStateEvent(userID string, event socialStateEvent) {
	for _, conn := range s.socialPresence.userConnections(userID) {
		logEmitFailure(conn, "social_state", event, "write social state failed", "userID", userID)
	}
}

func (s *wsServer) socialEventFromRecord(userID string, record database.SocialSnapshotRecord) socialStateEvent {
	event := socialStateEvent{
		UserID:                       userID,
		Friends:                      make([]socialUserSnapshot, 0, len(record.Friends)),
		IncomingFriendRequests:       make([]friendRequestSnapshot, 0, len(record.IncomingFriendRequests)),
		OutgoingFriendRequestUserIDs: append([]string(nil), record.OutgoingFriendRequestUserIDs...),
		GameInvites:                  make([]gameInviteSnapshot, 0, len(record.GameInvites)),
	}
	for _, friend := range record.Friends {
		event.Friends = append(event.Friends, s.socialFriendFromRecord(friend))
	}
	for _, request := range record.IncomingFriendRequests {
		event.IncomingFriendRequests = append(event.IncomingFriendRequests, friendRequestSnapshot{
			ID: request.ID, User: s.socialUserFromRecord(request.User), CreatedAt: request.CreatedAt,
		})
	}
	for _, invite := range record.GameInvites {
		event.GameInvites = append(event.GameInvites, gameInviteSnapshot{
			ID: invite.ID, User: s.socialUserFromRecord(invite.User), RoomCode: invite.RoomCode,
			CreatedAt: invite.CreatedAt, ExpiresAt: invite.ExpiresAt,
		})
	}
	return event
}

func (s *wsServer) socialUserFromRecord(user database.SocialUserRecord) socialUserSnapshot {
	return socialUserSnapshot{ID: user.ID, Name: user.Name, ImageURL: user.ImageURL, Online: s.socialPresence.isOnline(user.ID)}
}

func (s *wsServer) socialFriendFromRecord(user database.SocialUserRecord) socialUserSnapshot {
	snapshot := s.socialUserFromRecord(user)
	if s.lobby != nil {
		if startedAt, ok := s.lobby.activeGameForUser(user.ID); ok {
			snapshot.ActiveGame = &activeGameSnapshot{StartedAt: startedAt}
		}
	}
	return snapshot
}

func (s *wsServer) handleSpectateGame(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, userID, ok := decodeSocialRequest[spectateGameRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	record, err := s.socialStore.ListSocialSnapshot(ctx, userID)
	if err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	isFriend := false
	for _, friend := range record.Friends {
		if friend.ID == req.UserID {
			isFriend = true
			break
		}
	}
	if !isFriend {
		s.writeActionError(conn, envelope.Type, database.ErrUsersNotFriends)
		return
	}
	if _, active := s.lobby.activeGameForUser(req.UserID); !active {
		s.writeActionError(conn, envelope.Type, errors.New("friend is not in an active game"))
		return
	}

	if oldRoomState, oldRecipients := s.lobby.stopSpectating(conn); oldRoomState != nil {
		s.broadcastRoomState(*oldRoomState, oldRecipients)
	}
	event, roomState, recipients, err := s.lobby.spectateGame(sessionID, req.UserID, conn)
	if err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	s.broadcastRoomState(roomState, recipients)
	logEmitFailure(conn, "game_state", event, "write spectator game state failed", "roomCode", roomState.Code, "userID", userID)
	s.writeSocialActionSuccess(conn, envelope.Type, userID)
}

func (s *wsServer) handleStopSpectating(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	if _, _, ok := decodeSocialRequest[stopSpectatingRequest](s, conn, sessionID, envelope); !ok {
		return
	}
	roomState, recipients := s.lobby.stopSpectating(conn)
	logEmitFailure(conn, "spectating_ended", struct{}{}, "write spectating ended failed")
	if roomState != nil {
		s.broadcastRoomState(*roomState, recipients)
	}
	s.writeSocialActionSuccess(conn, envelope.Type, "")
}

func (s *wsServer) spectatorDisconnected(conn *websocket.Conn) {
	roomState, recipients := s.lobby.stopSpectating(conn)
	if roomState != nil {
		s.broadcastRoomState(*roomState, recipients)
	}
}

func (s *wsServer) refreshFriendsOfPlayers(players []playerSnapshot) {
	if s == nil || s.socialStore == nil {
		return
	}
	userIDs := make([]string, 0, len(players))
	for _, player := range players {
		if player.UserID != "" {
			userIDs = append(userIDs, player.UserID)
		}
	}
	friendIDs := make([]string, 0)
	if store, ok := s.socialStore.(batchSocialStore); ok {
		ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
		records, err := store.ListSocialSnapshots(ctx, userIDs)
		cancel()
		if err != nil {
			slog.Warn("load friends after game state change failed", "userCount", len(userIDs), "error", err)
			return
		}
		for _, userID := range userIDs {
			for _, friend := range records[userID].Friends {
				friendIDs = append(friendIDs, friend.ID)
			}
		}
		s.refreshSocialUsers(friendIDs...)
		return
	}
	for _, userID := range userIDs {
		ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
		record, err := s.socialStore.ListSocialSnapshot(ctx, userID)
		cancel()
		if err != nil {
			slog.Warn("load friends after game state change failed", "userID", userID, "error", err)
			continue
		}
		for _, friend := range record.Friends {
			friendIDs = append(friendIDs, friend.ID)
		}
	}
	s.refreshSocialUsers(friendIDs...)
}

func (s *wsServer) handleSendFriendRequest(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, userID, ok := decodeSocialRequest[sendFriendRequestRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	if _, err := s.socialStore.SendFriendRequest(ctx, userID, req.UserID); err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	s.refreshSocialUsers(userID, req.UserID)
	s.writeSocialActionSuccess(conn, envelope.Type, userID)
}

func (s *wsServer) handleRespondFriendRequest(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, userID, ok := decodeSocialRequest[respondFriendRequestRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	senderID, err := s.socialStore.RespondFriendRequest(ctx, userID, req.RequestID, req.Accept)
	if err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	s.refreshSocialUsers(userID, senderID)
	s.writeSocialActionSuccess(conn, envelope.Type, userID)
}

func (s *wsServer) handleRemoveFriend(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, userID, ok := decodeSocialRequest[removeFriendRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	if err := s.socialStore.RemoveFriend(ctx, userID, req.UserID); err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	s.refreshSocialUsers(userID, req.UserID)
	s.writeSocialActionSuccess(conn, envelope.Type, userID)
}

func (s *wsServer) handleSendGameInvite(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, userID, ok := decodeSocialRequest[sendGameInviteRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	if !s.socialPresence.isOnline(req.UserID) {
		s.writeActionError(conn, envelope.Type, errors.New("friend is not available"))
		return
	}
	roomCode, err := s.lobby.joinableRoomCode(sessionID)
	if err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	if _, err := s.socialStore.SendGameInvite(ctx, userID, req.UserID, roomCode, time.Now().Add(gameInviteTTL)); err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	s.refreshSocialUsers(req.UserID)
	s.writeSocialActionSuccess(conn, envelope.Type, userID)
}

func (s *wsServer) handleRespondGameInvite(conn *websocket.Conn, sessionID string, envelope wsEnvelope) {
	req, userID, ok := decodeSocialRequest[respondGameInviteRequest](s, conn, sessionID, envelope)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	invite, err := s.socialStore.GetGameInvite(ctx, userID, req.InviteID)
	if err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	if req.Accept {
		roomState, recipients, err := s.lobby.joinRoom(sessionID, invite.RoomCode, "")
		if err != nil {
			s.writeActionError(conn, envelope.Type, err)
			return
		}
		if _, err := s.socialStore.DeleteGameInvite(ctx, userID, req.InviteID); err != nil {
			slog.Warn("delete accepted game invite failed", "inviteID", req.InviteID, "userID", userID, "error", err)
		}
		s.broadcastRoomState(roomState, recipients)
	} else if _, err := s.socialStore.DeleteGameInvite(ctx, userID, req.InviteID); err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return
	}
	s.refreshSocialUsers(userID, invite.User.ID)
	s.writeSocialActionSuccess(conn, envelope.Type, userID)
}

func decodeSocialRequest[T any](s *wsServer, conn *websocket.Conn, sessionID string, envelope wsEnvelope) (T, string, bool) {
	var empty T
	if s.socialStore == nil {
		s.writeActionError(conn, envelope.Type, errors.New("social features are unavailable"))
		return empty, "", false
	}
	req, ok := decodeSessionRequest[T](s, conn, sessionID, envelope)
	if !ok {
		return empty, "", false
	}
	userID, err := s.lobby.authenticatedUserID(sessionID)
	if err != nil {
		s.writeActionError(conn, envelope.Type, err)
		return empty, "", false
	}
	return req, userID, true
}

func (s *wsServer) writeSocialActionSuccess(conn *websocket.Conn, action, userID string) {
	logEmitFailure(conn, "action_result", actionResultEvent{Action: action, PlayerID: userID, OK: true}, "write social action result failed", "action", action, "userID", userID)
}

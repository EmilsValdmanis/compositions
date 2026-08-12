package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/gorilla/websocket"
)

type socialStoreStub struct {
	snapshot    database.SocialSnapshotRecord
	invite      database.GameInviteRecord
	err         error
	deleteErr   error
	sendErr     error
	getErr      error
	batchErr    error
	respondedBy string
	listCalls   int
	batchCalls  int
}

func (s *socialStoreStub) ListSocialSnapshot(context.Context, string) (database.SocialSnapshotRecord, error) {
	s.listCalls++
	return s.snapshot, s.err
}
func (s *socialStoreStub) ListSocialSnapshots(_ context.Context, userIDs []string) (map[string]database.SocialSnapshotRecord, error) {
	s.batchCalls++
	if s.batchErr != nil {
		return nil, s.batchErr
	}
	result := make(map[string]database.SocialSnapshotRecord, len(userIDs))
	for _, userID := range userIDs {
		result[userID] = s.snapshot
	}
	return result, nil
}
func (s *socialStoreStub) SendFriendRequest(context.Context, string, string) (database.FriendRequestRecord, error) {
	return database.FriendRequestRecord{}, s.err
}
func (s *socialStoreStub) RespondFriendRequest(context.Context, string, string, bool) (string, error) {
	return s.respondedBy, s.err
}
func (s *socialStoreStub) RemoveFriend(context.Context, string, string) error { return s.err }
func (s *socialStoreStub) SendGameInvite(context.Context, string, string, string, time.Time) (database.GameInviteRecord, error) {
	if s.sendErr != nil {
		return database.GameInviteRecord{}, s.sendErr
	}
	return s.invite, s.err
}
func (s *socialStoreStub) GetGameInvite(context.Context, string, string) (database.GameInviteRecord, error) {
	if s.getErr != nil {
		return database.GameInviteRecord{}, s.getErr
	}
	return s.invite, s.err
}
func (s *socialStoreStub) DeleteGameInvite(context.Context, string, string) (string, error) {
	return s.invite.User.ID, s.deleteErr
}

func newSocialHandlerFixture(t *testing.T, store socialStore) (*wsServer, *websocket.Conn, *websocket.Conn, connectedEvent) {
	t.Helper()
	server := newWSServer()
	server.socialStore = store
	serverConn, clientConn, cleanup := newSocketPair(t)
	t.Cleanup(cleanup)
	event, _, _, err := server.lobby.connectWithUser("", authenticatedUser{ID: "user-1", Name: "User One", Email: "one@example.com"}, serverConn)
	if err != nil {
		t.Fatal(err)
	}
	return server, serverConn, clientConn, event
}

func readSocialResult(t *testing.T, conn *websocket.Conn, wantType string) {
	t.Helper()
	if envelope := mustReadEnvelopeFromConn(t, conn); envelope.Type != wantType {
		t.Fatalf("event type = %q; want %q", envelope.Type, wantType)
	}
}

func readSocialResults(t *testing.T, conn *websocket.Conn, wantTypes ...string) {
	t.Helper()
	wanted := make(map[string]bool, len(wantTypes))
	for _, wantType := range wantTypes {
		wanted[wantType] = false
	}
	for range wantTypes {
		envelope := mustReadEnvelopeFromConn(t, conn)
		if _, ok := wanted[envelope.Type]; !ok {
			t.Fatalf("unexpected event type %q; want %v", envelope.Type, wantTypes)
		}
		wanted[envelope.Type] = true
	}
	for wantType, seen := range wanted {
		if !seen {
			t.Fatalf("missing event type %q", wantType)
		}
	}
}

type batchSocialStoreStub struct {
	socialStore
	listCalls  int
	batchCalls int
}

type nonBatchSocialStore struct {
	socialStore
	snapshot database.SocialSnapshotRecord
	err      error
}

func (s nonBatchSocialStore) ListSocialSnapshot(context.Context, string) (database.SocialSnapshotRecord, error) {
	return s.snapshot, s.err
}

func (s *batchSocialStoreStub) ListSocialSnapshot(context.Context, string) (database.SocialSnapshotRecord, error) {
	s.listCalls++
	return database.SocialSnapshotRecord{}, nil
}

func (s *batchSocialStoreStub) ListSocialSnapshots(_ context.Context, userIDs []string) (map[string]database.SocialSnapshotRecord, error) {
	s.batchCalls++
	result := make(map[string]database.SocialSnapshotRecord, len(userIDs))
	for _, userID := range userIDs {
		result[userID] = database.SocialSnapshotRecord{}
	}
	return result, nil
}

func TestRefreshSocialUsersLoadsSnapshotsInOneBatch(t *testing.T) {
	store := &batchSocialStoreStub{}
	server := &wsServer{socialStore: store, socialPresence: newSocialPresence()}

	server.refreshSocialUsers("user-1", "user-2", "user-1", " ")

	if store.batchCalls != 1 || store.listCalls != 0 {
		t.Fatalf("batch/list calls = %d/%d; want 1/0", store.batchCalls, store.listCalls)
	}
}

func TestRefreshFriendsOfPlayersLoadsSnapshotsInOneBatch(t *testing.T) {
	store := &socialStoreStub{snapshot: database.SocialSnapshotRecord{Friends: []database.SocialUserRecord{{ID: "friend"}}}}
	server := &wsServer{socialStore: store, socialPresence: newSocialPresence()}
	server.refreshFriendsOfPlayers([]playerSnapshot{{UserID: "user-1"}, {UserID: "user-2"}, {}})
	if store.batchCalls != 2 || store.listCalls != 0 {
		t.Fatalf("batch/list calls = %d/%d; want two constant-size batch queries and no per-user queries", store.batchCalls, store.listCalls)
	}
	store.batchErr = errors.New("batch failed")
	server.refreshFriendsOfPlayers([]playerSnapshot{{UserID: "user-1"}})
}

func TestSocialMutationHandlers(t *testing.T) {
	t.Run("friend operations succeed", func(t *testing.T) {
		store := &socialStoreStub{respondedBy: "user-2"}
		server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
		cases := []struct {
			name   string
			handle func()
		}{
			{"send", func() {
				server.handleSendFriendRequest(serverConn, event.SessionID, wsEnvelope{Type: "send_friend_request", Data: mustMarshalRawMessage(sendFriendRequestRequest{UserID: "user-2"})})
			}},
			{"respond", func() {
				server.handleRespondFriendRequest(serverConn, event.SessionID, wsEnvelope{Type: "respond_friend_request", Data: mustMarshalRawMessage(respondFriendRequestRequest{RequestID: "request-1", Accept: true})})
			}},
			{"remove", func() {
				server.handleRemoveFriend(serverConn, event.SessionID, wsEnvelope{Type: "remove_friend", Data: mustMarshalRawMessage(removeFriendRequest{UserID: "user-2"})})
			}},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				test.handle()
				readSocialResult(t, clientConn, "action_result")
			})
		}
	})

	t.Run("friend operation errors", func(t *testing.T) {
		store := &socialStoreStub{err: database.ErrSocialRelationshipExists}
		server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
		server.handleSendFriendRequest(serverConn, event.SessionID, wsEnvelope{Type: "send_friend_request", Data: mustMarshalRawMessage(sendFriendRequestRequest{UserID: "user-2"})})
		readSocialResult(t, clientConn, "error")
		server.handleRespondFriendRequest(serverConn, event.SessionID, wsEnvelope{Type: "respond_friend_request", Data: mustMarshalRawMessage(respondFriendRequestRequest{RequestID: "request-1"})})
		readSocialResult(t, clientConn, "error")
		server.handleRemoveFriend(serverConn, event.SessionID, wsEnvelope{Type: "remove_friend", Data: mustMarshalRawMessage(removeFriendRequest{UserID: "user-2"})})
		readSocialResult(t, clientConn, "error")
	})

	t.Run("invite operations", func(t *testing.T) {
		store := &socialStoreStub{invite: database.GameInviteRecord{ID: "invite-1", RoomCode: "ROOM42", User: database.SocialUserRecord{ID: "user-2"}}}
		server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
		if _, _, err := server.lobby.createRoom(event.SessionID, "User One"); err != nil {
			t.Fatal(err)
		}
		targetConn, _, cleanup := newSocketPair(t)
		defer cleanup()
		server.socialPresence.add("user-2", targetConn)
		server.handleSendGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "send_game_invite", Data: mustMarshalRawMessage(sendGameInviteRequest{UserID: "user-2"})})
		readSocialResult(t, clientConn, "action_result")
		server.handleRespondGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "respond_game_invite", Data: mustMarshalRawMessage(respondGameInviteRequest{InviteID: "invite-1", Accept: false})})
		readSocialResult(t, clientConn, "action_result")
	})

	t.Run("invite errors", func(t *testing.T) {
		store := &socialStoreStub{err: database.ErrGameInviteNotFound}
		server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
		server.handleSendGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "send_game_invite", Data: mustMarshalRawMessage(sendGameInviteRequest{UserID: "offline"})})
		readSocialResult(t, clientConn, "error")
		targetConn, _, cleanup := newSocketPair(t)
		defer cleanup()
		server.socialPresence.add("user-2", targetConn)
		server.handleSendGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "send_game_invite", Data: mustMarshalRawMessage(sendGameInviteRequest{UserID: "user-2"})})
		readSocialResult(t, clientConn, "error")
		server.handleRespondGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "respond_game_invite", Data: mustMarshalRawMessage(respondGameInviteRequest{InviteID: "missing"})})
		readSocialResult(t, clientConn, "error")
	})
}

func TestSocialPresenceAndRefreshBranches(t *testing.T) {
	var nilPresence *socialPresence
	nilPresence.add("user", &websocket.Conn{})
	if _, offline := nilPresence.remove(&websocket.Conn{}); offline || nilPresence.isOnline("user") || nilPresence.userConnections("user") != nil {
		t.Fatal("nil social presence returned live state")
	}
	presence := newSocialPresence()
	conn := &websocket.Conn{}
	presence.add("", conn)
	presence.add("user-1", nil)
	if _, offline := presence.remove(conn); offline {
		t.Fatal("unknown connection went offline")
	}
	presence.add("user-1", conn)
	presence.add("user-2", conn)
	if presence.isOnline("user-1") || !presence.isOnline("user-2") {
		t.Fatal("moving a connection did not update presence")
	}

	store := &socialStoreStub{batchErr: errors.New("batch unavailable")}
	server := &wsServer{socialStore: store, socialPresence: newSocialPresence()}
	server.refreshSocialUsers("", "user-1")
	server.refreshSocialUsers()
	store.batchErr = nil
	server.refreshSocialUsers("user-1", " user-1 ")

	fallback := nonBatchSocialStore{socialStore: &socialStoreStub{}, err: errors.New("load unavailable")}
	fallbackServer := &wsServer{socialStore: fallback, socialPresence: newSocialPresence()}
	fallbackServer.refreshSocialUsers("user-1")
	fallbackServer.refreshFriendsOfPlayers([]playerSnapshot{{UserID: "user-1"}})
	fallback.err = nil
	fallback.snapshot = database.SocialSnapshotRecord{Friends: []database.SocialUserRecord{{ID: "friend-1"}}}
	fallbackServer.socialStore = fallback
	fallbackServer.refreshFriendsOfPlayers([]playerSnapshot{{UserID: "user-1"}})
}

func TestSocialConnectionLifecycle(t *testing.T) {
	store := &socialStoreStub{snapshot: database.SocialSnapshotRecord{
		Friends: []database.SocialUserRecord{{ID: "friend-1", Name: "Friend"}},
	}}
	server := newWSServer()
	server.socialStore = store
	serverConn, clientConn, cleanup := newSocketPair(t)
	defer cleanup()
	server.socialConnected("user-1", serverConn)
	readSocialResult(t, clientConn, "social_state")
	if !server.socialPresence.isOnline("user-1") {
		t.Fatal("connected user is offline")
	}
	server.socialDisconnected(serverConn)
	if server.socialPresence.isOnline("user-1") {
		t.Fatal("disconnected user remains online")
	}

	server.socialConnected("", serverConn)
	server.socialStore = nil
	server.socialConnected("user-1", serverConn)
	server.socialDisconnected(serverConn)

	server.socialStore = &socialStoreStub{err: errors.New("load failed")}
	server.socialConnected("user-2", serverConn)
	server.socialDisconnected(serverConn)

	server.socialStore = store
	first, _, cleanupFirst := newSocketPair(t)
	defer cleanupFirst()
	second, _, cleanupSecond := newSocketPair(t)
	defer cleanupSecond()
	server.socialPresence.add("user-3", first)
	server.socialPresence.add("user-3", second)
	server.socialDisconnected(first)
}

func TestSocialSpectatingHandlers(t *testing.T) {
	lobby, events, roomCode := newActiveLobbyForExitTests(t, 2)
	friend := lobby.sessions[events[0].SessionID]
	friend.authenticated = true
	friend.authUserID = "friend-user"
	lobby.rooms[roomCode].players[0].authUserID = "friend-user"

	viewerConn, viewerPeer, cleanup := newSocketPair(t)
	defer cleanup()
	viewer, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "viewer-user", Name: "Viewer", Email: "viewer@example.com"}, viewerConn)
	if err != nil {
		t.Fatal(err)
	}
	store := &socialStoreStub{snapshot: database.SocialSnapshotRecord{Friends: []database.SocialUserRecord{{ID: "friend-user"}}}}
	server := &wsServer{lobby: lobby, socialStore: store, socialPresence: newSocialPresence()}

	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "friend-user"})})
	readSocialResults(t, viewerPeer, "room_state", "game_state", "action_result")
	server.handleStopSpectating(viewerConn, viewer.SessionID, wsEnvelope{Type: "stop_spectating", Data: mustMarshalRawMessage(stopSpectatingRequest{})})
	readSocialResults(t, viewerPeer, "spectating_ended", "action_result")
	server.spectatorDisconnected(viewerConn)

	store.err = errors.New("load failed")
	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "friend-user"})})
	readSocialResult(t, viewerPeer, "error")
	store.err = nil
	store.snapshot.Friends = nil
	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "stranger"})})
	readSocialResult(t, viewerPeer, "error")
	store.snapshot.Friends = []database.SocialUserRecord{{ID: "inactive-user"}}
	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "inactive-user"})})
	readSocialResult(t, viewerPeer, "error")

	store.snapshot.Friends = []database.SocialUserRecord{{ID: "friend-user"}}
	if _, _, err := lobby.createRoom(viewer.SessionID, "Viewer"); err != nil {
		t.Fatal(err)
	}
	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "friend-user"})})
	readSocialResult(t, viewerPeer, "error")
	if _, _, _, err := lobby.leaveRoom(viewer.SessionID); err != nil {
		t.Fatal(err)
	}

	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "friend-user"})})
	readSocialResults(t, viewerPeer, "room_state", "game_state", "action_result")
	server.handleSpectateGame(viewerConn, viewer.SessionID, wsEnvelope{Type: "spectate_game", Data: mustMarshalRawMessage(spectateGameRequest{UserID: "friend-user"})})
	readSocialResults(t, viewerPeer, "room_state", "game_state", "action_result")
	server.spectatorDisconnected(viewerConn)
}

func TestSocialDecodeAndFriendRefreshBranches(t *testing.T) {
	server, serverConn, clientConn, event := newSocialHandlerFixture(t, &socialStoreStub{})
	server.socialStore = nil
	server.handleSendFriendRequest(serverConn, event.SessionID, wsEnvelope{Type: "send_friend_request", Data: mustMarshalRawMessage(sendFriendRequestRequest{})})
	readSocialResult(t, clientConn, "error")
	server.socialStore = &socialStoreStub{}
	server.handleSendFriendRequest(serverConn, "", wsEnvelope{Type: "send_friend_request", Data: mustMarshalRawMessage(sendFriendRequestRequest{})})
	readSocialResult(t, clientConn, "error")

	server.refreshFriendsOfPlayers(nil)
	server.socialStore = nil
	server.refreshFriendsOfPlayers([]playerSnapshot{{UserID: "user-1"}})
	store := &socialStoreStub{snapshot: database.SocialSnapshotRecord{Friends: []database.SocialUserRecord{{ID: "friend-1"}}}}
	server.socialStore = store
	server.refreshFriendsOfPlayers([]playerSnapshot{{}, {UserID: "user-1"}})
	store.err = errors.New("load failed")
	server.refreshFriendsOfPlayers([]playerSnapshot{{UserID: "user-1"}})
}

func TestSocialEventReflectsLivePresence(t *testing.T) {
	presence := newSocialPresence()
	server := &wsServer{socialPresence: presence}
	friendConnection := &websocket.Conn{}
	presence.add("friend-1", friendConnection)

	event := server.socialEventFromRecord("viewer-1", database.SocialSnapshotRecord{
		Friends: []database.SocialUserRecord{{ID: "friend-1", Name: "Avery"}},
		IncomingFriendRequests: []database.FriendRequestRecord{{
			ID:   "request-1",
			User: database.SocialUserRecord{ID: "requester-1", Name: "Blake"},
		}},
		GameInvites: []database.GameInviteRecord{{ID: "invite-1", RoomCode: "ROOM", User: database.SocialUserRecord{ID: "inviter"}}},
	})

	if len(event.Friends) != 1 || !event.Friends[0].Online {
		t.Fatalf("friends = %#v; want online friend", event.Friends)
	}
	if len(event.IncomingFriendRequests) != 1 || event.IncomingFriendRequests[0].User.Online {
		t.Fatalf("requests = %#v; want offline requester", event.IncomingFriendRequests)
	}
	if len(event.GameInvites) != 1 || event.GameInvites[0].ID != "invite-1" {
		t.Fatalf("invites = %#v", event.GameInvites)
	}

	if userID, wentOffline := presence.remove(friendConnection); userID != "friend-1" || !wentOffline {
		t.Fatalf("remove() = (%q, %t); want (friend-1, true)", userID, wentOffline)
	}
	if presence.isOnline("friend-1") {
		t.Fatal("friend remains online after their last connection is removed")
	}
}

func TestSocialFriendIncludesActiveGame(t *testing.T) {
	lobby, _, code := newActiveLobbyForExitTests(t, 2)
	lobby.rooms[code].players[0].authUserID = "friend-1"
	server := &wsServer{lobby: lobby, socialPresence: newSocialPresence()}
	friend := server.socialFriendFromRecord(database.SocialUserRecord{ID: "friend-1"})
	if friend.ActiveGame == nil || friend.ActiveGame.StartedAt.IsZero() {
		t.Fatalf("friend = %#v; want active game", friend)
	}
}

func TestSocialHandlerRemainingBranches(t *testing.T) {
	t.Run("malformed payloads", func(t *testing.T) {
		server, serverConn, clientConn, event := newSocialHandlerFixture(t, &socialStoreStub{})
		cases := []struct {
			typeName string
			handle   func(wsEnvelope)
		}{
			{"respond_friend_request", func(e wsEnvelope) { server.handleRespondFriendRequest(serverConn, event.SessionID, e) }},
			{"remove_friend", func(e wsEnvelope) { server.handleRemoveFriend(serverConn, event.SessionID, e) }},
			{"send_game_invite", func(e wsEnvelope) { server.handleSendGameInvite(serverConn, event.SessionID, e) }},
			{"respond_game_invite", func(e wsEnvelope) { server.handleRespondGameInvite(serverConn, event.SessionID, e) }},
			{"spectate_game", func(e wsEnvelope) { server.handleSpectateGame(serverConn, event.SessionID, e) }},
			{"stop_spectating", func(e wsEnvelope) { server.handleStopSpectating(serverConn, event.SessionID, e) }},
		}
		for _, testCase := range cases {
			testCase.handle(wsEnvelope{Type: testCase.typeName, Data: []byte(`{"broken":`)})
			readSocialResult(t, clientConn, "error")
		}
	})

	t.Run("authenticated user required", func(t *testing.T) {
		server := newWSServer()
		server.socialStore = &socialStoreStub{}
		serverConn, clientConn, cleanup := newSocketPair(t)
		defer cleanup()
		event, _, _, err := server.lobby.connect("", serverConn)
		if err != nil {
			t.Fatal(err)
		}
		server.handleRemoveFriend(serverConn, event.SessionID, wsEnvelope{Type: "remove_friend", Data: mustMarshalRawMessage(removeFriendRequest{UserID: "friend"})})
		readSocialResult(t, clientConn, "error")
	})

	t.Run("send invite storage failure", func(t *testing.T) {
		store := &socialStoreStub{sendErr: errors.New("send failed")}
		server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
		if _, _, err := server.lobby.createRoom(event.SessionID, "User One"); err != nil {
			t.Fatal(err)
		}
		target := &websocket.Conn{}
		server.socialPresence.add("user-2", target)
		server.handleSendGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "send_game_invite", Data: mustMarshalRawMessage(sendGameInviteRequest{UserID: "user-2"})})
		readSocialResult(t, clientConn, "error")
	})

	t.Run("respond invite branches", func(t *testing.T) {
		t.Run("join failure", func(t *testing.T) {
			store := &socialStoreStub{invite: database.GameInviteRecord{RoomCode: "MISSING", User: database.SocialUserRecord{ID: "friend"}}}
			server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
			server.handleRespondGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "respond_game_invite", Data: mustMarshalRawMessage(respondGameInviteRequest{InviteID: "invite", Accept: true})})
			readSocialResult(t, clientConn, "error")
		})
		t.Run("accepted delete failure remains successful", func(t *testing.T) {
			store := &socialStoreStub{deleteErr: errors.New("delete failed"), invite: database.GameInviteRecord{User: database.SocialUserRecord{ID: "friend"}}}
			server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
			host, _, _, _ := server.lobby.connect("", nil)
			roomState, _, err := server.lobby.createRoom(host.SessionID, "Host")
			if err != nil {
				t.Fatal(err)
			}
			store.invite.RoomCode = roomState.Code
			server.handleRespondGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "respond_game_invite", Data: mustMarshalRawMessage(respondGameInviteRequest{InviteID: "invite", Accept: true})})
			readSocialResults(t, clientConn, "room_state", "action_result")
		})
		t.Run("declined delete failure", func(t *testing.T) {
			store := &socialStoreStub{deleteErr: errors.New("delete failed"), invite: database.GameInviteRecord{User: database.SocialUserRecord{ID: "friend"}}}
			server, serverConn, clientConn, event := newSocialHandlerFixture(t, store)
			server.handleRespondGameInvite(serverConn, event.SessionID, wsEnvelope{Type: "respond_game_invite", Data: mustMarshalRawMessage(respondGameInviteRequest{InviteID: "invite"})})
			readSocialResult(t, clientConn, "error")
		})
	})
}

func TestSocialPresenceKeepsUserOnlineAcrossConnections(t *testing.T) {
	presence := newSocialPresence()
	first := &websocket.Conn{}
	second := &websocket.Conn{}
	presence.add("user-1", first)
	presence.add("user-1", second)

	if _, wentOffline := presence.remove(first); wentOffline {
		t.Fatal("first disconnect marked a user with another connection offline")
	}
	if !presence.isOnline("user-1") {
		t.Fatal("user should remain online while a second connection is active")
	}
	if _, wentOffline := presence.remove(second); !wentOffline {
		t.Fatal("last disconnect did not mark the user offline")
	}
}

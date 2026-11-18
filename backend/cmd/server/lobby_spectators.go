package main

import (
	"time"

	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/gorilla/websocket"
)

func (l *lobbyServer) activeGameForUser(userID string) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	room := l.activeRoomForUserLocked(userID)
	if room == nil {
		return time.Time{}, false
	}
	if room.statisticsStartedAt.IsZero() {
		return time.Now().UTC(), true
	}
	return room.statisticsStartedAt, true
}

func (l *lobbyServer) activeRoomForUserLocked(userID string) *room {
	for _, room := range l.rooms {
		if room == nil || !isSpectatablePhase(room.gameStatePhase()) {
			continue
		}
		for _, player := range room.players {
			if player != nil && player.authUserID == userID && player.connected && !player.forfeited {
				return room
			}
		}
	}
	return nil
}

func (l *lobbyServer) spectateGame(sessionID, friendUserID string, conn *websocket.Conn) (gameStateEvent, roomSnapshot, []*websocket.Conn, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	session, err := l.requireSession(sessionID)
	if err != nil {
		return gameStateEvent{}, roomSnapshot{}, nil, err
	}
	if !session.authenticated || session.authUserID == "" {
		return gameStateEvent{}, roomSnapshot{}, nil, errAuthenticationRequired
	}
	if session.authUserID == friendUserID {
		return gameStateEvent{}, roomSnapshot{}, nil, clientError{"cannot_spectate_self", "cannot spectate yourself"}
	}
	if l.sessionRoom(session) != nil {
		return gameStateEvent{}, roomSnapshot{}, nil, clientError{"spectate_while_in_room", "leave your room before spectating"}
	}

	target := l.activeRoomForUserLocked(friendUserID)
	if target == nil {
		return gameStateEvent{}, roomSnapshot{}, nil, errFriendNotInGame
	}

	l.removeSpectatorLocked(conn)
	if l.spectatorsByRoom[target.code] == nil {
		l.spectatorsByRoom[target.code] = make(map[*websocket.Conn]struct{})
	}
	l.spectatorsByRoom[target.code][conn] = struct{}{}
	l.spectatingRoomByConn[conn] = target.code

	event := l.spectatorGameStateLocked(target)
	recipients := append(target.connectedConns(l.sessions), l.spectatorConnectionsLocked(target.code)...)
	return event, event.Room, recipients, nil
}

func (l *lobbyServer) stopSpectating(conn *websocket.Conn) (*roomSnapshot, []*websocket.Conn) {
	l.mu.Lock()
	defer l.mu.Unlock()

	roomCode := l.removeSpectatorLocked(conn)
	room := l.rooms[roomCode]
	if room == nil {
		return nil, nil
	}
	snapshot := room.snapshot()
	snapshot.SpectatorCount = len(l.spectatorsByRoom[roomCode])
	recipients := append(room.connectedConns(l.sessions), l.spectatorConnectionsLocked(roomCode)...)
	return &snapshot, recipients
}

func (l *lobbyServer) withSpectatorCount(snapshot roomSnapshot) roomSnapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	snapshot.SpectatorCount = len(l.spectatorsByRoom[snapshot.Code])
	return snapshot
}

func (l *lobbyServer) spectatorConnections(roomCode string) []*websocket.Conn {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.spectatorConnectionsLocked(roomCode)
}

func (l *lobbyServer) spectatorGameRecipients(roomCode string) []gameStateRecipient {
	l.mu.Lock()
	defer l.mu.Unlock()

	room := l.rooms[roomCode]
	if room == nil || room.gameState == nil || len(l.spectatorsByRoom[roomCode]) == 0 {
		return nil
	}
	event := l.spectatorGameStateLocked(room)
	recipients := make([]gameStateRecipient, 0, len(l.spectatorsByRoom[roomCode]))
	for conn := range l.spectatorsByRoom[roomCode] {
		recipients = append(recipients, gameStateRecipient{conn: conn, event: event})
	}
	return recipients
}

func (l *lobbyServer) spectatorGameStateLocked(room *room) gameStateEvent {
	state, _ := room.gameState.SnapshotForSpectator()
	state.TurnActivity = room.turnActivityForPlayer("")
	snapshot := room.snapshot()
	snapshot.SpectatorCount = len(l.spectatorsByRoom[room.code])
	return gameStateEvent{Room: snapshot, Game: state, Spectating: true}
}

func (l *lobbyServer) endSpectatingRoom(roomCode string) []*websocket.Conn {
	l.mu.Lock()
	defer l.mu.Unlock()
	connections := l.spectatorConnectionsLocked(roomCode)
	for _, conn := range connections {
		delete(l.spectatingRoomByConn, conn)
	}
	delete(l.spectatorsByRoom, roomCode)
	return connections
}

func (l *lobbyServer) spectatorConnectionsLocked(roomCode string) []*websocket.Conn {
	connections := make([]*websocket.Conn, 0, len(l.spectatorsByRoom[roomCode]))
	for conn := range l.spectatorsByRoom[roomCode] {
		connections = append(connections, conn)
	}
	return connections
}

func (l *lobbyServer) removeSpectatorLocked(conn *websocket.Conn) string {
	roomCode := l.spectatingRoomByConn[conn]
	if roomCode == "" {
		return ""
	}
	delete(l.spectatingRoomByConn, conn)
	delete(l.spectatorsByRoom[roomCode], conn)
	if len(l.spectatorsByRoom[roomCode]) == 0 {
		delete(l.spectatorsByRoom, roomCode)
	}
	return roomCode
}

func isSpectatablePhase(phase game.GamePhase) bool {
	return phase == game.PhaseInProgress || phase == game.PhaseRoundOver
}

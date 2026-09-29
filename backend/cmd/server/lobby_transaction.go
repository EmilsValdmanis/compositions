package main

import (
	"maps"
	"slices"
)

type lobbyMutationState struct {
	sessions          map[string]*playerSession
	rooms             map[string]*room
	pendingStatistics map[string]pendingGameStatistics
}

// Commands are serialized across the singleton persisted lobby. Read-only
// operations keep using mu and remain available while persistence waits on I/O.
func (l *lobbyServer) lockMutation() func() {
	l.commandMu.Lock()
	l.mu.Lock()
	before := &lobbyMutationState{
		sessions: maps.Clone(l.sessions), rooms: maps.Clone(l.rooms),
		pendingStatistics: maps.Clone(l.pendingStatistics),
	}
	for id, session := range before.sessions {
		if session != nil {
			copy := *session
			before.sessions[id] = &copy
		}
	}
	for code, room := range before.rooms {
		if room == nil {
			continue
		}
		copy := *room
		copy.gameState = room.gameState.Clone()
		copy.players = slices.Clone(room.players)
		for i, player := range copy.players {
			if player == nil {
				continue
			}
			playerCopy := *player
			copy.players[i] = &playerCopy
		}
		copy.turnBaseline = cloneGameSnapshot(room.turnBaseline)
		copy.turnActivity = cloneTurnActivitySnapshot(room.turnActivity)
		copy.issueReportCooldowns = maps.Clone(room.issueReportCooldowns)
		if room.endProposal != nil {
			proposal := *room.endProposal
			proposal.eligiblePlayerIDs = slices.Clone(proposal.eligiblePlayerIDs)
			proposal.agreedPlayerIDs = maps.Clone(proposal.agreedPlayerIDs)
			copy.endProposal = &proposal
		}
		before.rooms[code] = &copy
	}
	l.mutationBefore = before
	return func() {
		l.mutationBefore = nil
		l.mu.Unlock()
		l.commandMu.Unlock()
	}
}

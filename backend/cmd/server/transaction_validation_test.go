package main

import (
	"reflect"
	"testing"

	"github.com/EmilsValdmanis/compositions/internal/game"
)

func TestInvalidNextDealRollsBackAndCanRetry(t *testing.T) {
	lobby, events, code := newActiveLobbyForExitTests(t, 2)
	setGameStatePhaseForTest(t, lobby.rooms[code].gameState, game.PhaseRoundOver)
	if _, _, err := lobby.startNextRound(events[0].SessionID); err != nil {
		t.Fatal(err)
	}
	lobby.store = &jsonLobbyStateStore{}
	before := lobby.rooms[code].gameState.PersistenceSnapshot()
	gameID := lobby.rooms[code].statisticsGameID
	chooser := lobby.rooms[code].players[lobby.rooms[code].pendingDealChoice.chooserIndex].sessionID
	cut := -1
	if _, _, err := lobby.chooseDealing(chooser, "round_robin", dealingChoiceOptions{cutSize: &cut}); err == nil {
		t.Fatal("invalid cut succeeded")
	}
	if !reflect.DeepEqual(before, lobby.rooms[code].gameState.PersistenceSnapshot()) {
		t.Fatalf("failed next-round deal changed live state: phase before=%v, after=%v", game.PhaseRoundOver, lobby.rooms[code].gameState.Phase())
	}

	cut = 0
	if _, _, err := lobby.chooseDealing(chooser, "round_robin", dealingChoiceOptions{cutSize: &cut}); err != nil {
		t.Fatal(err)
	}
	if lobby.rooms[code].gameState.RoundNumber() != before.Round+1 || lobby.rooms[code].statisticsGameID != gameID {
		t.Fatal("retry started a different game instead of the next round")
	}
}

func TestInMemoryCommandCommitsWithoutStore(t *testing.T) {
	lobby := newLobbyServer()
	lobby.store = nil
	event, _, _, err := lobby.connect("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := lobby.createRoom(event.SessionID, "Host"); err != nil {
		t.Fatal(err)
	}
	if len(lobby.rooms) != 1 || lobby.sessions[event.SessionID].roomCode == "" {
		t.Fatal("in-memory command was rolled back")
	}
}

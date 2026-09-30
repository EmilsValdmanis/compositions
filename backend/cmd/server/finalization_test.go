package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

func TestFinalizationDurableBeforeReset(t *testing.T) {
	lobby, events, _ := newActiveLobbyForExitTests(t, 2)
	store := &statisticsRecordingStore{gameErr: errors.New("statistics unavailable")}
	lobby.store = store
	for i, event := range events {
		lobby.sessions[event.SessionID].authenticated = true
		lobby.sessions[event.SessionID].authUserID = fmt.Sprintf("user-%d", i)
	}
	if _, _, _, _, err := lobby.forfeitGame(events[0].SessionID); err != nil {
		t.Fatal(err)
	}
	var persisted persistedLobbyState
	if err := json.Unmarshal(store.data, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted.PendingStatistics) != 1 {
		t.Fatal("terminal job missing from authoritative snapshot")
	}
	var original pendingGameStatistics
	for _, record := range persisted.PendingStatistics {
		original = record
	}
	if original.Sequence == 0 {
		t.Fatal("missing durable replay sequence")
	}
	restored := newLobbyServerWithStore(store)
	if err := restored.restorePersistedState(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(restored.pendingStatistics) != 1 {
		t.Errorf("restart before reset restored %d terminal jobs; want 1", len(restored.pendingStatistics))
	}
	for _, record := range restored.pendingStatistics {
		if !record.CompletedAt.Equal(original.CompletedAt) || record.Sequence != original.Sequence {
			t.Fatal("restart changed completion metadata")
		}
	}
	store.gameErr = nil
	restored.retryPendingStatistics()
	if len(store.games) != 1 {
		t.Errorf("maintenance finalized %d games after recovery; want 1", len(store.games))
	}
}

func TestLegacyFinalizationReplayPreservesCompletionOrder(t *testing.T) {
	lobby := newLobbyServer()
	store := &statisticsRecordingStore{}
	lobby.store = store
	lobby.pendingStatistics = make(map[string]pendingGameStatistics)
	for i := 7; i >= 0; i-- {
		id := fmt.Sprintf("game-%d", i)
		completedAt := time.Unix(int64(100+i), 0).UTC()
		lobby.pendingStatistics[id] = pendingGameStatistics{
			Kind: "completed", CompletedAt: completedAt,
			Completed: database.CompletedGameRecord{ID: id, CompletedAt: completedAt},
		}
	}
	saveStatisticsForTest(lobby)
	for i := 1; i < len(store.games); i++ {
		if store.games[i].CompletedAt.Before(store.games[i-1].CompletedAt) {
			t.Fatalf("replayed newer %s before older %s; Elo and streak updates are order-dependent", store.games[i-1].ID, store.games[i].ID)
		}
	}
}

func TestLegacyFinishedRoomRecoversFinalization(t *testing.T) {
	lobby, events, _ := newActiveLobbyForExitTests(t, 2)
	store := &statisticsRecordingStore{gameErr: errors.New("offline")}
	lobby.store = store
	for i, event := range events {
		lobby.sessions[event.SessionID].authenticated = true
		lobby.sessions[event.SessionID].authUserID = fmt.Sprintf("user-%d", i)
	}
	if _, _, _, _, err := lobby.forfeitGame(events[0].SessionID); err != nil {
		t.Fatal(err)
	}
	var persisted persistedLobbyState
	if err := json.Unmarshal(store.data, &persisted); err != nil {
		t.Fatal(err)
	}
	persisted.PendingStatistics = nil // A snapshot written by the previous server.
	store.data, _ = json.Marshal(persisted)
	restored := newLobbyServerWithStore(store)
	if err := restored.restorePersistedState(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.gameErr = nil
	restored.retryPendingStatistics()
	if len(store.games) != 1 || len(restored.pendingStatistics) != 0 {
		t.Fatal("maintenance failed to finalize legacy finished room")
	}
}

type orderedFinalizationStore struct {
	statisticsRecordingStore
	failID   string
	attempts []string
}

func (s *orderedFinalizationStore) SaveCompletedGame(ctx context.Context, record database.CompletedGameRecord) error {
	s.attempts = append(s.attempts, record.ID)
	if record.ID == s.failID {
		return errors.New("temporary finalization failure")
	}
	return s.statisticsRecordingStore.SaveCompletedGame(ctx, record)
}

func TestFinalizationFailureBlocksDependentGamesAcrossRestart(t *testing.T) {
	store := &orderedFinalizationStore{failID: "first"}
	lobby := newLobbyServerWithStore(store)
	lobby.pendingStatistics = make(map[string]pendingGameStatistics)
	// Timestamps intentionally go backward: durable sequence defines commit order.
	for i, item := range []struct {
		id    string
		users []string
	}{
		{"first", []string{"a", "b"}}, {"second", []string{"b", "c"}},
		{"third", []string{"c", "d"}}, {"unrelated", []string{"x", "y"}},
	} {
		players := make([]database.CompletedGamePlayerRecord, len(item.users))
		for j, user := range item.users {
			players[j].UserID = user
		}
		completedAt := time.Unix(int64(100-i), 0).UTC()
		lobby.pendingStatistics[item.id] = pendingGameStatistics{Sequence: uint64(i + 1), Kind: "completed", CompletedAt: completedAt,
			Completed: database.CompletedGameRecord{ID: item.id, CompletedAt: completedAt, Players: players}}
	}
	lobby.retryPendingStatistics()
	if fmt.Sprint(store.attempts) != "[first unrelated]" {
		t.Fatalf("attempts = %v", store.attempts)
	}
	restored := newLobbyServerWithStore(store)
	if err := restored.restorePersistedState(context.Background()); err != nil {
		t.Fatal(err)
	}
	store.failID = ""
	store.attempts = nil
	restored.retryPendingStatistics()
	if fmt.Sprint(store.attempts) != "[first second third]" {
		t.Fatalf("recovery order = %v", store.attempts)
	}
	if len(restored.pendingStatistics) != 0 {
		t.Fatal("recovered jobs remain queued")
	}
	restored.retryPendingStatistics()
	if len(store.games) != 4 {
		t.Fatalf("completed %d games; want exactly four", len(store.games))
	}
}

func TestMultipleFinalizationsAllocateReplaySequence(t *testing.T) {
	lobby, events, code := newActiveLobbyForExitTests(t, 2)
	store := &statisticsRecordingStore{gameErr: errors.New("offline")}
	lobby.store = store
	lobby.pendingStatistics = map[string]pendingGameStatistics{"earlier": {Kind: "completed", Sequence: 41}}
	for i, event := range events {
		lobby.sessions[event.SessionID].authenticated = true
		lobby.sessions[event.SessionID].authUserID = fmt.Sprintf("user-%d", i)
	}
	gameID := lobby.rooms[code].statisticsGameID
	if _, _, _, _, err := lobby.forfeitGame(events[0].SessionID); err != nil {
		t.Fatal(err)
	}
	var persisted persistedLobbyState
	if err := json.Unmarshal(store.data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.PendingStatistics[gameID].Sequence != 42 {
		t.Fatalf("sequence = %d; want 42", persisted.PendingStatistics[gameID].Sequence)
	}
}

func TestLegacyFinalizationTimestampTiesHaveStableOrder(t *testing.T) {
	lobby := newLobbyServer()
	store := &statisticsRecordingStore{}
	lobby.store = store
	completedAt := time.Unix(100, 0).UTC()
	lobby.pendingStatistics = map[string]pendingGameStatistics{}
	for _, id := range []string{"c", "b", "a"} {
		lobby.pendingStatistics[id] = pendingGameStatistics{Kind: "completed", CompletedAt: completedAt, Completed: database.CompletedGameRecord{ID: id}}
	}
	saveStatisticsForTest(lobby)
	if len(store.games) != 3 || store.games[0].ID != "a" || store.games[1].ID != "b" || store.games[2].ID != "c" {
		t.Fatalf("unstable order: %+v", store.games)
	}
}

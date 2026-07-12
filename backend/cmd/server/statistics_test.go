package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/EmilsValdmanis/compositions/internal/game"
	"github.com/gorilla/websocket"
)

type statisticsRecordingStore struct {
	jsonLobbyStateStore
	games            []database.CompletedGameRecord
	checkpoints      []database.GameCheckpointRecord
	unranked         []database.GameCheckpointRecord
	unrankedStatuses []string
	gameErr          error
	lobbySaveCalls   int
	lobbySaveErrors  map[int]error
}

func (s *statisticsRecordingStore) SaveLobbyState(ctx context.Context, state persistedLobbyState) error {
	s.lobbySaveCalls++
	if err := s.lobbySaveErrors[s.lobbySaveCalls]; err != nil {
		return err
	}
	return s.jsonLobbyStateStore.SaveLobbyState(ctx, state)
}

func (s *statisticsRecordingStore) SaveCompletedGame(_ context.Context, completed database.CompletedGameRecord) error {
	if s.gameErr != nil {
		return s.gameErr
	}
	s.games = append(s.games, completed)
	return nil
}

func (s *statisticsRecordingStore) SaveGameCheckpoint(_ context.Context, checkpoint database.GameCheckpointRecord) error {
	if s.gameErr != nil {
		return s.gameErr
	}
	s.checkpoints = append(s.checkpoints, checkpoint)
	return nil
}

func (s *statisticsRecordingStore) SaveUnrankedGame(_ context.Context, checkpoint database.GameCheckpointRecord, status string, _ time.Time) error {
	if s.gameErr != nil {
		return s.gameErr
	}
	s.unranked = append(s.unranked, checkpoint)
	s.unrankedStatuses = append(s.unrankedStatuses, status)
	return nil
}

func TestStatisticsPersistenceIsCrashConsistent(t *testing.T) {
	newLobby := func(t *testing.T, store *statisticsRecordingStore) (*lobbyServer, *room) {
		t.Helper()
		lobby, events, roomCode := newActiveLobbyForExitTests(t, 2)
		lobby.store = store
		gameRoom := lobby.rooms[roomCode]
		gameRoom.statisticsGameID = "00000000-0000-0000-0000-000000000001"
		gameRoom.statisticsStartedAt = time.Now().UTC()
		gameRoom.statisticsDirty = true
		for i, event := range events {
			session := lobby.sessions[event.SessionID]
			session.authenticated = true
			session.authUserID = fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		}
		return lobby, gameRoom
	}

	t.Run("authoritative state failure prevents checkpoint", func(t *testing.T) {
		store := &statisticsRecordingStore{lobbySaveErrors: map[int]error{1: errors.New("state unavailable")}}
		lobby, gameRoom := newLobby(t, store)

		lobby.persistLocked("test")

		if len(store.checkpoints) != 0 {
			t.Fatalf("saved %d checkpoints after state failure; want 0", len(store.checkpoints))
		}
		if !gameRoom.statisticsDirty {
			t.Fatal("state failure cleared the dirty marker")
		}
	})

	t.Run("successful checkpoint clears persisted marker", func(t *testing.T) {
		store := &statisticsRecordingStore{}
		lobby, gameRoom := newLobby(t, store)

		lobby.persistLocked("test")

		if store.lobbySaveCalls != 2 || len(store.checkpoints) != 1 {
			t.Fatalf("lobby saves/checkpoints = %d/%d; want 2/1", store.lobbySaveCalls, len(store.checkpoints))
		}
		if gameRoom.statisticsDirty {
			t.Fatal("successful checkpoint left dirty marker set")
		}
		var persisted persistedLobbyState
		if err := json.Unmarshal(store.data, &persisted); err != nil {
			t.Fatal(err)
		}
		if len(persisted.Rooms) != 1 || persisted.Rooms[0].StatisticsDirty {
			t.Fatalf("persisted marker = %+v; want clean", persisted.Rooms)
		}
	})

	t.Run("marker failure leaves safe retry on disk", func(t *testing.T) {
		store := &statisticsRecordingStore{lobbySaveErrors: map[int]error{2: errors.New("marker unavailable")}}
		lobby, _ := newLobby(t, store)

		lobby.persistLocked("test")

		if len(store.checkpoints) != 1 {
			t.Fatalf("saved %d checkpoints; want 1", len(store.checkpoints))
		}
		var persisted persistedLobbyState
		if err := json.Unmarshal(store.data, &persisted); err != nil {
			t.Fatal(err)
		}
		if len(persisted.Rooms) != 1 || !persisted.Rooms[0].StatisticsDirty {
			t.Fatalf("persisted marker = %+v; want dirty retry marker", persisted.Rooms)
		}
	})
}

func TestStatisticsCheckpointLifecycle(t *testing.T) {
	store := &statisticsRecordingStore{}
	lobby := newLobbyServerWithStore(store)
	events := make([]connectedEvent, 3)
	for i := range events {
		event, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: fmt.Sprintf("user-%d", i), Name: fmt.Sprintf("Player %d", i+1)}, nil)
		if err != nil {
			t.Fatalf("connectWithUser(%d) error = %v", i, err)
		}
		events[i] = event
	}
	roomState, _, err := lobby.createRoom(events[0].SessionID, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(events); i++ {
		if _, _, err := lobby.joinRoom(events[i].SessionID, roomState.Code, "ignored"); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := lobby.startGame(events[0].SessionID, 0); err != nil {
		t.Fatal(err)
	}
	cutSize := 0
	if _, _, err := lobby.chooseDealing(events[2].SessionID, "round_robin", dealingChoiceOptions{cutSize: &cutSize}); err != nil {
		t.Fatal(err)
	}
	if len(store.checkpoints) != 1 || store.checkpoints[0].RoundsPlayed != 1 {
		t.Fatalf("initial checkpoints = %+v", store.checkpoints)
	}

	room := lobby.rooms[roomState.Code]
	snapshot := room.gameState.PersistenceSnapshot()
	currentIndex := snapshot.Turn.PlayerIndex
	currentPlayerID := snapshot.Players[currentIndex].ID
	snapshot.Turn.HasDrawn = true
	snapshot.Players[currentIndex].Hand = []game.CardSnapshot{{Rank: game.Two, Suit: game.Clubs}}
	restored, err := game.RestoreGameState(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	room.gameState = restored
	currentSessionID := ""
	for _, event := range events {
		if event.PlayerID == currentPlayerID {
			currentSessionID = event.SessionID
			break
		}
	}
	if _, _, _, err := lobby.discard(currentSessionID, 0); err != nil {
		t.Fatalf("discard() error = %v", err)
	}
	if len(store.checkpoints) != 2 || store.checkpoints[1].Players[currentIndex].RoundsWon != 1 {
		t.Fatalf("round checkpoint = %+v", store.checkpoints)
	}

	forfeitIndex := 0
	if events[forfeitIndex].PlayerID == currentPlayerID {
		forfeitIndex = 1
	}
	if _, _, _, _, err := lobby.forfeitGame(events[forfeitIndex].SessionID); err != nil {
		t.Fatalf("forfeitGame() error = %v", err)
	}
	if len(store.checkpoints) != 3 {
		t.Fatalf("checkpoint count after forfeit = %d; want 3", len(store.checkpoints))
	}
	foundForfeit := false
	for _, player := range store.checkpoints[2].Players {
		foundForfeit = foundForfeit || player.Forfeited
	}
	if !foundForfeit {
		t.Fatal("forfeit was not retained in immediate checkpoint")
	}

	active := make([]connectedEvent, 0, 2)
	for i, event := range events {
		if i != forfeitIndex {
			active = append(active, event)
		}
	}
	proposal, _, _, err := lobby.reportIssue(active[0].SessionID, "technical problem", true)
	if err != nil {
		t.Fatalf("reportIssue() error = %v", err)
	}
	if _, _, _, err := lobby.voteEndGame(active[1].SessionID, proposal.EndProposal.ID, true); err != nil {
		t.Fatalf("voteEndGame() error = %v", err)
	}
	if len(store.unranked) != 1 || store.unrankedStatuses[0] != "technical_abort" || len(store.games) != 0 {
		t.Fatalf("unranked saves = %+v statuses=%v ranked=%+v", store.unranked, store.unrankedStatuses, store.games)
	}
	foundForfeit = false
	for _, player := range store.unranked[0].Players {
		foundForfeit = foundForfeit || player.Forfeited
	}
	if !foundForfeit {
		t.Fatal("technical abort lost the earlier forfeit checkpoint")
	}
}

func TestMutualEndRetainsUnrankedCheckpoint(t *testing.T) {
	store := &statisticsRecordingStore{}
	lobby := newLobbyServerWithStore(store)
	host, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "host-user", Name: "Host"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	guest, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "guest-user", Name: "Guest"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err := lobby.createRoom(host.SessionID, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := lobby.joinRoom(guest.SessionID, state.Code, "ignored"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lobby.startGame(host.SessionID, 0); err != nil {
		t.Fatal(err)
	}
	cut := 0
	if _, _, err := lobby.chooseDealing(guest.SessionID, "round_robin", dealingChoiceOptions{cutSize: &cut}); err != nil {
		t.Fatal(err)
	}
	proposal, _, _, err := lobby.requestEndGame(host.SessionID, "mutual_end")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := lobby.voteEndGame(guest.SessionID, proposal.EndProposal.ID, true); err != nil {
		t.Fatal(err)
	}
	if len(store.unranked) != 1 || store.unrankedStatuses[0] != "mutual_end" || len(store.unranked[0].Players) != 2 || len(store.games) != 0 {
		t.Fatalf("mutual-end persistence = checkpoints:%+v statuses:%v ranked:%+v", store.unranked, store.unrankedStatuses, store.games)
	}
}

func TestInitialSpecialWinCreatesCheckpoint(t *testing.T) {
	store := &statisticsRecordingStore{}
	lobby := newLobbyServerWithStore(store)
	host, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "special-host", Name: "Host"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	guest, _, _, err := lobby.connectWithUser("", authenticatedUser{ID: "special-guest", Name: "Guest"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, _, err := lobby.createRoom(host.SessionID, "ignored")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := lobby.joinRoom(guest.SessionID, state.Code, "ignored"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lobby.startGame(host.SessionID, 0); err != nil {
		t.Fatal(err)
	}

	cards := make([]game.Card, 0, game.InitialHandSize*2+1)
	for rank := game.Ace; rank <= game.Queen; rank++ {
		cards = append(cards, game.NewCard(rank, game.Hearts), game.NewCard(rank, game.Diamonds))
	}
	cards = append(cards, game.NewCard(game.King, game.Spades))
	prepared := game.NewGameStateWithDeck(cards)
	room := lobby.rooms[state.Code]
	for _, player := range room.players {
		if err := prepared.AddPlayer(newPlayerWithID(player.player.ID)); err != nil {
			t.Fatal(err)
		}
	}
	room.gameState = prepared
	cut := 0
	result, _, err := lobby.chooseDealing(guest.SessionID, "round_robin", dealingChoiceOptions{cutSize: &cut})
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "round_over" || len(store.checkpoints) != 1 {
		t.Fatalf("special opening result = phase:%q checkpoints:%+v", result.Phase, store.checkpoints)
	}
}

func TestRemainingSmallServerBranches(t *testing.T) {
	invalidPath := base64.RawURLEncoding.EncodeToString([]byte("https://evil.example"))
	if got := frontendRedirectURL("https://app.example", invalidPath); got != "https://app.example" {
		t.Fatalf("frontendRedirectURL(invalid path) = %q", got)
	}
	validPath := base64.RawURLEncoding.EncodeToString([]byte("/game"))
	if got := frontendRedirectURL("://bad", validPath); got != "://bad" {
		t.Fatalf("frontendRedirectURL(invalid frontend) = %q", got)
	}
	if _, err := normalizeIssueDescription("   "); err == nil {
		t.Fatal("blank issue description accepted")
	}
	if _, err := normalizeIssueDescription(strings.Repeat("x", maxIssueLength+1)); err == nil {
		t.Fatal("long issue description accepted")
	}

	expired := &room{endProposal: &endGameProposal{expiresAt: time.Now().Add(-time.Second)}}
	if expired.activeEndProposal(time.Now()) != nil || expired.endProposal != nil || expired.endProposalCooldownUntil.IsZero() {
		t.Fatal("expired proposal was not cleared with cooldown")
	}
	if (&endGameProposal{}).unanimouslyApproved() {
		t.Fatal("empty proposal was unanimously approved")
	}
	var nilRoom *room
	nilRoom.removePlayerFromProposal("x")
	nilRoom.transferHostToFirstActive()
	proposalRoom := &room{endProposal: &endGameProposal{
		proposerPlayerID: "host", eligiblePlayerIDs: []string{"host", "guest", "third"},
		agreedPlayerIDs: map[string]bool{"host": true, "guest": true},
	}}
	proposalRoom.removePlayerFromProposal("host")
	if proposalRoom.endProposal != nil {
		t.Fatal("proposal with fewer than two remaining voters was retained")
	}
	if (&room{players: []*roomPlayer{nil}}).allPlayersConnected() {
		t.Fatal("nil room player counted as connected")
	}
	if !(&room{players: []*roomPlayer{{forfeited: true}}}).allPlayersConnected() {
		t.Fatal("forfeited player required a connection")
	}

	report := database.GameBugReportRecord{ID: "report"}
	if got, err := (noopUserStore{}).CreateGameBugReport(context.Background(), report); err != nil || got.ID != report.ID {
		t.Fatalf("noop CreateGameBugReport() = (%+v, %v)", got, err)
	}
	var nilPostgres *postgresUserStore
	if err := nilPostgres.SaveCompletedGame(context.Background(), database.CompletedGameRecord{}); err == nil {
		t.Fatal("nil SaveCompletedGame accepted")
	}
	configured := &postgresUserStore{store: &database.UserStore{}}
	if err := configured.SaveCompletedGame(context.Background(), database.CompletedGameRecord{}); err == nil {
		t.Fatal("empty database store accepted completed game")
	}
}

func TestRestorePersistedConclusionAndProposal(t *testing.T) {
	store := &jsonLobbyStateStore{}
	lobby, events, roomCode := newActiveLobbyForExitTests(t, 2)
	state := lobby.persistenceSnapshotLocked()
	state.Rooms[0].Conclusion = &persistedGameConclusion{Kind: "normal", WinnerPlayerID: events[0].PlayerID, ReportID: "r"}
	state.Rooms[0].EndProposal = &persistedEndGameProposal{
		ID: "proposal", Kind: "mutual_end", ProposerPlayerID: events[0].PlayerID,
		EligiblePlayerIDs: []string{events[0].PlayerID, events[1].PlayerID},
		AgreedPlayerIDs:   []string{events[0].PlayerID}, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}
	if err := store.SaveLobbyState(context.Background(), state); err != nil {
		t.Fatalf("SaveLobbyState() error = %v", err)
	}
	restored := newLobbyServerWithStore(store)
	if err := restored.restorePersistedState(context.Background()); err != nil {
		t.Fatalf("restorePersistedState() error = %v", err)
	}
	room := restored.rooms[roomCode]
	if room.conclusion == nil || room.endProposal == nil || !room.endProposal.agreedPlayerIDs[events[0].PlayerID] {
		t.Fatalf("restored room metadata = %+v", room)
	}
}

func TestSaveCompletedStatisticsBranches(t *testing.T) {
	store := &statisticsRecordingStore{}
	lobby, events, roomCode := newActiveLobbyForExitTests(t, 2)
	lobby.store = store
	completedRoom := lobby.rooms[roomCode]
	for _, event := range events {
		session := lobby.sessions[event.SessionID]
		session.authenticated = true
		session.authUserID = "user-" + event.PlayerID
	}
	completedRoom.statisticsGameID = "game-id"
	completedRoom.statisticsStartedAt = time.Time{}
	if _, _, _, _, err := lobby.forfeitGame(events[0].SessionID); err != nil {
		t.Fatalf("forfeitGame() error = %v", err)
	}
	if len(store.games) != 1 || store.games[0].CompletionKind != "forfeit" || len(store.games[0].Players) != 2 || !completedRoom.statisticsSaved {
		t.Fatalf("saved games = %+v, saved=%v", store.games, completedRoom.statisticsSaved)
	}

	// Retrying a saved room and irrelevant room entries are skipped.
	lobby.rooms["nil"] = nil
	lobby.rooms["lobby"] = &room{gameState: game.NewGameState(), statisticsGameID: "pending"}
	lobby.rooms["no-id"] = &room{gameState: completedRoom.gameState}
	lobby.saveStatisticsLocked(context.Background())
	if len(store.games) != 1 {
		t.Fatalf("retry saved %d games; want 1", len(store.games))
	}

	// An unranked game and a ranked game with no authenticated mappings save no player rows.
	abortedState := completedRoom.gameState.PersistenceSnapshot()
	abortedState.RoundWinnerIndex = -1
	aborted, err := game.RestoreGameState(abortedState)
	if err != nil {
		t.Fatalf("RestoreGameState(aborted) error = %v", err)
	}
	lobby.rooms["abort"] = &room{gameState: aborted, statisticsGameID: "abort"}
	noUsers := &room{gameState: completedRoom.gameState, statisticsGameID: "no-users"}
	for _, result := range completedRoom.gameState.CompletedPlayerStatistics() {
		noUsers.players = append(noUsers.players, &roomPlayer{player: newPlayerWithID(result.PlayerID), sessionID: "missing"})
	}
	lobby.rooms["no-users"] = noUsers
	lobby.saveStatisticsLocked(context.Background())
	if !lobby.rooms["abort"].statisticsSaved || !noUsers.statisticsSaved {
		t.Fatal("unranked/no-user rooms were not finalized")
	}
	partial := &room{gameState: completedRoom.gameState, statisticsGameID: "partial"}
	firstResult := completedRoom.gameState.CompletedPlayerStatistics()[0]
	partial.players = []*roomPlayer{{player: newPlayerWithID(firstResult.PlayerID), sessionID: "partial-session"}}
	lobby.sessions["partial-session"] = &playerSession{sessionID: "partial-session", authenticated: true, authUserID: "partial-user"}
	lobby.rooms["partial"] = partial
	lobby.saveStatisticsLocked(context.Background())
	if !partial.statisticsSaved {
		t.Fatal("partial authenticated result was not saved")
	}

	// A storage failure remains retryable.
	store.gameErr = errors.New("save failed")
	failing := &room{gameState: completedRoom.gameState, statisticsGameID: "failure", statisticsStartedAt: time.Now(), conclusion: &gameConclusion{kind: "forfeit"}}
	for _, result := range completedRoom.gameState.CompletedPlayerStatistics() {
		sid := "failure-" + result.PlayerID
		failing.players = append(failing.players, &roomPlayer{player: newPlayerWithID(result.PlayerID), sessionID: sid})
		lobby.sessions[sid] = &playerSession{sessionID: sid, authenticated: true, authUserID: "user-" + result.PlayerID}
	}
	lobby.rooms["failure"] = failing
	lobby.saveStatisticsLocked(context.Background())
	if failing.statisticsSaved {
		t.Fatal("failed statistics save was marked complete")
	}
}

func TestLobbyRemainingErrorBranches(t *testing.T) {
	lobby := newLobbyServer()
	event, _, _, err := lobby.connect("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := lobby.forfeitGame("missing"); err == nil {
		t.Fatal("missing forfeit session accepted")
	}
	if _, _, _, _, err := lobby.forfeitGame(event.SessionID); err == nil {
		t.Fatal("roomless forfeit accepted")
	}
	if _, _, _, err := lobby.reportIssue("missing", "x", false); err == nil {
		t.Fatal("missing report session accepted")
	}
	if _, _, _, err := lobby.reportIssue(event.SessionID, "x", false); err == nil {
		t.Fatal("roomless report accepted")
	}
	if _, _, _, err := lobby.createEndProposal("missing", "mutual_end", "", ""); err == nil {
		t.Fatal("missing proposal session accepted")
	}
	if _, _, _, err := lobby.createEndProposal(event.SessionID, "mutual_end", "", ""); err == nil {
		t.Fatal("roomless proposal accepted")
	}
	if _, _, _, err := lobby.voteEndGame("missing", "x", true); err == nil {
		t.Fatal("missing vote session accepted")
	}
	if _, _, _, err := lobby.voteEndGame(event.SessionID, "x", true); err == nil {
		t.Fatal("roomless vote accepted")
	}

	active, events, roomCode := newActiveLobbyForExitTests(t, 3)
	room := active.rooms[roomCode]
	if _, _, _, err := active.reportIssue(events[0].SessionID, "", false); err == nil {
		t.Fatal("blank report accepted")
	}
	room.endProposal = room.newEndProposal("mutual_end", events[0].PlayerID, "", "")
	if _, _, _, err := active.reportIssue(events[0].SessionID, "x", true); err == nil {
		t.Fatal("report alongside proposal accepted")
	}
	room.endProposal = nil
	room.endProposalCooldownUntil = time.Now().Add(time.Minute)
	if _, _, _, err := active.reportIssue(events[0].SessionID, "x", true); err == nil {
		t.Fatal("report during cooldown accepted")
	}
	room.endProposalCooldownUntil = time.Time{}
	if _, _, _, err := active.createEndProposal(events[0].SessionID, "bad", "", ""); err == nil {
		t.Fatal("unknown proposal accepted")
	}
	room.endProposal = room.newEndProposal("mutual_end", events[0].PlayerID, "", "")
	if _, _, _, err := active.createEndProposal(events[0].SessionID, "mutual_end", "", ""); err == nil {
		t.Fatal("duplicate proposal accepted")
	}
	if _, _, _, err := active.voteEndGame(events[0].SessionID, "wrong", true); err == nil {
		t.Fatal("wrong proposal vote accepted")
	}
	room.endProposal.eligiblePlayerIDs = []string{events[1].PlayerID}
	if _, _, _, err := active.voteEndGame(events[0].SessionID, room.endProposal.id, true); err == nil {
		t.Fatal("ineligible vote accepted")
	}

	// Force the unanimous vote's game-ending operation to fail.
	room.endProposal = &endGameProposal{id: "forced", eligiblePlayerIDs: []string{events[0].PlayerID}, agreedPlayerIDs: map[string]bool{}, expiresAt: time.Now().Add(time.Minute)}
	setGameStatePhaseForTest(t, room.gameState, game.PhaseLobby)
	if _, _, _, err := active.voteEndGame(events[0].SessionID, "forced", true); err == nil {
		t.Fatal("vote ending a lobby game succeeded")
	}
}

func addRecipientGhost(lobby *lobbyServer, target *room) {
	const sessionID = "ghost-session"
	target.players = append(target.players, &roomPlayer{player: newPlayerWithID("ghost-player"), sessionID: sessionID, connected: true})
	lobby.sessions[sessionID] = &playerSession{sessionID: sessionID, playerID: "ghost-player", roomCode: target.code, conn: &websocket.Conn{}}
}

func TestLobbyRemainingStateAndRecipientErrors(t *testing.T) {
	t.Run("next round has too few active players", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 2)
		room := lobby.rooms[code]
		snapshot := room.gameState.PersistenceSnapshot()
		snapshot.Phase = game.PhaseRoundOver
		snapshot.Players[1].Forfeited = true
		restored, err := game.RestoreGameState(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		room.gameState = restored
		room.players[1].forfeited = true
		if _, _, err := lobby.startNextRound(events[0].SessionID); !errors.Is(err, game.ErrNotEnoughPlayers) {
			t.Fatalf("startNextRound() error = %v", err)
		}
	})

	for _, test := range []struct {
		name   string
		mutate func(*lobbyServer, *room)
	}{
		{"inactive room player", func(_ *lobbyServer, room *room) { room.players[0].forfeited = true }},
		{"missing game state", func(_ *lobbyServer, room *room) { room.gameState = nil }},
		{"game rejects forfeit", func(_ *lobbyServer, room *room) { setGameStatePhaseForTest(t, room.gameState, game.PhaseLobby) }},
	} {
		t.Run("forfeit "+test.name, func(t *testing.T) {
			lobby, events, code := newActiveLobbyForExitTests(t, 3)
			test.mutate(lobby, lobby.rooms[code])
			if _, _, _, _, err := lobby.forfeitGame(events[0].SessionID); err == nil {
				t.Fatal("forfeitGame() error = nil")
			}
		})
	}

	t.Run("forfeit completes unanimous proposal", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 4)
		room := lobby.rooms[code]
		room.endProposal = &endGameProposal{
			id: "proposal", kind: "mutual_end", proposerPlayerID: events[1].PlayerID,
			eligiblePlayerIDs: []string{events[1].PlayerID, events[2].PlayerID, events[3].PlayerID},
			agreedPlayerIDs:   map[string]bool{events[1].PlayerID: true, events[2].PlayerID: true, events[3].PlayerID: true},
			expiresAt:         time.Now().Add(time.Minute),
		}
		state, _, _, _, err := lobby.forfeitGame(events[0].SessionID)
		if err != nil || state.Conclusion == nil || state.Conclusion.Kind != "mutual_end" {
			t.Fatalf("forfeitGame() = (%+v, %v)", state, err)
		}
	})

	t.Run("forfeit recipient snapshot error", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 3)
		addRecipientGhost(lobby, lobby.rooms[code])
		if _, _, _, _, err := lobby.forfeitGame(events[0].SessionID); err == nil {
			t.Fatal("recipient snapshot error = nil")
		}
	})

	t.Run("report state validation and recipient error", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 2)
		room := lobby.rooms[code]
		setGameStatePhaseForTest(t, room.gameState, game.PhaseLobby)
		if _, _, _, err := lobby.reportIssue(events[0].SessionID, "x", false); !errors.Is(err, game.ErrGameNotInProgress) {
			t.Fatalf("phase error = %v", err)
		}
		setGameStatePhaseForTest(t, room.gameState, game.PhaseInProgress)
		room.players[0].forfeited = true
		if _, _, _, err := lobby.reportIssue(events[0].SessionID, "x", false); err == nil {
			t.Fatal("inactive reporter accepted")
		}
		room.players[0].forfeited = false
		addRecipientGhost(lobby, room)
		if _, _, _, err := lobby.reportIssue(events[0].SessionID, "x", false); err == nil {
			t.Fatal("report recipient snapshot error = nil")
		}
	})

	t.Run("proposal state validation and recipient error", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 2)
		room := lobby.rooms[code]
		setGameStatePhaseForTest(t, room.gameState, game.PhaseLobby)
		if _, _, _, err := lobby.createEndProposal(events[0].SessionID, "mutual_end", "", ""); !errors.Is(err, game.ErrGameNotInProgress) {
			t.Fatalf("phase error = %v", err)
		}
		setGameStatePhaseForTest(t, room.gameState, game.PhaseInProgress)
		room.players[0].forfeited = true
		if _, _, _, err := lobby.createEndProposal(events[0].SessionID, "mutual_end", "", ""); err == nil {
			t.Fatal("inactive proposer accepted")
		}
		room.players[0].forfeited = false
		addRecipientGhost(lobby, room)
		if _, _, _, err := lobby.createEndProposal(events[0].SessionID, "mutual_end", "", ""); err == nil {
			t.Fatal("proposal recipient snapshot error = nil")
		}
	})

	t.Run("vote recipient snapshot error", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 2)
		room := lobby.rooms[code]
		room.endProposal = room.newEndProposal("mutual_end", events[0].PlayerID, "", "")
		addRecipientGhost(lobby, room)
		if _, _, _, err := lobby.voteEndGame(events[0].SessionID, room.endProposal.id, false); err == nil {
			t.Fatal("vote recipient snapshot error = nil")
		}
	})

	t.Run("reset replaces missing host", func(t *testing.T) {
		room := &room{gameState: game.NewGameState(), hostID: "missing", players: []*roomPlayer{{player: newPlayerWithID("active")}}}
		if err := room.resetForLobby(); err != nil || room.hostID != "active" {
			t.Fatalf("resetForLobby() = (%q, %v)", room.hostID, err)
		}
	})
}

func TestWebSocketHandlerDecodeFailures(t *testing.T) {
	server := newWSServer()
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = func(_ *websocket.Conn, _ string, _ any) error { return nil }
	empty := wsEnvelope{}
	server.handleForfeitGame(nil, "", empty)
	server.handleRequestEndGame(nil, "", empty)
	server.handleVoteEndGame(nil, "", empty)
	server.handleReportIssue(nil, "", empty)
	server.handlePlayAndDiscard(nil, "", empty)
}

func TestWebSocketRemainingActionHandlers(t *testing.T) {
	originalEmit := emitEvent
	defer func() { emitEvent = originalEmit }()
	emitEvent = func(_ *websocket.Conn, _ string, _ any) error { return nil }

	t.Run("successful simple action handlers", func(t *testing.T) {
		forfeitLobby, forfeitEvents, _ := newActiveLobbyForExitTests(t, 2)
		server := newWSServer()
		server.lobby = forfeitLobby
		server.handleForfeitGame(nil, forfeitEvents[0].SessionID, wsEnvelope{Data: mustMarshalRawMessage(forfeitGameRequest{})})

		requestLobby, requestEvents, _ := newActiveLobbyForExitTests(t, 2)
		server.lobby = requestLobby
		server.handleRequestEndGame(nil, requestEvents[0].SessionID, wsEnvelope{Data: mustMarshalRawMessage(requestEndGameRequest{Kind: "mutual_end"})})

		voteLobby, voteEvents, voteCode := newActiveLobbyForExitTests(t, 2)
		voteRoom := voteLobby.rooms[voteCode]
		voteRoom.endProposal = voteRoom.newEndProposal("mutual_end", voteEvents[0].PlayerID, "", "")
		server.lobby = voteLobby
		server.handleVoteEndGame(nil, voteEvents[1].SessionID, wsEnvelope{Data: mustMarshalRawMessage(voteEndGameRequest{ProposalID: voteRoom.endProposal.id, Approve: true})})

		reportLobby, reportEvents, _ := newActiveLobbyForExitTests(t, 2)
		reportLobby.store = &jsonLobbyStateStore{}
		server.lobby = reportLobby
		server.handleReportIssue(nil, reportEvents[0].SessionID, wsEnvelope{Data: mustMarshalRawMessage(reportIssueRequest{Description: "test issue"})})
	})

	t.Run("handler lobby errors", func(t *testing.T) {
		server := newWSServer()
		event, _, _, err := server.lobby.connect("", nil)
		if err != nil {
			t.Fatal(err)
		}
		server.handleForfeitGame(nil, event.SessionID, wsEnvelope{Data: mustMarshalRawMessage(forfeitGameRequest{})})
		server.handleRequestEndGame(nil, event.SessionID, wsEnvelope{Data: mustMarshalRawMessage(requestEndGameRequest{Kind: "mutual_end"})})
		server.handleVoteEndGame(nil, event.SessionID, wsEnvelope{Data: mustMarshalRawMessage(voteEndGameRequest{ProposalID: "x"})})
		server.handleReportIssue(nil, event.SessionID, wsEnvelope{Data: mustMarshalRawMessage(reportIssueRequest{Description: "x"})})
	})

	t.Run("play and discard conversions, lobby error, and success", func(t *testing.T) {
		lobby, events, code := newActiveLobbyForExitTests(t, 2)
		server := newWSServer()
		server.lobby = lobby
		room := lobby.rooms[code]
		currentID := room.gameState.PersistenceSnapshot().Players[room.gameState.CurrentPlayerIndex()].ID
		currentSession := events[0].SessionID
		otherSession := events[1].SessionID
		if events[1].PlayerID == currentID {
			currentSession, otherSession = otherSession, currentSession
		}

		invalidCard := cardRequest{Rank: 99, Suit: int(game.Hearts)}
		server.handlePlayAndDiscard(nil, currentSession, wsEnvelope{Data: mustMarshalRawMessage(playAndDiscardRequest{playRequest: playRequest{Compositions: []compositionRequest{{Cards: []cardRequest{invalidCard}}}}})})
		server.handlePlayAndDiscard(nil, currentSession, wsEnvelope{Data: mustMarshalRawMessage(playAndDiscardRequest{playRequest: playRequest{Additions: []compositionAdditionRequest{{Cards: []cardRequest{invalidCard}}}}})})
		server.handlePlayAndDiscard(nil, currentSession, wsEnvelope{Data: mustMarshalRawMessage(playAndDiscardRequest{playRequest: playRequest{Reclaims: []reclaimRequest{{ReplacementCard: invalidCard}}}})})

		validRun := []cardRequest{
			{Rank: int(game.Ten), Suit: int(game.Hearts)}, {Rank: int(game.Jack), Suit: int(game.Hearts)},
			{Rank: int(game.Queen), Suit: int(game.Hearts)}, {Rank: int(game.King), Suit: int(game.Hearts)},
		}
		request := playAndDiscardRequest{playRequest: playRequest{Compositions: []compositionRequest{{Cards: validRun}}}, CardIndex: 0}
		server.handlePlayAndDiscard(nil, otherSession, wsEnvelope{Data: mustMarshalRawMessage(request)})

		snapshot := room.gameState.PersistenceSnapshot()
		index := snapshot.Turn.PlayerIndex
		snapshot.Turn.HasDrawn = true
		snapshot.Turn.MustUseDiscardDraw = false
		snapshot.Players[index].HasOpened = false
		snapshot.Players[index].Hand = []game.CardSnapshot{
			{Rank: game.Ten, Suit: game.Hearts}, {Rank: game.Jack, Suit: game.Hearts},
			{Rank: game.Queen, Suit: game.Hearts}, {Rank: game.King, Suit: game.Hearts},
			{Rank: game.Two, Suit: game.Clubs},
		}
		restored, err := game.RestoreGameState(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		room.gameState = restored
		server.handlePlayAndDiscard(nil, currentSession, wsEnvelope{Data: mustMarshalRawMessage(request)})
	})
}

func TestHandleConnectionDispatchesNewActions(t *testing.T) {
	server := newWSServer()
	httpServer := httptest.NewServer(server.routes())
	defer httpServer.Close()
	conn := mustDialWS(t, httpServer.URL)
	defer conn.Close()
	connected := mustConnectSession(t, conn, "")
	_ = connected

	cases := []struct {
		kind string
		data any
	}{
		{"forfeit_game", forfeitGameRequest{}},
		{"request_end_game", requestEndGameRequest{Kind: "mutual_end"}},
		{"vote_end_game", voteEndGameRequest{ProposalID: "x"}},
		{"report_issue", reportIssueRequest{Description: "x"}},
		{"play_and_discard", playAndDiscardRequest{CardIndex: 0}},
	}
	for _, test := range cases {
		mustSendEnvelope(t, conn, test.kind, test.data)
		mustReadError(t, conn, "join a room first")
	}

	// Keep imports and the HTTP upgrade path explicit in this coverage-focused test.
	_ = httptest.NewRequest(http.MethodGet, "/ws", nil)
}

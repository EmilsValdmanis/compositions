package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type playerProfileTestStore struct {
	userStore
	profile       database.PlayerProfileRecord
	history       database.PlayerGameHistoryPage
	err           error
	historyErr    error
	historyLimit  int
	historyOffset int
}

func (s playerProfileTestStore) GetPlayerProfile(context.Context, string) (database.PlayerProfileRecord, error) {
	return s.profile, s.err
}

func (s *playerProfileTestStore) GetPlayerGameHistory(_ context.Context, _ string, limit, offset int) (database.PlayerGameHistoryPage, error) {
	s.historyLimit = limit
	s.historyOffset = offset
	return s.history, s.historyErr
}

func TestHandlePlayerProfile(t *testing.T) {
	profile := database.PlayerProfileRecord{
		ID: "00000000-0000-0000-0000-000000000002", Name: "Avery", ImageURL: "https://example.com/avatar.png",
		GamesPlayed: 8, GamesWon: 3, TotalPlacement: 14, RoundsPlayed: 22, RoundsWon: 9, Forfeits: 2,
		TotalPlaytimeSeconds: 5_430,
		CompositionsCreated:  41, SetsCreated: 17, RunsCreated: 24, PointsInflicted: 230,
		LongestGameWinStreak: 2,
	}
	store := &playerProfileTestStore{userStore: noopUserStore{}, profile: profile}
	server := newWSServerWithDependencies(nil, store, "")
	request := httptest.NewRequest(http.MethodGet, "/api/players/"+profile.ID, nil)
	response := httptest.NewRecorder()

	server.routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
	if strings.Contains(response.Body.String(), `"forfeits"`) {
		t.Fatalf("public profile response exposes forfeits: %s", response.Body.String())
	}
	var payload playerProfileResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != profile.ID || payload.Name != profile.Name || payload.GamesPlayed != 8 || payload.GamesWon != 3 || payload.TotalPlaytimeSeconds != 5_430 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestHandlePlayerGameHistory(t *testing.T) {
	playerID := "00000000-0000-0000-0000-000000000002"
	completedAt := time.Date(2026, time.July, 15, 12, 30, 0, 0, time.UTC)
	store := &playerProfileTestStore{
		userStore: noopUserStore{},
		history: database.PlayerGameHistoryPage{
			Total: 23,
			Games: []database.PlayerGameHistoryRecord{{
				GameID: "00000000-0000-0000-0000-000000000003", Status: "completed",
				CompletedAt: completedAt, Placement: 1, PlayerCount: 3, Won: true,
				TotalPoints: 42, RoundsPlayed: 4, RoundsWon: 2, PlaytimeSeconds: 1800,
			}},
		},
	}
	server := newWSServerWithDependencies(nil, store, "")
	response := httptest.NewRecorder()
	server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/"+playerID+"/games?page=2&pageSize=10", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
	if store.historyLimit != 10 || store.historyOffset != 10 {
		t.Fatalf("pagination = limit:%d offset:%d; want 10/10", store.historyLimit, store.historyOffset)
	}
	var payload playerGameHistoryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Page != 2 || payload.TotalItems != 23 || payload.TotalPages != 3 || len(payload.Games) != 1 || !payload.Games[0].Won {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestHandlePlayerGameHistoryRejectsInvalidPagination(t *testing.T) {
	playerID := "00000000-0000-0000-0000-000000000002"
	server := newWSServerWithDependencies(nil, &playerProfileTestStore{userStore: noopUserStore{}}, "")
	for _, query := range []string{"page=0", "page=nope", "pageSize=0", "pageSize=51"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/players/"+playerID+"/games?"+query, nil)
		server.routes().ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("query %q status = %d; want %d", query, response.Code, http.StatusBadRequest)
		}
	}
}

func TestHandlePlayerProfileRejectsInvalidID(t *testing.T) {
	server := newWSServer()
	response := httptest.NewRecorder()
	server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/player-id", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusNotFound)
	}
}

func TestHandlePlayerProfileBranches(t *testing.T) {
	validID := "00000000-0000-0000-0000-000000000002"

	t.Run("handles preflight", func(t *testing.T) {
		server := newWSServer()
		request := httptest.NewRequest(http.MethodOptions, "/api/players/"+validID, nil)
		request.Header.Set("Origin", "http://localhost:3000")
		response := httptest.NewRecorder()
		server.handlePlayerProfile(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNoContent)
		}
	})

	t.Run("rejects methods other than get", func(t *testing.T) {
		server := newWSServer()
		response := httptest.NewRecorder()
		server.handlePlayerProfile(response, httptest.NewRequest(http.MethodPost, "/api/players/"+validID, nil))
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
			t.Fatalf("status = %d allow = %q; want 405 and GET", response.Code, response.Header().Get("Allow"))
		}
	})

	for name, path := range map[string]string{
		"empty id":     "/api/players/",
		"nested path":  "/api/players/" + validID + "/extra",
		"malformed id": "/api/players/not-a-uuid",
	} {
		t.Run(name, func(t *testing.T) {
			server := newWSServer()
			response := httptest.NewRecorder()
			server.handlePlayerProfile(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d; want %d", response.Code, http.StatusNotFound)
			}
		})
	}

	t.Run("requires profile storage", func(t *testing.T) {
		server := newWSServerWithDependencies(nil, noopUserStore{}, "")
		response := httptest.NewRecorder()
		server.handlePlayerProfile(response, httptest.NewRequest(http.MethodGet, "/api/players/"+validID, nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusServiceUnavailable)
		}
	})

	for name, testCase := range map[string]struct {
		err        error
		wantStatus int
	}{
		"missing profile": {err: database.ErrPlayerProfileNotFound, wantStatus: http.StatusNotFound},
		"storage failure": {err: errors.New("profile store boom"), wantStatus: http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			server := newWSServerWithDependencies(nil, playerProfileTestStore{userStore: noopUserStore{}, err: testCase.err}, "")
			response := httptest.NewRecorder()
			server.handlePlayerProfile(response, httptest.NewRequest(http.MethodGet, "/api/players/"+validID, nil))
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d; want %d", response.Code, testCase.wantStatus)
			}
		})
	}

	t.Run("handles response write failures", func(t *testing.T) {
		server := newWSServerWithDependencies(nil, playerProfileTestStore{
			userStore: noopUserStore{},
			profile:   database.PlayerProfileRecord{ID: validID},
		}, "")
		server.handlePlayerProfile(&failingProfileResponseWriter{header: http.Header{}}, httptest.NewRequest(http.MethodGet, "/api/players/"+validID, nil))
	})
}

type failingProfileResponseWriter struct {
	header http.Header
}

func (w *failingProfileResponseWriter) Header() http.Header { return w.header }
func (*failingProfileResponseWriter) WriteHeader(int)       {}
func (*failingProfileResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRoomSnapshotIncludesAuthenticatedProfileID(t *testing.T) {
	gameState := makeGameState()
	player := newPlayer()
	if err := addPlayerToGameState(gameState, player); err != nil {
		t.Fatalf("add player: %v", err)
	}
	room := &room{gameState: gameState, players: []*roomPlayer{{
		player: player, authUserID: "profile-id", name: "Avery", connected: true,
	}}}

	snapshot := room.snapshot()
	if len(snapshot.Players) != 1 || snapshot.Players[0].UserID != "profile-id" {
		t.Fatalf("players = %+v", snapshot.Players)
	}
}

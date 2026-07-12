package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type playerProfileTestStore struct {
	userStore
	profile database.PlayerProfileRecord
	err     error
}

func (s playerProfileTestStore) GetPlayerProfile(context.Context, string) (database.PlayerProfileRecord, error) {
	return s.profile, s.err
}

func TestHandlePlayerProfile(t *testing.T) {
	profile := database.PlayerProfileRecord{
		ID: "00000000-0000-0000-0000-000000000002", Name: "Avery", ImageURL: "https://example.com/avatar.png",
		GamesPlayed: 8, GamesWon: 3, TotalPlacement: 14, RoundsPlayed: 22, RoundsWon: 9,
		CompositionsCreated: 41, SetsCreated: 17, RunsCreated: 24, PointsInflicted: 230,
		LongestGameWinStreak: 2,
	}
	store := playerProfileTestStore{userStore: noopUserStore{}, profile: profile}
	authStore := &recordingSessionStore{user: authenticatedUser{
		ID: "00000000-0000-0000-0000-000000000001", Name: "Viewer", Email: "viewer@example.com",
	}}
	server := newWSServerWithDependencies(&authHandler{store: authStore, now: time.Now}, store, "")
	request := httptest.NewRequest(http.MethodGet, "/api/players/"+profile.ID, nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
	response := httptest.NewRecorder()

	server.routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
	var payload playerProfileResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != profile.ID || payload.Name != profile.Name || payload.GamesPlayed != 8 || payload.GamesWon != 3 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestHandlePlayerProfileRequiresAuthentication(t *testing.T) {
	server := newWSServer()
	response := httptest.NewRecorder()
	server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/players/player-id", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusUnauthorized)
	}
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

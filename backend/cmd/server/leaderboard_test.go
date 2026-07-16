package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type leaderboardTestStore struct {
	userStore
	page     database.LeaderboardPage
	err      error
	cursor   *database.LeaderboardCursor
	limit    int
	viewerID string
}

func (s *leaderboardTestStore) GetLeaderboard(_ context.Context, cursor *database.LeaderboardCursor, limit int, viewerID string) (database.LeaderboardPage, error) {
	s.cursor = cursor
	s.limit = limit
	s.viewerID = viewerID
	return s.page, s.err
}

func TestHandleLeaderboard(t *testing.T) {
	firstID := "00000000-0000-0000-0000-000000000001"
	secondID := "00000000-0000-0000-0000-000000000002"
	store := &leaderboardTestStore{
		userStore: noopUserStore{},
		page: database.LeaderboardPage{
			Players: []database.LeaderboardPlayerRecord{{
				Rank: 1, PlayerID: firstID, Name: "Avery", Wins: 12, GamesPlayed: 20,
				RoundsWon: 48, PointsInflicted: 920, TotalPlaytimeSeconds: 18_400,
			}},
			NextCursor: &database.LeaderboardCursor{Score: 12, PlayerID: firstID},
			Placement: &database.LeaderboardPlayerRecord{
				Rank: 8, PlayerID: secondID, Name: "Casey", Wins: 4, GamesPlayed: 9,
			},
		},
	}
	server := newWSServerWithDependencies(nil, store, "")
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/leaderboard?playerId="+secondID, nil)
	server.routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
	if store.limit != leaderboardPageSize || store.viewerID != secondID || store.cursor != nil {
		t.Fatalf("request = limit:%d viewer:%q cursor:%+v", store.limit, store.viewerID, store.cursor)
	}
	var payload leaderboardResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Players) != 1 || payload.Players[0].Wins != 12 || payload.Placement == nil || payload.Placement.Rank != 8 || payload.NextCursor == nil {
		t.Fatalf("payload = %+v", payload)
	}

	nextResponse := httptest.NewRecorder()
	nextURL := "/api/leaderboard?limit=25&cursor=" + url.QueryEscape(*payload.NextCursor)
	server.routes().ServeHTTP(nextResponse, httptest.NewRequest(http.MethodGet, nextURL, nil))
	if nextResponse.Code != http.StatusOK || store.limit != 25 || store.cursor == nil || store.cursor.Score != 12 || store.cursor.PlayerID != firstID {
		t.Fatalf("next page status:%d limit:%d cursor:%+v", nextResponse.Code, store.limit, store.cursor)
	}
}

func TestHandleLeaderboardRejectsInvalidRequests(t *testing.T) {
	server := newWSServerWithDependencies(nil, &leaderboardTestStore{userStore: noopUserStore{}}, "")
	for _, path := range []string{
		"/api/leaderboard?limit=0",
		"/api/leaderboard?limit=51",
		"/api/leaderboard?cursor=not-a-cursor",
		"/api/leaderboard?playerId=not-a-player",
	} {
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path %q status = %d; want %d", path, response.Code, http.StatusBadRequest)
		}
	}

	response := httptest.NewRecorder()
	server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/leaderboard", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("post status = %d allow = %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestHandleLeaderboardUnavailableAndFailure(t *testing.T) {
	t.Run("unavailable", func(t *testing.T) {
		server := newWSServerWithDependencies(nil, noopUserStore{}, "")
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/leaderboard", nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("store failure", func(t *testing.T) {
		store := &leaderboardTestStore{userStore: noopUserStore{}, err: errors.New("database failed")}
		server := newWSServerWithDependencies(nil, store, "")
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/leaderboard", nil))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})
}

func TestLeaderboardCursorValidation(t *testing.T) {
	for _, raw := range []string{
		"eyJzY29yZSI6LTEsInBsYXllcklkIjoiMDAwMDAwMDAtMDAwMC0wMDAwLTAwMDAtMDAwMDAwMDAwMDAxIn0",
		"eyJzY29yZSI6MSwicGxheWVySWQiOiJub3QtYS11dWlkIn0",
		"eyJzY29yZSI6MSwicGxheWVySWQiOiIwMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDEiLCJleHRyYSI6dHJ1ZX0",
	} {
		if _, err := decodeLeaderboardCursor(raw); err == nil {
			t.Fatalf("decodeLeaderboardCursor(%q) succeeded; want error", raw)
		}
	}
}

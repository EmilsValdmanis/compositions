package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type leaderboardTestStore struct {
	userStore
	page     database.LeaderboardPage
	err      error
	cursor   *database.LeaderboardCursor
	limit    int
	viewerID string
	metric   database.LeaderboardMetric
	scope    database.LeaderboardScope
}

func newAuthenticatedLeaderboardServer(store userStore, userID string) *wsServer {
	authStore := &stubAuthStore{sessionUser: database.SessionUserRecord{
		ID: userID, Name: "Leaderboard Viewer", Email: "viewer@example.com",
	}}
	return newWSServerWithDependencies(&authHandler{store: authStore, now: time.Now}, store, "")
}

func authenticatedLeaderboardRequest(method, path string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	request.AddCookie(&http.Cookie{Name: authCookieName, Value: "leaderboard-session"})
	return request
}

func (s *leaderboardTestStore) GetLeaderboard(_ context.Context, cursor *database.LeaderboardCursor, limit int, viewerID string, metric database.LeaderboardMetric, scope database.LeaderboardScope) (database.LeaderboardPage, error) {
	s.cursor = cursor
	s.limit = limit
	s.viewerID = viewerID
	s.metric = metric
	s.scope = scope
	return s.page, s.err
}

func TestHandleLeaderboard(t *testing.T) {
	firstID := "00000000-0000-0000-0000-000000000001"
	secondID := "00000000-0000-0000-0000-000000000002"
	store := &leaderboardTestStore{
		userStore: noopUserStore{},
		page: database.LeaderboardPage{
			Players: []database.LeaderboardPlayerRecord{{
				Rank: 1, Score: 12, PlayerID: firstID, Name: "Avery", Wins: 12, GamesPlayed: 20,
				RoundsWon: 48, PointsInflicted: 920, TotalPlaytimeSeconds: 18_400,
			}},
			NextCursor: &database.LeaderboardCursor{Score: 12, PlayerID: firstID},
			Placement: &database.LeaderboardPlayerRecord{
				Rank: 8, PlayerID: secondID, Name: "Casey", Wins: 4, GamesPlayed: 9,
			},
		},
	}
	server := newAuthenticatedLeaderboardServer(store, secondID)
	response := httptest.NewRecorder()
	request := authenticatedLeaderboardRequest(http.MethodGet, "/api/leaderboard?playerId="+firstID)
	server.routes().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
	if store.limit != leaderboardPageSize || store.viewerID != secondID || store.cursor != nil || store.metric != database.LeaderboardMetricWins || store.scope != database.LeaderboardScopeFriends {
		t.Fatalf("request = limit:%d viewer:%q cursor:%+v metric:%q scope:%q", store.limit, store.viewerID, store.cursor, store.metric, store.scope)
	}
	var payload leaderboardResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Players) != 1 || payload.Players[0].Wins != 12 || payload.Placement == nil || payload.Placement.Rank != 8 || payload.NextCursor == nil {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Scope != database.LeaderboardScopeFriends {
		t.Fatalf("payload scope = %q; want %q", payload.Scope, database.LeaderboardScopeFriends)
	}

	nextResponse := httptest.NewRecorder()
	nextURL := "/api/leaderboard?limit=25&metric=wins&cursor=" + url.QueryEscape(*payload.NextCursor)
	server.routes().ServeHTTP(nextResponse, authenticatedLeaderboardRequest(http.MethodGet, nextURL))
	if nextResponse.Code != http.StatusOK || store.limit != 25 || store.cursor == nil || store.cursor.Score != 12 || store.cursor.PlayerID != firstID {
		t.Fatalf("next page status:%d limit:%d cursor:%+v", nextResponse.Code, store.limit, store.cursor)
	}
}

func TestHandleLeaderboardRejectsInvalidRequests(t *testing.T) {
	viewerID := "00000000-0000-0000-0000-000000000001"
	server := newAuthenticatedLeaderboardServer(&leaderboardTestStore{userStore: noopUserStore{}}, viewerID)
	for _, path := range []string{
		"/api/leaderboard?limit=0",
		"/api/leaderboard?limit=51",
		"/api/leaderboard?cursor=not-a-cursor",
		"/api/leaderboard?metric=unknown",
		"/api/leaderboard?scope=unknown",
	} {
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, authenticatedLeaderboardRequest(http.MethodGet, path))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path %q status = %d; want %d", path, response.Code, http.StatusBadRequest)
		}
	}

	response := httptest.NewRecorder()
	server.routes().ServeHTTP(response, authenticatedLeaderboardRequest(http.MethodPost, "/api/leaderboard"))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("post status = %d allow = %q", response.Code, response.Header().Get("Allow"))
	}
}

func TestHandleLeaderboardUnavailableAndFailure(t *testing.T) {
	viewerID := "00000000-0000-0000-0000-000000000001"

	t.Run("requires authentication", func(t *testing.T) {
		server := newAuthenticatedLeaderboardServer(&leaderboardTestStore{userStore: noopUserStore{}}, viewerID)
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/leaderboard", nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		server := newAuthenticatedLeaderboardServer(noopUserStore{}, viewerID)
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, authenticatedLeaderboardRequest(http.MethodGet, "/api/leaderboard?scope=global"))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("store failure", func(t *testing.T) {
		store := &leaderboardTestStore{userStore: noopUserStore{}, err: errors.New("database failed")}
		server := newAuthenticatedLeaderboardServer(store, viewerID)
		response := httptest.NewRecorder()
		server.routes().ServeHTTP(response, authenticatedLeaderboardRequest(http.MethodGet, "/api/leaderboard?scope=global"))
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
		if _, err := decodeLeaderboardCursor(raw, database.LeaderboardMetricWins, database.LeaderboardScopeFriends); err == nil {
			t.Fatalf("decodeLeaderboardCursor(%q) succeeded; want error", raw)
		}
	}

	cursor, err := encodeLeaderboardCursor(database.LeaderboardCursor{
		Score: 9, PlayerID: "00000000-0000-0000-0000-000000000001",
	}, database.LeaderboardMetricGames, database.LeaderboardScopeFriends)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	if _, err := decodeLeaderboardCursor(cursor, database.LeaderboardMetricWins, database.LeaderboardScopeFriends); err == nil {
		t.Fatal("cursor for a different metric succeeded; want error")
	}
	if _, err := decodeLeaderboardCursor(cursor, database.LeaderboardMetricGames, database.LeaderboardScopeGlobal); err == nil {
		t.Fatal("cursor for a different scope succeeded; want error")
	}
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type stubAdminAnalyticsStore struct {
	noopUserStore
	record    database.AdminAnalyticsRecord
	err       error
	dateRange database.AdminAnalyticsRange
}

func (s *stubAdminAnalyticsStore) GetAdminAnalytics(_ context.Context, dateRange database.AdminAnalyticsRange) (database.AdminAnalyticsRecord, error) {
	s.dateRange = dateRange
	return s.record, s.err
}

func TestHandleAdminAnalytics(t *testing.T) {
	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	newServer := func(admin bool, store userStore) *wsServer {
		auth := &authHandler{
			store: &stubAuthStore{sessionUser: database.SessionUserRecord{
				ID: "user-1", Email: "player@example.com", IsAdmin: admin, ExpiresAt: now.Add(time.Hour),
			}},
			now: func() time.Time { return now },
		}
		return newWSServerWithDependencies(auth, store, "")
	}
	newRequest := func(path string) *http.Request {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		return request
	}

	t.Run("returns analytics for an inclusive Riga date range", func(t *testing.T) {
		store := &stubAdminAnalyticsStore{record: database.AdminAnalyticsRecord{
			Current:  database.AdminAnalyticsTotalsRecord{Games: 12, ActivePlayers: 7},
			Previous: database.AdminAnalyticsTotalsRecord{Games: 8, ActivePlayers: 5},
			Points:   []database.AdminAnalyticsPointRecord{{Date: "2026-07-01", Games: 2}},
		}}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminAnalytics(response, newRequest("/api/admin/analytics?from=2026-07-01&to=2026-07-30"))

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
		}
		if got := store.dateRange.To.Sub(store.dateRange.From); got != 720*time.Hour {
			t.Fatalf("current range duration = %s; want 720h", got)
		}
		if !store.dateRange.PreviousTo.Equal(store.dateRange.From) {
			t.Fatal("previous range does not end at the current range boundary")
		}
		var payload adminAnalyticsResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.From != "2026-07-01" || payload.To != "2026-07-30" || payload.Current.Games != 12 || len(payload.Points) != 1 {
			t.Fatalf("payload = %#v", payload)
		}
	})

	t.Run("rejects invalid ranges before querying the store", func(t *testing.T) {
		for _, query := range []string{
			"",
			"?from=invalid&to=2026-07-30",
			"?from=2026-07-30&to=2026-07-01",
			"?from=2025-01-01&to=2026-07-30",
		} {
			store := &stubAdminAnalyticsStore{}
			server := newServer(true, store)
			response := httptest.NewRecorder()
			server.handleAdminAnalytics(response, newRequest("/api/admin/analytics"+query))
			if response.Code != http.StatusBadRequest {
				t.Fatalf("query %q status = %d; want %d", query, response.Code, http.StatusBadRequest)
			}
			if !store.dateRange.From.IsZero() {
				t.Fatalf("query %q reached analytics store", query)
			}
		}
	})

	t.Run("rejects non-admins", func(t *testing.T) {
		store := &stubAdminAnalyticsStore{}
		server := newServer(false, store)
		response := httptest.NewRecorder()
		server.handleAdminAnalytics(response, newRequest("/api/admin/analytics?from=2026-07-01&to=2026-07-30"))
		if response.Code != http.StatusForbidden || !store.dateRange.From.IsZero() {
			t.Fatalf("status = %d range = %#v", response.Code, store.dateRange)
		}
	})

	t.Run("rejects unsupported methods", func(t *testing.T) {
		server := newServer(true, &stubAdminAnalyticsStore{})
		request := newRequest("/api/admin/analytics?from=2026-07-01&to=2026-07-30")
		request.Method = http.MethodPost
		response := httptest.NewRecorder()
		server.handleAdminAnalytics(response, request)
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
			t.Fatalf("status = %d allow = %q", response.Code, response.Header().Get("Allow"))
		}
	})

	t.Run("maps storage errors", func(t *testing.T) {
		server := newServer(true, &stubAdminAnalyticsStore{err: errors.New("database unavailable")})
		response := httptest.NewRecorder()
		server.handleAdminAnalytics(response, newRequest("/api/admin/analytics?from=2026-07-01&to=2026-07-30"))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})

	t.Run("reports unavailable stores", func(t *testing.T) {
		server := newServer(true, &noopUserStore{})
		response := httptest.NewRecorder()
		server.handleAdminAnalytics(response, newRequest("/api/admin/analytics?from=2026-07-01&to=2026-07-30"))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusServiceUnavailable)
		}
	})
}

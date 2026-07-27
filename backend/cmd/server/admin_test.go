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
	"github.com/google/uuid"
)

type stubAdminBugReportStore struct {
	noopUserStore
	reports      []database.GameBugReportRecord
	report       database.GameBugReportRecord
	total        int64
	listErr      error
	countErr     error
	getErr       error
	limit        int
	offset       int
	loadedReport string
}

func (s *stubAdminBugReportStore) ListGameBugReportsPage(_ context.Context, limit, offset int) ([]database.GameBugReportRecord, error) {
	s.limit = limit
	s.offset = offset
	return s.reports, s.listErr
}

func (s *stubAdminBugReportStore) CountGameBugReports(context.Context) (int64, error) {
	return s.total, s.countErr
}

func (s *stubAdminBugReportStore) GetGameBugReport(_ context.Context, reportID string) (database.GameBugReportRecord, error) {
	s.loadedReport = reportID
	return s.report, s.getErr
}

func TestHandleAdminBugReports(t *testing.T) {
	now := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	reportID := uuid.NewString()
	report := database.GameBugReportRecord{
		ID: reportID, RoomCode: "ABC123", ReporterPlayerID: "player-1",
		ReporterUserID: uuid.NewString(), Description: "The turn stopped responding",
		GameState: json.RawMessage(`{"version":1}`), Round: 2, Turn: 9,
		RequestedAbort: true, CreatedAt: now,
	}

	newServer := func(admin bool, store *stubAdminBugReportStore) *wsServer {
		authStore := &stubAuthStore{sessionUser: database.SessionUserRecord{
			ID: "user-1", Email: "player@example.com", IsAdmin: admin, ExpiresAt: now.Add(time.Hour),
		}}
		auth := &authHandler{
			store: authStore, now: func() time.Time { return now },
		}
		return newWSServerWithDependencies(auth, store, "")
	}
	newRequest := func(path string) *http.Request {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		return request
	}

	t.Run("rejects an unauthenticated request without caching the response", func(t *testing.T) {
		server := newServer(false, &stubAdminBugReportStore{})
		response := httptest.NewRecorder()
		server.handleAdminBugReports(
			response,
			httptest.NewRequest(http.MethodGet, "/api/admin/bug-reports", nil),
		)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusUnauthorized)
		}
		if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
			t.Fatalf("Cache-Control = %q; want no-store", cacheControl)
		}
	})

	t.Run("rejects an authenticated non-admin", func(t *testing.T) {
		store := &stubAdminBugReportStore{}
		server := newServer(false, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports"))
		if response.Code != http.StatusForbidden {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusForbidden)
		}
		if store.limit != 0 {
			t.Fatal("non-admin request reached the bug report store")
		}
	})

	t.Run("rejects unsupported methods", func(t *testing.T) {
		server := newServer(true, &stubAdminBugReportStore{})
		request := httptest.NewRequest(http.MethodPost, "/api/admin/bug-reports", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusMethodNotAllowed)
		}
		if allow := response.Header().Get("Allow"); allow != http.MethodGet {
			t.Fatalf("Allow = %q; want %q", allow, http.MethodGet)
		}
	})

	t.Run("rejects invalid pagination before querying the store", func(t *testing.T) {
		for _, query := range []string{
			"?page=0",
			"?page=invalid",
			"?page=1000001",
			"?pageSize=0",
			"?pageSize=101",
		} {
			store := &stubAdminBugReportStore{}
			server := newServer(true, store)
			response := httptest.NewRecorder()
			server.handleAdminBugReports(
				response,
				newRequest("/api/admin/bug-reports"+query),
			)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("query %q status = %d; want %d", query, response.Code, http.StatusBadRequest)
			}
			if store.limit != 0 {
				t.Fatalf("query %q reached the bug report store", query)
			}
		}
	})

	t.Run("returns a paginated summary without game state", func(t *testing.T) {
		store := &stubAdminBugReportStore{reports: []database.GameBugReportRecord{report}, total: 42}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports?page=2&pageSize=10"))

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
		}
		if store.limit != 10 || store.offset != 10 {
			t.Fatalf("pagination = limit:%d offset:%d; want 10/10", store.limit, store.offset)
		}
		var payload adminBugReportPageResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if payload.TotalPages != 5 || len(payload.Reports) != 1 || payload.Reports[0].ID != reportID {
			t.Fatalf("payload = %#v", payload)
		}
		if string(response.Body.Bytes()) == "" || json.Valid(report.GameState) && containsJSONField(response.Body.Bytes(), "gameState") {
			t.Fatal("list response unexpectedly exposed game state")
		}
	})

	t.Run("returns the full report detail", func(t *testing.T) {
		store := &stubAdminBugReportStore{report: report}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports/"+reportID))

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
		}
		var payload adminBugReportDetailResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if store.loadedReport != reportID || payload.ID != reportID || string(payload.GameState) != `{"version":1}` {
			t.Fatalf("payload = %#v, loaded report = %q", payload, store.loadedReport)
		}
	})

	t.Run("maps a missing report to not found", func(t *testing.T) {
		store := &stubAdminBugReportStore{getErr: database.ErrGameBugReportNotFound}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports/"+reportID))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("rejects a malformed report id before querying the store", func(t *testing.T) {
		store := &stubAdminBugReportStore{}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports/not-a-uuid"))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNotFound)
		}
		if store.loadedReport != "" {
			t.Fatal("malformed report id reached the bug report store")
		}
	})

	t.Run("handles list storage errors", func(t *testing.T) {
		store := &stubAdminBugReportStore{listErr: errors.New("database unavailable")}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports"))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handles count storage errors", func(t *testing.T) {
		store := &stubAdminBugReportStore{countErr: errors.New("database unavailable")}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports"))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})

	t.Run("handles detail storage errors", func(t *testing.T) {
		store := &stubAdminBugReportStore{getErr: errors.New("database unavailable")}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports/"+reportID))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})
}

func containsJSONField(data []byte, field string) bool {
	var payload map[string]json.RawMessage
	if json.Unmarshal(data, &payload) != nil {
		return false
	}
	_, ok := payload[field]
	if ok {
		return true
	}
	var nested struct {
		Reports []map[string]json.RawMessage `json:"reports"`
	}
	if json.Unmarshal(data, &nested) != nil || len(nested.Reports) == 0 {
		return false
	}
	_, ok = nested.Reports[0][field]
	return ok
}

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
	completeErr  error
	limit        int
	offset       int
	loadedReport string
	completedID  string
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

func (s *stubAdminBugReportStore) CompleteGameBugReport(_ context.Context, reportID string) error {
	s.completedID = reportID
	return s.completeErr
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
			"?page=10002&pageSize=1",
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
		if response.Body.String() == "" || json.Valid(report.GameState) && containsJSONField(response.Body.Bytes(), "gameState") {
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

	t.Run("completes a report", func(t *testing.T) {
		store := &stubAdminBugReportStore{}
		server := newServer(true, store)
		request := httptest.NewRequest(http.MethodPost, "/api/admin/bug-reports/"+reportID+"/complete", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, request)

		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNoContent)
		}
		if store.completedID != reportID {
			t.Fatalf("completed report = %q; want %q", store.completedID, reportID)
		}
	})

	t.Run("requires post when completing a report", func(t *testing.T) {
		store := &stubAdminBugReportStore{}
		server := newServer(true, store)
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest("/api/admin/bug-reports/"+reportID+"/complete"))

		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusMethodNotAllowed)
		}
		if allow := response.Header().Get("Allow"); allow != http.MethodPost {
			t.Fatalf("Allow = %q; want %q", allow, http.MethodPost)
		}
		if store.completedID != "" {
			t.Fatal("unsupported method reached the bug report store")
		}
	})

	t.Run("maps a missing report on completion to not found", func(t *testing.T) {
		store := &stubAdminBugReportStore{completeErr: database.ErrGameBugReportNotFound}
		server := newServer(true, store)
		request := httptest.NewRequest(http.MethodPost, "/api/admin/bug-reports/"+reportID+"/complete", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, request)

		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNotFound)
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

	t.Run("handles completion storage errors", func(t *testing.T) {
		store := &stubAdminBugReportStore{completeErr: errors.New("database unavailable")}
		server := newServer(true, store)
		request := httptest.NewRequest(http.MethodPost, "/api/admin/bug-reports/"+reportID+"/complete", nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session-token"})
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, request)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})
}

func TestHandleAdminBugReportsRemainingBranches(t *testing.T) {
	now := time.Now()
	reportID := uuid.NewString()
	newRequest := func(method, path string) *http.Request {
		request := httptest.NewRequest(method, path, nil)
		request.AddCookie(&http.Cookie{Name: authCookieName, Value: "session"})
		return request
	}
	newAdminServer := func(store userStore) *wsServer {
		auth := &authHandler{store: &stubAuthStore{sessionUser: database.SessionUserRecord{ID: "admin", IsAdmin: true, ExpiresAt: now.Add(time.Hour)}}, now: func() time.Time { return now }}
		return newWSServerWithDependencies(auth, store, "")
	}

	t.Run("preflight", func(t *testing.T) {
		request := newRequest(http.MethodOptions, "/api/admin/bug-reports")
		request.Header.Set("Origin", "http://localhost:3000")
		response := httptest.NewRecorder()
		newAdminServer(&stubAdminBugReportStore{}).handleAdminBugReports(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNoContent)
		}
	})

	t.Run("auth not configured", func(t *testing.T) {
		response := httptest.NewRecorder()
		newWSServer().handleAdminBugReports(response, newRequest(http.MethodGet, "/api/admin/bug-reports"))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})

	t.Run("auth storage failure", func(t *testing.T) {
		server := newWSServerWithDependencies(&authHandler{store: &stubAuthStore{sessionErr: errors.New("auth failed")}, now: time.Now}, &stubAdminBugReportStore{}, "")
		response := httptest.NewRecorder()
		server.handleAdminBugReports(response, newRequest(http.MethodGet, "/api/admin/bug-reports"))
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusInternalServerError)
		}
	})

	t.Run("store unavailable", func(t *testing.T) {
		response := httptest.NewRecorder()
		newAdminServer(noopUserStore{}).handleAdminBugReports(response, newRequest(http.MethodGet, "/api/admin/bug-reports"))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusServiceUnavailable)
		}
	})

	for name, testCase := range map[string]struct {
		method string
		path   string
		status int
	}{
		"nested path":   {http.MethodGet, "/api/admin/bug-reports/" + reportID + "/unexpected", http.StatusNotFound},
		"detail method": {http.MethodPost, "/api/admin/bug-reports/" + reportID, http.StatusMethodNotAllowed},
	} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			newAdminServer(&stubAdminBugReportStore{}).handleAdminBugReports(response, newRequest(testCase.method, testCase.path))
			if response.Code != testCase.status {
				t.Fatalf("status = %d; want %d", response.Code, testCase.status)
			}
		})
	}

	t.Run("malformed completion id", func(t *testing.T) {
		response := httptest.NewRecorder()
		newAdminServer(&stubAdminBugReportStore{}).handleAdminBugReports(response, newRequest(http.MethodPost, "/api/admin/bug-reports/bad/complete"))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d; want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("response write failures", func(t *testing.T) {
		store := &stubAdminBugReportStore{report: database.GameBugReportRecord{ID: reportID}}
		server := newAdminServer(store)
		server.handleAdminBugReportList(failingResponseWriter{}, httptest.NewRequest(http.MethodGet, "/api/admin/bug-reports", nil), store)
		server.handleAdminBugReportDetail(failingResponseWriter{}, httptest.NewRequest(http.MethodGet, "/api/admin/bug-reports/"+reportID, nil), store, reportID)
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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/google/uuid"
)

const (
	defaultAdminBugReportPageSize = 20
	maxAdminBugReportPageSize     = 100
)

type adminBugReportStore interface {
	ListGameBugReportsPage(ctx context.Context, limit, offset int) ([]database.GameBugReportRecord, error)
	CountGameBugReports(ctx context.Context) (int64, error)
	GetGameBugReport(ctx context.Context, reportID string) (database.GameBugReportRecord, error)
	CompleteGameBugReport(ctx context.Context, reportID string) error
}

type adminBugReportSummaryResponse struct {
	ID               string    `json:"id"`
	RoomCode         string    `json:"roomCode"`
	ReporterPlayerID string    `json:"reporterPlayerId"`
	Description      string    `json:"description"`
	Round            int       `json:"round"`
	Turn             int       `json:"turn"`
	RequestedAbort   bool      `json:"requestedAbort"`
	CreatedAt        time.Time `json:"createdAt"`
}

type adminBugReportPageResponse struct {
	Reports    []adminBugReportSummaryResponse `json:"reports"`
	Page       int                             `json:"page"`
	PageSize   int                             `json:"pageSize"`
	TotalItems int64                           `json:"totalItems"`
	TotalPages int                             `json:"totalPages"`
}

type adminBugReportDetailResponse struct {
	ID               string          `json:"id"`
	RoomCode         string          `json:"roomCode"`
	ReporterPlayerID string          `json:"reporterPlayerId"`
	ReporterUserID   string          `json:"reporterUserId,omitempty"`
	Description      string          `json:"description"`
	GameState        json.RawMessage `json:"gameState"`
	Round            int             `json:"round"`
	Turn             int             `json:"turn"`
	RequestedAbort   bool            `json:"requestedAbort"`
	CreatedAt        time.Time       `json:"createdAt"`
}

func (s *wsServer) handleAdminBugReports(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if handleCORSPreflight(w, r) {
		return
	}
	setNoStore(w)
	if !s.requireAdmin(w, r) {
		return
	}

	store, ok := s.userStore.(adminBugReportStore)
	if !ok {
		writeHTTPError(w, http.StatusServiceUnavailable, "bug_reports_unavailable", "bug reports are unavailable")
		return
	}

	reportPath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/bug-reports"), "/")
	if reportPath == "" {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.handleAdminBugReportList(w, r, store)
		return
	}
	pathParts := strings.Split(reportPath, "/")
	if len(pathParts) == 2 && pathParts[1] == "complete" {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		s.handleAdminBugReportComplete(w, r, store, pathParts[0])
		return
	}
	if len(pathParts) != 1 {
		writeHTTPError(w, http.StatusNotFound, "not_found", "bug report not found")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	s.handleAdminBugReportDetail(w, r, store, reportPath)
}

func (s *wsServer) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s == nil || s.auth == nil {
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "auth is not configured")
		return false
	}
	session, err := s.auth.sessionFromRequest(r)
	if errors.Is(err, errAuthenticationRequired) {
		writeHTTPError(w, http.StatusUnauthorized, "auth_required", "authentication required")
		return false
	}
	if err != nil {
		slog.Error("resolve admin session failed", "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to authorize request")
		return false
	}
	if !session.user.IsAdmin {
		writeHTTPError(w, http.StatusForbidden, "admin_required", "administrator access required")
		return false
	}
	return true
}

func (s *wsServer) handleAdminBugReportList(w http.ResponseWriter, r *http.Request, store adminBugReportStore) {
	page, ok := positiveQueryInt(r, "page", 1)
	if !ok {
		writeHTTPError(w, http.StatusBadRequest, "invalid_pagination", "page must be a positive integer")
		return
	}
	pageSize, ok := positiveQueryInt(r, "pageSize", defaultAdminBugReportPageSize)
	if !ok || pageSize > maxAdminBugReportPageSize {
		writeHTTPError(w, http.StatusBadRequest, "invalid_pagination", "page size must be between 1 and 100")
		return
	}
	if page > maxOffsetPagination/pageSize+1 {
		writeHTTPError(w, http.StatusBadRequest, "invalid_pagination", "page offset is too large")
		return
	}

	reports, err := store.ListGameBugReportsPage(r.Context(), pageSize, (page-1)*pageSize)
	if err != nil {
		slog.Error("list admin bug reports failed", "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load bug reports")
		return
	}
	totalItems, err := store.CountGameBugReports(r.Context())
	if err != nil {
		slog.Error("count admin bug reports failed", "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load bug reports")
		return
	}

	items := make([]adminBugReportSummaryResponse, 0, len(reports))
	for _, report := range reports {
		items = append(items, adminBugReportSummaryResponse{
			ID: report.ID, RoomCode: report.RoomCode, ReporterPlayerID: report.ReporterPlayerID,
			Description: report.Description, Round: report.Round, Turn: report.Turn,
			RequestedAbort: report.RequestedAbort, CreatedAt: report.CreatedAt,
		})
	}
	totalPages := 0
	if totalItems > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(adminBugReportPageResponse{
		Reports: items, Page: page, PageSize: pageSize, TotalItems: totalItems, TotalPages: totalPages,
	}); err != nil {
		slog.Error("write admin bug report list failed", "error", err)
	}
}

func (s *wsServer) handleAdminBugReportDetail(w http.ResponseWriter, r *http.Request, store adminBugReportStore, reportID string) {
	if _, err := uuid.Parse(reportID); err != nil {
		writeHTTPError(w, http.StatusNotFound, "not_found", "bug report not found")
		return
	}
	report, err := store.GetGameBugReport(r.Context(), reportID)
	if errors.Is(err, database.ErrGameBugReportNotFound) {
		writeHTTPError(w, http.StatusNotFound, "not_found", "bug report not found")
		return
	}
	if err != nil {
		slog.Error("load admin bug report failed", "reportID", reportID, "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load bug report")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(adminBugReportDetailResponse{
		ID: report.ID, RoomCode: report.RoomCode, ReporterPlayerID: report.ReporterPlayerID,
		ReporterUserID: report.ReporterUserID, Description: report.Description,
		GameState: report.GameState, Round: report.Round, Turn: report.Turn,
		RequestedAbort: report.RequestedAbort, CreatedAt: report.CreatedAt,
	}); err != nil {
		slog.Error("write admin bug report detail failed", "error", err)
	}
}

func (s *wsServer) handleAdminBugReportComplete(w http.ResponseWriter, r *http.Request, store adminBugReportStore, reportID string) {
	if _, err := uuid.Parse(reportID); err != nil {
		writeHTTPError(w, http.StatusNotFound, "not_found", "bug report not found")
		return
	}
	if err := store.CompleteGameBugReport(r.Context(), reportID); errors.Is(err, database.ErrGameBugReportNotFound) {
		writeHTTPError(w, http.StatusNotFound, "not_found", "bug report not found")
		return
	} else if err != nil {
		slog.Error("complete admin bug report failed", "reportID", reportID, "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to complete bug report")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

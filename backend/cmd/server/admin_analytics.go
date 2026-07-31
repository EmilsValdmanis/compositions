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
)

const (
	adminAnalyticsDateLayout = "2006-01-02"
	adminAnalyticsTimezone   = "Europe/Riga"
	maxAdminAnalyticsDays    = 366
)

type adminAnalyticsStore interface {
	GetAdminAnalytics(context.Context, database.AdminAnalyticsRange) (database.AdminAnalyticsRecord, error)
}

type adminAnalyticsResponse struct {
	From     string                               `json:"from"`
	To       string                               `json:"to"`
	Current  database.AdminAnalyticsTotalsRecord  `json:"current"`
	Previous database.AdminAnalyticsTotalsRecord  `json:"previous"`
	Points   []database.AdminAnalyticsPointRecord `json:"points"`
}

func (s *wsServer) handleAdminAnalytics(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if handleCORSPreflight(w, r) {
		return
	}
	setNoStore(w)
	if !s.requireAdmin(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	store, ok := s.userStore.(adminAnalyticsStore)
	if !ok {
		writeHTTPError(w, http.StatusServiceUnavailable, "analytics_unavailable", "analytics are unavailable")
		return
	}

	dateRange, from, to, err := parseAdminAnalyticsRange(r)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, "invalid_date_range", err.Error())
		return
	}
	record, err := store.GetAdminAnalytics(r.Context(), dateRange)
	if err != nil {
		slog.Error("load admin analytics failed", "from", from, "to", to, "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load analytics")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(adminAnalyticsResponse{
		From: from, To: to, Current: record.Current, Previous: record.Previous, Points: record.Points,
	}); err != nil {
		slog.Error("write admin analytics failed", "error", err)
	}
}

func parseAdminAnalyticsRange(r *http.Request) (database.AdminAnalyticsRange, string, string, error) {
	fromValue := strings.TrimSpace(r.URL.Query().Get("from"))
	toValue := strings.TrimSpace(r.URL.Query().Get("to"))
	fromDate, err := time.Parse(adminAnalyticsDateLayout, fromValue)
	if err != nil {
		return database.AdminAnalyticsRange{}, "", "", errors.New("from must use YYYY-MM-DD")
	}
	toDate, err := time.Parse(adminAnalyticsDateLayout, toValue)
	if err != nil {
		return database.AdminAnalyticsRange{}, "", "", errors.New("to must use YYYY-MM-DD")
	}
	endDate := toDate.AddDate(0, 0, 1)
	days := int(endDate.Sub(fromDate).Hours() / 24)
	if days < 1 || days > maxAdminAnalyticsDays {
		return database.AdminAnalyticsRange{}, "", "", errors.New("date range must be between 1 and 366 days")
	}
	location, err := time.LoadLocation(adminAnalyticsTimezone)
	if err != nil {
		return database.AdminAnalyticsRange{}, "", "", errors.New("analytics timezone is unavailable")
	}
	localDate := func(value time.Time) time.Time {
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
	}
	from := localDate(fromDate)
	to := localDate(endDate)
	previousFrom := from.AddDate(0, 0, -days)
	return database.AdminAnalyticsRange{
		From: from.UTC(), To: to.UTC(), PreviousFrom: previousFrom.UTC(), PreviousTo: from.UTC(),
	}, fromValue, toValue, nil
}

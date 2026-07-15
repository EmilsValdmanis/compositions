package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/google/uuid"
)

type playerProfileStore interface {
	GetPlayerProfile(ctx context.Context, userID string) (database.PlayerProfileRecord, error)
}

type playerGameHistoryStore interface {
	GetPlayerGameHistory(ctx context.Context, userID string, limit, offset int) (database.PlayerGameHistoryPage, error)
}

const (
	defaultGameHistoryPageSize = 10
	maxGameHistoryPageSize     = 50
)

type playerGameHistoryItemResponse struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	CompletedAt     time.Time `json:"completedAt"`
	Placement       int       `json:"placement"`
	PlayerCount     int       `json:"playerCount"`
	Won             bool      `json:"won"`
	Forfeited       bool      `json:"forfeited"`
	TotalPoints     int       `json:"totalPoints"`
	RoundsPlayed    int       `json:"roundsPlayed"`
	RoundsWon       int       `json:"roundsWon"`
	PlaytimeSeconds int64     `json:"playtimeSeconds"`
}

type playerGameHistoryResponse struct {
	Games      []playerGameHistoryItemResponse `json:"games"`
	Page       int                             `json:"page"`
	PageSize   int                             `json:"pageSize"`
	TotalItems int64                           `json:"totalItems"`
	TotalPages int                             `json:"totalPages"`
}

type playerProfileResponse struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	ImageURL              string `json:"imageUrl"`
	GamesPlayed           int64  `json:"gamesPlayed"`
	GamesWon              int64  `json:"gamesWon"`
	TotalPlacement        int64  `json:"totalPlacement"`
	TotalPlaytimeSeconds  int64  `json:"totalPlaytimeSeconds"`
	RoundsPlayed          int64  `json:"roundsPlayed"`
	RoundsWon             int64  `json:"roundsWon"`
	CompositionsCreated   int64  `json:"compositionsCreated"`
	SetsCreated           int64  `json:"setsCreated"`
	RunsCreated           int64  `json:"runsCreated"`
	PointsInflicted       int64  `json:"pointsInflicted"`
	PenaltyPoints         int64  `json:"penaltyPoints"`
	CurrentGameWinStreak  int    `json:"currentGameWinStreak"`
	LongestGameWinStreak  int    `json:"longestGameWinStreak"`
	CurrentRoundWinStreak int    `json:"currentRoundWinStreak"`
	LongestRoundWinStreak int    `json:"longestRoundWinStreak"`
}

func (s *wsServer) handlePlayerProfile(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if handleCORSPreflight(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	profilePath := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/players/"), "/")
	parts := strings.Split(profilePath, "/")
	if len(parts) == 0 || len(parts) > 2 || (len(parts) == 2 && parts[1] != "games") {
		writeHTTPError(w, http.StatusNotFound, "not_found", "player profile not found")
		return
	}
	userID := strings.TrimSpace(parts[0])
	if _, err := uuid.Parse(userID); err != nil {
		writeHTTPError(w, http.StatusNotFound, "not_found", "player profile not found")
		return
	}
	if len(parts) == 2 {
		s.handlePlayerGameHistory(w, r, userID)
		return
	}
	store, ok := s.userStore.(playerProfileStore)
	if !ok {
		writeHTTPError(w, http.StatusServiceUnavailable, "player_profiles_unavailable", "player profiles are unavailable")
		return
	}
	profile, err := store.GetPlayerProfile(r.Context(), userID)
	if errors.Is(err, database.ErrPlayerProfileNotFound) {
		writeHTTPError(w, http.StatusNotFound, "not_found", "player profile not found")
		return
	}
	if err != nil {
		slog.Error("load player profile failed", "userID", userID, "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load player profile")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	response := playerProfileResponse{
		ID: profile.ID, Name: profile.Name, ImageURL: profile.ImageURL,
		GamesPlayed: profile.GamesPlayed, GamesWon: profile.GamesWon, TotalPlacement: profile.TotalPlacement,
		TotalPlaytimeSeconds: profile.TotalPlaytimeSeconds,
		RoundsPlayed:         profile.RoundsPlayed, RoundsWon: profile.RoundsWon,
		CompositionsCreated: profile.CompositionsCreated, SetsCreated: profile.SetsCreated, RunsCreated: profile.RunsCreated,
		PointsInflicted: profile.PointsInflicted, PenaltyPoints: profile.PenaltyPoints,
		CurrentGameWinStreak: profile.CurrentGameWinStreak, LongestGameWinStreak: profile.LongestGameWinStreak,
		CurrentRoundWinStreak: profile.CurrentRoundWinStreak, LongestRoundWinStreak: profile.LongestRoundWinStreak,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("write player profile response failed", "error", err)
	}
}

func (s *wsServer) handlePlayerGameHistory(w http.ResponseWriter, r *http.Request, userID string) {
	page, ok := positiveQueryInt(r, "page", 1)
	if !ok || page > 1_000_000 {
		writeHTTPError(w, http.StatusBadRequest, "invalid_pagination", "page must be a positive integer")
		return
	}
	pageSize, ok := positiveQueryInt(r, "pageSize", defaultGameHistoryPageSize)
	if !ok || pageSize > maxGameHistoryPageSize {
		writeHTTPError(w, http.StatusBadRequest, "invalid_pagination", "page size must be between 1 and 50")
		return
	}
	store, ok := s.userStore.(playerGameHistoryStore)
	if !ok {
		writeHTTPError(w, http.StatusServiceUnavailable, "player_profiles_unavailable", "player profiles are unavailable")
		return
	}

	history, err := store.GetPlayerGameHistory(r.Context(), userID, pageSize, (page-1)*pageSize)
	if errors.Is(err, database.ErrPlayerProfileNotFound) {
		writeHTTPError(w, http.StatusNotFound, "not_found", "player profile not found")
		return
	}
	if err != nil {
		slog.Error("load player game history failed", "userID", userID, "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load player game history")
		return
	}

	games := make([]playerGameHistoryItemResponse, 0, len(history.Games))
	for _, game := range history.Games {
		games = append(games, playerGameHistoryItemResponse{
			ID: game.GameID, Status: game.Status, CompletedAt: game.CompletedAt,
			Placement: game.Placement, PlayerCount: game.PlayerCount,
			Won: game.Won, Forfeited: game.Forfeited, TotalPoints: game.TotalPoints,
			RoundsPlayed: game.RoundsPlayed, RoundsWon: game.RoundsWon,
			PlaytimeSeconds: game.PlaytimeSeconds,
		})
	}
	totalPages := 0
	if history.Total > 0 {
		totalPages = int((history.Total + int64(pageSize) - 1) / int64(pageSize))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=30, stale-while-revalidate=120")
	response := playerGameHistoryResponse{
		Games: games, Page: page, PageSize: pageSize,
		TotalItems: history.Total, TotalPages: totalPages,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("write player game history response failed", "error", err)
	}
}

func positiveQueryInt(r *http.Request, name string, fallback int) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	return value, err == nil && value > 0
}

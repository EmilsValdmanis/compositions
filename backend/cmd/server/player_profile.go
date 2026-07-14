package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/google/uuid"
)

type playerProfileStore interface {
	GetPlayerProfile(ctx context.Context, userID string) (database.PlayerProfileRecord, error)
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
	userID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/players/"))
	if userID == "" || strings.Contains(userID, "/") {
		writeHTTPError(w, http.StatusNotFound, "not_found", "player profile not found")
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeHTTPError(w, http.StatusNotFound, "not_found", "player profile not found")
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

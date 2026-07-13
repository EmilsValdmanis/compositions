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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s == nil || s.auth == nil {
		http.Error(w, errAuthenticationRequired.Error(), http.StatusUnauthorized)
		return
	}
	if _, err := s.auth.sessionFromRequest(r); err != nil {
		if errors.Is(err, errAuthenticationRequired) {
			http.Error(w, errAuthenticationRequired.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "failed to load session", http.StatusInternalServerError)
		return
	}

	userID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/players/"))
	if userID == "" || strings.Contains(userID, "/") {
		http.NotFound(w, r)
		return
	}
	if _, err := uuid.Parse(userID); err != nil {
		http.NotFound(w, r)
		return
	}
	store, ok := s.userStore.(playerProfileStore)
	if !ok {
		http.Error(w, "player profiles are unavailable", http.StatusServiceUnavailable)
		return
	}
	profile, err := store.GetPlayerProfile(r.Context(), userID)
	if errors.Is(err, database.ErrPlayerProfileNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		slog.Error("load player profile failed", "userID", userID, "error", err)
		http.Error(w, "failed to load player profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-cache")
	response := playerProfileResponse{
		ID: profile.ID, Name: profile.Name, ImageURL: profile.ImageURL,
		GamesPlayed: profile.GamesPlayed, GamesWon: profile.GamesWon, TotalPlacement: profile.TotalPlacement,
		RoundsPlayed: profile.RoundsPlayed, RoundsWon: profile.RoundsWon,
		CompositionsCreated: profile.CompositionsCreated, SetsCreated: profile.SetsCreated, RunsCreated: profile.RunsCreated,
		PointsInflicted: profile.PointsInflicted, PenaltyPoints: profile.PenaltyPoints,
		CurrentGameWinStreak: profile.CurrentGameWinStreak, LongestGameWinStreak: profile.LongestGameWinStreak,
		CurrentRoundWinStreak: profile.CurrentRoundWinStreak, LongestRoundWinStreak: profile.LongestRoundWinStreak,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("write player profile response failed", "error", err)
	}
}

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/EmilsValdmanis/compositions/internal/database"
	"github.com/google/uuid"
)

const leaderboardPageSize = 50

type leaderboardStore interface {
	GetLeaderboard(ctx context.Context, cursor *database.LeaderboardCursor, limit int, viewerUserID string, metric database.LeaderboardMetric) (database.LeaderboardPage, error)
}

type leaderboardPlayerResponse struct {
	Rank                 int64  `json:"rank"`
	Score                int64  `json:"score"`
	PlayerID             string `json:"playerId"`
	Name                 string `json:"name"`
	ImageURL             string `json:"imageUrl"`
	Wins                 int64  `json:"wins"`
	GamesPlayed          int64  `json:"gamesPlayed"`
	RoundsWon            int64  `json:"roundsWon"`
	PointsInflicted      int64  `json:"pointsInflicted"`
	TotalPlaytimeSeconds int64  `json:"totalPlaytimeSeconds"`
}

type leaderboardResponse struct {
	Metric     database.LeaderboardMetric  `json:"metric"`
	Players    []leaderboardPlayerResponse `json:"players"`
	NextCursor *string                     `json:"nextCursor"`
	Placement  *leaderboardPlayerResponse  `json:"placement"`
}

type leaderboardCursorPayload struct {
	Metric   database.LeaderboardMetric `json:"metric"`
	Score    int64                      `json:"score"`
	PlayerID string                     `json:"playerId"`
}

func (s *wsServer) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w, r)
	if handleCORSPreflight(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeHTTPError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	limit, ok := positiveQueryInt(r, "limit", leaderboardPageSize)
	if !ok || limit > leaderboardPageSize {
		writeHTTPError(w, http.StatusBadRequest, "invalid_pagination", "limit must be between 1 and 50")
		return
	}
	metric, ok := database.ParseLeaderboardMetric(r.URL.Query().Get("metric"))
	if !ok {
		writeHTTPError(w, http.StatusBadRequest, "invalid_metric", "leaderboard metric is invalid")
		return
	}
	cursor, err := decodeLeaderboardCursor(r.URL.Query().Get("cursor"), metric)
	if err != nil {
		writeHTTPError(w, http.StatusBadRequest, "invalid_cursor", "leaderboard cursor is invalid")
		return
	}
	viewerID := strings.TrimSpace(r.URL.Query().Get("playerId"))
	if viewerID != "" {
		if _, err := uuid.Parse(viewerID); err != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid_player", "player id is invalid")
			return
		}
	}

	store, ok := s.userStore.(leaderboardStore)
	if !ok {
		writeHTTPError(w, http.StatusServiceUnavailable, "leaderboard_unavailable", "leaderboard is unavailable")
		return
	}
	page, err := store.GetLeaderboard(r.Context(), cursor, limit, viewerID, metric)
	if err != nil {
		slog.Error("load leaderboard failed", "error", err)
		writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load leaderboard")
		return
	}

	players := make([]leaderboardPlayerResponse, 0, len(page.Players))
	for _, player := range page.Players {
		players = append(players, leaderboardPlayerFromRecord(player))
	}
	var nextCursor *string
	if page.NextCursor != nil {
		encoded, err := encodeLeaderboardCursor(*page.NextCursor, metric)
		if err != nil {
			slog.Error("encode leaderboard cursor failed", "error", err)
			writeHTTPError(w, http.StatusInternalServerError, clientErrorInternal, "failed to load leaderboard")
			return
		}
		nextCursor = &encoded
	}
	var placement *leaderboardPlayerResponse
	if page.Placement != nil {
		value := leaderboardPlayerFromRecord(*page.Placement)
		placement = &value
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=30, stale-while-revalidate=120")
	if err := json.NewEncoder(w).Encode(leaderboardResponse{
		Metric: metric, Players: players, NextCursor: nextCursor, Placement: placement,
	}); err != nil {
		slog.Error("write leaderboard response failed", "error", err)
	}
}

func leaderboardPlayerFromRecord(player database.LeaderboardPlayerRecord) leaderboardPlayerResponse {
	return leaderboardPlayerResponse{
		Rank: player.Rank, Score: player.Score, PlayerID: player.PlayerID, Name: player.Name, ImageURL: player.ImageURL,
		Wins: player.Wins, GamesPlayed: player.GamesPlayed, RoundsWon: player.RoundsWon,
		PointsInflicted: player.PointsInflicted, TotalPlaytimeSeconds: player.TotalPlaytimeSeconds,
	}
}

func encodeLeaderboardCursor(cursor database.LeaderboardCursor, metric database.LeaderboardMetric) (string, error) {
	payload, err := json.Marshal(leaderboardCursorPayload{Metric: metric, Score: cursor.Score, PlayerID: cursor.PlayerID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeLeaderboardCursor(raw string, metric database.LeaderboardMetric) (*database.LeaderboardCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var cursor leaderboardCursorPayload
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return nil, err
	}
	if cursor.Metric != metric || cursor.Score < 0 || strings.TrimSpace(cursor.PlayerID) == "" {
		return nil, errors.New("invalid leaderboard cursor values")
	}
	if _, err := uuid.Parse(cursor.PlayerID); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid leaderboard cursor payload")
	}
	return &database.LeaderboardCursor{Score: cursor.Score, PlayerID: cursor.PlayerID}, nil
}

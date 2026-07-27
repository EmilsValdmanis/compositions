package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

type postgresUserStore struct {
	store *database.UserStore
}

var _ leaderboardStore = (*postgresUserStore)(nil)
var _ adminBugReportStore = (*postgresUserStore)(nil)

func newConfiguredUserStore() (userStore, error) {
	databaseURL, err := databaseURLFromEnv()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	store, err := newDatabaseUserStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &postgresUserStore{store: store}, nil
}

func (s *postgresUserStore) UpsertUser(ctx context.Context, user authenticatedUser) (authenticatedUser, error) {
	if s == nil || s.store == nil {
		return authenticatedUser{}, errors.New("user store is not configured")
	}
	storedUser, err := upsertStoredUser(ctx, s.store, database.UserRecord{
		ID: strings.TrimSpace(user.ID), Name: strings.TrimSpace(user.Name),
		Email: strings.ToLower(strings.TrimSpace(user.Email)), ImageURL: strings.TrimSpace(user.Image),
		Provider: strings.TrimSpace(user.Provider), ProviderAccountID: strings.TrimSpace(user.ProviderAccountID),
	})
	if err != nil {
		return authenticatedUser{}, err
	}
	return authenticatedUser{
		ID: storedUser.ID, Name: storedUser.Name, Email: storedUser.Email, Image: storedUser.ImageURL,
		Provider: strings.TrimSpace(user.Provider), ProviderAccountID: strings.TrimSpace(user.ProviderAccountID),
	}, nil
}

func (s *postgresUserStore) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	closeStoredUserStore(s.store)
	return nil
}

func (s *postgresUserStore) CreateSession(ctx context.Context, session authSessionRecord) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.CreateSession(ctx, database.SessionRecord{
		Token: strings.TrimSpace(session.Token), UserID: strings.TrimSpace(session.UserID), ExpiresAt: session.ExpiresAt,
	})
}

func (s *postgresUserStore) GetSessionUserByToken(ctx context.Context, sessionToken string, now time.Time) (database.SessionUserRecord, error) {
	if s == nil || s.store == nil {
		return database.SessionUserRecord{}, errors.New("user store is not configured")
	}
	return s.store.GetSessionUserByToken(ctx, strings.TrimSpace(sessionToken), now)
}

func (s *postgresUserStore) DeleteSession(ctx context.Context, sessionToken string) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.DeleteSession(ctx, strings.TrimSpace(sessionToken))
}

func (s *postgresUserStore) SaveLobbyState(ctx context.Context, state persistedLobbyState) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	data, _ := json.Marshal(state)
	return saveStoredLobbyState(ctx, s.store, data)
}

func (s *postgresUserStore) LoadLobbyState(ctx context.Context) (persistedLobbyState, error) {
	if s == nil || s.store == nil {
		return persistedLobbyState{}, errors.New("user store is not configured")
	}
	data, err := loadStoredLobbyState(ctx, s.store)
	if errors.Is(err, database.ErrLobbyStateNotFound) {
		return persistedLobbyState{}, nil
	}
	if err != nil {
		return persistedLobbyState{}, err
	}
	var state persistedLobbyState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedLobbyState{}, err
	}
	return state, nil
}

func (s *postgresUserStore) CreateGameBugReport(ctx context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	if s == nil || s.store == nil {
		return database.GameBugReportRecord{}, errors.New("user store is not configured")
	}
	return createStoredGameBugReport(ctx, s.store, report)
}

func (s *postgresUserStore) ListGameBugReportsPage(ctx context.Context, limit, offset int) ([]database.GameBugReportRecord, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("user store is not configured")
	}
	return s.store.ListGameBugReportsPage(ctx, limit, offset)
}

func (s *postgresUserStore) CountGameBugReports(ctx context.Context) (int64, error) {
	if s == nil || s.store == nil {
		return 0, errors.New("user store is not configured")
	}
	return s.store.CountGameBugReports(ctx)
}

func (s *postgresUserStore) GetGameBugReport(ctx context.Context, reportID string) (database.GameBugReportRecord, error) {
	if s == nil || s.store == nil {
		return database.GameBugReportRecord{}, errors.New("user store is not configured")
	}
	return s.store.GetGameBugReport(ctx, strings.TrimSpace(reportID))
}

func (s *postgresUserStore) CompleteGameBugReport(ctx context.Context, reportID string) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.CompleteGameBugReport(ctx, strings.TrimSpace(reportID))
}

func (s *postgresUserStore) GetPlayerProfile(ctx context.Context, userID string) (database.PlayerProfileRecord, error) {
	if s == nil || s.store == nil {
		return database.PlayerProfileRecord{}, errors.New("user store is not configured")
	}
	return s.store.GetPlayerProfile(ctx, strings.TrimSpace(userID))
}

func (s *postgresUserStore) GetPlayerGameHistory(ctx context.Context, userID string, limit, offset int, filters ...database.GameHistoryFilter) (database.PlayerGameHistoryPage, error) {
	if s == nil || s.store == nil {
		return database.PlayerGameHistoryPage{}, errors.New("user store is not configured")
	}
	return s.store.GetPlayerGameHistory(ctx, strings.TrimSpace(userID), limit, offset, filters...)
}

func (s *postgresUserStore) GetLeaderboard(ctx context.Context, cursor *database.LeaderboardCursor, limit int, viewerUserID string, metric database.LeaderboardMetric, scope database.LeaderboardScope) (database.LeaderboardPage, error) {
	if s == nil || s.store == nil {
		return database.LeaderboardPage{}, errors.New("user store is not configured")
	}
	return s.store.GetLeaderboard(ctx, cursor, limit, strings.TrimSpace(viewerUserID), metric, scope)
}

func (s *postgresUserStore) SaveCompletedGame(ctx context.Context, game database.CompletedGameRecord) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.SaveCompletedGame(ctx, game)
}

func (s *postgresUserStore) SaveGameCheckpoint(ctx context.Context, checkpoint database.GameCheckpointRecord) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.SaveGameCheckpoint(ctx, checkpoint)
}

func (s *postgresUserStore) SaveUnrankedGame(ctx context.Context, checkpoint database.GameCheckpointRecord, status string, completedAt time.Time) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.SaveUnrankedGame(ctx, checkpoint, status, completedAt)
}

func (s *postgresUserStore) ListSocialSnapshot(ctx context.Context, userID string) (database.SocialSnapshotRecord, error) {
	if s == nil || s.store == nil {
		return database.SocialSnapshotRecord{}, errors.New("user store is not configured")
	}
	return s.store.ListSocialSnapshot(ctx, strings.TrimSpace(userID))
}

func (s *postgresUserStore) SendFriendRequest(ctx context.Context, senderID, recipientID string) (database.FriendRequestRecord, error) {
	if s == nil || s.store == nil {
		return database.FriendRequestRecord{}, errors.New("user store is not configured")
	}
	return s.store.SendFriendRequest(ctx, strings.TrimSpace(senderID), strings.TrimSpace(recipientID))
}

func (s *postgresUserStore) RespondFriendRequest(ctx context.Context, recipientID, requestID string, accept bool) (string, error) {
	if s == nil || s.store == nil {
		return "", errors.New("user store is not configured")
	}
	return s.store.RespondFriendRequest(ctx, strings.TrimSpace(recipientID), strings.TrimSpace(requestID), accept)
}

func (s *postgresUserStore) RemoveFriend(ctx context.Context, userID, friendID string) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}
	return s.store.RemoveFriend(ctx, strings.TrimSpace(userID), strings.TrimSpace(friendID))
}

func (s *postgresUserStore) SendGameInvite(ctx context.Context, senderID, recipientID, roomCode string, expiresAt time.Time) (database.GameInviteRecord, error) {
	if s == nil || s.store == nil {
		return database.GameInviteRecord{}, errors.New("user store is not configured")
	}
	return s.store.SendGameInvite(ctx, strings.TrimSpace(senderID), strings.TrimSpace(recipientID), strings.TrimSpace(roomCode), expiresAt)
}

func (s *postgresUserStore) GetGameInvite(ctx context.Context, recipientID, inviteID string) (database.GameInviteRecord, error) {
	if s == nil || s.store == nil {
		return database.GameInviteRecord{}, errors.New("user store is not configured")
	}
	return s.store.GetGameInvite(ctx, strings.TrimSpace(recipientID), strings.TrimSpace(inviteID))
}

func (s *postgresUserStore) DeleteGameInvite(ctx context.Context, recipientID, inviteID string) (string, error) {
	if s == nil || s.store == nil {
		return "", errors.New("user store is not configured")
	}
	return s.store.DeleteGameInvite(ctx, strings.TrimSpace(recipientID), strings.TrimSpace(inviteID))
}

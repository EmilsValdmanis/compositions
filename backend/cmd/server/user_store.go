package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

const defaultUserStoreTimeout = 5 * time.Second

var openConfiguredUserStore = newConfiguredUserStore
var databaseURLFromEnv = database.URLFromEnv
var newDatabaseUserStore = database.NewUserStore
var upsertStoredUser = func(ctx context.Context, store *database.UserStore, user database.UserRecord) (database.UserRecord, error) {
	return store.UpsertUser(ctx, user)
}
var closeStoredUserStore = func(store *database.UserStore) {
	store.Close()
}
var saveStoredLobbyState = func(ctx context.Context, store *database.UserStore, data []byte) error {
	return store.SaveLobbyState(ctx, data)
}
var loadStoredLobbyState = func(ctx context.Context, store *database.UserStore) ([]byte, error) {
	return store.LoadLobbyState(ctx)
}
var createStoredGameBugReport = func(ctx context.Context, store *database.UserStore, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return store.CreateGameBugReport(ctx, report)
}

type userStore interface {
	UpsertUser(ctx context.Context, user authenticatedUser) (authenticatedUser, error)
	CreateSession(ctx context.Context, session authSessionRecord) error
	GetSessionUserByToken(ctx context.Context, sessionToken string, now time.Time) (database.SessionUserRecord, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	SaveLobbyState(ctx context.Context, state persistedLobbyState) error
	LoadLobbyState(ctx context.Context) (persistedLobbyState, error)
	CreateGameBugReport(ctx context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error)
	Close() error
}

type gameStatisticsStore interface {
	SaveGameCheckpoint(ctx context.Context, checkpoint database.GameCheckpointRecord) error
	SaveUnrankedGame(ctx context.Context, checkpoint database.GameCheckpointRecord, status string, completedAt time.Time) error
	SaveCompletedGame(ctx context.Context, game database.CompletedGameRecord) error
}

type noopUserStore struct{}

func (noopUserStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	return user, nil
}

func (noopUserStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (noopUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (noopUserStore) DeleteSession(context.Context, string) error { return database.ErrSessionNotFound }

func (noopUserStore) SaveLobbyState(context.Context, persistedLobbyState) error { return nil }

func (noopUserStore) LoadLobbyState(context.Context) (persistedLobbyState, error) {
	return persistedLobbyState{}, nil
}

func (noopUserStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return report, nil
}

func (noopUserStore) Close() error { return nil }

type postgresUserStore struct {
	store *database.UserStore
}

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
		ID:                strings.TrimSpace(user.ID),
		Name:              strings.TrimSpace(user.Name),
		Email:             strings.ToLower(strings.TrimSpace(user.Email)),
		ImageURL:          strings.TrimSpace(user.Image),
		Provider:          strings.TrimSpace(user.Provider),
		ProviderAccountID: strings.TrimSpace(user.ProviderAccountID),
	})
	if err != nil {
		return authenticatedUser{}, err
	}

	return authenticatedUser{
		ID:                storedUser.ID,
		Name:              storedUser.Name,
		Email:             storedUser.Email,
		Image:             storedUser.ImageURL,
		Provider:          strings.TrimSpace(user.Provider),
		ProviderAccountID: strings.TrimSpace(user.ProviderAccountID),
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
		Token:     strings.TrimSpace(session.Token),
		UserID:    strings.TrimSpace(session.UserID),
		ExpiresAt: session.ExpiresAt,
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

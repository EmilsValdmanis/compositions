package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

const defaultUserStoreTimeout = 5 * time.Second

var openConfiguredUserStore = newConfiguredUserStore
var databaseURLFromEnv = database.URLFromEnv
var newDatabaseUserStore = database.NewUserStore
var upsertStoredUser = func(ctx context.Context, store *database.UserStore, user database.UserRecord) error {
	return store.UpsertUser(ctx, user)
}
var closeStoredUserStore = func(store *database.UserStore) {
	store.Close()
}

type userStore interface {
	UpsertUser(ctx context.Context, user authenticatedUser) error
	CreateSession(ctx context.Context, session authSessionRecord) error
	GetSessionUserByToken(ctx context.Context, sessionToken string, now time.Time) (database.SessionUserRecord, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	Close() error
}

type noopUserStore struct{}

func (noopUserStore) UpsertUser(context.Context, authenticatedUser) error { return nil }

func (noopUserStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (noopUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (noopUserStore) DeleteSession(context.Context, string) error { return database.ErrSessionNotFound }

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

func (s *postgresUserStore) UpsertUser(ctx context.Context, user authenticatedUser) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}

	return upsertStoredUser(ctx, s.store, database.UserRecord{
		ID:       strings.TrimSpace(user.ID),
		Name:     strings.TrimSpace(user.Name),
		Email:    strings.ToLower(strings.TrimSpace(user.Email)),
		ImageURL: strings.TrimSpace(user.Image),
	})
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

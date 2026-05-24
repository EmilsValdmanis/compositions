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
	Close() error
}

type noopUserStore struct{}

func (noopUserStore) UpsertUser(context.Context, authenticatedUser) error { return nil }

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
		Email:    strings.TrimSpace(user.Email),
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

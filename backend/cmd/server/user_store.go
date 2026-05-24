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
	databaseURL, err := database.URLFromEnv()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()

	store, err := database.NewUserStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	return &postgresUserStore{store: store}, nil
}

func (s *postgresUserStore) UpsertUser(ctx context.Context, user authenticatedUser) error {
	if s == nil || s.store == nil {
		return errors.New("user store is not configured")
	}

	return s.store.UpsertUser(ctx, database.UserRecord{
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

	s.store.Close()
	return nil
}

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

func TestNewConfiguredUserStore(t *testing.T) {
	originalDatabaseURLFromEnv := databaseURLFromEnv
	defer func() { databaseURLFromEnv = originalDatabaseURLFromEnv }()
	originalNewDatabaseUserStore := newDatabaseUserStore
	defer func() { newDatabaseUserStore = originalNewDatabaseUserStore }()

	t.Run("returns env error", func(t *testing.T) {
		databaseURLFromEnv = func() (string, error) {
			return "", errors.New("missing database url")
		}

		store, err := newConfiguredUserStore()
		if err == nil || err.Error() != "missing database url" {
			t.Fatalf("newConfiguredUserStore() error = %v; want missing database url", err)
		}
		if store != nil {
			t.Fatalf("store = %#v; want nil", store)
		}
	})

	t.Run("returns constructor error", func(t *testing.T) {
		databaseURLFromEnv = func() (string, error) {
			return "postgres://configured", nil
		}
		newDatabaseUserStore = func(ctx context.Context, databaseURL string) (*database.UserStore, error) {
			if databaseURL != "postgres://configured" {
				t.Fatalf("databaseURL = %q; want postgres://configured", databaseURL)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("ctx has no deadline; want timeout")
			}
			return nil, errors.New("open store boom")
		}

		store, err := newConfiguredUserStore()
		if err == nil || err.Error() != "open store boom" {
			t.Fatalf("newConfiguredUserStore() error = %v; want open store boom", err)
		}
		if store != nil {
			t.Fatalf("store = %#v; want nil", store)
		}
	})

	t.Run("returns postgres store", func(t *testing.T) {
		databaseURLFromEnv = func() (string, error) {
			return "postgres://configured", nil
		}
		newDatabaseUserStore = func(context.Context, string) (*database.UserStore, error) {
			return &database.UserStore{}, nil
		}

		store, err := newConfiguredUserStore()
		if err != nil {
			t.Fatalf("newConfiguredUserStore() error = %v", err)
		}
		postgresStore, ok := store.(*postgresUserStore)
		if !ok {
			t.Fatalf("store type = %T; want *postgresUserStore", store)
		}
		if postgresStore.store == nil {
			t.Fatal("postgresStore.store = nil; want configured database store")
		}
	})
}

func TestPostgresUserStoreUpsertUser(t *testing.T) {
	originalUpsertStoredUser := upsertStoredUser
	defer func() { upsertStoredUser = originalUpsertStoredUser }()

	t.Run("requires configured store", func(t *testing.T) {
		var store *postgresUserStore
		if _, err := store.UpsertUser(context.Background(), authenticatedUser{}); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("UpsertUser() error = %v; want user store is not configured", err)
		}
	})

	t.Run("trims fields before upsert", func(t *testing.T) {
		called := false
		upsertStoredUser = func(ctx context.Context, store *database.UserStore, user database.UserRecord) (database.UserRecord, error) {
			called = true
			if store == nil {
				t.Fatal("store = nil; want configured store")
			}
			if user.ID != "user-1" || user.Name != "Player One" || user.Email != "player@example.com" || user.ImageURL != "https://cdn.example.com/player.png" {
				t.Fatalf("user = %#v; want trimmed fields", user)
			}
			return user, nil
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		storedUser, err := store.UpsertUser(context.Background(), authenticatedUser{
			ID:    " user-1 ",
			Name:  " Player One ",
			Email: " Player@Example.com ",
			Image: " https://cdn.example.com/player.png ",
		})
		if err != nil {
			t.Fatalf("UpsertUser() error = %v", err)
		}
		if storedUser.ID != "user-1" {
			t.Fatalf("stored user id = %q; want user-1", storedUser.ID)
		}
		if !called {
			t.Fatal("upsertStoredUser was not called")
		}
	})
}

func TestPostgresUserStoreClose(t *testing.T) {
	originalCloseStoredUserStore := closeStoredUserStore
	defer func() { closeStoredUserStore = originalCloseStoredUserStore }()

	var nilStore *postgresUserStore
	if err := nilStore.Close(); err != nil {
		t.Fatalf("nilStore.Close() error = %v", err)
	}

	store := &postgresUserStore{}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}

	closed := false
	closeStoredUserStore = func(store *database.UserStore) {
		closed = true
		if store == nil {
			t.Fatal("store = nil; want configured store")
		}
	}

	store = &postgresUserStore{store: &database.UserStore{}}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}
	if !closed {
		t.Fatal("closeStoredUserStore was not called")
	}
}

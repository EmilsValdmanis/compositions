package main

import (
	"context"
	"errors"
	"testing"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

func TestNewConfiguredUserStore(t *testing.T) {
	originalNewDatabaseUserStore := newDatabaseUserStore
	defer func() { newDatabaseUserStore = originalNewDatabaseUserStore }()

	t.Run("returns env error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")

		store, err := newConfiguredUserStore()
		if err == nil || err.Error() != "DATABASE_URL is required" {
			t.Fatalf("newConfiguredUserStore() error = %v; want missing database url", err)
		}
		if store != nil {
			t.Fatalf("store = %#v; want nil", store)
		}
	})

	t.Run("returns constructor error", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://configured")
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
		t.Setenv("DATABASE_URL", "postgres://configured")
		newDatabaseUserStore = func(context.Context, string) (*database.UserStore, error) {
			return &database.UserStore{}, nil
		}

		store, err := newConfiguredUserStore()
		if err != nil {
			t.Fatalf("newConfiguredUserStore() error = %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("close configured store: %v", err)
		}
	})
}

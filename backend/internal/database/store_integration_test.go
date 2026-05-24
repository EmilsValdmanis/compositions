//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestUserStoreUpsertUser(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up) error = %v", err)
	}
	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up second pass) error = %v", err)
	}

	store, err := NewUserStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	defer store.Close()

	createdUser := UserRecord{
		ID:       "user-1",
		Name:     "Player One",
		Email:    "player1@example.com",
		ImageURL: "https://cdn.example.com/player-1.png",
	}
	if err := store.UpsertUser(ctx, createdUser); err != nil {
		t.Fatalf("UpsertUser(create) error = %v", err)
	}

	createdRecord, err := store.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID(create) error = %v", err)
	}
	if createdRecord.Name != createdUser.Name {
		t.Fatalf("created name = %q; want %q", createdRecord.Name, createdUser.Name)
	}
	if createdRecord.Email != createdUser.Email {
		t.Fatalf("created email = %q; want %q", createdRecord.Email, createdUser.Email)
	}
	if createdRecord.ImageUrl != createdUser.ImageURL {
		t.Fatalf("created image_url = %q; want %q", createdRecord.ImageUrl, createdUser.ImageURL)
	}

	updatedUser := UserRecord{
		ID:       "user-1",
		Name:     "Updated Player",
		Email:    "updated@example.com",
		ImageURL: "https://cdn.example.com/player-1-updated.png",
	}
	if err := store.UpsertUser(ctx, updatedUser); err != nil {
		t.Fatalf("UpsertUser(update) error = %v", err)
	}

	updatedRecord, err := store.GetUserByID(ctx, updatedUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID(update) error = %v", err)
	}
	if updatedRecord.Name != updatedUser.Name {
		t.Fatalf("updated name = %q; want %q", updatedRecord.Name, updatedUser.Name)
	}
	if updatedRecord.Email != updatedUser.Email {
		t.Fatalf("updated email = %q; want %q", updatedRecord.Email, updatedUser.Email)
	}
	if updatedRecord.ImageUrl != updatedUser.ImageURL {
		t.Fatalf("updated image_url = %q; want %q", updatedRecord.ImageUrl, updatedUser.ImageURL)
	}
	if !updatedRecord.CreatedAt.Valid || !updatedRecord.UpdatedAt.Valid {
		t.Fatalf("expected valid timestamps, got created_at=%#v updated_at=%#v", updatedRecord.CreatedAt, updatedRecord.UpdatedAt)
	}
	if updatedRecord.UpdatedAt.Time.Before(updatedRecord.CreatedAt.Time) {
		t.Fatalf("updated timestamps are inconsistent: created_at=%v updated_at=%v", updatedRecord.CreatedAt, updatedRecord.UpdatedAt)
	}

	if err := RunMigrations(ctx, databaseURL, MigrationDown); err != nil {
		t.Fatalf("RunMigrations(down) error = %v", err)
	}
}

func startPostgresContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "compositions",
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container.Host() error = %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("container.MappedPort() error = %v", err)
	}

	return fmt.Sprintf("postgres://postgres:postgres@%s:%s/compositions?sslmode=disable", host, port.Port())
}

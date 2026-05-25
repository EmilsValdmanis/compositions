//go:build integration

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
		Email:    "player1@example.com",
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

	conflictingUser := UserRecord{
		ID:       "user-2",
		Name:     " Conflicting Player ",
		Email:    " PLAYER1@EXAMPLE.COM ",
		ImageURL: " https://cdn.example.com/player-2.png ",
	}
	if err := store.UpsertUser(ctx, conflictingUser); !errors.Is(err, ErrUserConflict) {
		t.Fatalf("UpsertUser(conflicting) error = %v; want %v", err, ErrUserConflict)
	}

	if got := countUsers(t, ctx, pool); got != 1 {
		t.Fatalf("user row count = %d; want 1", got)
	}

	preservedRecord, err := store.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID(preserved) error = %v", err)
	}
	if preservedRecord.Name != updatedUser.Name {
		t.Fatalf("preserved name = %q; want %q", preservedRecord.Name, updatedUser.Name)
	}
	if preservedRecord.Email != updatedUser.Email {
		t.Fatalf("preserved email = %q; want %q", preservedRecord.Email, updatedUser.Email)
	}
	if preservedRecord.ImageUrl != updatedUser.ImageURL {
		t.Fatalf("preserved image_url = %q; want %q", preservedRecord.ImageUrl, updatedUser.ImageURL)
	}
	if _, err := store.GetUserByID(ctx, strings.TrimSpace(conflictingUser.ID)); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetUserByID(conflicting id) error = %v; want %v", err, pgx.ErrNoRows)
	}
}

func TestUserEmailUniquenessMigrationDeduplicatesExistingRows(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	migrationDB, err := OpenMigrationDB(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenMigrationDB() error = %v", err)
	}
	defer migrationDB.Close()

	migrator := newTestMigrator(t, migrationDB)
	if err := migrator.Steps(1); err != nil {
		t.Fatalf("migrator.Steps(1 first migration) error = %v", err)
	}

	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, image_url, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6),
			($7, $8, $9, $10, $11, $12)
	`,
		"user-1",
		" First Player ",
		"Player@example.com",
		" https://cdn.example.com/player-1.png ",
		time.Date(2026, time.May, 24, 15, 11, 1, 0, time.UTC),
		time.Date(2026, time.May, 24, 15, 11, 56, 0, time.UTC),
		"user-2",
		" Second Player ",
		" PLAYER@EXAMPLE.COM ",
		" https://cdn.example.com/player-2.png ",
		time.Date(2026, time.May, 24, 15, 51, 3, 0, time.UTC),
		time.Date(2026, time.May, 24, 15, 51, 3, 0, time.UTC),
	); err != nil {
		t.Fatalf("seed duplicate users error = %v", err)
	}

	if err := migrator.Steps(1); err != nil {
		t.Fatalf("migrator.Steps(1 second migration) error = %v", err)
	}

	if got := countUsers(t, ctx, pool); got != 1 {
		t.Fatalf("user row count after migration = %d; want 1", got)
	}

	var record struct {
		ID       string
		Name     string
		Email    string
		ImageURL string
	}
	if err := pool.QueryRow(ctx, `
		SELECT id, name, email, image_url
		FROM users
		LIMIT 1
	`).Scan(&record.ID, &record.Name, &record.Email, &record.ImageURL); err != nil {
		t.Fatalf("select deduplicated user error = %v", err)
	}
	if record.ID != "user-2" {
		t.Fatalf("deduplicated id = %q; want %q", record.ID, "user-2")
	}
	if record.Name != "Second Player" {
		t.Fatalf("deduplicated name = %q; want %q", record.Name, "Second Player")
	}
	if record.Email != "player@example.com" {
		t.Fatalf("deduplicated email = %q; want %q", record.Email, "player@example.com")
	}
	if record.ImageURL != "https://cdn.example.com/player-2.png" {
		t.Fatalf("deduplicated image_url = %q; want %q", record.ImageURL, "https://cdn.example.com/player-2.png")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, image_url)
		VALUES ($1, $2, $3, $4)
	`, "user-3", "Third Player", "PLAYER@example.com", "https://cdn.example.com/player-3.png"); err == nil {
		t.Fatal("expected duplicate email insert to fail")
	}
}

func countUsers(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("count users error = %v", err)
	}

	return count
}

func newTestMigrator(t *testing.T, db *sql.DB) *migrate.Migrate {
	t.Helper()

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("iofs.New() error = %v", err)
	}

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		t.Fatalf("pgx.WithInstance() error = %v", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		t.Fatalf("migrate.NewWithInstance() error = %v", err)
	}

	t.Cleanup(func() {
		_, _ = migrator.Close()
	})

	return migrator
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

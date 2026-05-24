package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

func TestMigrationDirectionFromArgs(t *testing.T) {
	t.Run("requires exactly one arg", func(t *testing.T) {
		direction, err := migrationDirectionFromArgs(nil)
		if err == nil || err.Error() != usageText() {
			t.Fatalf("migrationDirectionFromArgs(nil) error = %v; want usage text", err)
		}
		if direction != "" {
			t.Fatalf("direction = %q; want empty", direction)
		}
	})

	t.Run("accepts up case insensitively", func(t *testing.T) {
		direction, err := migrationDirectionFromArgs([]string{" Up "})
		if err != nil {
			t.Fatalf("migrationDirectionFromArgs(up) error = %v", err)
		}
		if direction != database.MigrationUp {
			t.Fatalf("direction = %q; want %q", direction, database.MigrationUp)
		}
	})

	t.Run("accepts down case insensitively", func(t *testing.T) {
		direction, err := migrationDirectionFromArgs([]string{"DOWN"})
		if err != nil {
			t.Fatalf("migrationDirectionFromArgs(down) error = %v", err)
		}
		if direction != database.MigrationDown {
			t.Fatalf("direction = %q; want %q", direction, database.MigrationDown)
		}
	})

	t.Run("rejects unsupported direction", func(t *testing.T) {
		direction, err := migrationDirectionFromArgs([]string{"sideways"})
		want := "unsupported migration direction \"sideways\"\n\n" + usageText()
		if err == nil || err.Error() != want {
			t.Fatalf("migrationDirectionFromArgs(sideways) error = %v; want %q", err, want)
		}
		if direction != "" {
			t.Fatalf("direction = %q; want empty", direction)
		}
	})
}

func TestErrorsNewUsageAndUsageText(t *testing.T) {
	if got := usageText(); got != "usage: go run ./cmd/migrate [up|down]" {
		t.Fatalf("usageText() = %q; want usage string", got)
	}
	if err := errorsNewUsage(); err == nil || err.Error() != usageText() {
		t.Fatalf("errorsNewUsage() error = %v; want usage text", err)
	}
}

func TestMain(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	originalFatal := fatalOnRunError
	defer func() { fatalOnRunError = originalFatal }()
	originalDatabaseURLFromEnv := databaseURLFromEnv
	defer func() { databaseURLFromEnv = originalDatabaseURLFromEnv }()
	originalRunDatabaseMigrations := runDatabaseMigrations
	defer func() { runDatabaseMigrations = originalRunDatabaseMigrations }()

	t.Run("reports invalid args", func(t *testing.T) {
		os.Args = []string{"migrate"}
		fatalArgs := []any(nil)
		fatalOnRunError = func(v ...any) { fatalArgs = v }

		main()

		if len(fatalArgs) != 1 {
			t.Fatalf("fatal args = %v; want single error", fatalArgs)
		}
		err, ok := fatalArgs[0].(error)
		if !ok || err == nil || err.Error() != usageText() {
			t.Fatalf("fatal error = %v; want usage text", fatalArgs[0])
		}
	})

	t.Run("reports database url error", func(t *testing.T) {
		os.Args = []string{"migrate", "up"}
		databaseURLFromEnv = func() (string, error) { return "", errors.New("missing database url") }
		fatalArgs := []any(nil)
		fatalOnRunError = func(v ...any) { fatalArgs = v }

		main()

		if len(fatalArgs) != 1 {
			t.Fatalf("fatal args = %v; want single error", fatalArgs)
		}
		err, ok := fatalArgs[0].(error)
		if !ok || err == nil || err.Error() != "missing database url" {
			t.Fatalf("fatal error = %v; want missing database url", fatalArgs[0])
		}
	})

	t.Run("reports migration error", func(t *testing.T) {
		os.Args = []string{"migrate", "down"}
		databaseURLFromEnv = func() (string, error) { return "postgres://configured", nil }
		runDatabaseMigrations = func(ctx context.Context, databaseURL string, direction database.MigrationDirection) error {
			if databaseURL != "postgres://configured" {
				t.Fatalf("databaseURL = %q; want postgres://configured", databaseURL)
			}
			if direction != database.MigrationDown {
				t.Fatalf("direction = %q; want %q", direction, database.MigrationDown)
			}
			if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) <= 0 || time.Until(deadline) > migrationTimeout {
				t.Fatalf("ctx deadline = %v, %v; want timeout within migrationTimeout", deadline, ok)
			}
			return errors.New("migration boom")
		}
		fatalArgs := []any(nil)
		fatalOnRunError = func(v ...any) { fatalArgs = v }

		main()

		if len(fatalArgs) != 1 {
			t.Fatalf("fatal args = %v; want single error", fatalArgs)
		}
		err, ok := fatalArgs[0].(error)
		if !ok || err == nil || err.Error() != "migration boom" {
			t.Fatalf("fatal error = %v; want migration boom", fatalArgs[0])
		}
	})

	t.Run("runs migrations successfully", func(t *testing.T) {
		os.Args = []string{"migrate", "up"}
		databaseURLFromEnv = func() (string, error) { return "postgres://configured", nil }
		called := false
		runDatabaseMigrations = func(ctx context.Context, databaseURL string, direction database.MigrationDirection) error {
			called = true
			if databaseURL != "postgres://configured" {
				t.Fatalf("databaseURL = %q; want postgres://configured", databaseURL)
			}
			if direction != database.MigrationUp {
				t.Fatalf("direction = %q; want %q", direction, database.MigrationUp)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("ctx has no deadline; want timeout")
			}
			return nil
		}
		fatalCalled := false
		fatalOnRunError = func(v ...any) { fatalCalled = true }

		main()

		if !called {
			t.Fatal("runDatabaseMigrations was not called")
		}
		if fatalCalled {
			t.Fatal("fatalOnRunError was called; want successful run")
		}
	})
}

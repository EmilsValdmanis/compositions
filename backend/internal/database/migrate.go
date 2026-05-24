package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type MigrationDirection string

const (
	MigrationUp   MigrationDirection = "up"
	MigrationDown MigrationDirection = "down"
)

func OpenMigrationDB(ctx context.Context, databaseURL string) (*sql.DB, error) {
	cleanURL, err := URLFromString(databaseURL)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", cleanURL)
	if err != nil {
		return nil, fmt.Errorf("open migration db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping migration db: %w", err)
	}

	return db, nil
}

func RunMigrations(ctx context.Context, databaseURL string, direction MigrationDirection) error {
	db, err := OpenMigrationDB(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	return runMigrationsWithDB(db, direction)
}

func runMigrationsWithDB(db *sql.DB, direction MigrationDirection) error {
	if db == nil {
		return errors.New("migration db is required")
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migrate source: %w", err)
	}

	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("create migrate postgres driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	migrator.Log = migrationLogger{}
	defer func() {
		_, _ = migrator.Close()
	}()

	currentVersion, dirty, err := migrator.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			log.Printf("migration state direction=%s current_version=none dirty=false", direction)
		} else {
			return fmt.Errorf("read current migration version: %w", err)
		}
	} else {
		log.Printf("migration state direction=%s current_version=%d dirty=%t", direction, currentVersion, dirty)
	}

	log.Printf("migration run starting direction=%s", direction)

	switch direction {
	case MigrationUp:
		err = migrator.Up()
	case MigrationDown:
		err = migrator.Steps(-1)
	default:
		return fmt.Errorf("unsupported migration direction %q", direction)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations %s: %w", direction, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Printf("migration run finished direction=%s result=no_change", direction)
	} else {
		log.Printf("migration run finished direction=%s result=applied", direction)
	}

	currentVersion, dirty, err = migrator.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			log.Printf("migration state after run direction=%s current_version=none dirty=false", direction)
			return nil
		}
		return fmt.Errorf("read migration version after %s: %w", direction, err)
	}

	log.Printf("migration state after run direction=%s current_version=%d dirty=%t", direction, currentVersion, dirty)

	return nil
}

type migrationLogger struct{}

func (migrationLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (migrationLogger) Verbose() bool {
	return true
}

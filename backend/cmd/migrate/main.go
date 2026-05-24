package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

const migrationTimeout = 30 * time.Second

func main() {
	direction, err := migrationDirectionFromArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	databaseURL, err := database.URLFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	if err := database.RunMigrations(ctx, databaseURL, direction); err != nil {
		log.Fatal(err)
	}
}

func migrationDirectionFromArgs(args []string) (database.MigrationDirection, error) {
	if len(args) != 1 {
		return "", errorsNewUsage()
	}

	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case string(database.MigrationUp):
		return database.MigrationUp, nil
	case string(database.MigrationDown):
		return database.MigrationDown, nil
	default:
		return "", fmt.Errorf("unsupported migration direction %q\n\n%s", args[0], usageText())
	}
}

func errorsNewUsage() error {
	return errors.New(usageText())
}

func usageText() string {
	return "usage: go run ./cmd/migrate [up|down]"
}

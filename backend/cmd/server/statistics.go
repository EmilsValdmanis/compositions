package main

import (
	"context"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

// gameStatisticsStore is intentionally separate from userStore so local/no-op
// servers can run without pretending to persist competitive statistics.
type gameStatisticsStore interface {
	SaveGameCheckpoint(ctx context.Context, checkpoint database.GameCheckpointRecord) error
	SaveUnrankedGame(ctx context.Context, checkpoint database.GameCheckpointRecord, status string, completedAt time.Time) error
	SaveCompletedGame(ctx context.Context, game database.CompletedGameRecord) error
}

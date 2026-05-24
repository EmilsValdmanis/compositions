package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dbsqlc "github.com/EmilsValdmanis/compositions/internal/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRecord struct {
	ID       string
	Name     string
	Email    string
	ImageURL string
}

type UserStore struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

func URLFromEnv() (string, error) {
	return URLFromString(os.Getenv("DATABASE_URL"))
}

func URLFromString(databaseURL string) (string, error) {
	cleanURL := strings.TrimSpace(databaseURL)
	if cleanURL == "" {
		return "", errors.New("DATABASE_URL is required")
	}

	return cleanURL, nil
}

func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cleanURL := strings.TrimSpace(databaseURL)
	if cleanURL == "" {
		return nil, errors.New("database url is required")
	}

	pool, err := pgxpool.New(ctx, cleanURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func NewUserStore(ctx context.Context, databaseURL string) (*UserStore, error) {
	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	return &UserStore{
		pool:    pool,
		queries: dbsqlc.New(pool),
	}, nil
}

func (s *UserStore) UpsertUser(ctx context.Context, user UserRecord) error {
	if s == nil || s.queries == nil {
		return errors.New("user store is not configured")
	}

	if strings.TrimSpace(user.ID) == "" {
		return errors.New("user id is required")
	}

	return s.queries.UpsertUser(ctx, dbsqlc.UpsertUserParams{
		ID:       strings.TrimSpace(user.ID),
		Name:     strings.TrimSpace(user.Name),
		Email:    strings.TrimSpace(user.Email),
		ImageUrl: strings.TrimSpace(user.ImageURL),
	})
}

func (s *UserStore) GetUserByID(ctx context.Context, userID string) (dbsqlc.User, error) {
	if s == nil || s.queries == nil {
		return dbsqlc.User{}, errors.New("user store is not configured")
	}

	return s.queries.GetUserByID(ctx, strings.TrimSpace(userID))
}

func (s *UserStore) Close() {
	if s == nil || s.pool == nil {
		return
	}

	s.pool.Close()
}

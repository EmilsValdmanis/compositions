package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	dbsqlc "github.com/EmilsValdmanis/compositions/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

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

type SessionRecord struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

type SessionUserRecord struct {
	ID        string
	Name      string
	Email     string
	ImageURL  string
	ExpiresAt time.Time
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

	userID := strings.TrimSpace(user.ID)
	if userID == "" {
		return errors.New("user id is required")
	}

	email := normalizeEmail(user.Email)
	name := strings.TrimSpace(user.Name)
	imageURL := strings.TrimSpace(user.ImageURL)

	return s.queries.UpsertUser(ctx, dbsqlc.UpsertUserParams{
		ID:       userID,
		Name:     name,
		Email:    email,
		ImageUrl: imageURL,
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashSessionToken(sessionToken string) (string, error) {
	cleanToken := strings.TrimSpace(sessionToken)
	if cleanToken == "" {
		return "", errors.New("session token is required")
	}

	hash := sha256.Sum256([]byte(cleanToken))
	return hex.EncodeToString(hash[:]), nil
}

func (s *UserStore) CreateSession(ctx context.Context, session SessionRecord) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}

	tokenHash, err := hashSessionToken(session.Token)
	if err != nil {
		return err
	}
	userID := strings.TrimSpace(session.UserID)
	if userID == "" {
		return errors.New("user id is required")
	}
	if session.ExpiresAt.IsZero() {
		return errors.New("session expiry is required")
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
	`, tokenHash, userID, session.ExpiresAt.UTC())
	return err
}

func (s *UserStore) GetSessionUserByToken(ctx context.Context, sessionToken string, now time.Time) (SessionUserRecord, error) {
	if s == nil || s.pool == nil {
		return SessionUserRecord{}, errors.New("user store is not configured")
	}

	tokenHash, err := hashSessionToken(sessionToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionUserRecord{}, ErrSessionNotFound
		}
		return SessionUserRecord{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}

	var record SessionUserRecord
	err = s.pool.QueryRow(ctx, `
		SELECT u.id, u.name, u.email, u.image_url, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.expires_at > $2
	`, tokenHash, now.UTC()).Scan(
		&record.ID,
		&record.Name,
		&record.Email,
		&record.ImageURL,
		&record.ExpiresAt,
	)
	if err != nil {
		return SessionUserRecord{}, err
	}

	return record, nil
}

func (s *UserStore) DeleteSession(ctx context.Context, sessionToken string) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}

	tokenHash, err := hashSessionToken(sessionToken)
	if err != nil {
		return err
	}

	commandTag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	return nil
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

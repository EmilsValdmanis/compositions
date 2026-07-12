package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	dbsqlc "github.com/EmilsValdmanis/compositions/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrUserConflict = errors.New("user conflict")
var ErrLobbyStateNotFound = errors.New("lobby state not found")
var ErrPlayerProfileNotFound = errors.New("player profile not found")

const postgresUniqueViolation = "23505"

type UserRecord struct {
	ID                string
	Name              string
	Email             string
	ImageURL          string
	Provider          string
	ProviderAccountID string
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

type StoredUserRecord struct {
	ID       string
	Name     string
	Email    string
	ImageURL string
}

type PlayerProfileRecord struct {
	ID, Name, ImageURL                            string
	GamesPlayed, GamesWon, TotalPlacement         int64
	RoundsPlayed, RoundsWon, Forfeits             int64
	CompositionsCreated, SetsCreated, RunsCreated int64
	PointsInflicted, PenaltyPoints                int64
	CurrentGameWinStreak, LongestGameWinStreak    int
	CurrentRoundWinStreak, LongestRoundWinStreak  int
}

type GameBugReportRecord struct {
	ID               string
	RoomCode         string
	ReporterPlayerID string
	ReporterUserID   string
	Description      string
	GameState        json.RawMessage
	Round            int
	Turn             int
	RequestedAbort   bool
	CreatedAt        time.Time
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

func (s *UserStore) UpsertUser(ctx context.Context, user UserRecord) (UserRecord, error) {
	if s == nil || s.queries == nil {
		return UserRecord{}, errors.New("user store is not configured")
	}

	normalizedUser := normalizeUserRecord(user)
	if normalizedUser.Provider != "" || normalizedUser.ProviderAccountID != "" {
		if normalizedUser.Provider == "" || normalizedUser.ProviderAccountID == "" {
			return UserRecord{}, errors.New("provider and provider account id are required together")
		}

		return s.upsertAccountUser(ctx, normalizedUser)
	}

	if normalizedUser.ID == "" {
		return UserRecord{}, errors.New("user id is required")
	}

	uuidID, err := parseUUID(normalizedUser.ID)
	if err != nil {
		return UserRecord{}, err
	}

	dbUser, err := s.queries.UpsertUserByID(ctx, dbsqlc.UpsertUserByIDParams{
		ID:       uuidID,
		Name:     normalizedUser.Name,
		Email:    normalizedUser.Email,
		ImageUrl: normalizedUser.ImageURL,
	})
	if isUniqueViolation(err) {
		return UserRecord{}, ErrUserConflict
	}
	if err != nil {
		return UserRecord{}, err
	}

	return userRecordFromRow(dbUser.ID, dbUser.Name, dbUser.Email, dbUser.ImageUrl), nil
}

func (s *UserStore) GetPlayerProfile(ctx context.Context, userID string) (PlayerProfileRecord, error) {
	if s == nil || s.pool == nil {
		return PlayerProfileRecord{}, errors.New("user store is not configured")
	}

	uuidID, err := parseUUID(userID)
	if err != nil {
		return PlayerProfileRecord{}, err
	}

	var profile PlayerProfileRecord
	err = s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.name, u.image_url,
			COALESCE(ps.games_played, 0), COALESCE(ps.games_won, 0), COALESCE(ps.total_placement, 0),
			COALESCE(ps.rounds_played, 0), COALESCE(ps.rounds_won, 0), COALESCE(ps.forfeits, 0),
			COALESCE(ps.compositions_created, 0), COALESCE(ps.sets_created, 0), COALESCE(ps.runs_created, 0),
			COALESCE(ps.points_inflicted, 0), COALESCE(ps.penalty_points, 0),
			COALESCE(ps.current_game_win_streak, 0), COALESCE(ps.longest_game_win_streak, 0),
			COALESCE(ps.current_round_win_streak, 0), COALESCE(ps.longest_round_win_streak, 0)
		FROM users u
		LEFT JOIN player_statistics ps ON ps.user_id = u.id
		WHERE u.id = $1
	`, uuidID).Scan(
		&profile.ID, &profile.Name, &profile.ImageURL,
		&profile.GamesPlayed, &profile.GamesWon, &profile.TotalPlacement,
		&profile.RoundsPlayed, &profile.RoundsWon, &profile.Forfeits,
		&profile.CompositionsCreated, &profile.SetsCreated, &profile.RunsCreated,
		&profile.PointsInflicted, &profile.PenaltyPoints,
		&profile.CurrentGameWinStreak, &profile.LongestGameWinStreak,
		&profile.CurrentRoundWinStreak, &profile.LongestRoundWinStreak,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlayerProfileRecord{}, ErrPlayerProfileNotFound
	}
	if err != nil {
		return PlayerProfileRecord{}, err
	}
	return profile, nil
}

func (s *UserStore) SaveLobbyState(ctx context.Context, state json.RawMessage) error {
	if s == nil || s.pool == nil {
		return errors.New("user store is not configured")
	}
	if len(state) == 0 || !json.Valid(state) {
		return errors.New("lobby state must be valid json")
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO lobby_state (id, state)
		VALUES (TRUE, $1)
		ON CONFLICT (id) DO UPDATE SET
			state = EXCLUDED.state,
			updated_at = NOW()
	`, state)
	return err
}

func (s *UserStore) LoadLobbyState(ctx context.Context) (json.RawMessage, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("user store is not configured")
	}

	var state json.RawMessage
	err := s.pool.QueryRow(ctx, `SELECT state FROM lobby_state WHERE id = TRUE`).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLobbyStateNotFound
	}
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (s *UserStore) CreateGameBugReport(ctx context.Context, report GameBugReportRecord) (GameBugReportRecord, error) {
	if s == nil || s.queries == nil {
		return GameBugReportRecord{}, errors.New("user store is not configured")
	}

	normalized, err := normalizeGameBugReport(report)
	if err != nil {
		return GameBugReportRecord{}, err
	}
	var reportID pgtype.UUID
	if err := reportID.Scan(normalized.ID); err != nil {
		return GameBugReportRecord{}, fmt.Errorf("invalid bug report id: %w", err)
	}
	var reporterUserID pgtype.UUID
	if normalized.ReporterUserID != "" {
		if err := reporterUserID.Scan(normalized.ReporterUserID); err != nil {
			return GameBugReportRecord{}, fmt.Errorf("invalid reporter user id: %w", err)
		}
	}

	row, err := s.queries.CreateGameBugReport(ctx, dbsqlc.CreateGameBugReportParams{
		ID:               reportID,
		RoomCode:         normalized.RoomCode,
		ReporterPlayerID: normalized.ReporterPlayerID,
		ReporterUserID:   reporterUserID,
		Description:      normalized.Description,
		GameState:        normalized.GameState,
		Round:            int32(normalized.Round),
		Turn:             int32(normalized.Turn),
		RequestedAbort:   normalized.RequestedAbort,
		CreatedAt:        pgtype.Timestamptz{Time: normalized.CreatedAt, Valid: true},
	})
	if err != nil {
		return GameBugReportRecord{}, err
	}
	return gameBugReportFromRow(
		row.ID,
		row.RoomCode,
		row.ReporterPlayerID,
		row.ReporterUserID,
		row.Description,
		row.GameState,
		row.Round,
		row.Turn,
		row.RequestedAbort,
		row.CreatedAt,
	), nil
}

func (s *UserStore) ListGameBugReports(ctx context.Context, limit int) ([]GameBugReportRecord, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("user store is not configured")
	}
	if limit <= 0 || limit > 500 {
		return nil, errors.New("bug report limit must be between 1 and 500")
	}
	rows, err := s.queries.ListGameBugReports(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	reports := make([]GameBugReportRecord, 0, len(rows))
	for _, row := range rows {
		reports = append(reports, gameBugReportFromRow(
			row.ID,
			row.RoomCode,
			row.ReporterPlayerID,
			row.ReporterUserID,
			row.Description,
			row.GameState,
			row.Round,
			row.Turn,
			row.RequestedAbort,
			row.CreatedAt,
		))
	}
	return reports, nil
}

func (s *UserStore) GetGameBugReport(ctx context.Context, reportID string) (GameBugReportRecord, error) {
	if s == nil || s.queries == nil {
		return GameBugReportRecord{}, errors.New("user store is not configured")
	}
	var id pgtype.UUID
	if err := id.Scan(strings.TrimSpace(reportID)); err != nil {
		return GameBugReportRecord{}, fmt.Errorf("invalid bug report id: %w", err)
	}
	row, err := s.queries.GetGameBugReport(ctx, id)
	if err != nil {
		return GameBugReportRecord{}, err
	}
	return gameBugReportFromRow(
		row.ID,
		row.RoomCode,
		row.ReporterPlayerID,
		row.ReporterUserID,
		row.Description,
		row.GameState,
		row.Round,
		row.Turn,
		row.RequestedAbort,
		row.CreatedAt,
	), nil
}

func nullableUUIDString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	driverValue, err := value.Value()
	if err != nil {
		return ""
	}
	text, _ := driverValue.(string)
	return text
}

func normalizeUserRecord(user UserRecord) UserRecord {
	return UserRecord{
		ID:                strings.TrimSpace(user.ID),
		Name:              strings.TrimSpace(user.Name),
		Email:             normalizeEmail(user.Email),
		ImageURL:          strings.TrimSpace(user.ImageURL),
		Provider:          strings.TrimSpace(user.Provider),
		ProviderAccountID: strings.TrimSpace(user.ProviderAccountID),
	}
}

func userRecordFromRow(id, name, email, imageURL string) UserRecord {
	return UserRecord{
		ID:       strings.TrimSpace(id),
		Name:     strings.TrimSpace(name),
		Email:    normalizeEmail(email),
		ImageURL: strings.TrimSpace(imageURL),
	}
}

func storedUserRecordFromRow(id, name, email, imageURL string) StoredUserRecord {
	return StoredUserRecord{
		ID:       strings.TrimSpace(id),
		Name:     strings.TrimSpace(name),
		Email:    normalizeEmail(email),
		ImageURL: strings.TrimSpace(imageURL),
	}
}

func (s *UserStore) upsertAccountUser(ctx context.Context, user UserRecord) (UserRecord, error) {
	if s == nil || s.pool == nil || s.queries == nil {
		return UserRecord{}, errors.New("user store is not configured")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return UserRecord{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	storedUser, err := resolveAccountUser(ctx, queries, user)
	if err != nil {
		if isUniqueViolation(err) {
			return UserRecord{}, ErrUserConflict
		}
		return UserRecord{}, err
	}

	storedUserID, err := parseUUID(storedUser.ID)
	if err != nil {
		return UserRecord{}, err
	}

	err = queries.UpdateUserByID(ctx, dbsqlc.UpdateUserByIDParams{
		Name:     user.Name,
		Email:    user.Email,
		ImageUrl: user.ImageURL,
		ID:       storedUserID,
	})
	if isUniqueViolation(err) {
		return UserRecord{}, ErrUserConflict
	}
	if err != nil {
		return UserRecord{}, err
	}

	updatedUser, err := queries.GetUserByID(ctx, storedUserID)
	if err != nil {
		return UserRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return UserRecord{}, err
	}

	return userRecordFromRow(updatedUser.ID, updatedUser.Name, updatedUser.Email, updatedUser.ImageUrl), nil
}

func resolveAccountUser(ctx context.Context, queries *dbsqlc.Queries, user UserRecord) (StoredUserRecord, error) {
	storedUser, err := queries.GetUserByAccount(ctx, dbsqlc.GetUserByAccountParams{
		Provider:          user.Provider,
		ProviderAccountID: user.ProviderAccountID,
	})
	if err == nil {
		return storedUserRecordFromRow(storedUser.ID, storedUser.Name, storedUser.Email, storedUser.ImageUrl), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return StoredUserRecord{}, err
	}

	if user.Email != "" {
		emailUser, emailErr := queries.GetUserByEmail(ctx, user.Email)
		if emailErr == nil {
			storedUserID, parseErr := parseUUID(emailUser.ID)
			if parseErr != nil {
				return StoredUserRecord{}, parseErr
			}

			if err := queries.CreateAccount(ctx, dbsqlc.CreateAccountParams{
				UserID:            storedUserID,
				Provider:          user.Provider,
				ProviderAccountID: user.ProviderAccountID,
			}); err != nil && !isUniqueViolation(err) {
				return StoredUserRecord{}, err
			}

			linkedUser, linkErr := queries.GetUserByAccount(ctx, dbsqlc.GetUserByAccountParams{
				Provider:          user.Provider,
				ProviderAccountID: user.ProviderAccountID,
			})
			if linkErr == nil {
				return storedUserRecordFromRow(linkedUser.ID, linkedUser.Name, linkedUser.Email, linkedUser.ImageUrl), nil
			}
			if !errors.Is(linkErr, pgx.ErrNoRows) {
				return StoredUserRecord{}, linkErr
			}

			return storedUserRecordFromRow(emailUser.ID, emailUser.Name, emailUser.Email, emailUser.ImageUrl), nil
		}
		if !errors.Is(emailErr, pgx.ErrNoRows) {
			return StoredUserRecord{}, emailErr
		}
	}

	createdUser, err := queries.CreateUser(ctx, dbsqlc.CreateUserParams{
		Name:     user.Name,
		Email:    user.Email,
		ImageUrl: user.ImageURL,
	})
	var createdOrExistingUser StoredUserRecord
	if isUniqueViolation(err) && user.Email != "" {
		emailUser, lookupErr := queries.GetUserByEmail(ctx, user.Email)
		err = lookupErr
		if lookupErr == nil {
			createdOrExistingUser = storedUserRecordFromRow(emailUser.ID, emailUser.Name, emailUser.Email, emailUser.ImageUrl)
		}
	} else if err == nil {
		createdOrExistingUser = storedUserRecordFromRow(createdUser.ID, createdUser.Name, createdUser.Email, createdUser.ImageUrl)
	}
	if err != nil {
		return StoredUserRecord{}, err
	}

	storedUserID, err := parseUUID(createdOrExistingUser.ID)
	if err != nil {
		return StoredUserRecord{}, err
	}

	if err := queries.CreateAccount(ctx, dbsqlc.CreateAccountParams{
		UserID:            storedUserID,
		Provider:          user.Provider,
		ProviderAccountID: user.ProviderAccountID,
	}); err != nil && !isUniqueViolation(err) {
		return StoredUserRecord{}, err
	}

	linkedUser, err := queries.GetUserByAccount(ctx, dbsqlc.GetUserByAccountParams{
		Provider:          user.Provider,
		ProviderAccountID: user.ProviderAccountID,
	})
	if err == nil {
		return storedUserRecordFromRow(linkedUser.ID, linkedUser.Name, linkedUser.Email, linkedUser.ImageUrl), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return StoredUserRecord{}, err
	}

	return createdOrExistingUser, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(strings.TrimSpace(value)); err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid user id: %w", err)
	}
	return uuid, nil
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
		return SessionUserRecord{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}

	var record SessionUserRecord
	err = s.pool.QueryRow(ctx, `
		SELECT u.id::text, u.name, u.email, u.image_url, s.expires_at
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
		if errors.Is(err, pgx.ErrNoRows) {
			if _, deleteErr := s.pool.Exec(ctx, `
				DELETE FROM sessions
				WHERE token_hash = $1
				  AND expires_at <= $2
			`, tokenHash, now.UTC()); deleteErr != nil {
				return SessionUserRecord{}, deleteErr
			}
			return SessionUserRecord{}, ErrSessionNotFound
		}
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

func (s *UserStore) GetUserByID(ctx context.Context, userID string) (StoredUserRecord, error) {
	if s == nil || s.queries == nil {
		return StoredUserRecord{}, errors.New("user store is not configured")
	}

	uuidID, err := parseUUID(userID)
	if err != nil {
		return StoredUserRecord{}, err
	}

	user, err := s.queries.GetUserByID(ctx, uuidID)
	if err != nil {
		return StoredUserRecord{}, err
	}

	return storedUserRecordFromRow(user.ID, user.Name, user.Email, user.ImageUrl), nil
}

func (s *UserStore) Close() {
	if s == nil || s.pool == nil {
		return
	}

	s.pool.Close()
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == postgresUniqueViolation
}

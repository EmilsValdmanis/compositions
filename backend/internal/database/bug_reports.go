package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbsqlc "github.com/EmilsValdmanis/compositions/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrGameBugReportNotFound = errors.New("game bug report not found")

const GameBugReportDescriptionMaxLength = 500

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

func (s *UserStore) ListGameBugReportsPage(ctx context.Context, limit, offset int) ([]GameBugReportRecord, error) {
	if s == nil || s.queries == nil {
		return nil, errors.New("user store is not configured")
	}
	if limit <= 0 || limit > 100 {
		return nil, errors.New("bug report page size must be between 1 and 100")
	}
	if offset < 0 {
		return nil, errors.New("bug report offset must not be negative")
	}
	rows, err := s.queries.ListGameBugReportsPage(ctx, dbsqlc.ListGameBugReportsPageParams{
		ResultLimit:  int32(limit),
		ResultOffset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	reports := make([]GameBugReportRecord, 0, len(rows))
	for _, row := range rows {
		reports = append(reports, GameBugReportRecord{
			ID: row.ID, RoomCode: row.RoomCode, ReporterPlayerID: row.ReporterPlayerID,
			Description: row.Description, Round: int(row.Round), Turn: int(row.Turn),
			RequestedAbort: row.RequestedAbort, CreatedAt: row.CreatedAt.Time.UTC(),
		})
	}
	return reports, nil
}

func (s *UserStore) CountGameBugReports(ctx context.Context) (int64, error) {
	if s == nil || s.queries == nil {
		return 0, errors.New("user store is not configured")
	}
	return s.queries.CountGameBugReports(ctx)
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
	if errors.Is(err, pgx.ErrNoRows) {
		return GameBugReportRecord{}, ErrGameBugReportNotFound
	}
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

func (s *UserStore) CompleteGameBugReport(ctx context.Context, reportID string) error {
	if s == nil || s.queries == nil {
		return errors.New("user store is not configured")
	}
	var id pgtype.UUID
	if err := id.Scan(strings.TrimSpace(reportID)); err != nil {
		return fmt.Errorf("invalid bug report id: %w", err)
	}
	if _, err := s.queries.CompleteGameBugReport(ctx, id); errors.Is(err, pgx.ErrNoRows) {
		return ErrGameBugReportNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func normalizeGameBugReport(report GameBugReportRecord) (GameBugReportRecord, error) {
	report.ID = strings.TrimSpace(report.ID)
	report.RoomCode = strings.ToUpper(strings.TrimSpace(report.RoomCode))
	report.ReporterPlayerID = strings.TrimSpace(report.ReporterPlayerID)
	report.ReporterUserID = strings.TrimSpace(report.ReporterUserID)
	report.Description = strings.TrimSpace(report.Description)
	if report.ID == "" {
		return GameBugReportRecord{}, errors.New("bug report id is required")
	}
	if report.RoomCode == "" {
		return GameBugReportRecord{}, errors.New("room code is required")
	}
	if report.ReporterPlayerID == "" {
		return GameBugReportRecord{}, errors.New("reporter player id is required")
	}
	if report.Description == "" {
		return GameBugReportRecord{}, errors.New("bug report description is required")
	}
	if len([]rune(report.Description)) > GameBugReportDescriptionMaxLength {
		return GameBugReportRecord{}, fmt.Errorf("bug report description must be %d characters or fewer", GameBugReportDescriptionMaxLength)
	}
	if len(report.GameState) == 0 || !json.Valid(report.GameState) {
		return GameBugReportRecord{}, errors.New("bug report game state must be valid json")
	}
	if report.Round <= 0 || report.Turn <= 0 {
		return GameBugReportRecord{}, errors.New("bug report round and turn must be positive")
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	} else {
		report.CreatedAt = report.CreatedAt.UTC()
	}
	report.GameState = append(json.RawMessage(nil), report.GameState...)
	return report, nil
}

func gameBugReportFromRow(
	id, roomCode, reporterPlayerID string,
	reporterUserID pgtype.UUID,
	description string,
	gameState []byte,
	round, turn int32,
	requestedAbort bool,
	createdAt pgtype.Timestamptz,
) GameBugReportRecord {
	return GameBugReportRecord{
		ID: id, RoomCode: roomCode, ReporterPlayerID: reporterPlayerID,
		ReporterUserID: nullableUUIDString(reporterUserID), Description: description,
		GameState: append(json.RawMessage(nil), gameState...), Round: int(round), Turn: int(turn),
		RequestedAbort: requestedAbort, CreatedAt: createdAt.Time.UTC(),
	}
}

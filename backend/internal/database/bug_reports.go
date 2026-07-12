package database

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
	if len([]rune(report.Description)) > 500 {
		return GameBugReportRecord{}, errors.New("bug report description must be 500 characters or fewer")
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

-- name: CreateGameBugReport :one
INSERT INTO game_bug_reports (
    id,
    room_code,
    reporter_player_id,
    reporter_user_id,
    description,
    game_state,
    round,
    turn,
    requested_abort,
    created_at
)
VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(room_code),
    sqlc.arg(reporter_player_id),
    sqlc.narg(reporter_user_id)::uuid,
    sqlc.arg(description),
    sqlc.arg(game_state),
    sqlc.arg(round),
    sqlc.arg(turn),
    sqlc.arg(requested_abort),
    sqlc.arg(created_at)
)
RETURNING
    id::text AS id,
    room_code,
    reporter_player_id,
    reporter_user_id,
    description,
    game_state,
    round,
    turn,
    requested_abort,
    created_at;

-- name: ListGameBugReports :many
SELECT
    id::text AS id,
    room_code,
    reporter_player_id,
    reporter_user_id,
    description,
    game_state,
    round,
    turn,
    requested_abort,
    created_at
FROM game_bug_reports
ORDER BY created_at DESC
LIMIT sqlc.arg(result_limit);

-- name: ListGameBugReportsPage :many
SELECT
    id::text AS id,
    room_code,
    reporter_player_id,
    description,
    round,
    turn,
    requested_abort,
    created_at
FROM game_bug_reports
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(result_limit)
OFFSET sqlc.arg(result_offset);

-- name: CountGameBugReports :one
SELECT COUNT(*) FROM game_bug_reports;

-- name: GetGameBugReport :one
SELECT
    id::text AS id,
    room_code,
    reporter_player_id,
    reporter_user_id,
    description,
    game_state,
    round,
    turn,
    requested_abort,
    created_at
FROM game_bug_reports
WHERE id = sqlc.arg(id)::uuid;

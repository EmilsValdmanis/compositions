# Backend

This directory contains the Go backend for Compositions.

## What It Contains

- `cmd/server/`: the websocket server entrypoint
- `internal/game/`: core game rules, card logic, state handling, and tests
- `dist/`: build output for the server binary

## Development

Run commands from inside `backend/`.

Google OAuth is handled entirely by the Go backend. The frontend is just UI and talks to the backend directly.

Required auth configuration:

- `BASE_URL`: backend public URL, for example `http://localhost:8080`
- `FRONTEND_URL`: frontend URL allowed for CORS, for example `http://localhost:3000`
- `GOOGLE_CLIENT_ID`: Google OAuth client id
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret
- `COOKIE_SECURE`: set to `true` when serving over HTTPS

Admin access is controlled solely by `users.is_admin` in every environment. Promote a user directly
in Postgres:

```sql
UPDATE users
SET is_admin = TRUE
WHERE email = 'admin@example.com';
```

Authenticated websocket users are persisted to Postgres on connect. The backend uses:

- `pgx/v5` for the connection pool and driver
- `sqlc` to generate typed query code from SQL files
- `golang-migrate` to apply SQL migrations from `internal/database/migrations/`

Optional Sentry configuration:

- `SENTRY_DSN`: enables Sentry when set outside development
- `SENTRY_ENVIRONMENT`: deployment environment such as `development`, `staging`, or `production`

The backend initializes Sentry for error monitoring, request tracing, and structured log forwarding. It is disabled automatically in `development`, health checks are excluded from tracing, and production traffic is sampled more conservatively.

| Command      | What it does                              |
| :----------- | :---------------------------------------- |
| `make run`   | Starts the server on `:8080`              |
| `make build` | Builds the server binary to `dist/server` |
| `make test`  | Runs `go vet` and the test suite          |
| `make test-integration` | Runs Postgres integration tests with Testcontainers |
| `make bench` | Runs benchmarks for game logic            |
| `make db-up` | Starts local Postgres with Docker Compose |
| `make db-down` | Stops local Postgres                    |
| `make migrate-up` | Applies all pending database migrations |
| `make migrate-down` | Rolls back the latest migration     |
| `make sqlc` | Regenerates typed query code from SQL     |

## Local Postgres Setup

1. Copy `.env.sample` to `.env` if you have not already.
2. Start Postgres with `make db-up`.
3. Apply the schema with `make migrate-up`.
4. Start the backend with `make run`.

The default local connection string is:

`postgres://postgres:postgres@localhost:5432/compositions?sslmode=disable`

## How `pgx` + `sqlc` Fit Together

- SQL lives in `internal/database/queries/*.sql`.
- `sqlc` reads those queries plus `internal/database/schema.sql` and generates Go code into `internal/database/sqlc/`.
- `internal/database/store.go` wraps the generated code in a small `UserStore` used by the websocket server.

If you change the schema or queries:

1. Update the migration in `internal/database/migrations/`.
2. Update `internal/database/schema.sql` so `sqlc` sees the latest schema.
3. Run `make sqlc`.
4. Run `make test` and `make test-integration`.

`make migrate-up` and `make migrate-down` use `golang-migrate` under the hood via `cmd/migrate`.

## Testing

- `make test` runs the fast unit test suite.
- `make test-integration` starts a real Postgres container with Testcontainers and verifies migrations plus user persistence.
- The PR validation workflow runs both.

To run the integration tests locally you need Docker running.

## Server Notes

- the server currently listens on port `8080`
- websocket connections are handled at `/ws`
- the module root for the Go project is this directory

## Reviewing Bug Reports

Game problem reports are stored independently in `game_bug_reports`, even after the original room
is deleted. To review the newest reports:

```sql
SELECT
    id,
    created_at,
    room_code,
    reporter_player_id,
    description,
    round,
    turn,
    requested_abort,
    game_state
FROM game_bug_reports
ORDER BY created_at DESC;
```

`game_state` is the authoritative server persistence snapshot captured when the player submitted
the report.

## Player Statistics

Completed-game statistics, lifetime aggregates, derived metrics, badge ideas, and extension guidance
are documented in [STATISTICS.md](STATISTICS.md).

## Purpose

The backend is responsible for enforcing the game rules and managing multiplayer session state. The intent is to keep the rule logic testable and separate from frontend concerns.

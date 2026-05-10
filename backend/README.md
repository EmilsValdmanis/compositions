# Backend

This directory contains the Go backend for Compositions.

## What It Contains

- `cmd/server/`: the websocket server entrypoint
- `internal/game/`: core game rules, card logic, state handling, and tests
- `dist/`: build output for the server binary

## Development

Run commands from inside `backend/`.

The websocket server now requires `BETTER_AUTH_URL` to point at the frontend app's Better Auth base URL, for example `http://localhost:3000` in local development.

| Command      | What it does                              |
| :----------- | :---------------------------------------- |
| `make run`   | Starts the server on `:8080`              |
| `make build` | Builds the server binary to `dist/server` |
| `make test`  | Runs `go vet` and the test suite          |
| `make bench` | Runs benchmarks for game logic            |

## Server Notes

- the server currently listens on port `8080`
- websocket connections are handled at `/ws`
- the module root for the Go project is this directory

## Purpose

The backend is responsible for enforcing the game rules and managing multiplayer session state. The intent is to keep the rule logic testable and separate from frontend concerns.

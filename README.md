# Compositions

Compositions is a project for bringing a family card game online.

The repository is split into separate frontend and backend applications so the game rules, server behavior, and player experience can evolve independently without losing the shape of a single project.

## Project Goal

The main goal is to turn the rules of Compositions into a playable online experience that still feels faithful to the real game.

That includes:

- encoding the game rules clearly and correctly
- supporting room creation and multiplayer play
- building a frontend that makes the game easy to understand and join
- keeping the codebase small and maintainable while the product is still taking shape

## The Game

Compositions is a multi-round card game built around:

- forming valid sets and runs
- hitting the initial 40-point requirement
- managing jokers and high/low aces carefully
- getting rid of your hand before everyone else

The full rule set lives in [RULES.md](RULES.md).

## Repository Layout

- `frontend/`: the web app built with React, TanStack Start, and Vite+
- `backend/`: the Go server and game logic module, including Postgres persistence via `pgx` + `sqlc`
- `RULES.md`: the current written rules for the game

## Current Status

The project is still in active foundation-building.

- the backend already contains core game logic and a websocket server
- the frontend has been scaffolded and is ready to grow into the playable client
- the repo is organized so each side can be worked on independently

## Working In This Repo

- frontend-specific commands and dependencies live in `frontend/`
- backend-specific commands and Go tooling live in `backend/`
- shared project context and rules live at the repository root

See `backend/README.md` for backend details.

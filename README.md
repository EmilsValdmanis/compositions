# Compositions

Compositions is a project for bringing a family card game online.

The repository is split into separate frontend and backend applications so the game rules, server behavior, and player experience can evolve independently without losing the shape of a single project.

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

The project has the base of the game complete. Next steps include:

- need to massively improve the pre-game and post-game UI 
- improving in-game UI/UX as well as animations and sound effects (emotes, maybe chat etc.)
- building out statistics and friend groups (maybe basic elo system) now that we have a database

## Working In This Repo

- frontend-specific commands and dependencies live in `frontend/`
- backend-specific commands and Go tooling live in `backend/`
- shared project context and rules live at the repository root

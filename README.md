<p align="center">
  <img src="frontend/public/favicon.svg" alt="Compositions joker logo" width="96" height="96">
</p>

<h1 align="center">Compositions</h1>

<p align="center">
  A family card game brought online, with real-time rooms, server-backed game state, and careful joker handling.
</p>

<p align="center">
  <a href="RULES.md">Game Rules</a>
  ·
  <a href="frontend/README.md">Frontend</a>
  ·
  <a href="backend/README.md">Backend</a>
</p>

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

The base game is implemented and playable. Current focus areas:

- improving the pre-game and post-game experience
- refining in-game interactions, animations, and sound
- adding social/statistics features now that persistence is in place

## Working In This Repo

Frontend-specific commands and dependencies live in `frontend/`. Backend commands, database migrations, and Go tooling live in `backend/`. Shared project context and rules live at the repository root.

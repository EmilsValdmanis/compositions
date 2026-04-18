# Compositions

A small Go project for bringing a family card game online.

The goal of this repo is simple: capture the rules of Compositions in code and build a playable experience so it is easier to get a game going, even when everyone is not in the same room.

## What It Is

Compositions is a multi-round card game built around:

- forming valid sets and runs
- hitting the initial 40-point requirement
- managing jokers and high/low aces carefully
- getting rid of your hand before everyone else

The full rule set lives in [RULES.md](RULES.md).

## Project Status

This project is currently focused on the game logic and backend foundation. The intent is to keep the implementation clean and rule-driven first, then grow it into a smooth online version of the game.

## Running It

The Makefile includes the main project commands:

| Command | What it does |
| :-- | :-- |
| `make run` | Starts the backend server |
| `make build` | Builds the server binary to `backend/dist/server` |
| `make test` | Runs `go vet` and the test suite |


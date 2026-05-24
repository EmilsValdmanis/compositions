.PHONY: run build test test-integration bench sqlc migrate-up migrate-down db-up db-down db-logs

ifneq (,$(wildcard .env))
include .env
export
endif

BENCH ?= .
BENCHMEM ?= 1

run:
	go run ./cmd/server

build:
	go build -o dist/server ./cmd/server

test:
	go vet ./... && go test -cover ./...

test-integration:
	go test -tags=integration ./internal/database/...

bench:
	go test -run '^$$' -bench '$(BENCH)' $(if $(filter 1,$(BENCHMEM)),-benchmem,) ./...

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

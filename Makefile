.PHONY: run build test bench

BENCH ?= .
BENCHMEM ?= 1

run:
	cd backend && go run ./cmd/server

build:
	cd backend && go build -o dist/server ./cmd/server

test:
	cd backend && go vet ./... && go test -cover ./...

bench:
	cd backend && go test -run '^$$' -bench '$(BENCH)' $(if $(filter 1,$(BENCHMEM)),-benchmem,) ./...

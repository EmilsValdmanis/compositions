.PHONY: run build test bench

BENCH ?= .
BENCHMEM ?= 1

run:
	go run ./cmd/server

build:
	go build -o dist/server ./cmd/server

test:
	go vet ./... && go test -cover ./...

bench:
	go test -run '^$$' -bench '$(BENCH)' $(if $(filter 1,$(BENCHMEM)),-benchmem,) ./...

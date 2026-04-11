.PHONY: run build test

run:
	cd backend && go run ./cmd/server

build:
	cd backend && go build -o dist/server ./cmd/server

test:
	cd backend && go vet ./... && go test ./...


.PHONY: dev test test-cover build lint

dev:
	go run ./cmd/api/...

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

build:
	go build -o bin/server ./cmd/api/...

lint:
	golangci-lint run ./...

# migrate-up/migrate-down live in lms-membership-db now — this repo no longer owns the schema
# (ADR-006-repo-per-context-decomposition.md)

.PHONY: run build clean test test-integration test-all lint fmt generate \
        migrate-up migrate-down migrate-create migrate-status \
        docker-up docker-down seed install-tools

# ============================================================================
# Development
# ============================================================================

run:
	go run ./cmd/server/

build:
	go build -ldflags="-s -w" -o bin/server ./cmd/server/

clean:
	rm -rf bin/

# ============================================================================
# Testing
# ============================================================================

test:
	go test ./internal/... -v -race

test-integration:
	go test ./tests/integration/... -v -race -tags=integration

test-all: test test-integration

# ============================================================================
# Code Quality
# ============================================================================

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w .

# ============================================================================
# Code Generation
# ============================================================================

generate:
	cd db && sqlc generate

# ============================================================================
# Database
# ============================================================================

migrate-up:
	goose -dir db/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DATABASE_URL)" down

migrate-create:
	@if [ -z "$(name)" ]; then echo "Error: name is required. Usage: make migrate-create name=migration_name"; exit 1; fi
	goose -dir db/migrations create $(name) sql

migrate-status:
	goose -dir db/migrations postgres "$(DATABASE_URL)" status

# ============================================================================
# Docker
# ============================================================================

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

# ============================================================================
# Utilities
# ============================================================================

seed:
	@echo "Running seed script..."
	@bash scripts/seed.sh

install-tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

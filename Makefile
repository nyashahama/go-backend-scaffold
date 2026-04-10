.PHONY: run build clean test test-integration test-all test-ci smoke security-check \
        bootstrap-smoke ready-for-adopters lint fmt generate migrate-up migrate-down \
        migrate-create migrate-status docker-up docker-down seed install-tools

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
	DATABASE_URL="$${DATABASE_URL:-postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable}" \
	REDIS_URL="$${REDIS_URL:-redis://localhost:6379}" \
	go test ./tests/integration/... -v -race -tags=integration

test-ci:
	go test ./... -race
	DATABASE_URL="$${DATABASE_URL:-postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable}" \
	REDIS_URL="$${REDIS_URL:-redis://localhost:6379}" \
	go test ./tests/integration/... -v -race -tags=integration

smoke:
	go test ./cmd/server ./internal/server ./internal/auth -v

security-check:
	GOTOOLCHAIN=go1.25.9 govulncheck -scan package ./...

bootstrap-smoke:
	bash scripts/ci/bootstrap-smoke.sh

ready-for-adopters:
	$(MAKE) lint
	$(MAKE) bootstrap-smoke
	docker build -t go-backend-scaffold:ready .

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
	go install golang.org/x/vuln/cmd/govulncheck@latest

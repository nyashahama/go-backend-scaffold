.PHONY: run build clean test test-integration test-all test-ci smoke \
        bootstrap-smoke ready-for-adopters lint fmt vuln generate migrate-up migrate-down \
        migrate-create migrate-status docker-up docker-down docker-build image-scan seed install-tools

IMAGE_NAME ?= go-backend-scaffold
IMAGE_TAG ?= local
IMAGE ?= $(IMAGE_NAME):$(IMAGE_TAG)
TRIVY_IMAGE ?= aquasec/trivy:latest

ifneq (,$(wildcard .env))
include .env
export
endif

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

bootstrap-smoke:
	bash scripts/ci/bootstrap-smoke.sh

ready-for-adopters:
	$(MAKE) lint
	$(MAKE) vuln
	$(MAKE) bootstrap-smoke
	$(MAKE) docker-build IMAGE_TAG=ready
	$(MAKE) image-scan IMAGE_TAG=ready

test-all: test test-integration

# ============================================================================
# Code Quality
# ============================================================================

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w .

vuln:
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "govulncheck not found; install it with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	fi
	govulncheck ./...

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

docker-build:
	docker build -t $(IMAGE) .

image-scan:
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(TRIVY_IMAGE) \
		image --exit-code 1 --ignore-unfixed --severity HIGH,CRITICAL $(IMAGE)

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

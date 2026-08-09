.PHONY: help dev infra stop migrate migrate-down migrate-create seed test lint build

# ─── Variables ────────────────────────────────────────────────────────────────
DATABASE_URL   := $(or $(DATABASE_URL),postgres://starter:secret@localhost:5432/starter_kit?sslmode=disable)
MIGRATION_NAME ?= new_migration
MIGRATE        := docker run --rm --network host -v $(PWD)/backend/migrations:/migrations \
                  migrate/migrate -path=/migrations -database "$(DATABASE_URL)"

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ─── Development ──────────────────────────────────────────────────────────────
infra: ## Start infrastructure (Postgres + Redis) for local dev
	docker compose -f docker-compose.dev.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo "Infrastructure ready."

dev: infra ## Start backend + frontend with hot-reload
	@echo "Starting backend (hot-reload via 'go run')..."
	@cd backend && go run ./cmd/api & \
	go run ./cmd/worker & \
	echo "Starting frontend (vite dev)..." && \
	cd frontend && npm run dev

stop: ## Stop all dev infrastructure
	docker compose -f docker-compose.dev.yml down

# ─── Database Migrations ──────────────────────────────────────────────────────
migrate: ## Run all pending migrations (up)
	$(MIGRATE) up

migrate-down: ## Roll back the last migration
	$(MIGRATE) down 1

migrate-reset: ## Roll back ALL migrations (destructive!)
	$(MIGRATE) down -all

migrate-create: ## Create a new migration: make migrate-create MIGRATION_NAME=name
	$(MIGRATE) create -ext sql -dir /migrations -seq $(MIGRATION_NAME)

migrate-status: ## Show migration status
	$(MIGRATE) version

seed: ## Seed development data (roles & permissions already in migration 000012)
	@echo "Seed data is applied via migration 000012. Run 'make migrate' first."

# ─── Testing ──────────────────────────────────────────────────────────────────
test: ## Run all backend tests
	@cd backend && go test ./... -v -count=1

test-cover: ## Run tests with coverage report
	@cd backend && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

test-fe: ## Run frontend tests
	@cd frontend && npm run test

# ─── Linting ──────────────────────────────────────────────────────────────────
lint: ## Lint backend (golangci-lint) and frontend (eslint)
	@cd backend && golangci-lint run ./...
	@cd frontend && npm run lint

fmt: ## Format backend code
	@cd backend && gofmt -w .
	@cd frontend && npm run format

# ─── Build ────────────────────────────────────────────────────────────────────
build: ## Build production Docker images
	docker compose build

build-be: ## Build backend binary locally
	@cd backend && go build -o bin/api ./cmd/api && go build -o bin/worker ./cmd/worker
	@echo "Built: backend/bin/api, backend/bin/worker"

build-fe: ## Build frontend for production
	@cd frontend && npm run build

# ─── Setup ────────────────────────────────────────────────────────────────────
setup: ## First-time project setup
	@echo "Copying backend .env.example → backend/.env"
	@cp -n backend/.env.example backend/.env || true
	@echo "Copying frontend .env.example → frontend/.env"
	@cp -n frontend/.env.example frontend/.env || true
	@echo "Installing frontend deps..."
	@cd frontend && npm install
	@echo "Downloading Go modules..."
	@cd backend && go mod download
	@echo ""
	@echo "Done! Next steps:"
	@echo "  1. Edit backend/.env with your secrets"
	@echo "  2. Run: make infra"
	@echo "  3. Run: make migrate"
	@echo "  4. Run: make dev"

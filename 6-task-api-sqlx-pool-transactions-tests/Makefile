.PHONY: help build run test test-integration docker-up docker-down migrate migrate-test migrate-down

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Build ─────────────────────────────────────────────────────────────────────
build: ## Compile the server binary → bin/taskapi
	go build -o bin/taskapi ./cmd/server

run: ## Run the server locally (Postgres must be up)
	go run ./cmd/server

# ── Tests ─────────────────────────────────────────────────────────────────────
test: ## Vet all packages (no DB required)
	go vet ./...

test-integration: ## Run all integration tests against postgres_test on :5433
	TEST_DB_HOST=localhost \
	TEST_DB_PORT=5433 \
	TEST_DB_USER=postgres \
	TEST_DB_PASSWORD=postgres \
	TEST_DB_NAME=taskdb_test \
	go test ./tests/integration/... -v -count=1 -timeout 60s

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up: ## Start api + postgres + postgres_test
	docker compose up -d --build

docker-down: ## Stop and remove all containers and volumes
	docker compose down -v

# ── Migrations (no local psql required — pipes via stdin) ─────────────────────
migrate: ## Apply schema to primary DB (taskdb)
	cat migrations/001_create_tasks.up.sql | \
	  docker compose exec -T postgres psql -U postgres -d taskdb

migrate-test: ## Apply schema to test DB (taskdb_test)
	cat migrations/001_create_tasks.up.sql | \
	  docker compose exec -T postgres_test psql -U postgres -d taskdb_test

migrate-down: ## Roll back schema on primary DB
	cat migrations/001_create_tasks.down.sql | \
	  docker compose exec -T postgres psql -U postgres -d taskdb

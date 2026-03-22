.PHONY: help up down build rebuild logs ps shell-db clean test lint image-size

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
	  | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

# ── Compose ───────────────────────────────────────────────────────────────────

up: ## Build images and start all services in the background
	docker compose up --build -d

down: ## Stop and remove containers (volumes are kept)
	docker compose down

rebuild: ## Force-rebuild the api image without cache, then restart
	docker compose build --no-cache api
	docker compose up -d

logs: ## Tail logs from all services (Ctrl-C to stop)
	docker compose logs -f

ps: ## Show status of all containers
	docker compose ps

# ── Dev helpers ───────────────────────────────────────────────────────────────

shell-db: ## Open psql inside the running postgres container
	docker compose exec postgres \
	  psql -U "$${POSTGRES_USER:-appuser}" -d "$${POSTGRES_DB:-appdb}"

# ── Quality ───────────────────────────────────────────────────────────────────

test: ## Run Go unit tests
	go test ./... -v -race -count=1

lint: ## Run go vet (install golangci-lint for richer checks)
	go vet ./...

# ── Image size ────────────────────────────────────────────────────────────────

image-size: ## Print the uncompressed size of the api image
	@IMAGE=$$(docker compose config --images 2>/dev/null | grep api || echo "go-docker-api-api"); \
	 docker image inspect $$IMAGE --format='{{.Size}}' 2>/dev/null \
	   | awk '{printf "api image size: %.1f MB\n", $$1/1024/1024}' \
	 || echo "run 'make up' first to build the image"

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean: ## Remove containers, local images AND volumes (destructive!)
	docker compose down --volumes --rmi local

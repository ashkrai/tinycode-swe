BINARY     := blog-api
BUILD_DIR  := ./bin
CMD        := ./cmd/api
MIGRATIONS := ./migrations

DB_HOST     ?= localhost
DB_PORT     ?= 5432
DB_USER     ?= postgres
DB_PASSWORD ?= postgres
DB_NAME     ?= blog
DB_SSLMODE  ?= disable

DSN := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

DB_ENV := DB_HOST=$(DB_HOST) DB_PORT=$(DB_PORT) DB_USER=$(DB_USER) \
          DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) DB_SSLMODE=$(DB_SSLMODE) \
          MIGRATIONS_DIR=$(MIGRATIONS)

.PHONY: build run migrate rollback version explain seed seed-local \
        docker-up docker-down test clean help

## build: compile the binary into ./bin/
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)
	@echo "built $(BUILD_DIR)/$(BINARY)"

## run: start the API server locally (after make migrate)
run: build
	$(DB_ENV) $(BUILD_DIR)/$(BINARY) -addr :8080

## migrate: apply all pending up-migrations
migrate: build
	$(DB_ENV) $(BUILD_DIR)/$(BINARY) -migrate

## rollback: roll back the last migration (one step down)
rollback: build
	@echo "Rolling back ONE migration step..."
	$(DB_ENV) $(BUILD_DIR)/$(BINARY) -rollback

## version: print current schema version
version: build
	$(DB_ENV) $(BUILD_DIR)/$(BINARY) -version

## explain: run EXPLAIN ANALYZE on the list-posts query
explain: build
	$(DB_ENV) $(BUILD_DIR)/$(BINARY) -explain

## seed: insert sample data via psql inside the postgres container (no local psql needed)
seed:
	@echo "Waiting for postgres to be ready..."
	@until docker compose exec -T postgres pg_isready -U $(DB_USER) -d $(DB_NAME) > /dev/null 2>&1; do \
	    sleep 1; \
	done
	docker exec -i $$(docker compose ps -q postgres) \
	    psql -U $(DB_USER) -d $(DB_NAME) < scripts/seed.sql
	@echo "Seed data inserted"

## seed-local: insert sample data using a locally installed psql
seed-local:
	PGPASSWORD=$(DB_PASSWORD) psql \
	  -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) \
	  -f scripts/seed.sql
	@echo "Seed data inserted"

## migrate-cli: apply migrations using the standalone golang-migrate CLI
## Install: go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
migrate-cli:
	migrate -path $(MIGRATIONS) -database "$(DSN)" up

## rollback-cli: rollback 1 step using the standalone golang-migrate CLI
rollback-cli:
	migrate -path $(MIGRATIONS) -database "$(DSN)" down 1

## docker-up: build and start postgres + api via docker compose
##            The api container auto-runs migrations then starts the server.
docker-up:
	docker compose up --build -d
	@echo ""
	@echo "Stack is starting..."
	@echo "Waiting for API to be healthy at http://localhost:8080/health"
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do \
	    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then \
	        echo "API is up -> http://localhost:8080"; \
	        break; \
	    fi; \
	    echo "  still waiting ($$i/20)..."; \
	    sleep 3; \
	done

## docker-down: stop and remove containers
docker-down:
	docker compose down

## test: run all tests
test:
	go test ./... -v -race -timeout 60s

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/## //'

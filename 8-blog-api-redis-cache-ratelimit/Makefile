.PHONY: build run test test-integration migrate-up migrate-down lint docker-up docker-down

# ── Local build ──────────────────────────────────────────────────────────────
build:
	go build -o bin/api ./cmd/api

run: build
	DATABASE_URL=postgres://blog:blog@localhost:5432/blog?sslmode=disable \
	REDIS_URL=redis://localhost:6379/0 \
	./bin/api

# ── Unit tests (no DB/Redis required) ────────────────────────────────────────
test:
	go test ./... -run "^Test[^I]" -v -count=1

# ── Integration tests (requires docker-compose.test.yml to be up) ────────────
test-integration:
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for services..."
	@sleep 3
	TEST_DATABASE_URL=postgres://blog:blog@localhost:5433/blog_test?sslmode=disable \
	TEST_REDIS_URL=redis://localhost:6380/1 \
	go test ./tests/... -run "TestIntegration" -v -count=1
	docker-compose -f docker-compose.test.yml down -v

# ── Migrations (manual) ───────────────────────────────────────────────────────
migrate-up:
	migrate -path ./migrations \
	        -database "$$DATABASE_URL" up

migrate-down:
	migrate -path ./migrations \
	        -database "$$DATABASE_URL" down 1

migrate-version:
	migrate -path ./migrations \
	        -database "$$DATABASE_URL" version

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

# ── Lint ─────────────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

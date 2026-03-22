#!/usr/bin/env bash
# setup.sh — run once after cloning / unzipping to prepare the project.
set -euo pipefail

echo "==> Downloading Go module dependencies..."
go mod tidy

echo "==> Verifying build..."
go build ./...

echo "==> Running unit tests..."
go test ./... -run "^Test[^I]" -count=1 -v

echo ""
echo "✅  Setup complete."
echo ""
echo "Next steps:"
echo "  docker-compose up --build        # start Postgres + Redis + API"
echo "  make test-integration            # run integration tests"

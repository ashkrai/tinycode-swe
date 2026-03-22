#!/bin/sh
set -e

echo ">>> Running database migrations..."
/app/blog-api -migrate

echo ">>> Starting API server..."
exec /app/blog-api -addr :8080

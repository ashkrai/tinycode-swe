#!/bin/bash
# Run this after starting the server: go run main.go
# It walks through every endpoint in order.

BASE="http://localhost:8080"

echo ""
echo "── CREATE two tasks ─────────────────────────────"
curl -s -X POST $BASE/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy milk"}' | jq .

curl -s -X POST $BASE/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Write code"}' | jq .

echo ""
echo "── LIST all tasks ───────────────────────────────"
curl -s $BASE/tasks | jq .

echo ""
echo "── GET task 1 ───────────────────────────────────"
curl -s $BASE/tasks/1 | jq .

echo ""
echo "── UPDATE task 1 ────────────────────────────────"
curl -s -X PUT $BASE/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Buy oat milk","done":true}' | jq .

echo ""
echo "── DELETE task 2 ────────────────────────────────"
curl -s -X DELETE $BASE/tasks/2
echo "(no response body — 204 means deleted)"

echo ""
echo "── LIST again — only task 1 should remain ───────"
curl -s $BASE/tasks | jq .

echo ""
echo "── GET a task that does not exist ───────────────"
curl -s $BASE/tasks/99 | jq .
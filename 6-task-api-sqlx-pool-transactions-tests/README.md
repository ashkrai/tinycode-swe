# Task API — Go · PostgreSQL · sqlx · chi

A production-ready REST API for task management.

| Concern | Choice |
|---|---|
| Language | Go 1.22 |
| HTTP router | chi v5 |
| SQL driver | lib/pq |
| Query helper | sqlx |
| Database | PostgreSQL 16 |

---

## Project structure

```
taskapi/
├── cmd/
│   └── server/
│       └── main.go                  ← entry point, graceful shutdown
├── internal/
│   ├── db/
│   │   └── db.go                    ← sqlx.Open + connection pool config
│   ├── model/
│   │   └── task.go                  ← domain types + request/response structs
│   ├── repository/
│   │   └── task.go                  ← all SQL queries + BulkDelete transaction
│   └── handler/
│       ├── handler.go               ← HTTP handlers (CRUD + healthz + bulk)
│       └── router.go                ← chi router + middleware
├── tests/
│   └── integration/
│       ├── helper_test.go           ← shared DB setup / teardown
│       ├── repository_test.go       ← 14 repository integration tests
│       └── handler_test.go          ← 12 end-to-end HTTP tests
├── migrations/
│   ├── 001_create_tasks.up.sql
│   └── 001_create_tasks.down.sql
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

---

## Quick start

```bash
# 1. Start both Postgres containers
docker compose up -d postgres postgres_test

# 2. Apply schema (no local psql needed)
make migrate
make migrate-test

# 3. Run the server  →  http://localhost:8080
make run

# 4. Run all integration tests
make test-integration
```

---

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /healthz | DB connectivity check |
| GET | /tasks | List all active tasks |
| POST | /tasks | Create a task |
| GET | /tasks/{id} | Get one task |
| PUT | /tasks/{id} | Partial update |
| DELETE | /tasks/{id} | Soft delete |
| DELETE | /tasks/bulk | Bulk soft delete (transactional) |

### ashkrai requests

```bash
# Create
curl -s -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy milk","status":"pending"}' | jq

# List
curl -s http://localhost:8080/tasks | jq

# Update
curl -s -X PUT http://localhost:8080/tasks/<id> \
  -H 'Content-Type: application/json' \
  -d '{"status":"done"}' | jq

# Bulk delete (runs inside a transaction)
curl -s -X DELETE http://localhost:8080/tasks/bulk \
  -H 'Content-Type: application/json' \
  -d '{"ids":["<id1>","<id2>"]}' | jq

# Health
curl -s http://localhost:8080/healthz | jq
```

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| SERVER_ADDR | :8080 | Listen address |
| DB_HOST | localhost | Postgres host |
| DB_PORT | 5432 | Postgres port |
| DB_USER | postgres | DB user |
| DB_PASSWORD | postgres | DB password |
| DB_NAME | taskdb | Database name |
| DB_SSLMODE | disable | SSL mode |
| DB_MAX_OPEN_CONNS | 25 | Max open connections |
| DB_MAX_IDLE_CONNS | 10 | Max idle connections |
| DB_CONN_MAX_LIFETIME | 5m | Max connection lifetime |

---

## Connection pool

```go
db.SetMaxOpenConns(25)          // cap total connections
db.SetMaxIdleConns(10)          // keep idle connections warm
db.SetConnMaxLifetime(5*time.Minute)  // recycle to avoid stale handles
db.SetConnMaxIdleTime(2*time.Minute)  // evict long-idle connections
```

## Bulk delete transaction flow

```
BEGIN (ReadCommitted isolation)
  SELECT … FOR UPDATE        ← lock target rows
  UPDATE tasks
    SET deleted_at = NOW()
    WHERE id IN (…)
COMMIT
                             ← deferred ROLLBACK fires on any error
```

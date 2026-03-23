# 8. blog-api

A production-style Go REST API demonstrating:

- **chi** router with logging, recovery, and rate-limiting middleware  
- **PostgreSQL** via `sqlx` (no ORM)  
- **Redis** caching (60 s TTL) + sliding-window rate limiter (100 req/min per IP)  
- **Cache-busting** on every write  
- **golang-migrate** for SQL migrations  
- Service / Repository / Handler layering  
- Table-driven unit tests with `net/http/httptest`  
- Integration tests using a dedicated test DB + Redis  
- Multi-stage Dockerfile → distroless image (< 20 MB)  
- `docker-compose.yml` with healthchecks and a named Postgres volume  

---

## Architecture

```
HTTP Request
     │
     ▼
  chi Router
     │
  Middleware stack
  ├── Recoverer        (panic → 500)
  ├── Logger           (method, path, status, latency)
  └── RateLimiter      (Redis INCR+EXPIRE, 100 req/min/IP → 429)
     │
     ▼
  PostHandler
  (JSON decode, validation)
     │
     ▼
  PostService  ◄──────────────────────────┐
  ├── cache.GetJSON  (Redis, 60 s TTL)    │ cache miss
  │        hit ──────────────────────────►│
  └── cache miss → PostRepository         │
                     └── sqlx → Postgres  │
                              └── result──┘
                                   └── cache.SetJSON
                                   └── return to handler
```

---

## Quick Start

```bash
# 1. Start Postgres + Redis + API
go mod tidy
docker-compose up --build

# 2. Check health
curl http://localhost:8080/healthz

# 3. Create a post
curl -X POST http://localhost:8080/posts \
  -H 'Content-Type: application/json' \
  -d '{"user_id":1,"title":"Hello Redis","body":"First cached post"}'

# 4. List posts (first call → DB + cache write)
curl http://localhost:8080/posts

# 5. List again (cache hit, no DB query)
curl http://localhost:8080/posts

# 6. Bulk delete
curl -X DELETE http://localhost:8080/posts/bulk \
  -H 'Content-Type: application/json' \
  -d '{"ids":[1,2]}'
```

---

## Environment Variables

| Variable       | Default                                      | Description              |
|----------------|----------------------------------------------|--------------------------|
| `DATABASE_URL` | *(required)*                                 | Postgres connection string |
| `REDIS_URL`    | `redis://localhost:6379/0`                   | Redis connection string  |
| `PORT`         | `8080`                                       | HTTP listen port         |

---

## Endpoints

| Method | Path              | Description                        |
|--------|-------------------|------------------------------------|
| GET    | `/healthz`        | DB + Redis liveness check          |
| GET    | `/posts`          | List all posts (cached 60 s)       |
| POST   | `/posts`          | Create post (busts list cache)     |
| GET    | `/posts/:id`      | Get post by ID (cached 60 s)       |
| PUT    | `/posts/:id`      | Update post (busts item + list)    |
| DELETE | `/posts/:id`      | Delete post (busts item + list)    |
| DELETE | `/posts/bulk`     | Bulk delete in a TX (busts all)    |

---

## Rate Limiting

Every IP is limited to **100 requests per minute**.  
Exceeded requests receive:

```
HTTP 429 Too Many Requests
Retry-After: <seconds>
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
```

**Algorithm:** `INCR key` → if count == 1 set `EXPIRE key 60` → deny if count > 100.

---

## Caching Strategy

| Operation | Cache action                              |
|-----------|-------------------------------------------|
| GET list  | Read `posts:list`; on miss populate + TTL |
| GET :id   | Read `posts:<id>`; on miss populate + TTL |
| POST      | Bust `posts:list`                         |
| PUT :id   | Bust `posts:<id>` + `posts:list`          |
| DELETE    | Bust `posts:<id>` + `posts:list`          |
| Bulk DEL  | Bust all affected keys + `posts:list`     |

Cache failures are **fail-open** — the API falls through to the DB rather than returning 5xx.

---

## Tests

```bash
# Unit tests (no external services required)
make test

# Integration tests (spins up Docker services automatically)
make test-integration
```

---

## Migrations

```bash
# Apply all up
make migrate-up

# Roll back one step
make migrate-down

# Inspect query plan (connect to psql first)
EXPLAIN ANALYZE SELECT * FROM posts WHERE user_id = 1;
```

---

## Docker Image Size

The multi-stage build (Go builder → `distroless/static-debian12:nonroot`) produces an image well under 20 MB:

```
REPOSITORY   TAG       SIZE
blog-api     latest    ~16 MB
```

# Finally do a comprehensive test
chmod +x test.sh
./test.sh

docker-compose down
docker-compose up --build

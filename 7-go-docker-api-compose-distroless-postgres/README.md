# 7. go-docker-api

A production-ready **Go REST API** backed by **PostgreSQL**, fully Dockerised
with a **multi-stage Dockerfile** (Alpine builder → distroless final image) and
**Docker Compose** orchestration.

---

## Folder structure

```
go-docker-api/
├── cmd/
│   └── api/
│       └── main.go                 ← entry point (router, server, graceful shutdown)
├── internal/
│   ├── db/
│   │   └── db.go                   ← Connect() with retry + Migrate()
│   ├── handlers/
│   │   └── handlers.go             ← all HTTP handlers (chi router)
│   └── models/
│       └── user.go                 ← request / response structs
├── .dockerignore                   ← keeps build context lean & secrets out
├── .env.ashkrai                    ← copy → .env before running
├── .gitignore
├── docker-compose.yml              ← full stack definition
├── Dockerfile                      ← multi-stage build
├── go.mod
├── go.sum
├── Makefile                        ← convenience targets
└── README.md
```

---

## Prerequisites

| Tool           | Minimum version |
|----------------|-----------------|
| Docker         | 24              |
| Docker Compose | v2 (`docker compose`) |

---

## Quick start

```bash
# 1. Create your .env from the template
cp .env.ashkrai .env

# 2. Set a real password (mandatory)
#    Edit .env and change POSTGRES_PASSWORD

# 3. Build images and start the stack
go mod tidy
make up
# or: docker compose up --build -d

# 4. Verify everything is healthy
make ps
# or: docker compose ps

# 5. Hit the health endpoint
curl http://localhost:8080/healthz
# → {"status":"ok"}
```

---

## API reference

All endpoints return / accept `application/json`.

### Health

| Method | Path       | Description                        |
|--------|------------|------------------------------------|
| `GET`  | `/healthz` | 200 OK when API + DB are reachable |

### Users

| Method   | Path                  | Description            | Body                               |
|----------|-----------------------|------------------------|------------------------------------|
| `GET`    | `/api/v1/users`       | List all users         | —                                  |
| `POST`   | `/api/v1/users`       | Create a user          | `{"name":"…","email":"…"}`         |
| `GET`    | `/api/v1/users/{id}`  | Get one user           | —                                  |
| `PUT`    | `/api/v1/users/{id}`  | Replace a user         | `{"name":"…","email":"…"}`         |
| `DELETE` | `/api/v1/users/{id}`  | Delete a user          | —                                  |

### ashkrai curl commands

```bash
# Health check
curl http://localhost:8080/healthz

# Create a user
curl -s -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@ashkrai.com"}' | jq

# List all users
curl -s http://localhost:8080/api/v1/users | jq

# Get user by id
curl -s http://localhost:8080/api/v1/users/1 | jq

# Update user
curl -s -X PUT http://localhost:8080/api/v1/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Smith","email":"alice.smith@ashkrai.com"}' | jq

# Delete user
curl -X DELETE http://localhost:8080/api/v1/users/1
# → HTTP 204 No Content
```

---

## Docker architecture

### Multi-stage Dockerfile

```
┌─────────────────────────────────────────────────────┐
│  Stage 1 — builder  (golang:1.22-alpine, ~280 MB)   │
│  • go mod download (cached layer)                   │
│  • CGO_ENABLED=0 GOOS=linux GOARCH=amd64            │
│  • -ldflags="-s -w" -trimpath                       │
│  → /bin/api  (fully static, ~8 MB)                  │
└───────────────────┬─────────────────────────────────┘
                    │ COPY --from=builder /bin/api /api
┌───────────────────▼─────────────────────────────────┐
│  Stage 2 — final  (distroless/static:nonroot, ~2 MB)│
│  • No shell, no package manager                     │
│  • No libc, no OS packages                          │
│  • Runs as UID 65532 (nonroot)                      │
│  → Final image: ~10 MB  ✅ < 20 MB target           │
└─────────────────────────────────────────────────────┘
```

### Compose services

| Service    | Image                  | Host port    | Notes                              |
|------------|------------------------|--------------|------------------------------------|
| `postgres` | `postgres:16-alpine`   | none         | Internal only; data in `pgdata`    |
| `api`      | built from Dockerfile  | `8080`       | Waits for postgres healthcheck     |

### Networking

Both services share a private **`backend`** bridge network. Containers
reference each other by service name (`postgres`, `api`). Postgres is
**not** published to the host — only the API port is reachable externally.

### Volumes

| Name     | Mount point                    | Purpose                      |
|----------|--------------------------------|------------------------------|
| `pgdata` | `/var/lib/postgresql/data`     | Postgres data — survives `down` |

### Environment variables

| Variable            | Default       | Required | Description                    |
|---------------------|---------------|----------|--------------------------------|
| `POSTGRES_PASSWORD` | —             | ✅ yes   | Postgres password               |
| `POSTGRES_USER`     | `appuser`     | no       | Postgres username               |
| `POSTGRES_DB`       | `appdb`       | no       | Database name                   |
| `API_PORT`          | `8080`        | no       | Host-side port for the API      |
| `PORT`              | `8080`        | no       | Port the Go binary listens on   |

---

## Healthchecks

| Service    | Command                                      | Interval | Retries |
|------------|----------------------------------------------|----------|---------|
| `postgres` | `pg_isready -U <user> -d <db>`               | 10 s     | 5       |
| `api`      | `wget --spider http://localhost:8080/healthz`| 15 s     | 3       |

`api` uses `depends_on: postgres: condition: service_healthy` so Compose
waits for Postgres to be **healthy** before starting the API — no sleep hacks
needed.

---

## Makefile targets

```
make up           Build images and start stack (background)
make down         Stop containers (keep volumes)
make rebuild      Force-rebuild api image without cache
make logs         Tail all logs
make ps           Show container status
make image-size   Print final api image size
make shell-db     Open psql in the postgres container
make test         Run Go unit tests
make lint         Run go vet
make clean        Remove containers, images, AND volumes (destructive)
make help         Show all targets
```

---

## Verifying image size

```bash
make up
make image-size
# Expected: api image size: ~10.0 MB
```

The distroless base is ~2 MB; the stripped, static Go binary adds ~7–9 MB.
Total stays well under the **20 MB** requirement.

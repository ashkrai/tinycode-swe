# Blog API — PostgreSQL Schema Design in Go

A complete, runnable Go REST API demonstrating:

- **PostgreSQL schema design** — users, posts, tags, comments with proper types, PKs, FKs, and indexes
- **golang-migrate** — versioned up/down migrations, add-column practice, rollback
- **EXPLAIN ANALYZE** — query plan inspection via CLI flag and HTTP endpoint
- **Clean architecture** — models / repository / handlers layers

---

## Project Structure

```
blog-api/
├── cmd/
│   └── api/
│       └── main.go                   # Entry point; -migrate -rollback -version -explain flags
├── internal/
│   ├── db/
│   │   └── db.go                     # Connect, MigrateUp, MigrateDown, MigrateVersion
│   ├── models/
│   │   └── models.go                 # Domain structs: User, Post, Tag, Comment
│   ├── repository/
│   │   ├── user.go
│   │   ├── post.go                   # Includes ExplainListPublished()
│   │   ├── tag.go
│   │   ├── comment.go
│   │   └── helpers.go
│   └── handlers/
│       └── handlers.go               # HTTP handlers (gorilla/mux)
├── migrations/
│   ├── 000001_create_users.up.sql
│   ├── 000001_create_users.down.sql
│   ├── 000002_create_posts.up.sql
│   ├── 000002_create_posts.down.sql
│   ├── 000003_create_tags.up.sql
│   ├── 000003_create_tags.down.sql
│   ├── 000004_create_comments.up.sql
│   ├── 000004_create_comments.down.sql
│   ├── 000005_add_reading_time_to_posts.up.sql    <- add a column
│   └── 000005_add_reading_time_to_posts.down.sql  <- rollback practice
├── scripts/
│   └── seed.sql
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Prerequisites

| Tool | Notes |
|------|-------|
| Docker + Docker Compose | Required for the quickest path |
| Go 1.22+ | Only needed for local (non-Docker) builds |
| `psql` | Optional, only needed for `make seed` locally |

---

## Quick Start — Docker (recommended)

```bash
# 1. Start Postgres + build and run the API (auto-applies all migrations)
make docker-up

# 2. Verify it is healthy
curl http://localhost:8080/health

# 3. Load sample data
make seed
```

---

## Quick Start — Local

```bash
# 1. Start only Postgres
docker compose up postgres -d

# 2. Build the binary
go mod tidy

make build

# 3. Apply all migrations
make migrate

# 4. Seed sample data
make seed

# 5. Start the API
make run
```

---

## Migration Workflow

```bash
# Apply all pending up-migrations
make migrate

# Check current schema version
make version
# => schema version: 5  dirty: false

# Roll back the last migration (drops reading_time_minutes column)
make rollback

# Re-apply
make migrate

# Using the standalone golang-migrate CLI instead:
make migrate-cli
make rollback-cli
```

### Migration Summary

| # | Up | Down |
|---|----|----|
| 1 | CREATE TABLE users | DROP TABLE users |
| 2 | CREATE TABLE posts + post_status ENUM | DROP TABLE posts, DROP TYPE |
| 3 | CREATE TABLE tags + post_tags (M:N join) | DROP both tables |
| 4 | CREATE TABLE comments (self-ref FK for threads) | DROP TABLE comments |
| 5 | ALTER TABLE posts ADD COLUMN reading_time_minutes | ALTER TABLE posts DROP COLUMN |

---

## EXPLAIN ANALYZE

```bash
# Print the query plan for the list-published-posts query
make explain
```

Or via HTTP after seeding:

```bash
curl http://localhost:8080/api/admin/explain-posts | jq -r .plan
```

The planner will use the partial index `idx_posts_status_published` instead of a
sequential scan, confirming the index is correctly applied.

---

## API Endpoints

### Users
| Method | Path | Description |
|--------|------|-------------|
| GET    | `/api/users`      | List all active users |
| POST   | `/api/users`      | Create a user |
| GET    | `/api/users/{id}` | Get user by ID |

### Posts
| Method | Path | Description |
|--------|------|-------------|
| GET    | `/api/posts`                    | List published posts (`?limit=20&offset=0`) |
| POST   | `/api/posts`                    | Create a post |
| GET    | `/api/posts/{slug}`             | Get post by slug (increments view count) |
| GET    | `/api/admin/explain-posts`      | EXPLAIN ANALYZE output |

### Tags
| Method | Path | Description |
|--------|------|-------------|
| GET    | `/api/tags` | List all tags |
| POST   | `/api/tags` | Create / upsert a tag |

### Comments
| Method | Path | Description |
|--------|------|-------------|
| GET    | `/api/posts/{postID}/comments` | List approved comments (threaded) |
| POST   | `/api/comments`                | Create a comment |
| PATCH  | `/api/comments/{id}/approve`   | Approve a comment |

---

## Example Requests

```bash
# Create a user
curl -s -X POST http://localhost:8080/api/users \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"s3cret"}' | jq

# Create a tag
curl -s -X POST http://localhost:8080/api/tags \
  -H 'Content-Type: application/json' \
  -d '{"name":"PostgreSQL","slug":"postgresql"}' | jq

# Create a published post with a tag
curl -s -X POST http://localhost:8080/api/posts \
  -H 'Content-Type: application/json' \
  -d '{
    "author_id": 1,
    "title": "My First Post",
    "slug": "my-first-post",
    "body": "Hello world! This is my first blog post.",
    "status": "published",
    "tag_ids": [1]
  }' | jq

# List published posts
curl -s "http://localhost:8080/api/posts?limit=10&offset=0" | jq

# Get post by slug (also bumps view count)
curl -s http://localhost:8080/api/posts/my-first-post | jq

# Add a comment
curl -s -X POST http://localhost:8080/api/comments \
  -H 'Content-Type: application/json' \
  -d '{"post_id":1,"author_id":2,"body":"Great post!"}' | jq

# Approve the comment
curl -s -X PATCH http://localhost:8080/api/comments/1/approve | jq

# List approved comments for post 1 (threaded)
curl -s http://localhost:8080/api/posts/1/comments | jq
```

---

## Schema Diagram

```
users
 ├── id            BIGSERIAL PK
 ├── username      VARCHAR UNIQUE
 ├── email         VARCHAR UNIQUE
 └── ...
      │ 1:N
      ▼
posts ──────────────────── post_tags ──── tags
 ├── id            PK       ├── post_id    ├── id   PK
 ├── author_id     FK       └── tag_id     ├── name UNIQUE
 ├── slug          UNIQUE                  └── slug UNIQUE
 ├── status        ENUM
 └── reading_time_minutes
      │ 1:N
      ▼
comments
 ├── id          PK
 ├── post_id     FK -> posts
 ├── author_id   FK -> users
 └── parent_id   FK -> comments  (nullable — threaded replies)
```

---

## Index Strategy

| Index | Type | Purpose |
|-------|------|---------|
| `idx_users_email` | B-tree | Login lookups |
| `idx_posts_author_id` | B-tree | JOIN posts → users |
| `idx_posts_status_published` | Partial B-tree | Only published rows — list feed |
| `idx_post_tags_tag_id` | B-tree | Posts by tag queries |
| `idx_comments_post_id` | B-tree | Fetch comments for a post |
| `idx_comments_approved_created` | Partial B-tree | Approved comments per post |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | Postgres hostname |
| `DB_PORT` | `5432` | Postgres port |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_NAME` | `blog` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `MIGRATIONS_DIR` | auto-detected | Path to migrations folder |

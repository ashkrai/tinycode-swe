# ── build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod ./
RUN go mod download && go mod tidy || true

COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o /app/blog-api ./cmd/api

# ── runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/blog-api  ./blog-api
COPY migrations/                   ./migrations/
COPY entrypoint.sh                 ./entrypoint.sh

EXPOSE 8080

# entrypoint.sh: runs migrations first, then starts the API server
ENTRYPOINT ["/app/entrypoint.sh"]

# =============================================================================
# Stage 1 — builder
# =============================================================================
# golang:1.22-alpine is ~250 MB but NEVER ships to production.
# We use it only to compile a fully-static binary.
FROM golang:1.22-alpine AS builder

# git is required by `go mod download` for VCS-based modules.
# ca-certificates lets the binary make outbound TLS calls at runtime.
RUN apk add --no-cache ca-certificates git

WORKDIR /build

# -- Download dependencies first so Docker caches this layer separately.
#    Rebuilds after source changes won't re-download modules.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# -- Copy the rest of the source and compile.
COPY . .

RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
      -ldflags="-s -w" \
      -trimpath \
      -o /bin/api \
      ./cmd/api

# =============================================================================
# Stage 2 — final (distroless/static:nonroot)
# =============================================================================
# gcr.io/distroless/static:nonroot contains ONLY:
#   • CA certificates bundle
#   • /etc/passwd  (so the nonroot user exists)
#   • Timezone data
#
# There is NO shell, NO package manager, NO extra OS libraries.
# Result: ~2 MB base + ~8 MB binary = ~10 MB total image.
#         Attack surface is essentially zero.
FROM gcr.io/distroless/static:nonroot

# Copy only the compiled binary from the builder stage.
COPY --from=builder /bin/api /api

# Run as nonroot (uid=65532 gid=65532) — defined in distroless /etc/passwd
USER nonroot:nonroot

# Document the port — actual publishing is done in docker-compose.yml
EXPOSE 8080

ENTRYPOINT ["/api"]

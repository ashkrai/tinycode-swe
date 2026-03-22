FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /blog-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /blog-api /blog-api

EXPOSE 8080

ENTRYPOINT ["/blog-api"]
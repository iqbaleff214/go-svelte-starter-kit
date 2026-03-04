# ─── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# ─── API runtime stage ────────────────────────────────────────────────────────
FROM alpine:3.20 AS api

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/api ./api
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/templates ./templates

# Pre-create uploads directory so the app can write on first boot
RUN mkdir -p uploads/avatars && chown -R app:app uploads

EXPOSE 8080
USER app
ENTRYPOINT ["/app/api"]

# ─── Worker runtime stage ─────────────────────────────────────────────────────
FROM alpine:3.20 AS worker

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/worker ./worker
COPY --from=builder /app/templates ./templates

USER app
ENTRYPOINT ["/app/worker"]

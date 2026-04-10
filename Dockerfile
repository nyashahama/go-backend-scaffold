# Stage 1: Build
FROM golang:1.25.9-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server/

# Stage 2: Runtime
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

# Run as non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /app/db/migrations ./db/migrations

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:${PORT:-8080}/healthz || exit 1

ENTRYPOINT ["./server"]

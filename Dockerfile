# syntax=docker/dockerfile:1

# ==============================================================================
# Stage 1: Build Frontend (React + Vite + TypeScript)
# ==============================================================================
FROM oven/bun:1-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package specifications for layer caching
COPY frontend/package.json frontend/bun.lock* ./

# Install dependencies (respecting lockfile)
RUN bun install --frozen-lockfile

# Copy frontend source code
COPY frontend/ ./

# Build frontend (Vite outDir targets ../cmd/server/dist)
RUN bun run build

# ==============================================================================
# Stage 2: Build Backend (Go with CGO enabled for C libraries / ONNX Runtime)
# ==============================================================================
FROM golang:bookworm AS backend-builder

WORKDIR /build

# Enable CGO: essential for onnxruntime_go C bindings and native system libraries
ENV CGO_ENABLED=1 \
    GOOS=linux

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source code & embedded migrations
COPY db/ ./db/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/

# Copy built frontend assets into cmd/server/dist (required by //go:embed in server.go)
COPY --from=frontend-builder /app/cmd/server/dist ./cmd/server/dist

# Build production binary with stripped debugging symbols
RUN go build -trimpath -ldflags="-s -w" -o /build/bin/server ./cmd/server

# ==============================================================================
# Stage 3: Minimal Production Runtime
# ==============================================================================
FROM debian:bookworm-slim AS runner

# Install essential runtime libraries:
# - ca-certificates: required for outbound HTTPS connections (LLM APIs)
# - tzdata: accurate timezone support
# - libstdc++6: C++ standard library runtime (required by libonnxruntime.so)
# - curl: required for container healthcheck
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    libstdc++6 \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Create a non-root group and user (UID/GID 1000 aligns with standard host permissions)
RUN groupadd -g 1000 appgroup && \
    useradd -u 1000 -g appgroup -m -s /bin/bash appuser && \
    mkdir -p /app/data /app/data/logs /app/data/onnx && \
    chown -R appuser:appgroup /app

# Copy compiled single-binary server
COPY --from=backend-builder --chown=appuser:appgroup /build/bin/server /app/server

# Default environment configuration
ENV HOST=0.0.0.0 \
    PORT=3434 \
    DATA_DIR=/app/data \
    SQLITE_DB_PATH=/app/data/tluagent.db \
    APP_ENV=production

# Expose HTTP port
EXPOSE 3434

# Run container as non-root user
USER appuser

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:3434/api/health || exit 1

# Start the server
ENTRYPOINT ["/app/server"]

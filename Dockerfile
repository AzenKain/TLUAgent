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
FROM golang:1.27.3-trixie AS backend-builder

WORKDIR /build

# Runtime assets that the distroless runner stage cannot install itself (no apt there)
RUN apt-get update && apt-get install -y --no-install-recommends tzdata ca-certificates \
    && rm -rf /var/lib/apt/lists/*

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

# Writable data directories, pre-owned by the runtime UID (distroless has no shell to mkdir)
RUN mkdir -p /build/appdata/logs /build/appdata/onnx && chown -R 1000:1000 /build/appdata

# ==============================================================================
# Stage 3: Minimal Production Runtime (distroless: glibc + libstdc++, no shell/apt)
# ==============================================================================
FROM gcr.io/distroless/cc-debian13 AS runner

WORKDIR /app

# distroless ships no package manager, so timezone data and the CA bundle come from the builder
COPY --from=backend-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy compiled single-binary server and the writable data directories
COPY --from=backend-builder --chown=1000:1000 /build/appdata /app/data
COPY --from=backend-builder --chown=1000:1000 /build/bin/server /app/server

# Default environment configuration
ENV HOST=0.0.0.0 \
    PORT=3434 \
    DATA_DIR=/app/data \
    SQLITE_DB_PATH=/app/data/tluagent.db \
    APP_ENV=production

# Expose HTTP port
EXPOSE 3434

# Run container as the same UID as before (keeps host volume permissions working)
USER 1000:1000

# Healthcheck runs the binary itself, since curl is not available in distroless
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/server", "-healthcheck"]

# Start the server
ENTRYPOINT ["/app/server"]


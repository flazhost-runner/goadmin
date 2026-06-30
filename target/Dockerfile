# syntax=docker/dockerfile:1
# ── GoAdmin starter kit · FlazHost PaaS (CapRover) ──────────────────────────
# Multi-stage build. SQLite driver = github.com/glebarez/sqlite (PURE Go), so
# no CGO/gcc is required → CGO_ENABLED=0 gives a fully static binary.

# 1) Build stage
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache module downloads first.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build both binaries.
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
# server = HTTP entrypoint (cmd/server); migrate = DB migrate+seed (cmd/migrate).
RUN go build -trimpath -ldflags="-s -w" -o /out/server  ./cmd/server \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# 2) Runtime stage
FROM alpine:3.20
WORKDIR /app

# ca-certificates for outbound TLS (e.g. S3 storage / SMTP / optional FE remote);
# tzdata is harmless and the app pins UTC anyway.
RUN apk add --no-cache ca-certificates tzdata libcap \
 && adduser -D -u 10001 appuser

# Binaries.
COPY --from=build /out/server  /app/server
COPY --from=build /out/migrate /app/migrate

# Allow the non-root user to bind the privileged port 80 (CapRover default).
RUN setcap 'cap_net_bind_service=+ep' /app/server

# Disk assets read at runtime, relative to WORKDIR:
#   web/templates/{layouts,partials}/*.html  + web/assets (static)
#   internal/modules/*/view/*.html           (module view templates)
# (SQL migrations are embedded in the migrate binary — no need to copy them.)
COPY --from=build /src/web      /app/web
COPY --from=build /src/internal /app/internal

COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh \
 && mkdir -p /app/data /app/web/uploads /app/web/cache \
 && chown -R appuser:appuser /app

# ── Zero-config defaults (all overridable via env) ──────────────────────────
# Production mode → server does NOT self-migrate (entrypoint runs migrate first)
# and secrets are required, so we ship safe non-secret defaults (override them!).
ENV NODE_ENV=production \
    APP_MODE=full \
    APP_NAME=GoAdmin \
    APP_HOST=http://localhost \
    PORT=80 \
    DB_TYPE=sqlite \
    DB_DATABASE=/app/data/goadmin.db \
    SESSION_SECRET=change-me-session-secret \
    JWT_SECRET=change-me-jwt-secret \
    REDIS_URL= \
    FE_TEMPLATE_REMOTE=false \
    STORAGE_DRIVER=local \
    STORAGE_DIR=/app/web/uploads \
    STORAGE_URL=/uploads

USER appuser
EXPOSE 80
ENTRYPOINT ["/app/docker-entrypoint.sh"]

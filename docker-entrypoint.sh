#!/bin/sh
# GoAdmin container entrypoint (FlazHost / CapRover).
#   1) Map CapRover's $PORT → APP_PORT (the app reads APP_PORT via internal/config).
#   2) Run DB migrations + seed (idempotent) — replicates `make migrate`.
#   3) Exec the server — replicates `make run`.
set -eu

# ── Port: CapRover injects $PORT (default 80). The app listens on APP_PORT. ──
: "${PORT:=80}"
export APP_PORT="${APP_PORT:-$PORT}"

# ── SQLite: ensure the DB file lives on a writable path. ──
if [ "${DB_TYPE:-sqlite}" = "sqlite" ]; then
  export DB_DATABASE="${DB_DATABASE:-/app/data/goadmin.db}"
  mkdir -p "$(dirname "$DB_DATABASE")" 2>/dev/null || true
fi

echo "[entrypoint] DB_TYPE=${DB_TYPE:-sqlite} DB_DATABASE=${DB_DATABASE:-} APP_PORT=${APP_PORT} NODE_ENV=${NODE_ENV:-production}"

# ── Migrate + seed. Idempotent: sqlite→AutoMigrate, mysql/postgres→golang-migrate.
# A failure here (e.g. already-migrated edge case or transient DB) must NOT block
# boot — log and continue so the server can still come up against an existing schema.
echo "[entrypoint] running migrations (./migrate)..."
if /app/migrate; then
  echo "[entrypoint] migrations OK"
else
  echo "[entrypoint] WARN: migrate exited non-zero — continuing to start server"
fi

# ── Start the HTTP server (PID 1 for clean SIGTERM/graceful shutdown). ──
echo "[entrypoint] starting server on :${APP_PORT}"
exec /app/server

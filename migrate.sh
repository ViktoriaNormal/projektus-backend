#!/usr/bin/env bash
set -euo pipefail

DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgres@localhost:5432/projektus?sslmode=disable}"
MIGRATIONS_DIR="$(cd "$(dirname "$0")" && pwd)/internal/db/migrations"

# Создаём таблицу для отслеживания применённых миграций
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'SQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
SQL

applied=0
skipped=0

for file in "$MIGRATIONS_DIR"/*.up.sql; do
    version="$(basename "$file" | cut -d_ -f1)"

    # Пропускаем уже применённые миграции
    already=$(psql "$DATABASE_URL" -tAc "SELECT 1 FROM schema_migrations WHERE version = '$version'" 2>/dev/null)
    if [ "$already" = "1" ]; then
        skipped=$((skipped + 1))
        continue
    fi

    echo "Applying $(basename "$file") ..."
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file"
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version) VALUES ('$version')"
    applied=$((applied + 1))
done

echo ""
echo "Done. Applied: $applied, skipped (already applied): $skipped."

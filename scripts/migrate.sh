#!/bin/sh

set -e

echo "=== Migration Script Started ==="
echo "DB_HOST: $DB_HOST"
echo "DB_PORT: $DB_PORT"
echo "POSTGRES_USER: $POSTGRES_USER"
echo "POSTGRES_DB: $POSTGRES_DB"
echo ""

# Проверяем наличие миграций
echo "Checking migrations directory..."
if [ -d "/app/migrations" ]; then
  echo "Migrations directory exists"
  echo "Contents:"
  ls -la /app/migrations/
else
  echo "ERROR: Migrations directory not found!"
  exit 1
fi
echo ""

# Ждем, пока база данных будет готова
echo "Waiting for database to be ready..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do
  echo "Waiting for database to be ready..."
  sleep 2
done

echo "✓ Database is ready!"
echo ""
echo "Running migrations..."

# Применяем миграции в порядке их имен (сортировка по имени файла)
migration_count=0
for migration in $(ls /app/migrations/*.up.sql | sort); do
  if [ -f "$migration" ]; then
    echo "→ Applying: $(basename $migration)"
    if PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f "$migration"; then
      echo "  ✓ Success"
      migration_count=$((migration_count + 1))
    else
      echo "  ✗ Failed!"
      exit 1
    fi
  fi
done

echo ""
echo "✓ Successfully applied $migration_count migration(s)"
echo "=== Migrations completed successfully! ==="


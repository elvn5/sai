#!/usr/bin/env bash
# Drops the former NestJS database ($POSTGRES_DB, "parcel_bot") once every
# Go service has copied its data out of it. Run by the deploy workflows after
# `docker compose up`, from the deploy directory (with .env loaded).
#
# Nothing is dropped unless all of these hold — otherwise it exits non-zero
# and leaves the database alone:
#   - the database exists (otherwise there's nothing to do: exit 0);
#   - its name isn't one of the services' own databases;
#   - users, content and listings have each recorded their import
#     (legacy_imports), waiting up to WAIT_SECONDS for them to start;
#   - a pg_dump of it was written to ./backups and is non-empty.
#
# Usage: drop-legacy-db.sh <compose project name>
# PG_EXEC overrides how commands reach the postgres container (tests).
set -euo pipefail

PROJECT="${1:?usage: drop-legacy-db.sh <compose project>}"
LEGACY_DB="${POSTGRES_DB:?POSTGRES_DB is not set}"
PG_USER="${POSTGRES_USER:?POSTGRES_USER is not set}"
WAIT_SECONDS="${WAIT_SECONDS:-120}"
BACKUP_DIR="${BACKUP_DIR:-backups}"
read -r -a PG_EXEC <<< "${PG_EXEC:-docker compose -p $PROJECT -f docker-compose.deploy.yml --env-file .env exec -T postgres}"

# service database : import marker
IMPORTS=("users:nestjs-users" "content:nestjs-content" "listings:nestjs-listings")

query() { # query <database> <sql>
  "${PG_EXEC[@]}" psql -U "$PG_USER" -d "$1" -tAc "$2"
}

case "$LEGACY_DB" in
  postgres | template0 | template1 | users | content | listings)
    echo "Refusing to drop '$LEGACY_DB': not the legacy NestJS database." >&2
    exit 1
    ;;
esac

if [ "$(query postgres "SELECT 1 FROM pg_database WHERE datname = '$LEGACY_DB'")" != "1" ]; then
  echo "Legacy database '$LEGACY_DB' is already gone."
  exit 0
fi

# Each service imports on its first start, before serving.
deadline=$((SECONDS + WAIT_SECONDS))
for entry in "${IMPORTS[@]}"; do
  db="${entry%%:*}"
  marker="${entry#*:}"
  until [ "$(query postgres "SELECT 1 FROM pg_database WHERE datname = '$db'")" = "1" ] &&
    [ "$(query "$db" "SELECT 1 FROM legacy_imports WHERE name = '$marker'" 2>/dev/null)" = "1" ]; do
    if [ "$SECONDS" -ge "$deadline" ]; then
      echo "The $db service hasn't imported from '$LEGACY_DB' yet; keeping it. Check that service's logs." >&2
      exit 1
    fi
    sleep 3
  done
  echo "$db: import recorded."
done

mkdir -p "$BACKUP_DIR"
backup="$BACKUP_DIR/$LEGACY_DB-$(date -u +%Y%m%dT%H%M%SZ).dump"
"${PG_EXEC[@]}" pg_dump -U "$PG_USER" -Fc "$LEGACY_DB" > "$backup"
if [ ! -s "$backup" ]; then
  echo "Backup $backup is empty; keeping '$LEGACY_DB'." >&2
  exit 1
fi
echo "Backed up '$LEGACY_DB' to $backup ($(wc -c < "$backup") bytes)."

query postgres "DROP DATABASE \"$LEGACY_DB\" WITH (FORCE)"
echo "Dropped legacy database '$LEGACY_DB'."

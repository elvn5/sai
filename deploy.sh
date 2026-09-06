#!/usr/bin/env bash
# Релиз AsDi Cargo — выполняется на сервере (deploy.yml по SSH; можно и руками
# из каталога деплоя). Идемпотентен: применяются только новые миграции.
#
# Обязательно: IMAGE_TAG (staging | X.Y.Z).
# Изоляция окружений на одном сервере:
#   COMPOSE_PROJECT_NAME — имя compose-проекта (asdicargo | asdicargo-staging).
#     Разные проекты => отдельные контейнеры, тома БД и сети.
#   API_PORT / ADMIN_PORT / CLIENT_PORT — порты на 127.0.0.1 (за host-nginx).
# Для приватных образов GHCR: GHCR_USER, GHCR_TOKEN.
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[ -f .env ] && chmod 600 .env || true   # секьюрим env на сервере

IMAGE_PREFIX="ghcr.io/elvn5/asdicargo"
COMPOSE_FILE="docker-compose.deploy.yml"
COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-asdicargo}"
: "${IMAGE_TAG:?IMAGE_TAG не задан}"

# Порты по умолчанию соответствуют production; staging передаёт свои через env.
export IMAGE_PREFIX IMAGE_TAG COMPOSE_PROJECT_NAME
export API_PORT="${API_PORT:-3001}"
export ADMIN_PORT="${ADMIN_PORT:-3000}"
export CLIENT_PORT="${CLIENT_PORT:-3002}"

compose() { docker compose -p "$COMPOSE_PROJECT_NAME" -f "$COMPOSE_FILE" "$@"; }

echo "▶ Деплой ${IMAGE_PREFIX} @ ${IMAGE_TAG} (проект ${COMPOSE_PROJECT_NAME}, порты ${API_PORT}/${ADMIN_PORT}/${CLIENT_PORT})"

# Логин в GHCR (для приватных образов)
if [[ -n "${GHCR_TOKEN:-}" ]]; then
  echo "${GHCR_TOKEN}" | docker login ghcr.io -u "${GHCR_USER:?GHCR_USER не задан}" --password-stdin
  trap 'docker logout ghcr.io >/dev/null 2>&1 || true' EXIT
fi

compose pull
compose up -d --wait postgres
echo "▶ Миграции БД"
compose run --rm -w /repo api \
  node apps/api/node_modules/typeorm/cli.js migration:run \
  -d packages/database/dist/data-source.js
compose up -d --remove-orphans
docker image prune -f
echo "✔ Готово"
compose ps

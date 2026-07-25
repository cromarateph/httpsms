#!/usr/bin/env sh
set -eu

project_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_dir"

export GIT_COMMIT="${GIT_COMMIT:-unknown}"

docker compose -f compose.production.yml config --quiet
docker compose -f compose.production.yml up -d --build --remove-orphans
curl -fsS --retry 12 --retry-delay 5 --retry-all-errors \
  https://sms.evilmachine.tech/health >/dev/null
docker compose -f compose.production.yml ps

#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="${ROOT}/.run"
ADMIN="http://127.0.0.1:2019"
PUBLIC="http://127.0.0.1:8080"

cleanup() {
  if [[ -n "${COMPOSE_PID:-}" ]]; then
    docker compose -f "${ROOT}/docker-compose.yml" down -v --remove-orphans >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

rm -rf "${RUN_DIR}"
mkdir -p "${RUN_DIR}"

echo "==> starting caddy + backends"
docker compose -f "${ROOT}/docker-compose.yml" up -d
COMPOSE_PID=1

echo "==> waiting for caddy admin"
for _ in $(seq 1 30); do
  if curl -sf "${ADMIN}/config/" >/dev/null; then
    break
  fi
  sleep 1
done

echo "==> traffic should hit backend A"
body="$(curl -sf "${PUBLIC}/" | tr -d '\r\n')"
if [[ "${body}" != "backend A" ]]; then
  echo "expected backend A, got: ${body}"
  exit 1
fi

echo "==> patching upstream to backend B"
curl -sf -X PATCH "${ADMIN}/config/apps/http/servers/srv0/routes/0/handle/0/upstreams" \
  -H "Content-Type: application/json" \
  -d '[{"dial":"backend_b:80"}]' >/dev/null

sleep 1
body="$(curl -sf "${PUBLIC}/" | tr -d '\r\n')"
if [[ "${body}" != "backend B" ]]; then
  echo "expected backend B after swap, got: ${body}"
  exit 1
fi

echo "==> draining: new requests use B only"
for _ in $(seq 1 5); do
  curl -sf "${PUBLIC}/" | grep -q "backend B"
done

echo "==> swap back to A (rollback simulation)"
curl -sf -X PATCH "${ADMIN}/config/apps/http/servers/srv0/routes/0/handle/0/upstreams" \
  -H "Content-Type: application/json" \
  -d '[{"dial":"backend_a:80"}]' >/dev/null

sleep 1
body="$(curl -sf "${PUBLIC}/" | tr -d '\r\n')"
if [[ "${body}" != "backend A" ]]; then
  echo "rollback failed, got: ${body}"
  exit 1
fi

echo "==> caddy swap prototype OK"

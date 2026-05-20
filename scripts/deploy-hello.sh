#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HELLO="${ROOT}/agent/testdata/hello"
TMP="$(mktemp -d)"
TAR="${TMP}/source.tar.gz"
trap 'rm -rf "${TMP}"' EXIT

tar -czf "${TAR}" -C "${HELLO}" .
TOML="$(cat "${HELLO}/hangar.toml")"
B64="$(base64 -w0 "${TAR}" 2>/dev/null || base64 < "${TAR}" | tr -d '\n')"

AGENT="${AGENT_URL:-http://127.0.0.1:8741}"
TOKEN="${HANGAR_TOKEN:-}"

HDR=(-H "Content-Type: application/json")
if [[ -n "${TOKEN}" ]]; then
  HDR+=(-H "Authorization: Bearer ${TOKEN}")
fi

BUILD_ID="build-$(date +%s)"
export TOML BUILD_ID B64
BODY=$(python3 -c '
import json, os
print(json.dumps({
  "toml": os.environ["TOML"],
  "build_id": os.environ["BUILD_ID"],
  "source_base64": os.environ["B64"],
  "tenant": "default",
}))
')

echo "==> POST /deploys"
RESP=$(curl -sf "${HDR[@]}" -d "${BODY}" "${AGENT}/deploys")
ID=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])' <<< "${RESP}")
echo "deploy_id=${ID}"

echo "==> waiting for terminal state"
for _ in $(seq 1 180); do
  ST=$(curl -sf "${HDR[@]}" "${AGENT}/deploys/${ID}")
  PHASE=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["phase"])' <<< "${ST}")
  MSG=$(python3 -c 'import json,sys; print(json.load(sys.stdin).get("message",""))' <<< "${ST}")
  echo "  phase=${PHASE} msg=${MSG}"
  if [[ "${PHASE}" == "succeeded" ]]; then
    CADDY_PORT="${CADDY_HTTP_PORT:-8877}"
    echo "==> curl via caddy (Host: hello.hangar.local, port ${CADDY_PORT})"
    if curl -sf -H "Host: hello.hangar.local" "http://127.0.0.1:${CADDY_PORT}/"; then
      echo
      exit 0
    fi
    echo "caddy curl failed"
    echo "  listener on ${CADDY_PORT}: $(ss -tlnp 2>/dev/null | grep ":${CADDY_PORT} " || echo 'none')"
    echo "  hangar-caddy: $(docker port hangar-caddy 2>/dev/null | tr '\n' ' ' || echo 'not running')"
    echo "checking app container directly"
    CNAME=$(docker ps --format '{{.Names}}' | grep -E '^hello-web-' | head -1 || true)
    if [[ -n "${CNAME}" ]]; then
      docker exec "${CNAME}" wget -qO- "http://127.0.0.1:8080/" || true
      echo
    else
      echo "no hello-web container found"
      exit 1
    fi
    exit 0
  fi
  if [[ "${PHASE}" == "failed" ]]; then
    echo "${ST}"
    exit 1
  fi
  sleep 2
done
echo "timeout"
exit 1

#!/usr/bin/env bash
set -euo pipefail

VERSION="${GO_VERSION:-1.22.12}"
ARCH="${GO_ARCH:-linux-amd64}"
URL="https://go.dev/dl/go${VERSION}.${ARCH}.tar.gz"

if command -v go >/dev/null 2>&1; then
  ver="$(go version 2>/dev/null || true)"
  if echo "${ver}" | grep -qE 'go1\.(2[2-9]|[3-9][0-9])'; then
    echo "Go already OK: ${ver}"
    exit 0
  fi
  echo "Found ${ver}; installing Go ${VERSION} to /usr/local/go"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

echo "==> downloading ${URL}"
curl -fsSL "${URL}" -o "${tmp}/go.tar.gz"
echo "==> installing to /usr/local/go (requires sudo)"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "${tmp}/go.tar.gz"
echo "==> done: $(/usr/local/go/bin/go version)"
echo
echo "Add to ~/.bashrc:"
echo '  export PATH=/usr/local/go/bin:$PATH'

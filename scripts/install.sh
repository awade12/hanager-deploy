#!/usr/bin/env bash
set -eo pipefail

INSTALLER_VERSION=2
echo "hangar installer v${INSTALLER_VERSION}"

MODULE="${HANGAR_MODULE:-github.com/awade12/hanager-deploy}"
REF="${HANGAR_VERSION:-latest}"
REPO="${HANGAR_REPO:-awade12/hanager-deploy}"
INSTALL_DIR="${HANGAR_INSTALL_DIR:-}"
LOCAL_INSTALL=0

if [[ -t 0 ]] && [[ -n "${BASH_SOURCE[0]+x}" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
  if [[ -z "${HANGAR_MODULE:-}" && -f "${REPO_ROOT}/go.mod" ]]; then
    if grep -q 'module github.com/awade12/hanager-deploy' "${REPO_ROOT}/go.mod" 2>/dev/null; then
      LOCAL_INSTALL=1
    fi
  fi
fi

go_bin_dir() {
  if [[ -n "${GOBIN:-}" ]]; then
    echo "${GOBIN}"
    return
  fi
  local gopath="${GOPATH:-${HOME}/go}"
  echo "${gopath}/bin"
}

install_with_go() {
  if ! command -v go >/dev/null 2>&1; then
    return 1
  fi
  local ver
  ver="$(go version 2>/dev/null || true)"
  if ! echo "${ver}" | grep -qE 'go1\.(2[2-9]|[3-9][0-9])'; then
    echo "error: need Go 1.22+ (got: ${ver:-missing})" >&2
    exit 1
  fi
  if [[ "${LOCAL_INSTALL}" == "1" ]]; then
    echo "==> installing from local repo ${REPO_ROOT}"
    (cd "${REPO_ROOT}" && go install ./cli/cmd/hangar && go install ./agent/cmd/hangar-agent)
    return 0
  fi
  echo "==> installing hangar CLI and agent via go install (${MODULE}@${REF})"
  go install "${MODULE}/cli/cmd/hangar@${REF}"
  go install "${MODULE}/agent/cmd/hangar-agent@${REF}"
}

install_from_release() {
  local os arch base url tmpdir
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  arch="$(uname -m)"
  case "${arch}" in
    x86_64) arch="amd64" ;;
    aarch64|arm64) arch="arm64" ;;
    *) echo "unsupported arch: ${arch}" >&2; return 1 ;;
  esac
  case "${os}" in
    linux|darwin) ;;
    *) echo "unsupported os: ${os}" >&2; return 1 ;;
  esac

  if [[ "${REF}" == "latest" ]]; then
    base="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -o '"tag_name": "[^"]*"' | head -1 | cut -d'"' -f4)"
  else
    base="${REF}"
  fi
  if [[ -z "${base}" ]]; then
    echo "no GitHub release found for ${REPO}" >&2
    return 1
  fi

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "${tmpdir}"' RETURN
  for bin in hangar hangar-agent; do
    url="https://github.com/${REPO}/releases/download/${base}/${bin}-${os}-${arch}"
    echo "==> downloading ${url}"
    curl -fsSL "${url}" -o "${tmpdir}/${bin}"
    chmod +x "${tmpdir}/${bin}"
  done

  local dest="${INSTALL_DIR:-$(go_bin_dir)}"
  mkdir -p "${dest}"
  install -m 755 "${tmpdir}/hangar" "${dest}/hangar"
  install -m 755 "${tmpdir}/hangar-agent" "${dest}/hangar-agent"
  echo "installed to ${dest}"
}

verify_bins() {
  local bindir="${1}"
  local missing=0
  for bin in hangar hangar-agent; do
    if [[ ! -x "${bindir}/${bin}" ]]; then
      echo "error: ${bindir}/${bin} not found after install" >&2
      missing=1
    fi
  done
  return "${missing}"
}

main() {
  if ! install_with_go; then
    if ! install_from_release; then
      echo "install failed: need Go 1.22+ or a GitHub release binary" >&2
      echo "  go install ${MODULE}/cli/cmd/hangar@${REF}" >&2
      echo "  go install ${MODULE}/agent/cmd/hangar-agent@${REF}" >&2
      exit 1
    fi
  fi

  local bindir
  bindir="${INSTALL_DIR:-$(go_bin_dir)}"
  if ! verify_bins "${bindir}"; then
    exit 1
  fi

  echo
  echo "Installed:"
  echo "  ${bindir}/hangar"
  echo "  ${bindir}/hangar-agent"
  if [[ ":${PATH}:" != *":${bindir}:"* ]]; then
    echo
    echo "Add to PATH (e.g. ~/.zshrc):"
    echo "  export PATH=\"${bindir}:\$PATH\""
  fi
  echo
  echo "Next:"
  echo "  hangar config init"
  echo "  hangar init"
  echo "  cd my-app && hangar deploy"
}

main "$@"

#!/usr/bin/env bash
set -euo pipefail

# Template installer for GitHub Releases artifacts produced by GoReleaser.
# You can override defaults with env vars:
#   OWNER, REPO, BINARY, VERSION, INSTALL_DIR
#
# Examples:
#   ./scripts/install-release.sh
#   VERSION=v1.2.3 ./scripts/install-release.sh

OWNER="${OWNER:-hengyunabc}"
REPO="${REPO:-mcp2cli}"
BINARY="${BINARY:-mcp2cli}"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64 | amd64) echo "amd64" ;;
    arm64 | aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

latest_tag() {
  local url
  url="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
  if command -v jq >/dev/null 2>&1; then
    curl -fsSL "$url" | jq -r ".tag_name"
    return
  fi
  curl -fsSL "$url" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

resolve_tag_and_version() {
  local input="$1"
  if [[ "$input" == "latest" ]]; then
    TAG="$(latest_tag)"
    if [[ -z "$TAG" || "$TAG" == "null" ]]; then
      echo "failed to resolve latest release tag from GitHub API" >&2
      exit 1
    fi
    VERSION_NO_V="${TAG#v}"
    return
  fi

  if [[ "$input" == v* ]]; then
    TAG="$input"
    VERSION_NO_V="${input#v}"
    return
  fi

  TAG="v${input}"
  VERSION_NO_V="$input"
}

verify_checksum() {
  local expected
  local file="$1"
  local checksums="$2"
  local filename
  filename="$(basename "$file")"

  expected="$(grep "  ${filename}$" "$checksums" | awk '{print $1}')"
  if [[ -z "$expected" ]]; then
    echo "cannot find checksum entry for ${filename}" >&2
    exit 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    echo "${expected}  ${file}" | sha256sum -c -
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    local actual
    actual="$(shasum -a 256 "$file" | awk '{print $1}')"
    if [[ "$actual" != "$expected" ]]; then
      echo "checksum mismatch for ${filename}" >&2
      exit 1
    fi
    return
  fi

  echo "warning: skip checksum verify (sha256sum/shasum not found)" >&2
}

install_binary() {
  local source="$1"
  local target="$2"
  mkdir -p "$(dirname "$target")"
  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$source" "$target"
    return
  fi
  cp "$source" "$target"
  chmod 0755 "$target"
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd grep
  need_cmd awk

  local os arch tmpdir asset archive_url checksums_url archive_file checksums_file extracted
  os="$(detect_os)"
  arch="$(detect_arch)"
  resolve_tag_and_version "$VERSION"

  if [[ ! -w "$INSTALL_DIR" ]]; then
    INSTALL_DIR="${HOME}/.local/bin"
  fi

  asset="${REPO}_${VERSION_NO_V}_${os}_${arch}.tar.gz"
  archive_url="https://github.com/${OWNER}/${REPO}/releases/download/${TAG}/${asset}"
  checksums_url="https://github.com/${OWNER}/${REPO}/releases/download/${TAG}/checksums.txt"

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  archive_file="${tmpdir}/${asset}"
  checksums_file="${tmpdir}/checksums.txt"

  echo "Downloading ${archive_url}"
  curl -fsSL "$archive_url" -o "$archive_file"
  curl -fsSL "$checksums_url" -o "$checksums_file"

  verify_checksum "$archive_file" "$checksums_file"

  tar -xzf "$archive_file" -C "$tmpdir"
  extracted="$(find "$tmpdir" -type f -name "$BINARY" | head -n 1)"
  if [[ -z "$extracted" ]]; then
    echo "cannot find binary ${BINARY} in downloaded archive" >&2
    exit 1
  fi

  install_binary "$extracted" "${INSTALL_DIR}/${BINARY}"

  echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
  if ! echo ":$PATH:" | grep -q ":${INSTALL_DIR}:"; then
    echo "PATH does not include ${INSTALL_DIR}. Add this line to your shell rc:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi
  echo "Run: ${BINARY} version"
}

main "$@"

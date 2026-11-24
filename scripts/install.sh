#!/usr/bin/env bash
set -euo pipefail

REPO="${NAMELESS_REPO:-heisdezz/nameless}"
VERSION="${VERSION:-latest}"

if [ "$(id -u)" -eq 0 ]; then
  DEFAULT_INSTALL_DIR="/usr/local/bin"
else
  DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
fi
INSTALL_DIR="${INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"

os="$(uname -s)"
if [ "$os" != "Linux" ]; then
  echo "Error: this install script is for Linux (detected: $os)." >&2
  exit 1
fi

arch_raw="$(uname -m)"
case "$arch_raw" in
  x86_64|amd64)
    arch="amd64"
    ;;
  aarch64|arm64)
    arch="arm64"
    ;;
  *)
    echo "Error: unsupported architecture '$arch_raw' (supported: x86_64, aarch64)." >&2
    exit 1
    ;;
esac

asset="nameless-linux-${arch}"
if [ "$VERSION" = "latest" ]; then
  url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
  case "$VERSION" in
    v*) tag="$VERSION" ;;
    *) tag="v$VERSION" ;;
  esac
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"
fi

tmp_file="$(mktemp)"
trap 'rm -f "$tmp_file"' EXIT

echo "Downloading nameless (${VERSION}, linux/${arch}) from ${url}..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp_file"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_file" "$url"
else
  echo "Error: curl or wget is required to download nameless." >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
chmod +x "$tmp_file"
mv "$tmp_file" "${INSTALL_DIR}/nameless"

echo "Installed nameless to ${INSTALL_DIR}/nameless"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo "Note: ${INSTALL_DIR} is not in your PATH. Add it with:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

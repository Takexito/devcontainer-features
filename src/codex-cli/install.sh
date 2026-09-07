#!/usr/bin/env bash
# Ставит Codex CLI статическим бинарником из GitHub Releases — без Node
# и npm: агентам Node ни к чему, а тащить его ради одного пакета жалко.
set -euo pipefail

VERSION="${VERSION:-latest}"

case "$(uname -m)" in
    x86_64)  ARCH=x86_64 ;;
    aarch64) ARCH=aarch64 ;;
    *) echo "codex-cli: неподдерживаемая архитектура $(uname -m)" >&2; exit 1 ;;
esac
ASSET="codex-${ARCH}-unknown-linux-musl"
if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/openai/codex/releases/latest/download/${ASSET}.tar.gz"
else
    URL="https://github.com/openai/codex/releases/download/rust-v${VERSION}/${ASSET}.tar.gz"
fi

if ! command -v curl >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y --no-install-recommends curl ca-certificates
    rm -rf /var/lib/apt/lists/*
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$URL" -o "$TMP/codex.tar.gz"
tar -xzf "$TMP/codex.tar.gz" -C "$TMP"
# В архиве один файл, названный по платформе.
install -m 0755 "$TMP/$ASSET" /usr/local/bin/codex
codex --version

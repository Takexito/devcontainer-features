#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-8.30.1}"

if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -fsSL https://api.github.com/repos/gitleaks/gitleaks/releases/latest \
        | sed -n 's/.*"tag_name": *"v\([^"]*\)".*/\1/p' | head -1)
    [ -n "$VERSION" ] || { echo "не удалось определить последнюю версию" >&2; exit 1; }
fi

case "$(uname -m)" in
    x86_64)  ARCH=x64 ;;
    aarch64) ARCH=arm64 ;;
    *) echo "неподдерживаемая архитектура: $(uname -m)" >&2; exit 1 ;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fsSL -o "$tmp/g.tar.gz" \
    "https://github.com/gitleaks/gitleaks/releases/download/v${VERSION}/gitleaks_${VERSION}_linux_${ARCH}.tar.gz"
tar -xzf "$tmp/g.tar.gz" -C "$tmp" gitleaks
install -m 0755 "$tmp/gitleaks" /usr/local/bin/gitleaks

gitleaks version

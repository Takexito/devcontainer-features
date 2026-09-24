#!/usr/bin/env bash
# Ставит glab готовым бинарником из официального GitLab Release. Архив
# проверяется по checksums.txt того же релиза до установки.
set -euo pipefail

VERSION="${VERSION:-1.116.0}"

case "$(uname -m)" in
    x86_64)  ARCH=amd64 ;;
    aarch64) ARCH=arm64 ;;
    *) echo "gitlab-cli: неподдерживаемая архитектура $(uname -m)" >&2; exit 1 ;;
esac

if ! command -v curl >/dev/null 2>&1 \
    || ! command -v tar >/dev/null 2>&1 \
    || ! command -v sha256sum >/dev/null 2>&1 \
    || [ ! -f /etc/ssl/certs/ca-certificates.crt ]; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y --no-install-recommends curl ca-certificates tar coreutils
    rm -rf /var/lib/apt/lists/*
fi

if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -fsSL \
        'https://gitlab.com/api/v4/projects/gitlab-org%2Fcli/releases/permalink/latest' \
        | sed -n 's/.*"tag_name":[[:space:]]*"v\([^"]*\)".*/\1/p')
    [ -n "$VERSION" ] || {
        echo "gitlab-cli: не удалось определить последнюю версию" >&2
        exit 1
    }
fi

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "gitlab-cli: некорректная версия: $VERSION" >&2
    exit 1
fi

ASSET="glab_${VERSION}_linux_${ARCH}.tar.gz"
URL="https://gitlab.com/gitlab-org/cli/-/releases/v${VERSION}/downloads"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL/$ASSET" -o "$TMP/$ASSET"
curl -fsSL "$URL/checksums.txt" -o "$TMP/checksums.txt"

EXPECTED=$(awk -v asset="$ASSET" '$2 == asset { print $1; exit }' "$TMP/checksums.txt")
[ -n "$EXPECTED" ] || {
    echo "gitlab-cli: для $ASSET нет checksum" >&2
    exit 1
}
ACTUAL=$(sha256sum "$TMP/$ASSET" | awk '{ print $1 }')
[ "$ACTUAL" = "$EXPECTED" ] || {
    echo "gitlab-cli: checksum $ASSET не совпал" >&2
    exit 1
}

tar -xzf "$TMP/$ASSET" -C "$TMP" bin/glab
install -m 0755 "$TMP/bin/glab" /usr/local/bin/glab

if [ -d /etc/bash_completion.d ]; then
    glab completion -s bash > /etc/bash_completion.d/glab
fi

glab --version

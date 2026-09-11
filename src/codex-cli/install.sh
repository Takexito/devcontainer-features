#!/usr/bin/env bash
# Ставит полный статический Codex bundle из GitHub Releases — без Node
# и npm. Полный package нужен не только CLI, но и companion-бинарникам
# вроде codex-code-mode-host и встроенным runtime-resources.
set -euo pipefail

VERSION="${VERSION:-latest}"

if [ "$VERSION" != "latest" ] \
    && [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-alpha(\.[0-9]+){0,2}|-beta(\.[0-9]+)?)?$ ]]; then
    echo "codex-cli: некорректная версия: $VERSION" >&2
    exit 1
fi

case "$(uname -m)" in
    x86_64)  ARCH=x86_64 ;;
    aarch64) ARCH=aarch64 ;;
    *) echo "codex-cli: неподдерживаемая архитектура $(uname -m)" >&2; exit 1 ;;
esac
TARGET="${ARCH}-unknown-linux-musl"
ASSET="codex-package-${TARGET}.tar.gz"
CHECKSUMS="codex-package_SHA256SUMS"
if [ "$VERSION" = "latest" ]; then
    BASE_URL="https://github.com/openai/codex/releases/latest/download"
else
    BASE_URL="https://github.com/openai/codex/releases/download/rust-v${VERSION}"
fi

if ! command -v curl >/dev/null 2>&1 \
    || ! command -v tar >/dev/null 2>&1 \
    || ! command -v sha256sum >/dev/null 2>&1 \
    || [ ! -f /etc/ssl/certs/ca-certificates.crt ]; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y --no-install-recommends curl ca-certificates tar coreutils
    rm -rf /var/lib/apt/lists/*
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$BASE_URL/$ASSET" -o "$TMP/$ASSET"
curl -fsSL "$BASE_URL/$CHECKSUMS" -o "$TMP/$CHECKSUMS"

EXPECTED=$(awk -v asset="$ASSET" '$2 == asset { print $1; exit }' "$TMP/$CHECKSUMS")
[ -n "$EXPECTED" ] || {
    echo "codex-cli: для $ASSET нет checksum" >&2
    exit 1
}
ACTUAL=$(sha256sum "$TMP/$ASSET" | awk '{ print $1 }')
[ "$ACTUAL" = "$EXPECTED" ] || {
    echo "codex-cli: checksum $ASSET не совпал" >&2
    exit 1
}

mkdir "$TMP/package"
tar -xzf "$TMP/$ASSET" -C "$TMP/package"
for path in \
    bin/codex \
    bin/codex-code-mode-host \
    codex-package.json \
    codex-path/rg; do
    [ -e "$TMP/package/$path" ] || {
        echo "codex-cli: в $ASSET нет $path" >&2
        exit 1
    }
done

chmod 0755 \
    "$TMP/package/bin/codex" \
    "$TMP/package/bin/codex-code-mode-host" \
    "$TMP/package/codex-path/rg"
if [ -f "$TMP/package/codex-resources/bwrap" ]; then
    chmod 0755 "$TMP/package/codex-resources/bwrap"
fi

# Сохраняем package layout: Codex ищет host и resources рядом с
# реальным путём своего executable. Обе команды в /usr/local/bin — ссылки
# в bundle, а не отдельные копии бинарников.
rm -rf /opt/codex
mv "$TMP/package" /opt/codex
ln -sfn /opt/codex/bin/codex /usr/local/bin/codex
ln -sfn /opt/codex/bin/codex-code-mode-host /usr/local/bin/codex-code-mode-host

codex --version
codex-code-mode-host --help >/dev/null

#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-263.4421.0}"
ROOT=/opt/kotlin-lsp
DEST="$ROOT/$VERSION"
USERNAME="${_REMOTE_USER:-root}"

case "$(uname -m)" in
    x86_64)  SUFFIX="" ;;
    aarch64) SUFFIX="-aarch64" ;;
    *) echo "неподдерживаемая архитектура: $(uname -m)" >&2; exit 1 ;;
esac

mkdir -p "$DEST"
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT

# Сборка тащит свой JRE, поэтому системный JDK не задействуется и не конфликтует.
curl -fsSL -o "$tmp/k.tar.gz" \
    "https://download.jetbrains.com/language-server/kotlin-server/${VERSION}/kotlin-server-${VERSION}${SUFFIX}.tar.gz"
tar -xzf "$tmp/k.tar.gz" -C "$tmp"
mv "$tmp/kotlin-server-${VERSION}"/* "$DEST/"

# Стабильный путь, чтобы конфиги редактора не переписывать при смене версии.
ln -sfn "$DEST" "$ROOT/current"

# Через группу, а не владельцем: updateRemoteUserUID перемаппит uid после
# сборки, и chown этап установки не переживёт.
if [ "$USERNAME" != "root" ]; then
    groupadd -f kotlin-lsp
    chgrp -R kotlin-lsp "$ROOT"
    chmod -R g+rwX "$ROOT"
    usermod -aG kotlin-lsp "$USERNAME"
fi

echo "Kotlin LSP: $ROOT/current/bin/intellij-server"

#!/usr/bin/env bash
# Ставит cargo-nextest готовым бинарником в $CARGO_HOME/bin — туда же, куда
# фича rust кладёт cargo, так что PATH уже настроен. cargo install собирал бы
# его из исходников несколько минут.
set -euo pipefail

VERSION="${VERSION:-latest}"
CARGO_HOME="${CARGO_HOME:-/usr/local/cargo}"

case "$(uname -m)" in
    x86_64)  PLATFORM=linux ;;
    aarch64) PLATFORM=linux-arm ;;
    *) echo "cargo-nextest: неподдерживаемая архитектура $(uname -m)" >&2; exit 1 ;;
esac
if [ ! -x "$CARGO_HOME/bin/cargo" ]; then
    echo "cargo-nextest: нет $CARGO_HOME/bin/cargo — добавь фичу ghcr.io/devcontainers/features/rust перед этой" >&2
    exit 1
fi

curl -fsSL "https://get.nexte.st/${VERSION}/${PLATFORM}" | tar -xzf - -C "$CARGO_HOME/bin"
chmod 0755 "$CARGO_HOME/bin/cargo-nextest"
"$CARGO_HOME/bin/cargo-nextest" nextest --version

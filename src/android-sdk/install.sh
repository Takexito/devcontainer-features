#!/usr/bin/env bash
set -euo pipefail

PLATFORMS="${PLATFORMS:-34}"
BUILDTOOLS="${BUILDTOOLS:-34.0.0}"
CMDLINE="${CMDLINETOOLSVERSION:-15859902}"
SDK=/opt/android-sdk
USERNAME="${_REMOTE_USER:-root}"

command -v java >/dev/null 2>&1 || {
    echo "java не найден — добавь ghcr.io/devcontainers/features/java перед этой фичей" >&2
    exit 1
}
command -v unzip >/dev/null 2>&1 || {
    apt-get update -qq && apt-get install -y -qq --no-install-recommends unzip
}

mkdir -p "$SDK/cmdline-tools"
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
curl -fsSL -o "$tmp/t.zip" \
    "https://dl.google.com/android/repository/commandlinetools-linux-${CMDLINE}_latest.zip"
unzip -q "$tmp/t.zip" -d "$tmp"
rm -rf "$SDK/cmdline-tools/latest"
mv "$tmp/cmdline-tools" "$SDK/cmdline-tools/latest"

export PATH="$SDK/cmdline-tools/latest/bin:$PATH"
yes | sdkmanager --sdk_root="$SDK" --licenses >/dev/null 2>&1 || true

pkgs=("platform-tools")
IFS=',' read -ra P <<< "$PLATFORMS"
for p in "${P[@]}"; do pkgs+=("platforms;android-${p// /}"); done
IFS=',' read -ra B <<< "$BUILDTOOLS"
for b in "${B[@]}"; do pkgs+=("build-tools;${b// /}"); done

echo "ставлю: ${pkgs[*]}"
sdkmanager --sdk_root="$SDK" --install "${pkgs[@]}" >/dev/null

# Каталог должен оставаться писабельным: sdkmanager дописывает пакеты уже из
# контейнера. Права даём ЧЕРЕЗ ГРУППУ, а не владельцем: devcontainer по умолчанию
# перемаппит uid пользователя под хостовой (updateRemoteUserUID), и владение,
# выставленное на этапе сборки, после этого перестаёт действовать. Членство
# в группе перемаппинг переживает.
if [ "$USERNAME" != "root" ]; then
    groupadd -f android-sdk
    chgrp -R android-sdk "$SDK"
    chmod -R g+rwX "$SDK"
    find "$SDK" -type d -exec chmod g+s {} +
    usermod -aG android-sdk "$USERNAME"
fi

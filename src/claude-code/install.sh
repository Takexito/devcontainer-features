#!/usr/bin/env bash
# Ставит Claude Code нативным установщиком — без Node и npm. Бинарник живёт
# в ~/.local/bin/claude пользователя, версии в ~/.local/share/claude;
# конфиг и авторизация — в ~/.claude, его удобно держать в общем томе.
set -euo pipefail

VERSION="${VERSION:-stable}"
USERNAME="${_REMOTE_USER:-root}"

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends curl ca-certificates
rm -rf /var/lib/apt/lists/*

# Установщик отказывается работать под sudo от обычного пользователя, чтобы
# бинарник не уехал в /root. Поэтому root ставим с явным разрешением, а
# пользователю — через su: HOME и владелец файлов будут его.
if [ "$USERNAME" = "root" ]; then
    curl -fsSL https://claude.ai/install.sh | CLAUDE_INSTALL_ALLOW_SUDO=1 bash -s -- "$VERSION"
    "$HOME/.local/bin/claude" --version
else
    su - "$USERNAME" -c "curl -fsSL https://claude.ai/install.sh | bash -s -- '$VERSION'"
    # shellcheck disable=SC2016 # $HOME должен раскрыться в шелле пользователя
    su - "$USERNAME" -c '"$HOME/.local/bin/claude" --version'
fi

#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-latest}"
USERNAME="${_REMOTE_USER:-root}"

command -v npm >/dev/null 2>&1 || {
    echo "npm не найден — добавь ghcr.io/devcontainers/features/node перед этой фичей" >&2
    exit 1
}

# Ставим от имени пользователя, а не root: фича node кладёт nvm в каталог,
# принадлежащий ему, и установка под root там падает.
if [ "$USERNAME" != "root" ]; then
    su - "$USERNAME" -c "npm install -g @openai/codex@${VERSION}"
else
    npm install -g "@openai/codex@${VERSION}"
fi

echo "codex установлен"

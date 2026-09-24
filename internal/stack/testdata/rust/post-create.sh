#!/usr/bin/env bash
# Сгенерировано devc, стек rust. Не править руками: файл пересоздаётся
# командой «devc up demo». Тулчейн уже в образе, здесь только то, что
# зависит от конкретного контейнера.
set -euo pipefail
WS="/workspaces/${PROJECT_NAME:?}"

# Свежий том docker принадлежит root. Рекурсивно чиним только чужие каталоги:
# общий cargo-registry или ~/.claude обходить на каждом старте незачем.
own() {
  for d in "$@"; do
    [ -e "$d" ] || continue
    [ "$(stat -c %u "$d")" = "$(id -u)" ] || sudo chown -R dev:dev "$d"
  done
}
own "/home/dev/.ssh" "/home/dev/.local/share/mise" "/home/dev/target" "/usr/local/cargo/registry" "/home/dev/.claude" "/home/dev/.codex" "/home/dev/.config/gh" "/home/dev/.config/glab-cli"
# Недостающих родителей точек монтирования docker тоже создаёт от root.
sudo chown dev:dev "$HOME/.local" "$HOME/.local/share" "$HOME/.config" 2>/dev/null || true
chmod 700 "$HOME/.ssh" 2>/dev/null || true

# sshd не наследует окружение контейнера: ssh-сессии берут PATH из
# /etc/environment, который devcontainer CLI заполняет до этого хука, а sshd
# читает через PAM. Добавляем туда бинарники пользователя и шимы mise.
if ! grep -q 'mise/shims' /etc/environment; then
  sudo sed -i 's|^PATH="|PATH="/home/dev/.local/bin:/home/dev/.local/share/mise/shims:|' /etc/environment
fi

# SSH приземляется в домашний каталог, а проект примонтирован в /workspaces.
ln -sfn "$WS" "$HOME/$PROJECT_NAME"

# Ключ хоста храним в томе проекта. Иначе при каждой пересборке он меняется,
# и клиент отказывается подключаться, приняв это за подмену.
KEYDIR="$HOME/.ssh/hostkeys"
mkdir -p "$KEYDIR"
if [ ! -f "$KEYDIR/ssh_host_ed25519_key" ]; then
  ssh-keygen -q -t ed25519 -N '' -C "$PROJECT_NAME" -f "$KEYDIR/ssh_host_ed25519_key"
fi
sudo chown root:root "$KEYDIR"/ssh_host_ed25519_key*
sudo chmod 600 "$KEYDIR/ssh_host_ed25519_key"

# Фича sshd оставляет включёнными пароли и root-логин; паролей нет, но
# выключаем явно. Второй источник ключей — authorized_keys хоста: новое
# устройство достаточно прописать на VPS, в контейнер дублировать не нужно.
{
  printf 'PasswordAuthentication no\nPermitRootLogin no\nKbdInteractiveAuthentication no\n'
  printf 'AuthorizedKeysFile .ssh/authorized_keys /etc/ssh/authorized_keys.host\n'
  printf 'HostKey %s/ssh_host_ed25519_key\n' "$KEYDIR"
} | sudo tee /etc/ssh/sshd_config.d/99-harden.conf >/dev/null
sudo pkill -HUP -x sshd 2>/dev/null || true

echo ">> $(mise --version)"
echo ">> $(rustc --version), $(cargo nextest --version | head -1)"
echo ">> $(rust-analyzer --version)"

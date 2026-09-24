#!/usr/bin/env bash
# Сгенерировано devc, стек android. Не править руками: файл пересоздаётся
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
own "/home/dev/.ssh" "/home/dev/.local/share/mise" "/home/dev/.gradle" "/home/dev/.konan" "/home/dev/.android" "/home/dev/.claude" "/home/dev/.codex" "/home/dev/.config/gh" "/home/dev/.config/glab-cli"
# Недостающих родителей точек монтирования docker тоже создаёт от root.
sudo chown dev:dev "$HOME/.local" "$HOME/.local/share" "$HOME/.config" 2>/dev/null || true
chmod 700 "$HOME/.ssh" 2>/dev/null || true

# sshd не наследует окружение контейнера: ssh-сессии берут PATH из
# /etc/environment, который devcontainer CLI заполняет до этого хука, а sshd
# читает через PAM. Добавляем туда бинарники пользователя и шимы mise.
if ! grep -q 'mise/shims' /etc/environment; then
  sudo sed -i 's|^PATH="|PATH="/home/dev/.local/bin:/home/dev/.local/share/mise/shims:|' /etc/environment
fi

# JDK из mise.toml проекта. Шимам хватает java в PATH, но JAVA_HOME они не
# выставляют, а Gradle-обвязки и IDE-серверы ищут JDK именно по нему. Ставим
# JDK сразу, а не при первом вызове, иначе JAVA_HOME указывать некуда.
if [ -n "$(cd "$WS" && mise current java 2>/dev/null)" ]; then
  (cd "$WS" && mise install -q java)
  JH=$(cd "$WS" && mise where java)
  sudo sed -i '/^JAVA_HOME=/d' /etc/environment
  echo "JAVA_HOME=\"$JH\"" | sudo tee -a /etc/environment >/dev/null
  echo ">> JAVA_HOME=$JH (из mise.toml)"
fi

# sdk.dir нужен Gradle; local.properties обычно в .gitignore
if [ -f "$WS/settings.gradle.kts" ] || [ -f "$WS/settings.gradle" ]; then
  touch "$WS/local.properties"
  grep -q '^sdk.dir=' "$WS/local.properties" || echo "sdk.dir=$ANDROID_HOME" >> "$WS/local.properties"
fi

# Kotlin LSP. Штатный сервер не умеет импортировать source sets KMP — модуль
# приезжает пустым. Пропатченная сборка лежит в общем томе dev-kotlin-lsp-kmp
# (собирается на хосте командой devc kmplsp) и заодно тянет обычные JVM-проекты,
# поэтому берём её по умолчанию. Патч прибит к версии сервера, так что сверяем
# её с образом: если том пуст или отстал после обновления LSP, откатываемся на
# штатный сервер — без KMP, но лучше, чем совсем без подсказок.
WANT=$(cat /opt/kotlin-lsp/current/build.txt 2>/dev/null || echo "?")
HAVE=$(sed -n 's/^server=//p' /opt/kmp/current/kmp-manifest.properties 2>/dev/null || true)
if [ -x /opt/kmp/current/bin/intellij-server ] && [ "$HAVE" = "$WANT" ]; then
  LSP=/opt/kmp/current
  LSP_KIND="KMP-сборка $WANT"
else
  LSP=/opt/kotlin-lsp/current
  LSP_KIND="штатный $WANT, без KMP — собери KMP-сборку на хосте: devc kmplsp"
  [ -n "$HAVE" ] && LSP_KIND="штатный $WANT: KMP-сборка в томе от $HAVE, пересобери: devc kmplsp"
fi
# Стабильный путь для редактора, чтобы .zed/settings.json не зависел от выбора.
if [ -e /opt/kotlin-lsp/kmp ] && [ ! -L /opt/kotlin-lsp/kmp ]; then
  echo ">> /opt/kotlin-lsp/kmp занят каталогом, ссылку не ставлю" >&2
else
  ln -sfn "$LSP" /opt/kotlin-lsp/kmp
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
echo ">> $(java -version 2>&1 | head -1)"
echo ">> Kotlin LSP: $LSP_KIND"

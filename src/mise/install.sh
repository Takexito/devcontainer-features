#!/usr/bin/env bash
# Ставит mise из apt-репозитория jdx — тем же способом, что на хосте, чтобы
# версии и поведение совпадали. Тулчейны сам не ставит: их тянет mise.toml
# проекта при первом заходе в каталог (auto_install), в ~/.local/share/mise
# пользователя — этот каталог devc держит в томе проекта.
set -euo pipefail

VERSION="${VERSION:-latest}"

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends curl gpg ca-certificates

install -d -m 0755 /etc/apt/keyrings
curl -fsSL https://mise.jdx.dev/gpg-key.pub | gpg --dearmor --yes -o /etc/apt/keyrings/mise-archive-keyring.gpg
echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.gpg arch=$(dpkg --print-architecture)] https://mise.jdx.dev/deb stable main" \
    > /etc/apt/sources.list.d/mise.list
apt-get update
if [ "$VERSION" = "latest" ]; then
    apt-get install -y --no-install-recommends mise
else
    apt-get install -y --no-install-recommends "mise=$VERSION"
fi
rm -rf /var/lib/apt/lists/*

# Шимы на PATH для login-шеллов (ssh, su -, bash -l). Именно шимы, а не
# mise activate: они работают и в неинтерактивных командах, на которых
# держится работа агентов.
cat > /etc/profile.d/mise.sh <<'EOF'
case ":$PATH:" in
  *"/.local/share/mise/shims:"*) ;;
  *) export PATH="$HOME/.local/share/mise/shims:$PATH" ;;
esac
EOF

mise --version

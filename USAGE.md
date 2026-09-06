# Дев-контейнеры на build01

Шпаргалка: развернуть проект, подключиться, снести.
netcup VPS · 8 vCPU · 15 GiB · Debian 13 · порты 2222+

## Новый проект

```sh
git clone <url> ~/projects/my-app     # или mkdir, если с нуля
newdev my-app                         # ~15 с, порт подберёт сам
```

На своей машине — обновить ssh-конфиг (нужно, только когда список проектов
изменился):

```sh
ssh vps cat .config/devhosts > ~/.ssh/config.d/devhosts
```

В Zed: `Ctrl+Alt+Shift+O` → `my-app` → путь `/workspaces/my-app`.

## Команды

| | |
|---|---|
| `newdev <name> [port]` | создать контейнер и запись в ssh-конфиге |
| `lsdev [-a]` | что запущено: порт, CPU, память |
| `rmdev <name> [-y]` | снести контейнер и тома проекта. Код остаётся |

## Разовая настройка

На клиенте — первой строкой в `~/.ssh/config`:

```
Include ~/.ssh/config.d/devhosts
```

В любом контейнере — авторизации общие для всех проектов:

```sh
gh auth login
claude
codex login --device-auth
```

## Что где лежит

| | |
|---|---|
| `~/projects/<name>` | код. Внутри контейнера — `/workspaces/<name>`, те же файлы |
| `ghcr.io/takexito/android-dev:1` | образ: JDK 17, Android SDK 34+35, Kotlin LSP, gitleaks, tmux, gh, агенты |
| `Takexito/devcontainer-features` | фичи и конфиг образа в `images/android` |
| `Takexito/dotfiles` | git-identity, шелл, tmux, pre-commit |
| `~/.config/devhosts` | накопленные блоки для ssh-конфига |

## Грабли

**Сборки Kotlin LSP протухают через 30 дней.** Симптом — `This build of
intellij-server has expired`. Поднять версию в фиче `kotlin-lsp`, пересобрать
образ. Текущая от 2026-09-06, следующая правка — начало октября. Ловушка:
свежие сборки иногда появляются в комментариях к issue раньше, чем в GitHub
Releases.

**Тома `dev-cargo`, `dev-claude`, `dev-codex`, `dev-gh` общие.** В них
авторизации: снесёшь один — разлогинятся все проекты. `rmdev` их не трогает
и отказывается работать с именами, начинающимися на `dev-`.

**`newdev` заточен под Android и KMP.** Для другого стека скопируй
`.devcontainer/` из `compose-preview-anywhere` — там свой Dockerfile вместо
готового образа.

## Пересобрать образ

```sh
cd ~/projects/devcontainer-features
devcontainer build \
  --workspace-folder images/android \
  --config images/android/devcontainer.json \
  --image-name ghcr.io/takexito/android-dev:1 \
  --image-name ghcr.io/takexito/android-dev:latest \
  --push
```

Нужен вход в реестр: `gh auth token | docker login ghcr.io -u Takexito --password-stdin`.

## Незакрыто

- **Лимитов памяти нет.** `cambridge-dictionary` в `gradle.properties` просит
  до 8 ГБ из 15 — разогнавшийся Gradle способен уронить хост со всеми
  контейнерами.
- **redroid не восстановлен** после уборки. При настройке первым делом
  закрепить модули `binder_linux` и `loop` через `/etc/modules-load.d`,
  иначе после перезагрузки не поднимется.

---

Контейнеры переживают перезагрузку: `--restart=unless-stopped` плюс docker
в автозагрузке. Проверено рестартом демона — поднялись за секунду.

Веб-версия: https://claude.ai/code/artifact/57d615ee-9186-499a-8d46-46b93091f922

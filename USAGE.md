# Дев-контейнеры на build01

Шпаргалка: развернуть проект, подключиться, снести.
netcup VPS · 8 vCPU · 15 GiB · Debian 13

## Новый проект

```sh
devc up my-app --stack rust --clone <url>   # или --init, если с нуля; ~15 с
```

Всё. Подключаться — `ssh my-app.dev`, дописывать ничего не надо. Каталог уже
есть — просто `devc up my-app --stack rust`. Стек запоминается в конфиге,
дальше хватает `devc up my-app`.

В Zed: `Ctrl+Alt+Shift+O` → `my-app.dev` → путь `/workspaces/my-app`.

## Команды

| | |
|---|---|
| `devc up <name> [--stack …]` | поднять контейнер. `--clone URL` или `--init` создают каталог, `--rebuild` пересоздаёт контейнер, `--mem 8g --cpus 6` меняют лимиты |
| `devc ls [-a]` | что запущено: стек, адрес, лимит, CPU, память |
| `devc exec <name> [cmd]` | зайти с хоста без ssh |
| `devc rm <name> [-y]` | снести контейнер и тома проекта. Код остаётся |
| `devc build <образ\|all> [--push]` | пересобрать образы |
| `devc prune [--builder]` | убрать мусорные образы `vsc-*` и кеш сборки |
| `devc kmplsp` | пересобрать KMP-сборку Kotlin LSP в общий том |

Исходники — `Takexito/devcontainer-features`; `make install` кладёт бинарник
в `~/.local/bin` вместе с симлинком `devproxy`.

## Стеки

| стек | образ | что внутри | тома проекта | общие тома |
|---|---|---|---|---|
| `base` | `base-dev` | Debian 13, mise, gh, tmux, rg, fd, build-essential, gitleaks, Claude Code, Codex | `ssh`, `mise` | `dev-claude`, `dev-codex`, `dev-gh` |
| `web` | `web-dev` | base + Node 22, pnpm, yarn | те же | + `dev-npm` |
| `rust` | `rust-dev` | base + rustup stable, rust-analyzer, clippy, rustfmt, cargo-nextest, libssl-dev | + `target` | + `dev-cargo-registry` |
| `android` | `android-dev` | base + JDK 17, Android SDK 34/35, Kotlin LSP | + `gradle`, `konan`, `android` | + `dev-kotlin-lsp-kmp` (только чтение) |

Языковых тулчейнов в `base` нет намеренно: положи в проект `mise.toml`, и mise
поставит нужный Python, Go или Node при первом заходе в каталог — в том
`<name>-mise`, который переживает пересборку. Rust исключение: он через rustup,
как и на хосте.

Тома проекта зовутся `<name>-<что>`: `my-app-ssh`, `my-app-target`. В `my-app-ssh`
лежит ключ хоста, поэтому пересборка не выглядит для клиента подменой.

Лимиты по умолчанию: base и web — 4g и 4 CPU, rust — 8g и 6, android — 10g и 6.
Это потолок против одного разогнавшегося Gradle, а не резервирование: сумма
по контейнерам больше хоста.

**Конфиг, написанный руками.** Если в проекте свой `.devcontainer/devcontainer.json`
без маркера `customizations.devc`, `devc up` его не трогает и просто поднимает —
так живут `compose-preview-anywhere` и `kotlin-lsp-kmp`. В `devc ls` у них стек
`custom`. `--force` перезапишет конфиг сгенерированным.

## Разовая настройка

На клиенте — одна запись в ssh-конфиге на все проекты, навсегда:

```
Host *.dev
    User dev
    ProxyCommand ssh vps devproxy %h
```

`devproxy` на сервере находит контейнер по имени и пробрасывает поток на его
sshd. Поэтому портов нет вообще: ни выбирать, ни помнить, ни бояться
столкновений. Ключ хоста лежит в томе проекта и переживает пересборку, так что
клиент не примет её за подмену.

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
| `ghcr.io/takexito/<стек>-dev:1` | образы `base-dev`, `web-dev`, `rust-dev`, `android-dev` — см. «Стеки» |
| `Takexito/devcontainer-features` | `devc`, фичи и конфиги образов в `images/` |
| `Takexito/dotfiles` | git-identity, шелл, tmux, pre-commit |
| `Takexito/kotlin-lsp-kmp` | патч Kotlin LSP под KMP; том `dev-kotlin-lsp-kmp` |

## Kotlin LSP и KMP

Штатный сервер JetBrains не импортирует source sets KMP: модуль приезжает
пустым, и в `commonMain` нет ничего. Поэтому по умолчанию используется
пропатченная сборка из `Takexito/kotlin-lsp-kmp` — каждый source set сохраняет
свои roots, платформу и зависимости, работают common/JVM/Android/Native
(включая iOS), обычные JVM-проекты тоже. JS и Wasm отвергаются явно.

Живёт она в общем томе `dev-kotlin-lsp-kmp`, смонтированном в проекты только
на чтение как `/opt/kmp`. Один экземпляр на все проекты: 1.1 ГБ вместо копии
на каждый. `post-create` ставит ссылку `/opt/kotlin-lsp/kmp` на выбранный
сервер, и `.zed/settings.json` указывает на неё — то есть путь для редактора
один и тот же независимо от того, какой сервер выбран.

Патч прибит к конкретной сборке сервера и сверяется по хешам пяти JAR-ов.
`post-create` сравнивает версию в томе с версией в образе: не сошлось — молча
откатывается на штатный сервер и пишет, что делать. Совсем без подсказок
Kotlin не остаётся никогда.

Собрать или пересобрать том: `devc kmplsp`. Установщик патча написан на Node,
которого в `android-dev` больше нет, поэтому `devc kmplsp` берёт его через
`mise x node@22` в том `kotlin-lsp-kmp-mise`. Внутри `~/projects/kotlin-lsp-kmp`
том подключён на запись, поэтому там короче — `./install.sh /opt/kotlin-lsp/current
/opt/kmp/current`; Node там даёт `mise.toml` репозитория. Копию для отката
потом уберёт `devc kmplsp`.

## Грабли

**Сборки Kotlin LSP протухают через 30 дней.** Симптом — `This build of
intellij-server has expired`. Ритуал: поднять версию в фиче `kotlin-lsp` →
`devc build android --push` → поднять пин в `kotlin-lsp-kmp` → `devc kmplsp`.
Пропустишь последние два шага — проекты сами откатятся на штатный сервер и KMP
отвалится. Текущая от 2026-09-06, следующая правка — начало октября. Ловушка:
свежие сборки иногда появляются в комментариях к issue раньше, чем в GitHub
Releases.

**Тома `dev-*` общие.** В них авторизации агентов и gh, реестр crates, кеш npm
и KMP-сборка: снесёшь `dev-claude` — разлогинятся все проекты. `devc rm`
их не трогает и отказывается работать с именами, начинающимися на `dev-`.

**Переменные для ssh-сессий приходят из `/etc/environment`**, а не из окружения
контейнера: devcontainer CLI заполняет его при создании, sshd читает через PAM.
`post-create` добавляет туда `~/.local/bin` и шимы mise. Поменял `containerEnv`
или лимиты — нужен `devc up <name> --rebuild`, без него правка лежит в файле и
ждёт.

**В `rust-dev` `cargo install` и `cargo login` живут в слое контейнера** и
теряются при `--rebuild`. Реестр crates — в общем томе, его не теряем.
Нужен инструмент навсегда — добавь его в `images/rust` и пересобери образ.

**Агенты стоят нативными бинарниками**: `claude` в `~/.local/bin`, `codex`
в `/usr/local/bin`. Node в `base` нет; нужен `npx` — `mise use node@22`
в проекте или стек `web`.

## Пересобрать образы

```sh
cd ~/projects/devcontainer-features
devc build all --push          # base → web → rust → android, порядок важен
```

Нужен вход в реестр: `gh auth token | docker login ghcr.io -u Takexito --password-stdin`.
Образы ссылаются на локальный тег `base-dev:1`, поэтому `web`, `rust` и `android`
собираются только после `base`. Фичи, на которые ссылаются образы, публикуются
отдельно и заранее:

```sh
devcontainer features publish -r ghcr.io -n takexito/devcontainer-features ./src
```

Уборка после сборок и тестов фич — `devc prune --builder`: он считает место
по `du`, а не по `docker system df`, и никогда не трогает `ghcr.io/takexito/*`.

## Незакрыто

- **redroid не восстановлен** после уборки. При настройке первым делом
  закрепить модули `binder_linux` и `loop` через `/etc/modules-load.d`,
  иначе после перезагрузки не поднимется.

---

Контейнеры переживают перезагрузку: `--restart=unless-stopped` плюс docker
в автозагрузке. Проверено рестартом демона — поднялись за секунду.

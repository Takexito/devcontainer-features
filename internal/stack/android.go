package stack

// Фрагмент post-create для Android и KMP. Перенесён из newdev без изменений
// логики: sdk.dir для Gradle и выбор сборки Kotlin LSP.
const androidPostCreate = `# sdk.dir нужен Gradle; local.properties обычно в .gitignore
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
`

// Zed ходит по стабильной ссылке /opt/kotlin-lsp/kmp, которую ставит
// post-create, поэтому путь один и тот же при любом выбранном сервере.
const androidZed = `{
  "languages": {
    "Kotlin": {
      "language_servers": ["kotlin-lsp", "!kotlin-language-server", "..."]
    }
  },
  "lsp": {
    "kotlin-lsp": {
      "binary": {
        "path": "/opt/kotlin-lsp/kmp/bin/intellij-server",
        "arguments": ["--stdio"]
      }
    }
  }
}
`

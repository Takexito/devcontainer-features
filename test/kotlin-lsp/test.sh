#!/usr/bin/env bash
set -e
source dev-container-features-test-lib

check "бинарник на месте"      bash -lc "[ -x /opt/kotlin-lsp/current/bin/intellij-server ]"
check "свой JRE в комплекте"   bash -lc "[ -d /opt/kotlin-lsp/current/jbr ]"
check "симлинк current ведёт на версию" bash -lc "readlink /opt/kotlin-lsp/current | grep -qE '[0-9]+\.[0-9]+'"
check "каталог писабелен"      bash -lc "touch /opt/kotlin-lsp/.probe && rm /opt/kotlin-lsp/.probe"
reportResults

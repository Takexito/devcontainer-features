#!/bin/bash
set -e
source dev-container-features-test-lib

check "codex стоит" codex --version
check "полный Codex bundle" test -f /opt/codex/codex-package.json
check "Code Mode host стоит" test -x /opt/codex/bin/codex-code-mode-host
check "Code Mode host запускается" codex-code-mode-host --help
check "codex запускается из bundle" \
    test "$(readlink -f "$(command -v codex)")" = /opt/codex/bin/codex
check "codex — статический бинарник, не npm" bash -c '! command -v npm >/dev/null'

reportResults

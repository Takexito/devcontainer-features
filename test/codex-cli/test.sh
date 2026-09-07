#!/bin/bash
set -e
source dev-container-features-test-lib

check "codex стоит" codex --version
check "codex — статический бинарник, не npm" bash -c '! command -v npm >/dev/null'

reportResults

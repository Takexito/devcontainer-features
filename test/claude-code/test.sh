#!/bin/bash
set -e
source dev-container-features-test-lib

check "claude в ~/.local/bin пользователя" test -x "$HOME/.local/bin/claude"
check "claude запускается" "$HOME/.local/bin/claude" --version

reportResults

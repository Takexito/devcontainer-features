#!/bin/bash
set -e
source dev-container-features-test-lib

check "mise стоит" mise --version
check "шимы в PATH login-шелла" bash -lc 'case ":$PATH:" in *"/.local/share/mise/shims:"*) exit 0 ;; *) echo "$PATH"; exit 1 ;; esac'
check "mise работает от пользователя" bash -lc 'mise ls'

reportResults

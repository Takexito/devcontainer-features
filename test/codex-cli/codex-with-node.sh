#!/usr/bin/env bash
set -e
source dev-container-features-test-lib

check "бинарник на месте" bash -lc "command -v codex"
check "версия выводится"  bash -lc "codex --version | grep -qi codex"
reportResults

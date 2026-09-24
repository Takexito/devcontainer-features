#!/usr/bin/env bash
set -e
source dev-container-features-test-lib

check "glab стоит" glab --version
check "установлена закреплённая версия" bash -c "glab --version | grep -q '1.116.0'"
check "glab доступен пользователю" bash -lc 'command -v glab'

reportResults

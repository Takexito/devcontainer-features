#!/usr/bin/env bash
set -e
source dev-container-features-test-lib

check "бинарник на месте" bash -c "command -v gitleaks"
check "версия выводится"  bash -c "gitleaks version | grep -qE '^[0-9]+\.[0-9]+'"
check "находит секрет"    bash -c '
  d=$(mktemp -d); cd "$d"; git init -q .
  printf "token = \"ghp_%s\"\n" "0a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r" > c.txt
  git add c.txt
  ! gitleaks protect --staged --redact --no-banner
'
reportResults

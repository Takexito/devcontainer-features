#!/bin/bash
set -e
source dev-container-features-test-lib

check "cargo nextest доступен через cargo" bash -lc 'cargo nextest --version'

reportResults

#!/usr/bin/env bash
set -e
source dev-container-features-test-lib

check "ANDROID_HOME выставлен" bash -lc '[ -n "$ANDROID_HOME" ]'
check "sdkmanager в PATH"      bash -lc "command -v sdkmanager"
check "платформа 34 стоит"     bash -lc '[ -d "$ANDROID_HOME/platforms/android-34" ]'
check "build-tools стоят"      bash -lc '[ -d "$ANDROID_HOME/build-tools/34.0.0" ]'
check "adb на месте"           bash -lc "command -v adb"
check "каталог писабелен"      bash -lc 'touch "$ANDROID_HOME/.probe" && rm "$ANDROID_HOME/.probe"'
reportResults

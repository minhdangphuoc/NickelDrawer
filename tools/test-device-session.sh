#!/bin/bash
# Exercises tools/device-session.sh against a fake Kobo tree. Never touches a real volume.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
TOOL=$REPO_ROOT/tools/device-session.sh
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
export KOBO_ROOT=$WORK/KOBOeReader
export BACKUP_ROOT=$WORK/backups
FLAGS=$KOBO_ROOT/.adds/nickeldrawer
failures=0

check() {
    local description=$1
    shift
    if "$@"; then echo "ok   - $description"; else echo "FAIL - $description"; failures=$((failures + 1)); fi
}
fails() { ! "$@" >/dev/null 2>&1; }

makeTgz() { # name entry...
    local name=$1
    shift
    mkdir -p "$WORK/pack"
    (cd "$WORK/pack" && rm -rf ./* && for e in "$@"; do mkdir -p "$(dirname "$e")" && : >"$e"; done &&
        tar czf "$WORK/$name" .)
}

check "refuses without .kobo" fails "$TOOL" status
mkdir -p "$KOBO_ROOT/.kobo/Kobo" "$FLAGS" "$KOBO_ROOT/.adds/nm"
echo "N1,4.38.1,4.38.23828,x" >"$KOBO_ROOT/.kobo/version"
printf 'AvailableQt6Upgrade=http://x\r\n' >"$KOBO_ROOT/.kobo/Kobo/Kobo eReader.conf"
echo item >"$KOBO_ROOT/.adds/nm/config"

touch "$FLAGS/proto-a" "$FLAGS/uninstall-proto"
out=$("$TOOL" status)
check "status shows firmware" grep -q "4.38.23828" <<<"$out"
check "status warns on pending qt6" grep -q "PENDING" <<<"$out"
check "status lists flags" grep -q "proto-a" <<<"$out"

makeTgz good.tgz usr/local/Kobo/imageformats/libx.so
makeTgz bad.tgz usr/local/Kobo/imageformats/libx.so etc/passwd
check "stage rejects foreign paths" fails "$TOOL" stage "$WORK/bad.tgz"
check "bad tgz not staged" test ! -e "$KOBO_ROOT/.kobo/KoboRoot.tgz"
check "stage accepts good tgz" "$TOOL" stage "$WORK/good.tgz"
check "tgz staged" test -f "$KOBO_ROOT/.kobo/KoboRoot.tgz"
check "stage refuses second tgz" fails "$TOOL" stage "$WORK/good.tgz"
check "backup holds nm" bash -c "ls $BACKUP_ROOT/*/nm/config"

"$TOOL" flags proto-a debug >/dev/null
touch "$FLAGS/proto-c"
"$TOOL" flags proto-a debug >/dev/null
check "flags keeps requested" test -f "$FLAGS/debug"
check "flags removes others" test ! -e "$FLAGS/proto-c"
check "flags keeps uninstall*" test -f "$FLAGS/uninstall-proto"

echo "line" >"$FLAGS/debug.log"
check "log prints" grep -q line < <("$TOOL" log 2>/dev/null)
check "log removed from device" test ! -e "$FLAGS/debug.log"
check "log saved" bash -c "ls $BACKUP_ROOT/*/debug.log"

touch "$KOBO_ROOT/.adds/nm/._config" "$KOBO_ROOT/.kobo/._version"
"$TOOL" clean >/dev/null
check "clean removes ._ files" test ! -e "$KOBO_ROOT/.adds/nm/._config" -a ! -e "$KOBO_ROOT/.kobo/._version"
check "clean keeps real files" test -f "$KOBO_ROOT/.kobo/version"

if command -v shellcheck >/dev/null; then
    check "shellcheck" shellcheck "$TOOL"
else
    echo "skip - shellcheck not installed"
fi
[ "$failures" -eq 0 ] || { echo "$failures failed"; exit 1; }
echo "all passed"

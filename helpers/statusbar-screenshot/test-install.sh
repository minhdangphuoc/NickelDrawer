#!/bin/bash
# Exercises helpers/statusbar-screenshot/install.sh against a fake Kobo tree in a temp dir. Never
# touches a real volume. Needs the icon (go run ./cmd/iconsgen -repo .) but not a built library.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
INSTALLER=$REPO_ROOT/helpers/statusbar-screenshot/install.sh
WORK=$(mktemp -d)
export KOBO_ROOT=$WORK/KOBOeReader
export BACKUP_ROOT=$WORK/backups
export HELPER_TGZ=$WORK/KoboRoot-statusbarshot.tgz
TARGET=$KOBO_ROOT/.adds/helpers/screenshot
STAGED=$KOBO_ROOT/.kobo/KoboRoot.tgz
ARCHIVE_SOURCE=$WORK/archive-src
failures=0

trap 'rm -rf "$WORK"' EXIT

check() {
    local description=$1
    shift
    if "$@"; then
        echo "ok   - $description"
    else
        echo "FAIL - $description"
        failures=$((failures + 1))
    fi
}

contains() { grep -qF -- "$2" "$1"; }

# A stand-in archive holding the one path the real one holds.
make_archive() {
    rm -rf "$ARCHIVE_SOURCE"
    mkdir -p "$ARCHIVE_SOURCE/usr/local/Kobo/imageformats"
    echo fake-library >"$ARCHIVE_SOURCE/usr/local/Kobo/imageformats/libstatusbarshot.so"
    tar czf "$HELPER_TGZ" -C "$ARCHIVE_SOURCE" usr
}

mkdir -p "$KOBO_ROOT/.kobo"
make_archive

echo "--- dry run"
"$INSTALLER" --dry-run >"$WORK/dry.out"
check "dry run prints actions" contains "$WORK/dry.out" "[dry-run]"
check "dry run writes no helper folder" test ! -e "$TARGET"
check "dry run stages nothing" test ! -e "$STAGED"

echo "--- install"
mkdir -p "$TARGET"
touch "$TARGET/off" "$TARGET/disable"
"$INSTALLER" >"$WORK/install.out"
check "the library is staged as KoboRoot.tgz" cmp -s "$HELPER_TGZ" "$STAGED"
check "the English locale is installed" test -f "$TARGET/locales/en.json"
check "the Vietnamese locale is installed" test -f "$TARGET/locales/vi.json"
check "the icon is installed" test -f "$TARGET/icons/screenshot-64.png"
check "the Tabler notice is installed" test -f "$TARGET/LICENSES/tabler.txt"
check "the user's off flag stays" test -e "$TARGET/off"
check "the user's disable flag stays" test -e "$TARGET/disable"
check "the previous folder was backed up" test -e "$BACKUP_ROOT"/*/.adds/helpers/screenshot/off
check "install says to reboot" contains "$WORK/install.out" "reboot"
check "nothing of the drawer is touched" test ! -e "$KOBO_ROOT/.adds/nickeldrawer"

echo "--- refusals"
before=$(cksum <"$STAGED")
"$INSTALLER" >"$WORK/waiting.out" 2>&1 && waitingStatus=0 || waitingStatus=$?
check "refuses while a KoboRoot.tgz is waiting" test "$waitingStatus" -ne 0
check "...and says why" contains "$WORK/waiting.out" "already waiting"
check "...and leaves the waiting archive alone" test "$(cksum <"$STAGED")" = "$before"
rm -f "$STAGED"
echo "extra" >"$ARCHIVE_SOURCE/etc-passwd"
tar czf "$HELPER_TGZ" -C "$ARCHIVE_SOURCE" usr etc-passwd
"$INSTALLER" >"$WORK/bad.out" 2>&1 && badStatus=0 || badStatus=$?
check "refuses an archive with more than a mod library" test "$badStatus" -ne 0
check "...and stages nothing" test ! -e "$STAGED"
KOBO_ROOT=$WORK/nothing "$INSTALLER" >"$WORK/missing.out" 2>&1 && missingStatus=0 || missingStatus=$?
check "refuses a volume without .kobo" test "$missingStatus" -ne 0
HELPER_TGZ=$WORK/missing.tgz "$INSTALLER" >"$WORK/nobuild.out" 2>&1 && nobuildStatus=0 || nobuildStatus=$?
check "refuses without a built library" test "$nobuildStatus" -ne 0
check "...and says how to build it" contains "$WORK/nobuild.out" "docker-build"

echo "--- a stale uninstall flag"
make_archive
touch "$TARGET/uninstall"
"$INSTALLER" >/dev/null
check "installing clears it" test ! -e "$TARGET/uninstall"
rm -f "$STAGED"

echo "--- uninstall"
"$INSTALLER" --uninstall >"$WORK/uninstall.out"
check "writes NickelHook's uninstall flag" test -e "$TARGET/uninstall"
check "stages no archive" test ! -e "$STAGED"
check "says to reboot" contains "$WORK/uninstall.out" "reboot"
echo "--- off / on"
"$INSTALLER" --off >"$WORK/off.out"
check "--off creates the off flag" test -e "$TARGET/off"
check "--off backs the folder up first" test -d "$BACKUP_ROOT"
"$INSTALLER" --on >"$WORK/on.out"
check "--on removes the off flag" test ! -e "$TARGET/off"
check "--on stages no archive" test ! -e "$STAGED"
rm -rf "$TARGET"
"$INSTALLER" --off --dry-run >/dev/null
check "--off --dry-run writes nothing" test ! -e "$TARGET"
"$INSTALLER" --nope >/dev/null 2>&1 && flagStatus=0 || flagStatus=$?
check "an unknown flag is refused" test "$flagStatus" -ne 0

if [ "$failures" -ne 0 ]; then
    echo "$failures check(s) failed"
    exit 1
fi
echo "all checks passed"

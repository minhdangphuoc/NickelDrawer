#!/bin/bash
# Exercises helpers/extras/install.sh against fake Kobo trees in a temp dir. Never touches a real
# volume. Needs the generated icons (go run ./cmd/iconsgen -repo .) but not a built library: a stand-in
# archive and binary are used. The drawer-only and both-together sections also run
# tools/install-to-kobo.sh and need `make build dist`; without dist/ they are skipped with a note.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
INSTALLER=$REPO_ROOT/helpers/extras/install.sh
DRAWER_INSTALLER=$REPO_ROOT/tools/install-to-kobo.sh
WORK=$(mktemp -d "${TMPDIR:-/tmp}/extras-install-test.XXXXXX")
export BACKUP_ROOT=$WORK/backups
export HELPER_TGZ=$WORK/KoboRoot-extras.tgz
export TWEAKS_BINARY=$WORK/extras-tweaks
DRAWER_TGZ=$WORK/KoboRoot-nickeldrawer.tgz
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
    local library=$1 output=$2
    rm -rf "$ARCHIVE_SOURCE"
    mkdir -p "$ARCHIVE_SOURCE/usr/local/Kobo/imageformats"
    echo "fake-$library" >"$ARCHIVE_SOURCE/usr/local/Kobo/imageformats/$library"
    tar czf "$output" -C "$ARCHIVE_SOURCE" usr
}

# A fresh Kobo; the names say what it holds.
new_kobo() {
    export KOBO_ROOT=$WORK/$1/KOBOeReader
    TARGET=$KOBO_ROOT/.adds/helpers/extras
    STAGED=$KOBO_ROOT/.kobo/KoboRoot.tgz
    mkdir -p "$KOBO_ROOT/.kobo/Kobo"
    printf '[ApplicationPreferences]\nCurrentLocale=vi\n' >"$KOBO_ROOT/.kobo/Kobo/Kobo eReader.conf"
}

reboot_kobo() { rm -f "$STAGED"; }

make_archive libextras.so "$HELPER_TGZ"
make_archive libnickeldrawer.so "$DRAWER_TGZ"
printf '#!/bin/sh\necho stand-in\n' >"$TWEAKS_BINARY"

echo "--- helper only: dry run and install"
new_kobo helper-only
"$INSTALLER" --dry-run >"$WORK/dry.out"
check "dry run prints actions" contains "$WORK/dry.out" "[dry-run]"
check "dry run writes no helper folder" test ! -e "$TARGET"
check "dry run stages nothing" test ! -e "$STAGED"
mkdir -p "$TARGET"
touch "$TARGET/screenshot-off" "$TARGET/disable"
"$INSTALLER" >"$WORK/install.out"
check "the library is staged as KoboRoot.tgz" cmp -s "$HELPER_TGZ" "$STAGED"
check "the tweaks binary is installed and executable" test -x "$TARGET/bin/extras-tweaks"
check "the English locale is installed" test -f "$TARGET/locales/en.json"
check "the Vietnamese locale is installed" test -f "$TARGET/locales/vi.json"
check "no Go source reaches the Kobo" test -z "$(find "$TARGET" -name '*.go')"
check "the screenshot icon is installed" test -f "$TARGET/icons/screenshot.png"
check "the tweaks icon is installed" test -f "$TARGET/icons/tweaks.png"
check "the Tabler notice is installed" test -f "$TARGET/LICENSES/tabler.txt"
check "the user's screenshot-off flag stays" test -e "$TARGET/screenshot-off"
check "the user's disable flag stays" test -e "$TARGET/disable"
check "the previous folder was backed up" test -e "$BACKUP_ROOT"/*/.adds/helpers/extras/screenshot-off
check "install says to reboot" contains "$WORK/install.out" "reboot"
check "no drawer folder appears" test ! -e "$KOBO_ROOT/.adds/koboapps"
check "no drawer flag folder appears" test ! -e "$KOBO_ROOT/.adds/nickeldrawer"

echo "--- refusals"
before=$(cksum <"$STAGED")
"$INSTALLER" >"$WORK/waiting.out" 2>&1 && waitingStatus=0 || waitingStatus=$?
check "refuses while a KoboRoot.tgz is waiting" test "$waitingStatus" -ne 0
check "...and says why" contains "$WORK/waiting.out" "already waiting"
check "...and leaves the waiting archive alone" test "$(cksum <"$STAGED")" = "$before"
reboot_kobo
echo "extra" >"$ARCHIVE_SOURCE/etc-passwd"
tar czf "$WORK/bad.tgz" -C "$ARCHIVE_SOURCE" usr etc-passwd
HELPER_TGZ=$WORK/bad.tgz "$INSTALLER" >"$WORK/bad.out" 2>&1 && badStatus=0 || badStatus=$?
check "refuses an archive with more than a mod library" test "$badStatus" -ne 0
check "...and stages nothing" test ! -e "$STAGED"
KOBO_ROOT=$WORK/nothing "$INSTALLER" >"$WORK/missing.out" 2>&1 && missingStatus=0 || missingStatus=$?
check "refuses a volume without .kobo" test "$missingStatus" -ne 0
HELPER_TGZ=$WORK/missing.tgz "$INSTALLER" >"$WORK/nobuild.out" 2>&1 && nobuildStatus=0 || nobuildStatus=$?
check "refuses without a built library" test "$nobuildStatus" -ne 0
check "...and says how to build it" contains "$WORK/nobuild.out" "docker-build"
TWEAKS_BINARY=$WORK/missing-binary "$INSTALLER" >"$WORK/nobinary.out" 2>&1 && nobinaryStatus=0 || nobinaryStatus=$?
check "refuses without the tweaks binary" test "$nobinaryStatus" -ne 0
"$INSTALLER" --nope >/dev/null 2>&1 && flagStatus=0 || flagStatus=$?
check "an unknown flag is refused" test "$flagStatus" -ne 0
"$INSTALLER" --off >/dev/null 2>&1 && offStatus=0 || offStatus=$?
check "the retired --off flag is refused" test "$offStatus" -ne 0

echo "--- a stale uninstall flag"
touch "$TARGET/uninstall"
"$INSTALLER" >/dev/null
check "installing clears it" test ! -e "$TARGET/uninstall"
reboot_kobo

echo "--- helper only: uninstall"
"$INSTALLER" --uninstall >"$WORK/uninstall.out"
check "writes NickelHook's uninstall flag" test -e "$TARGET/uninstall"
check "stages no archive" test ! -e "$STAGED"
check "says to reboot" contains "$WORK/uninstall.out" "reboot"
check "keeps the files for the user to delete" test -x "$TARGET/bin/extras-tweaks"

echo "--- the retired status-bar helper"
new_kobo old-helper
OLD=$KOBO_ROOT/.adds/helpers/screenshot
mkdir -p "$OLD/locales" "$OLD/icons" "$OLD/LICENSES"
touch "$OLD/off" "$OLD/disable" "$OLD/debug" "$OLD/locales/en.json" "$OLD/icons/screenshot-64.png" "$OLD/LICENSES/tabler.txt"
"$INSTALLER" >"$WORK/migrate.out"
check "the old uninstall flag is written" test -e "$OLD/uninstall"
check "the old locales are gone" test ! -e "$OLD/locales"
check "the old icons are gone" test ! -e "$OLD/icons"
check "the old flag files are gone" test ! -e "$OLD/off" -a ! -e "$OLD/disable" -a ! -e "$OLD/debug"
check "the old folder was backed up" test -e "$BACKUP_ROOT"/*/.adds/helpers/screenshot/off
check "the new helper is staged in the same run" cmp -s "$HELPER_TGZ" "$STAGED"
check "the message names the old library" contains "$WORK/migrate.out" "libstatusbarshot.so"
new_kobo no-old-helper
"$INSTALLER" >/dev/null
check "without the old folder nothing of it is created" test ! -e "$KOBO_ROOT/.adds/helpers/screenshot"

# The drawer installer refuses a missing or stale dist/; then its sections are skipped, loudly.
drawer_is_testable() {
    [ -d "$REPO_ROOT/dist/onboard/.adds" ] || return 1
    new_kobo drawer-probe
    MOD_TGZ=$DRAWER_TGZ "$DRAWER_INSTALLER" --dry-run >/dev/null 2>&1
}

if drawer_is_testable; then
    echo "--- drawer only: it creates nothing of the helper"
    new_kobo drawer-only
    export MOD_TGZ=$DRAWER_TGZ
    "$DRAWER_INSTALLER" >/dev/null
    check "the drawer is installed" test -x "$KOBO_ROOT/.adds/koboapps/bin/appsctl"
    check "the drawer stages its own library" cmp -s "$DRAWER_TGZ" "$STAGED"
    check "no helper folder exists" test ! -e "$KOBO_ROOT/.adds/helpers"
    check "the staged archive has no helper library" test -z "$(tar tzf "$STAGED" | grep -i extras || true)"

    for order in extras-first drawer-first; do
        echo "--- both together ($order)"
        new_kobo "both-$order"
        install_extras() { "$INSTALLER" >/dev/null; }
        install_drawer() { "$DRAWER_INSTALLER" >/dev/null; }
        if [ "$order" = extras-first ]; then first=install_extras second=install_drawer; else first=install_drawer second=install_extras; fi
        "$first"
        check "the first install is staged" test -f "$STAGED"
        sleep 1
        reboot_kobo
        "$second"
        check "the second install is staged after the reboot" test -f "$STAGED"
        check "the helper folder is complete" test -x "$TARGET/bin/extras-tweaks"
        check "the drawer is complete" test -x "$KOBO_ROOT/.adds/koboapps/bin/appsctl"
        check "the drawer wrote nothing into the helper folder" test -z "$(find "$TARGET" -newer "$HELPER_TGZ" -name 'appsctl*')"
        check "the helper wrote nothing into the drawer's flag folder" test ! -e "$KOBO_ROOT/.adds/nickeldrawer/screenshot-off"

        echo "--- both together ($order): each uninstall leaves the other alone"
        reboot_kobo
        sleep 1
        "$INSTALLER" --uninstall >/dev/null
        check "extras uninstall flag is written" test -e "$TARGET/uninstall"
        check "extras uninstall leaves the drawer's files" test -x "$KOBO_ROOT/.adds/koboapps/bin/appsctl"
        check "extras uninstall writes no drawer uninstall flag" test ! -e "$KOBO_ROOT/.adds/nickeldrawer/uninstall"
        rm -f "$TARGET/uninstall"
        sleep 1
        "$DRAWER_INSTALLER" --uninstall --purge >/dev/null
        check "drawer uninstall flag is written" test -e "$KOBO_ROOT/.adds/nickeldrawer/uninstall"
        check "drawer uninstall leaves the helper folder" test -x "$TARGET/bin/extras-tweaks"
        check "drawer uninstall writes no helper uninstall flag" test ! -e "$TARGET/uninstall"
        check "drawer uninstall leaves the helper's locales" test -f "$TARGET/locales/en.json"
    done
else
    echo "--- SKIPPED: the drawer-only and both-together sections need a fresh dist/ (run: make check build dist)"
fi

if [ "$failures" -ne 0 ]; then
    echo "$failures check(s) failed"
    exit 1
fi
echo "all checks passed"

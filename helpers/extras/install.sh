#!/bin/bash
# Installs or removes the Extras helper (status-bar screenshot button and the Tweaks panel) on a
# mounted Kobo. Maintainer-built: the helper is not part of tools/install-to-kobo.sh, dist/ or the releases.
#   helpers/extras/install.sh [--dry-run]
#   helpers/extras/install.sh --uninstall [--dry-run]
# Env: KOBO_ROOT, the mounted Kobo (default: found like tools/install-to-kobo.sh does);
#      BACKUP_ROOT (default <repo>/backups); HELPER_TGZ (default build/helpers/KoboRoot-extras.tgz);
#      TWEAKS_BINARY (default build/helpers/extras-tweaks-linux-armv7).
# Run `make build`, `make -C helpers/extras docker-build` and `helpers/extras/check-symbols.sh` first.
# Never cd into the volume: it would block ejecting.
#
# Installing copies the tweaks binary, the locale files, the icons and the Tabler notice to
# .adds/helpers/extras (the user's flag files there stay), then stages the library as
# .kobo/KoboRoot.tgz; Nickel installs it at the next reboot. It refuses when another KoboRoot.tgz is
# already waiting. --uninstall writes NickelHook's uninstall flag: the library is deleted at the next
# boot (the folder stays until you delete it).
# The retired status-bar helper (.adds/helpers/screenshot, library libstatusbarshot.so) is removed in the
# same run when its folder exists. This script never touches .adds/koboapps or .adds/nickeldrawer.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
# shellcheck source=../../tools/lib/kobo-staging.sh
. "$REPO_ROOT/tools/lib/kobo-staging.sh"

HELPER_DIR=$REPO_ROOT/helpers/extras
KOBO_ROOT=${KOBO_ROOT:-$(detect_kobo_root)}
BACKUP_ROOT=${BACKUP_ROOT:-$REPO_ROOT/backups}
HELPER_TGZ=${HELPER_TGZ:-$REPO_ROOT/build/helpers/KoboRoot-extras.tgz}
TWEAKS_BINARY=${TWEAKS_BINARY:-$REPO_ROOT/build/helpers/extras-tweaks-linux-armv7}
TABLER_LICENSE=$REPO_ROOT/third_party/tabler-icons/LICENSE
HELPER_RELATIVE=.adds/helpers/extras
TARGET=$KOBO_ROOT/$HELPER_RELATIVE
STAGED_TGZ=$KOBO_ROOT/.kobo/KoboRoot.tgz
# Flag files the user controls; installing never writes or removes them, except a stale uninstall flag.
UNINSTALL_FLAG=$TARGET/uninstall
# The retired helper: its folder and the uninstall flag its NickelHook plugin (nh.statusbarshot) reads.
OLD_HELPER_RELATIVE=.adds/helpers/screenshot
OLD_TARGET=$KOBO_ROOT/$OLD_HELPER_RELATIVE

isDryRun=false
isUninstall=false
backupDir=

fail() {
    echo "Error: $*" >&2
    exit 1
}

usage() {
    echo "Usage: $0 [--dry-run]"
    echo "       $0 --uninstall [--dry-run]"
}

# Runs a command, or only prints it with --dry-run.
act() {
    if $isDryRun; then
        echo "[dry-run] $*"
    else
        "$@"
    fi
}

parse_flags() {
    local flag
    for flag in "$@"; do
        case $flag in
        --dry-run) isDryRun=true ;;
        --uninstall) isUninstall=true ;;
        *) fail "unknown flag '$flag'. $(usage)" ;;
        esac
    done
}

require_mounted_kobo() {
    [ -d "$KOBO_ROOT/.kobo" ] ||
        fail "no Kobo found at $KOBO_ROOT (missing .kobo). Plug in the Kobo, tap Connect, and try again, or set KOBO_ROOT=/path/to/the/mounted/Kobo."
}

require_built_helper() {
    [ -f "$HELPER_TGZ" ] || fail "$HELPER_TGZ is missing. Run: make -C helpers/extras docker-build"
    [ -f "$TWEAKS_BINARY" ] || fail "$TWEAKS_BINARY is missing. Run: make build"
    [ -f "$HELPER_DIR/icons/screenshot.png" ] || fail "the screenshot icon is missing. Run: go run ./cmd/iconsgen -repo ."
    [ -f "$HELPER_DIR/icons/tweaks.png" ] || fail "the tweaks icon is missing. Run: go run ./cmd/iconsgen -repo ."
    [ -f "$TABLER_LICENSE" ] || fail "$TABLER_LICENSE is missing (the icons' MIT notice ships with them)"
}

# Only one archive can wait for a reboot: checked before anything is written.
require_no_waiting_archive() {
    [ ! -e "$STAGED_TGZ" ] ||
        fail "a KoboRoot.tgz is already waiting on the Kobo. Only one can be installed per reboot; reboot the Kobo first."
}

backup_path() {
    local source=$1
    [ -e "$source" ] || return 0
    local relative=${source#"$KOBO_ROOT"/}
    echo "backup: $relative"
    act mkdir -p "$backupDir/$(dirname "$relative")"
    act cp -R "$source" "$backupDir/$relative"
}

backup_before_writing() {
    backupDir=$BACKUP_ROOT/$(date +%Y-%m-%d-%H%M%S)
    backup_path "$TARGET"
    backup_path "$OLD_TARGET"
}

# Replaces bin/, locales/, icons/ and LICENSES/ and leaves the user's flag files alone.
install_data_files() {
    act mkdir -p "$TARGET/bin" "$TARGET/icons" "$TARGET/LICENSES"
    act cp "$TWEAKS_BINARY" "$TARGET/bin/extras-tweaks"
    act chmod 755 "$TARGET/bin/extras-tweaks"
    act rm -rf "$TARGET/locales"
    act mkdir -p "$TARGET/locales"
    act cp "$HELPER_DIR"/locales/*.json "$TARGET/locales/"
    act cp "$HELPER_DIR/icons/screenshot.png" "$TARGET/icons/screenshot.png"
    act cp "$HELPER_DIR/icons/tweaks.png" "$TARGET/icons/tweaks.png"
    act cp "$TABLER_LICENSE" "$TARGET/LICENSES/tabler.txt"
}

# A stale uninstall flag would remove the helper we are installing at the next boot.
clear_stale_uninstall_flag() {
    if [ -e "$UNINSTALL_FLAG" ]; then
        act rm -f "$UNINSTALL_FLAG"
        echo "removed the stale uninstall flag (backed up)"
    fi
}

# The old helper hooks the same status bar; it must be gone before the new library loads. Its files go
# and its own NickelHook uninstall flag stays, so libstatusbarshot.so is deleted at the same reboot.
retire_old_helper() {
    [ -d "$OLD_TARGET" ] || return 0
    local leftover
    for leftover in locales icons LICENSES off disable debug; do
        act rm -rf "${OLD_TARGET:?}/$leftover"
    done
    act touch "$OLD_TARGET/uninstall"
    echo "retired the old status-bar helper: wrote $OLD_HELPER_RELATIVE/uninstall (libstatusbarshot.so goes at the next boot)"
}

stage_library() {
    assert_only_mod_libraries "$HELPER_TGZ"
    act cp "$HELPER_TGZ" "$STAGED_TGZ"
}

install_helper() {
    require_built_helper
    require_no_waiting_archive
    backup_before_writing
    install_data_files
    clear_stale_uninstall_flag
    retire_old_helper
    stage_library
    echo "Installed $HELPER_RELATIVE and staged the library as .kobo/KoboRoot.tgz."
    echo "Eject the Kobo, then reboot it to install. Then run the device checklist in helpers/extras/README.md."
}

uninstall_helper() {
    backup_before_writing
    act mkdir -p "$TARGET"
    act touch "$UNINSTALL_FLAG"
    echo "Wrote the uninstall flag. Eject, then reboot the Kobo: NickelHook deletes libextras.so at boot."
    echo "Delete $HELPER_RELATIVE over USB afterwards if you want the files gone too."
}

main() {
    parse_flags "$@"
    require_mounted_kobo
    if $isUninstall; then
        uninstall_helper
    else
        install_helper
    fi
}

main "$@"

#!/bin/bash
# Installs or removes the status-bar screenshot helper on a mounted Kobo (maintainer only; the helper is
# not part of tools/install-to-kobo.sh, dist/ or the releases).
#   helpers/statusbar-screenshot/install.sh [--dry-run]
#   helpers/statusbar-screenshot/install.sh --uninstall [--dry-run]
#   helpers/statusbar-screenshot/install.sh --off [--dry-run]   # hide the button (creates the "off" flag)
#   helpers/statusbar-screenshot/install.sh --on [--dry-run]    # show it again (removes the flag)
# Env: KOBO_ROOT, the mounted Kobo (default: found like tools/install-to-kobo.sh does);
#      BACKUP_ROOT (default <repo>/backups); HELPER_TGZ (default build/helpers/KoboRoot-statusbarshot.tgz).
# Run `make -C helpers/statusbar-screenshot docker-build` and `helpers/statusbar-screenshot/check-symbols.sh`
# first. Never cd into the volume: it would block ejecting.
#
# Installing copies the locale files, the icon and the Tabler notice to .adds/helpers/screenshot (the
# user's flag files there stay), then stages the library as .kobo/KoboRoot.tgz; Nickel installs it at the
# next reboot. It refuses when another KoboRoot.tgz is already waiting. --uninstall writes NickelHook's
# uninstall flag: the library is deleted at the next boot (the folder is left for you to remove).
# --off and --on only create or remove the "off" flag file (backed up first); the Kobo picks it up the
# next time the status bar is shown or the button is tapped, no reboot.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
# shellcheck source=../../tools/lib/kobo-staging.sh
. "$REPO_ROOT/tools/lib/kobo-staging.sh"

HELPER_DIR=$REPO_ROOT/helpers/statusbar-screenshot
KOBO_ROOT=${KOBO_ROOT:-$(detect_kobo_root)}
BACKUP_ROOT=${BACKUP_ROOT:-$REPO_ROOT/backups}
HELPER_TGZ=${HELPER_TGZ:-$REPO_ROOT/build/helpers/KoboRoot-statusbarshot.tgz}
TABLER_LICENSE=$REPO_ROOT/third_party/tabler-icons/LICENSE
HELPER_RELATIVE=.adds/helpers/screenshot
TARGET=$KOBO_ROOT/$HELPER_RELATIVE
STAGED_TGZ=$KOBO_ROOT/.kobo/KoboRoot.tgz
# Flag files the user controls; installing never writes or removes them, except a stale uninstall flag.
UNINSTALL_FLAG=$TARGET/uninstall
OFF_FLAG=$TARGET/off

isDryRun=false
isUninstall=false
buttonSwitch=
backupDir=

fail() {
    echo "Error: $*" >&2
    exit 1
}

usage() {
    echo "Usage: $0 [--dry-run]"
    echo "       $0 --uninstall [--dry-run]"
    echo "       $0 --off|--on [--dry-run]"
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
        --off) buttonSwitch=off ;;
        --on) buttonSwitch=on ;;
        *) fail "unknown flag '$flag'. $(usage)" ;;
        esac
    done
}

require_mounted_kobo() {
    [ -d "$KOBO_ROOT/.kobo" ] ||
        fail "no Kobo found at $KOBO_ROOT (missing .kobo). Plug in the Kobo, tap Connect, and try again, or set KOBO_ROOT=/path/to/the/mounted/Kobo."
}

require_built_helper() {
    [ -f "$HELPER_TGZ" ] || fail "$HELPER_TGZ is missing. Run: make -C helpers/statusbar-screenshot docker-build"
    [ -f "$HELPER_DIR/icons/screenshot-64.png" ] || fail "the icon is missing. Run: go run ./cmd/iconsgen -repo ."
    [ -f "$TABLER_LICENSE" ] || fail "$TABLER_LICENSE is missing (the icon's MIT notice ships with it)"
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
}

# Replaces locales/, icons/ and LICENSES/ and leaves the user's flag files alone.
install_data_files() {
    act mkdir -p "$TARGET/icons" "$TARGET/LICENSES"
    act rm -rf "$TARGET/locales"
    act cp -R "$HELPER_DIR/locales" "$TARGET/locales"
    act cp "$HELPER_DIR/icons/screenshot-64.png" "$TARGET/icons/screenshot-64.png"
    act cp "$TABLER_LICENSE" "$TARGET/LICENSES/tabler.txt"
}

# A stale uninstall flag would remove the helper we are installing at the next boot.
clear_stale_uninstall_flag() {
    if [ -e "$UNINSTALL_FLAG" ]; then
        act rm -f "$UNINSTALL_FLAG"
        echo "removed the stale uninstall flag (backed up)"
    fi
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
    stage_library
    echo "Installed .adds/helpers/screenshot and staged the library as .kobo/KoboRoot.tgz."
    echo "Eject the Kobo, then reboot it to install. Then run the G1 checklist in re/notes/statusbar-screenshot.md (section 7)."
}

uninstall_helper() {
    backup_before_writing
    act mkdir -p "$TARGET"
    act touch "$UNINSTALL_FLAG"
    echo "Wrote the uninstall flag. Eject, then reboot the Kobo: NickelHook deletes libstatusbarshot.so at boot."
    echo "Delete .adds/helpers/screenshot over USB afterwards if you want the files gone too."
}

# The button's on/off switch is just the "off" flag file: present = hidden, absent = shown.
switch_button() {
    backup_before_writing
    if [ "$buttonSwitch" = off ]; then
        act mkdir -p "$TARGET"
        act touch "$OFF_FLAG"
        echo "Button off: wrote the off flag. Eject; the status bar updates the next time it is shown."
    else
        act rm -f "$OFF_FLAG"
        echo "Button on: removed the off flag. Eject; the status bar updates the next time it is shown."
    fi
}

main() {
    parse_flags "$@"
    require_mounted_kobo
    if [ -n "$buttonSwitch" ]; then
        switch_button
    elif $isUninstall; then
        uninstall_helper
    else
        install_helper
    fi
}

main "$@"

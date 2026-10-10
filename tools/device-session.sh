#!/bin/bash
# Orchestrator helper for on-device sessions. Never cd's into the volume; refuses to run without a Kobo.
# Usage: device-session.sh status | stage <tgz> | flags <flag>... | log | clean | eject
# Env: KOBO_ROOT (default /Volumes/KOBOeReader) for tests; BACKUP_ROOT (default <repo>/backups).
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
KOBO_ROOT=${KOBO_ROOT:-/Volumes/KOBOeReader}
BACKUP_ROOT=${BACKUP_ROOT:-$REPO_ROOT/backups}
FLAG_DIR=$KOBO_ROOT/.adds/nickeldrawer
STAGED_TGZ=$KOBO_ROOT/.kobo/KoboRoot.tgz
KOBO_CONF=$KOBO_ROOT/.kobo/Kobo/Kobo\ eReader.conf

fail() {
    echo "device-session: $*" >&2
    exit 1
}

# shellcheck source=lib/kobo-staging.sh
. "$REPO_ROOT/tools/lib/kobo-staging.sh"

requireKobo() {
    [ -d "$KOBO_ROOT/.kobo" ] ||
        fail "no Kobo at $KOBO_ROOT (missing .kobo). Plug in the Kobo, tap Connect, and try again."
}

backupDeviceState() {
    local target=$BACKUP_ROOT/$(date +%Y-%m-%d-%H%M%S)
    mkdir -p "$target"
    local source
    for source in "$KOBO_ROOT/.adds/nickeldrawer" "$KOBO_ROOT/.adds/nm"; do
        [ -d "$source" ] && cp -R "$source" "$target/"
    done
    echo "backup: $target"
}

confValue() {
    [ -f "$KOBO_CONF" ] || return 0
    sed -n "s/^$1=//p" "$KOBO_CONF" | head -1 | tr -d '\r'
}

commandStatus() {
    local firmware="unknown"
    [ -f "$KOBO_ROOT/.kobo/version" ] && firmware=$(cut -d, -f3 "$KOBO_ROOT/.kobo/version")
    echo "firmware: $firmware"

    local offered skipped
    offered=$(confValue AvailableQt6Upgrade)
    skipped=$(confValue SkippedQt6UpgradeUrl)
    if [ -z "$offered" ]; then
        echo "qt6 offer: none"
    elif [ "$offered" = "$skipped" ]; then
        echo "qt6 offer: present but declined (OK)"
    else
        echo "qt6 offer: PENDING and not declined. Do not sync; decline it on the device first (docs/DECISIONS.md)."
    fi

    if [ -f "$STAGED_TGZ" ]; then
        echo "staged KoboRoot.tgz: yes ($(wc -c <"$STAGED_TGZ" | tr -d ' ') bytes)"
    else
        echo "staged KoboRoot.tgz: no"
    fi

    echo "flags in .adds/nickeldrawer:"
    local flag found=0
    for flag in "$FLAG_DIR"/*; do
        [ -f "$flag" ] || continue
        echo "  $(basename "$flag")"
        found=1
    done
    [ "$found" = 1 ] || echo "  (none)"
}

commandStage() {
    local tgz=${1:-}
    [ -n "$tgz" ] || fail "usage: stage <KoboRoot-*.tgz>"
    [ -f "$tgz" ] || fail "no such file: $tgz"
    [ ! -e "$STAGED_TGZ" ] ||
        fail "a KoboRoot.tgz is already staged. Only one can be installed per reboot; reboot the Kobo first or remove it by hand."
    assert_only_mod_libraries "$tgz"
    backupDeviceState
    stage_mod_archive "$tgz"
    echo "staged $tgz. Eject, unplug, and wait for the Kobo to install and reboot."
}

commandFlags() {
    mkdir -p "$FLAG_DIR"
    backupDeviceState
    local flag wanted keep
    for flag in "$FLAG_DIR"/*; do
        [ -f "$flag" ] || continue
        case "$(basename "$flag")" in uninstall* | debug.log | debug.log.1) continue ;; esac
        keep=0
        for wanted in "$@"; do
            [ "$wanted" = "$(basename "$flag")" ] && keep=1
        done
        [ "$keep" = 1 ] || rm "$flag"
    done
    for wanted in "$@"; do
        case "$wanted" in */* | "" | .*) fail "bad flag name '$wanted'" ;; esac
        touch "$FLAG_DIR/$wanted"
    done
    echo "flags set: ${*:-(none)}. Reboot the Kobo to apply."
}

commandLog() {
    local log=$FLAG_DIR/debug.log
    [ -f "$log" ] || fail "no debug.log in $FLAG_DIR (was the debug flag set and the Kobo rebooted?)"
    local target=$BACKUP_ROOT/$(date +%Y-%m-%d-%H%M%S)
    mkdir -p "$target"
    # The mod rotates a full log to debug.log.1; keep it ahead of the current one.
    if [ -f "$log.1" ]; then
        cp "$log.1" "$target/debug.log.1"
        rm "$log.1"
        cat "$target/debug.log.1"
    fi
    cp "$log" "$target/debug.log"
    rm "$log"
    cat "$target/debug.log"
    echo "--- saved to $target/debug.log and removed from the Kobo" >&2
}

commandClean() {
    find "$KOBO_ROOT/.adds" -name '._*' -exec rm -rf {} + 2>/dev/null || true
    find "$KOBO_ROOT/.kobo" -maxdepth 1 -name '._*' -exec rm -rf {} + 2>/dev/null || true
    echo "removed ._* files"
}

commandEject() {
    sync
    if diskutil eject "$KOBO_ROOT"; then
        echo "ejected. Unplug the Kobo."
    else
        fail "eject failed. If it says loginwindow/another process dissented, close Finder windows and terminals showing the Kobo, make sure no shell is inside it, then run eject again."
    fi
}

requireKobo
command=${1:-}
[ $# -gt 0 ] && shift
case "$command" in
status) commandStatus ;;
stage) commandStage "$@" ;;
flags) commandFlags "$@" ;;
log) commandLog ;;
clean) commandClean ;;
eject) commandEject ;;
*) fail "usage: device-session.sh status | stage <tgz> | flags <flag>... | log | clean | eject" ;;
esac

#!/bin/bash
# Installs the koboapps dist/ tree onto a mounted Kobo (macOS).
#   tools/install-to-kobo.sh [--dry-run] [--remove-legacy]
# Env: KOBO_ROOT (default /Volumes/KOBOeReader) for tests or a differently named volume.
# Run `make check build dist` first. Never cd into the volume: it would block ejecting.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
KOBO_ROOT=${KOBO_ROOT:-/Volumes/KOBOeReader}
ADDS=$KOBO_ROOT/.adds
DIST_ADDS=$REPO_ROOT/dist/onboard/.adds
APPSCTL_HOST=$REPO_ROOT/build/appsctl-host
NMSTRIP_HOST=$REPO_ROOT/build/nmstrip-host
DEVICE_APPS_ROOT=/mnt/onboard/.adds/apps
DEVICE_KOBOAPPS_ROOT=/mnt/onboard/.adds/koboapps

isDryRun=false
shouldRemoveLegacy=false
backupDir=
changes=()

fail() {
    echo "Error: $*" >&2
    exit 1
}

note_change() {
    changes+=("$*")
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
    for flag in "$@"; do
        case $flag in
        --dry-run) isDryRun=true ;;
        --remove-legacy) shouldRemoveLegacy=true ;;
        *) fail "unknown flag '$flag'. Usage: $0 [--dry-run] [--remove-legacy]" ;;
        esac
    done
}

require_mounted_kobo() {
    [ -d "$KOBO_ROOT/.kobo" ] ||
        fail "no Kobo found at $KOBO_ROOT (missing .kobo). Plug in the Kobo, tap Connect, and try again."
}

require_fresh_dist() {
    [ -f "$REPO_ROOT/dist/.stamp" ] && [ -x "$APPSCTL_HOST" ] && [ -x "$NMSTRIP_HOST" ] ||
        fail "dist/ is missing. Run: make check build dist"
    local newer
    newer=$(find "$REPO_ROOT/apps" "$REPO_ROOT/host" "$REPO_ROOT/cmd" "$REPO_ROOT/internal" \
        "$REPO_ROOT/go.mod" "$REPO_ROOT/Makefile" -type f -newer "$REPO_ROOT/dist/.stamp" \
        ! -path '*/data/*' ! -name '.DS_Store' | head -n 3)
    [ -z "$newer" ] ||
        fail "dist/ is older than the sources (e.g. $newer). Run: make check build dist"
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
    backupDir=$REPO_ROOT/backups/$(date +%Y-%m-%d-%H%M%S)
    backup_path "$ADDS/nm"
    backup_path "$ADDS/apps"
    backup_path "$ADDS/koboapps"
    if $shouldRemoveLegacy; then
        backup_path "$ADDS/localsend"
    fi
    note_change "backup written to $backupDir"
}

# Replaces bin/, locales/ (and lib/ for the host); leaves data/ alone, creating it when missing.
install_folder() {
    local source=$1 target=$2
    shift 2
    act mkdir -p "$target"
    for subfolder in "$@"; do
        [ -d "$source/$subfolder" ] || continue
        act rm -rf "$target/$subfolder"
        act cp -R "$source/$subfolder" "$target/$subfolder"
    done
    [ ! -f "$source/app.json" ] || act cp "$source/app.json" "$target/app.json"
    act mkdir -p "$target/data"
}

install_example_config() {
    local source=$1 target=$2
    [ -f "$source/data/config.example.json" ] || return 0
    act cp "$source/data/config.example.json" "$target/data/config.example.json"
}

install_host() {
    install_folder "$DIST_ADDS/koboapps" "$ADDS/koboapps" bin lib locales
    note_change "installed .adds/koboapps (bin, lib, locales)"
}

install_apps() {
    local appSource appId
    act mkdir -p "$ADDS/apps"
    for appSource in "$DIST_ADDS"/apps/*/; do
        appSource=${appSource%/}
        appId=$(basename "$appSource")
        if [ -d "$ADDS/apps/$appId" ]; then
            note_change "updated app $appId (data/ kept)"
        else
            note_change "installed new app $appId"
        fi
        install_folder "$appSource" "$ADDS/apps/$appId" bin locales
        install_example_config "$appSource" "$ADDS/apps/$appId"
    done
}

generate_menu() {
    act "$APPSCTL_HOST" gen-menu \
        --apps-dir "$ADDS/apps" \
        --out "$ADDS/nm/koboapps" \
        --errors-file "$ADDS/koboapps/data/errors.txt" \
        --device-apps-root "$DEVICE_APPS_ROOT" \
        --device-koboapps-root "$DEVICE_KOBOAPPS_ROOT" \
        --koboapps-dir "$ADDS/koboapps" \
        --conf "$KOBO_ROOT/.kobo/Kobo/Kobo eReader.conf"
    note_change "generated .adds/nm/koboapps"
}

remove_legacy() {
    if [ -d "$ADDS/localsend" ]; then
        act rm -rf "$ADDS/localsend"
        note_change "removed legacy .adds/localsend"
    fi
    if [ -f "$ADDS/nm/items" ]; then
        act "$NMSTRIP_HOST" "$ADDS/nm/items"
        note_change "removed legacy LocalSend items from .adds/nm/items"
    fi
}

# macOS writes ._* and .DS_Store next to anything it copies; Nickel and the menu generator must not see them.
# Sweeps whole trees (not only the copied folders) so stray files from earlier copies go too.
clean_appledouble_files() {
    local folder
    for folder in "$ADDS/apps" "$ADDS/koboapps" "$ADDS/nm"; do
        [ -d "$folder" ] || continue
        act find "$folder" \( -name '._*' -o -name '.DS_Store' \) -type f -delete
    done
}

print_summary() {
    echo
    echo "Summary:"
    printf '  - %s\n' "${changes[@]}"
    $isDryRun && echo "(dry run: nothing was changed)"
    echo "Backup: $backupDir"
    echo "Eject the Kobo before unplugging."
}

main() {
    parse_flags "$@"
    require_mounted_kobo
    require_fresh_dist
    backup_before_writing
    install_host
    install_apps
    if $shouldRemoveLegacy; then
        remove_legacy
    fi
    generate_menu
    clean_appledouble_files
    act sync
    print_summary
}

main "$@"

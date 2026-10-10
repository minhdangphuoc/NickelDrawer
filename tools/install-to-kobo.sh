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
# NickelMenu's nav-bar tab (the "Ứng dụng" tab) is switched off with this setting; the drawer
# replaces it. The reader-menu items (Dark Mode) are not touched.
TAB_KEY_PATTERN='^experimental[[:space:]]*:[[:space:]]*menu_main_15505_enabled[[:space:]]*:'
TAB_OFF_LINE='experimental:menu_main_15505_enabled:0'
TAB_OFF_VALUE_PATTERN=':[[:space:]]*0[[:space:]]*$'

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

# Everything in .adds/koboapps except inbox/, which holds received books and could be huge.
backup_koboapps() {
    local entry
    [ -d "$ADDS/koboapps" ] || return 0
    for entry in "$ADDS/koboapps"/*; do
        [ "$(basename "$entry")" != inbox ] || continue
        backup_path "$entry"
    done
}

backup_before_writing() {
    backupDir=$REPO_ROOT/backups/$(date +%Y-%m-%d-%H%M%S)
    backup_path "$ADDS/nm"
    backup_path "$ADDS/apps"
    backup_koboapps
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
    [ ! -f "$source/ui.json" ] || act cp "$source/ui.json" "$target/ui.json"
    [ ! -f "$source/icon.png" ] || act cp "$source/icon.png" "$target/icon.png"
    act mkdir -p "$target/data"
}

install_example_config() {
    local source=$1 target=$2
    [ -f "$source/data/config.example.json" ] || return 0
    act cp "$source/data/config.example.json" "$target/data/config.example.json"
}

# The inbox and its rejected/ folder must exist before the first file arrives. The user's
# data/routing.json is never written here: install_folder only replaces the folders it is given.
create_inbox_folders() {
    act mkdir -p "$ADDS/koboapps/inbox/rejected"
}

install_host() {
    install_folder "$DIST_ADDS/koboapps" "$ADDS/koboapps" bin lib locales config icons
    create_inbox_folders
    note_change "installed .adds/koboapps (bin, lib, locales, config, icons, inbox)"
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

# Older versions generated NickelMenu items for the apps; the drawer replaces them. The whole
# .adds/nm folder was backed up first (backup_before_writing).
remove_generated_menu() {
    local generated=$ADDS/nm/koboapps
    if [ -e "$generated" ]; then
        act rm -f "$generated"
        note_change "removed generated .adds/nm/koboapps (backed up)"
    fi
    if [ -e "$ADDS/koboapps/data/errors.txt" ]; then
        act rm -f "$ADDS/koboapps/data/errors.txt"
    fi
}

# Writes the file with every tab line replaced by the "off" line, via a temp file and a rename.
rewrite_tab_lines_off() {
    local file=$1 temp="$1.tmp"
    if $isDryRun; then
        echo "[dry-run] set $TAB_OFF_LINE in $file"
        return 0
    fi
    awk -v pattern="$TAB_KEY_PATTERN" -v off="$TAB_OFF_LINE" '$0 ~ pattern { print off; next } { print }' "$file" >"$temp"
    mv "$temp" "$file"
}

append_tab_off_line() {
    local file=$1
    if $isDryRun; then
        echo "[dry-run] append $TAB_OFF_LINE to $file"
        return 0
    fi
    mkdir -p "$(dirname "$file")"
    # A file that does not end in a newline would glue our line onto its last one.
    if [ -s "$file" ] && [ -n "$(tail -c 1 "$file")" ]; then
        echo >>"$file"
    fi
    echo "$TAB_OFF_LINE" >>"$file"
}

# True when the file has a tab line whose value is not 0.
has_tab_line_not_off() {
    grep -E "$TAB_KEY_PATTERN" "$1" | grep -Eqv "$TAB_OFF_VALUE_PATTERN"
}

# Ensures the NickelMenu tab setting is 0 in the user's .adds/nm config. Idempotent: a file that
# already says 0 is left alone; with no tab line anywhere the line is added to .adds/nm/items.
disable_nickelmenu_tab() {
    local file hasTabLine=false
    for file in "$ADDS"/nm/*; do
        [ -f "$file" ] && grep -Eq "$TAB_KEY_PATTERN" "$file" || continue
        hasTabLine=true
        if has_tab_line_not_off "$file"; then
            rewrite_tab_lines_off "$file"
            note_change "NickelMenu tab switched off in .adds/nm/$(basename "$file")"
        fi
    done
    if ! $hasTabLine; then
        append_tab_off_line "$ADDS/nm/items"
        note_change "NickelMenu tab switched off (line added to .adds/nm/items)"
    fi
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

# macOS writes ._* and .DS_Store next to anything it copies; Nickel and NickelMenu must not see them.
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
    remove_generated_menu
    disable_nickelmenu_tab
    clean_appledouble_files
    act sync
    print_summary
}

main "$@"

#!/bin/bash
# Installs, updates or removes NickelDrawer on a mounted Kobo (macOS).
#   tools/install-to-kobo.sh [--dry-run] [--remove-legacy] [--with-mod | --no-mod]
#   tools/install-to-kobo.sh --uninstall [--purge] [--dry-run]
# Env: KOBO_ROOT (default /Volumes/KOBOeReader) for tests or a differently named volume;
#      BACKUP_ROOT (default <repo>/backups); MOD_TGZ (default build/mod/KoboRoot-nickeldrawer.tgz).
# Run `make check build dist` first. Never cd into the volume: it would block ejecting.
#
# The Nickel mod (build/mod/KoboRoot-nickeldrawer.tgz) is staged as .kobo/KoboRoot.tgz when it
# exists and differs from the one installed last time; Nickel installs it at the next reboot.
# --with-mod forces it, --no-mod skips it. --uninstall removes the drawer, the apps and the mod.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
KOBO_ROOT=${KOBO_ROOT:-/Volumes/KOBOeReader}
BACKUP_ROOT=${BACKUP_ROOT:-$REPO_ROOT/backups}
ADDS=$KOBO_ROOT/.adds
FLAG_DIR=$ADDS/nickeldrawer
STAGED_TGZ=$KOBO_ROOT/.kobo/KoboRoot.tgz
DIST_ADDS=$REPO_ROOT/dist/onboard/.adds
APPSCTL_HOST=$REPO_ROOT/build/appsctl-host
MOD_TGZ=${MOD_TGZ:-$REPO_ROOT/build/mod/KoboRoot-nickeldrawer.tgz}
# The archive staged last time, kept on the Kobo so the next install can tell whether the mod changed.
INSTALLED_MOD_RECORD=$ADDS/koboapps/mod/KoboRoot-nickeldrawer.tgz
# NickelMenu's nav-bar tab is switched off with this setting; the drawer replaces it. The
# reader-menu items (Dark Mode) are not touched. --uninstall puts the old value back.
TAB_KEY_PATTERN='^experimental[[:space:]]*:[[:space:]]*menu_main_15505_enabled[[:space:]]*:'
TAB_KEY_NAME='experimental:menu_main_15505_enabled'
TAB_OFF_VALUE_PATTERN=':[[:space:]]*0[[:space:]]*$'
TAB_DEFAULT_VALUE=1
# Hand-written NickelMenu items that the LocalSend app replaced.
LEGACY_ITEM_PATTERN=':(Nhận sách LocalSend|Dừng LocalSend) '

isDryRun=false
shouldRemoveLegacy=false
modMode=auto # auto | force | skip
isUninstall=false
isPurge=false
backupDir=
changes=()

fail() {
    echo "Error: $*" >&2
    exit 1
}

# shellcheck source=lib/kobo-staging.sh
. "$REPO_ROOT/tools/lib/kobo-staging.sh"

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

usage() {
    echo "Usage: $0 [--dry-run] [--remove-legacy] [--with-mod | --no-mod]"
    echo "       $0 --uninstall [--purge] [--dry-run]"
}

parse_flags() {
    for flag in "$@"; do
        case $flag in
        --dry-run) isDryRun=true ;;
        --remove-legacy) shouldRemoveLegacy=true ;;
        --with-mod) modMode=force ;;
        --no-mod) modMode=skip ;;
        --uninstall) isUninstall=true ;;
        --purge) isPurge=true ;;
        *) fail "unknown flag '$flag'. $(usage)" ;;
        esac
    done
    if $isPurge && ! $isUninstall; then
        fail "--purge only goes with --uninstall."
    fi
}

require_mounted_kobo() {
    [ -d "$KOBO_ROOT/.kobo" ] ||
        fail "no Kobo found at $KOBO_ROOT (missing .kobo). Plug in the Kobo, tap Connect, and try again."
}

require_fresh_dist() {
    [ -f "$REPO_ROOT/dist/.stamp" ] && [ -x "$APPSCTL_HOST" ] ||
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
    backupDir=$BACKUP_ROOT/$(date +%Y-%m-%d-%H%M%S)
    backup_path "$ADDS/nm"
    backup_path "$ADDS/apps"
    backup_path "$FLAG_DIR"
    backup_path "$STAGED_TGZ"
    backup_koboapps
    if $shouldRemoveLegacy; then
        backup_path "$ADDS/localsend"
    fi
    note_change "backup written to $backupDir"
}

# --- the Nickel mod ---------------------------------------------------------------------------

# True when the mod archive differs from the one the last install staged.
is_mod_changed() {
    [ ! -f "$INSTALLED_MOD_RECORD" ] || ! cmp -s "$MOD_TGZ" "$INSTALLED_MOD_RECORD"
}

# Decides before anything is written whether the mod gets staged, and fails early on a conflict.
should_stage_mod() {
    case $modMode in
    skip) return 1 ;;
    force) [ -f "$MOD_TGZ" ] || fail "--with-mod needs $MOD_TGZ. Build the mod first." ;;
    auto) [ -f "$MOD_TGZ" ] && is_mod_changed || return 1 ;;
    esac
    if [ -e "$STAGED_TGZ" ]; then
        cmp -s "$MOD_TGZ" "$STAGED_TGZ" && return 1
        if [ "$modMode" = force ]; then
            fail "a different KoboRoot.tgz is already waiting on the Kobo. Reboot the Kobo first, then run again."
        fi
        echo "warning: a different KoboRoot.tgz is already waiting on the Kobo; the mod is not staged this time." >&2
        return 1
    fi
    return 0
}

stage_mod() {
    assert_only_mod_libraries "$MOD_TGZ"
    act cp "$MOD_TGZ" "$STAGED_TGZ"
    act mkdir -p "$(dirname "$INSTALLED_MOD_RECORD")"
    act cp "$MOD_TGZ" "$INSTALLED_MOD_RECORD"
    note_change "Nickel mod staged as .kobo/KoboRoot.tgz: eject, then reboot the Kobo to install it"
}

# The flag folder only needs to exist: every flag is an opt-in or opt-out the user adds by hand
# (disable, no-screens, debug, ...), so nothing is created in it. A stale uninstall flag would
# remove the mod we are installing, and the old "screens" flag is ignored now.
prepare_flag_folder() {
    act mkdir -p "$FLAG_DIR"
    local stale
    for stale in uninstall screens; do
        if [ -e "$FLAG_DIR/$stale" ]; then
            act rm -f "$FLAG_DIR/$stale"
            note_change "removed stale flag .adds/nickeldrawer/$stale (backed up)"
        fi
    done
}

# --- installing -------------------------------------------------------------------------------

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
    install_folder "$DIST_ADDS/koboapps" "$ADDS/koboapps" bin lib locales config icons screens
    create_inbox_folders
    note_change "installed .adds/koboapps (bin, lib, locales, config, icons, screens, inbox)"
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

# --- the NickelMenu tab setting ---------------------------------------------------------------

# Writes the file with every tab line replaced by "<key>:<value>", via a temp file and a rename.
rewrite_tab_lines() {
    local file=$1 value=$2 temp="$1.tmp"
    if $isDryRun; then
        echo "[dry-run] set $TAB_KEY_NAME:$value in $file"
        return 0
    fi
    awk -v pattern="$TAB_KEY_PATTERN" -v line="$TAB_KEY_NAME:$value" '$0 ~ pattern { print line; next } { print }' "$file" >"$temp"
    mv "$temp" "$file"
}

append_tab_line() {
    local file=$1 value=$2
    if $isDryRun; then
        echo "[dry-run] append $TAB_KEY_NAME:$value to $file"
        return 0
    fi
    mkdir -p "$(dirname "$file")"
    # A file that does not end in a newline would glue our line onto its last one.
    if [ -s "$file" ] && [ -n "$(tail -c 1 "$file")" ]; then
        echo >>"$file"
    fi
    echo "$TAB_KEY_NAME:$value" >>"$file"
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
            rewrite_tab_lines "$file" 0
            note_change "NickelMenu tab switched off in .adds/nm/$(basename "$file")"
        fi
    done
    if ! $hasTabLine; then
        append_tab_line "$ADDS/nm/items" 0
        note_change "NickelMenu tab switched off (line added to .adds/nm/items)"
    fi
}

# Prints the tab value the user had before the first install: the newest backup that still has
# a tab line whose value is not 0. Prints the default when no backup says.
backed_up_tab_value() {
    local backup file line
    for backup in $(ls -d "$BACKUP_ROOT"/*/ 2>/dev/null | sort -r); do
        for file in "$backup".adds/nm/*; do
            [ -f "$file" ] || continue
            line=$(grep -E "$TAB_KEY_PATTERN" "$file" | grep -Ev "$TAB_OFF_VALUE_PATTERN" | tail -n 1 || true)
            if [ -n "$line" ]; then
                echo "$line" | sed -E 's/.*:[[:space:]]*([^:[:space:]]+)[[:space:]]*$/\1/'
                return 0
            fi
        done
    done
    echo "$TAB_DEFAULT_VALUE"
}

restore_nickelmenu_tab() {
    local value file hasTabLine=false
    value=$(backed_up_tab_value)
    for file in "$ADDS"/nm/*; do
        [ -f "$file" ] && grep -Eq "$TAB_KEY_PATTERN" "$file" || continue
        hasTabLine=true
        rewrite_tab_lines "$file" "$value"
    done
    if $hasTabLine; then
        note_change "NickelMenu tab setting restored to $value"
    fi
}

# --- legacy files -----------------------------------------------------------------------------

# Drops the hand-written LocalSend items (the item line and its indented chain_ lines).
strip_legacy_items() {
    local file=$1 temp="$1.tmp"
    if $isDryRun; then
        echo "[dry-run] remove legacy LocalSend items from $file"
        return 0
    fi
    awk -v pattern="^menu_item .*$LEGACY_ITEM_PATTERN" '
        $0 ~ pattern { dropping = 1; next }
        dropping && /^[ \t].*chain_/ { next }
        { dropping = 0; print }' "$file" >"$temp"
    mv "$temp" "$file"
}

remove_legacy() {
    if [ -d "$ADDS/localsend" ]; then
        act rm -rf "$ADDS/localsend"
        note_change "removed legacy .adds/localsend"
    fi
    if [ -f "$ADDS/nm/items" ]; then
        strip_legacy_items "$ADDS/nm/items"
        note_change "removed legacy LocalSend items from .adds/nm/items"
    fi
}

# macOS writes ._* and .DS_Store next to anything it copies; Nickel and NickelMenu must not see them.
# Sweeps whole trees (not only the copied folders) so stray files from earlier copies go too.
clean_appledouble_files() {
    local folder
    for folder in "$ADDS/apps" "$ADDS/koboapps" "$ADDS/nm" "$FLAG_DIR"; do
        [ -d "$folder" ] || continue
        act find "$folder" \( -name '._*' -o -name '.DS_Store' \) -type f -delete
    done
}

# --- uninstalling -----------------------------------------------------------------------------

# Asks whether the apps' data/ folders (tokens, settings, trash) stay. --purge answers no.
should_keep_app_data() {
    $isPurge && return 1
    local answer=y
    if [ -t 0 ] || ! $isDryRun; then
        printf "Keep the data/ folders of the apps (tokens, settings, trash)? [Y/n] " >&2
        read -r answer || answer=y
    fi
    case $answer in n | N | no | NO) return 1 ;; *) return 0 ;; esac
}

# Deletes everything in a folder except the named entries; removes the folder when nothing stays.
remove_except() {
    local folder=$1 entry keep name
    shift
    [ -d "$folder" ] || return 0
    for entry in "$folder"/* "$folder"/.[!.]*; do
        [ -e "$entry" ] || continue
        name=$(basename "$entry")
        keep=false
        for kept in "$@"; do
            [ "$name" != "$kept" ] || keep=true
        done
        $keep || act rm -rf "$entry"
    done
    act rmdir "$folder" 2>/dev/null || true
}

remove_apps() {
    local keepData=$1 appFolder
    for appFolder in "$ADDS/apps"/*/; do
        [ -d "$appFolder" ] || continue
        if $keepData; then
            remove_except "${appFolder%/}" data
        else
            act rm -rf "${appFolder%/}"
        fi
    done
    act rmdir "$ADDS/apps" 2>/dev/null || true
    if $keepData; then
        note_change "removed the apps (their data/ folders kept)"
    else
        note_change "removed the apps and their data"
    fi
}

# Received books waiting in the inbox are the user's files and stay, whatever is chosen.
remove_host() {
    local keepData=$1 keepers=(inbox)
    $keepData && keepers+=(data)
    remove_except "$ADDS/koboapps" "${keepers[@]}"
    if [ -d "$ADDS/koboapps/inbox" ] && [ -z "$(find "$ADDS/koboapps/inbox" -type f | head -n 1)" ]; then
        act rm -rf "$ADDS/koboapps/inbox"
        act rmdir "$ADDS/koboapps" 2>/dev/null || true
    fi
    note_change "removed .adds/koboapps (files in its inbox/ are kept)"
}

# The mod removes itself at the next boot when this flag exists. A waiting copy of our own
# archive would put it straight back, so that one is withdrawn.
request_mod_uninstall() {
    act mkdir -p "$FLAG_DIR"
    act touch "$FLAG_DIR/uninstall"
    note_change "mod uninstall flag written: reboot the Kobo to remove the mod"
    if [ -f "$STAGED_TGZ" ] && [ -f "$MOD_TGZ" ] && cmp -s "$MOD_TGZ" "$STAGED_TGZ"; then
        act rm -f "$STAGED_TGZ"
        note_change "withdrew the waiting KoboRoot.tgz of this mod (backed up)"
    fi
}

uninstall() {
    backup_before_writing
    local keepData=true
    should_keep_app_data || keepData=false
    remove_apps "$keepData"
    remove_host "$keepData"
    request_mod_uninstall
    restore_nickelmenu_tab
    clean_appledouble_files
}

print_summary() {
    echo
    echo "Summary:"
    printf '  - %s\n' "${changes[@]}"
    $isDryRun && echo "(dry run: nothing was changed)"
    echo "Backup: $backupDir"
    echo "Eject the Kobo before unplugging."
}

install() {
    require_fresh_dist
    local isModStaged=false
    should_stage_mod && isModStaged=true
    backup_before_writing
    install_host
    install_apps
    prepare_flag_folder
    if $isModStaged; then
        stage_mod
    fi
    if $shouldRemoveLegacy; then
        remove_legacy
    fi
    remove_generated_menu
    disable_nickelmenu_tab
    clean_appledouble_files
}

main() {
    parse_flags "$@"
    require_mounted_kobo
    if $isUninstall; then
        uninstall
    else
        install
    fi
    act sync
    print_summary
}

main "$@"

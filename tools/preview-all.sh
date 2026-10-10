#!/bin/sh
# Renders every app screen, the host settings screens, the drawer list and grid views and the
# Tweaks modal with ndpreview, in English and Vietnamese, into build/preview/<locale>/ (gitignored).
#   tools/preview-all.sh [locale ...]        # default: en vi
# Each app screen is rendered for every state file apps/<app>/preview-state*.json (English runs use
# the preview-state*.en.json sibling automatically) and every page of a long screen. Needs a
# TrueType font with Vietnamese letters (default: Arial Unicode on macOS; override with FONT=...).
# The wallpaper and UI Kit states point at sample pictures: go run ./tools/wallpaper-samples
# --out build/preview/wallpaper-samples makes them (this script does when they are missing).
set -eu
cd "$(dirname "$0")/.."
FONT=${FONT:-/System/Library/Fonts/Supplemental/Arial Unicode.ttf}
OUT=${OUT:-build/preview}
LOCALES=${*:-en vi}
mkdir -p build "$OUT"
go build -o build/ndpreview-host ./cmd/ndpreview
NDPREVIEW=build/ndpreview-host
[ -d "$OUT/wallpaper-samples/data" ] || go run ./tools/wallpaper-samples --out "$OUT/wallpaper-samples"

# screens_of APP_DIR: the screen ids of its ui.json and of its screens/*.json files, one per line.
screens_of() {
    python3 -c 'import json,os,sys
ids = list(json.load(open(sys.argv[1] + "/ui.json"))["screens"])
folder = sys.argv[1] + "/screens"
if os.path.isdir(folder):
    ids += sorted(name[:-5] for name in os.listdir(folder) if name.endswith(".json"))
print("\n".join(ids))' "$1"
}

# render_pages APP_DIR LOCALE SCREEN STATE_FILE NAME: page 1, then the pages ndpreview says follow.
render_pages() {
    dir=$1 locale=$2 screen=$3 state=$4 name=$5
    page=1
    while :; do
        suffix=""
        [ "$page" -eq 1 ] || suffix="-p$page"
        message=$($NDPREVIEW render "$dir" --locale "$locale" --state "$state" --screen "$screen" --page "$page" \
            --font "$FONT" --out "$OUT/$locale/$name$suffix.png")
        pages=${message##*/}
        pages=${pages%%)*}
        page=$((page + 1))
        [ "$page" -le "$pages" ] || break
    done
}

for locale in $LOCALES; do
    mkdir -p "$OUT/$locale"
    for dir in apps/*/; do
        app=$(basename "$dir")
        # An app without a state file is rendered with the ui.json defaults.
        for state in "$dir"preview-state*.json; do
            case $state in *.en.json) continue ;; esac
            variant=${state##*/preview-state}
            variant=${variant%.json}
            [ -e "$state" ] || { state=""; variant=""; }
            screens_of "$dir" | while read -r screen; do
                render_pages "$dir" "$locale" "$screen" "$state" "$app-$screen$variant"
            done
        done
    done
    $NDPREVIEW drawer-list --locale "$locale" --font "$FONT" --out "$OUT/$locale/drawer-list.png"
    $NDPREVIEW drawer-list --locale "$locale" --count 4 --problems --qt6 --font "$FONT" --out "$OUT/$locale/drawer-list-warning.png"
    $NDPREVIEW drawer-empty --locale "$locale" --font "$FONT" --out "$OUT/$locale/drawer-empty.png"
    $NDPREVIEW drawer-empty --locale "$locale" --loading --font "$FONT" --out "$OUT/$locale/drawer-loading.png"
    $NDPREVIEW drawer-grid --locale "$locale" --font "$FONT" --out "$OUT/$locale/drawer-grid.png"
    $NDPREVIEW tweaks --locale "$locale" --font "$FONT" --out "$OUT/$locale/tweaks.png"
done
NDPREVIEW="$PWD/$NDPREVIEW" OUT="$OUT" PER_LOCALE=1 FONT="$FONT" sh tools/preview-settings.sh >/dev/null
echo "previews in $OUT/<locale>/"

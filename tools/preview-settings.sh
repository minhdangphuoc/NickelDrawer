#!/bin/sh
# Renders the settings and About screens with ndpreview, in English and Vietnamese.
#   tools/preview-settings.sh                 # build/preview/settings-<screen>-<locale>.png
#   OUT=build/preview PER_LOCALE=1 tools/preview-settings.sh   # build/preview/<locale>/settings-<screen>.png
# Needs a TrueType font with Vietnamese letters (default: Arial Unicode on macOS; override with FONT=...).
set -eu
cd "$(dirname "$0")/.."
FONT=${FONT:-/System/Library/Fonts/Supplemental/Arial Unicode.ttf}
OUT=${OUT:-build/preview}
PER_LOCALE=${PER_LOCALE:-}
NDPREVIEW=${NDPREVIEW:-"go run ./cmd/ndpreview"}
# ndpreview wants an app folder: borrow the host screen's files and add the manifest the host
# builds in code (internal/drawer/host_settings.go), so bindings to its actions validate.
SCREEN_DIR=$OUT/settings
rm -rf "$SCREEN_DIR"
mkdir -p "$SCREEN_DIR/bin"
: >"$SCREEN_DIR/bin/x.sh"
chmod 755 "$SCREEN_DIR/bin/x.sh"
cp host/screens/settings/ui.json "$SCREEN_DIR/ui.json"
cp -R host/screens/settings/locales "$SCREEN_DIR/locales"
cat >"$SCREEN_DIR/app.json" <<'JSON'
{"schema":1,"id":"settings","name":"Settings","version":"1.0.0","actions":[
{"id":"check","label":"Check","run":"bin/x.sh","output":"show"},
{"id":"apply","label":"Update","run":"bin/x.sh","output":"show"},
{"id":"debug","label":"Debug","run":"bin/x.sh","output":"show"}]}
JSON

# render LOCALE SCREEN PAGE NAME [STATE]: the state file is tools/preview/settings-state-<STATE>.json
render() {
    locale=$1 screen=$2 page=$3 name=$4 state=${5:-$1}
    target="$OUT/$name-$locale.png"
    if [ -n "$PER_LOCALE" ]; then
        mkdir -p "$OUT/$locale"
        target="$OUT/$locale/$name.png"
    fi
    $NDPREVIEW render "$SCREEN_DIR" --locale "$locale" --state "tools/preview/settings-state-$state.json" \
        --screen "$screen" --page "$page" --font "$FONT" --out "$target"
}

for locale in vi en; do
    render "$locale" main 1 settings-main
    render "$locale" main 2 settings-main-p2
    render "$locale" about 1 settings-about
    render "$locale" about 2 settings-about-p2
    # What the Updates section says after a check, while one runs, and when one fails.
    render "$locale" main 1 settings-uptodate "$locale-uptodate"
    render "$locale" main 1 settings-checking "$locale-checking"
    render "$locale" main 1 settings-failed "$locale-failed"
done

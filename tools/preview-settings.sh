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
{"id":"debug","label":"Debug","run":"bin/x.sh","output":"show"},
{"id":"apps-check","label":"Check","run":"bin/x.sh","output":"show"},
{"id":"apps-update-all","label":"Update all","run":"bin/x.sh","output":"show"},
{"id":"apps-list","label":"Apps","run":"bin/x.sh","output":"show"}]}
JSON

# render LOCALE SCREEN PAGE NAME [STATE]: the state file is tools/preview/settings-<STATE>.json (default state-<LOCALE>)
render() {
    locale=$1 screen=$2 page=$3 name=$4 state=${5:-state-$1}
    target="$OUT/$name-$locale.png"
    if [ -n "$PER_LOCALE" ]; then
        mkdir -p "$OUT/$locale"
        target="$OUT/$locale/$name.png"
    fi
    $NDPREVIEW render "$SCREEN_DIR" --locale "$locale" --state "tools/preview/settings-$state.json" \
        --screen "$screen" --page "$page" --font "$FONT" --out "$target"
}

for locale in vi en; do
    render "$locale" main 1 settings-main
    render "$locale" main 2 settings-main-p2
    render "$locale" about 1 settings-about
    render "$locale" about 2 settings-about-p2
    render "$locale" about 3 settings-about-p3
    # What the Updates section says after a check, while one runs, and when one fails.
    render "$locale" main 1 settings-uptodate "state-$locale-uptodate"
    render "$locale" main 1 settings-checking "state-$locale-checking"
    render "$locale" main 1 settings-failed "state-$locale-failed"
    # The App updates screen: updates waiting, all current, a job running, a job that failed.
    render "$locale" apps 1 settings-apps "apps-state-$locale"
    render "$locale" apps 2 settings-apps-p2 "apps-state-$locale"
    render "$locale" apps 1 settings-apps-uptodate "apps-state-$locale-uptodate"
    render "$locale" apps 1 settings-apps-working "apps-state-$locale-working"
    render "$locale" apps 1 settings-apps-failed "apps-state-$locale-failed"
done

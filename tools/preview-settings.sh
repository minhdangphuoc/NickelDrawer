#!/bin/sh
# Renders the settings and About screens to build/preview/settings-*.png with ndpreview.
# Needs a TrueType font with Vietnamese letters (default: Arial Unicode on macOS; override with FONT=...).
set -eu
cd "$(dirname "$0")/.."
FONT=${FONT:-/System/Library/Fonts/Supplemental/Arial Unicode.ttf}
OUT=build/preview
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

render() {
    locale=$1 screen=$2 page=$3 name=$4 state=${5:-$1}
    go run ./cmd/ndpreview render "$SCREEN_DIR" --locale "$locale" --state "tools/preview/settings-state-$state.json" \
        --screen "$screen" --page "$page" --font "$FONT" --out "$OUT/$name.png"
}

render vi main 1 settings-main-vi
render vi main 2 settings-main-vi-p2
render vi about 1 settings-about-vi
render vi about 2 settings-about-vi-p2
render en main 1 settings-main-en
render en about 1 settings-about-en
# What the Updates section says after a check, while one runs, and when one fails.
render vi main 1 settings-uptodate-vi vi-uptodate
render vi main 1 settings-checking-vi vi-checking
render vi main 1 settings-failed-vi vi-failed

#!/bin/sh
# Prints the GitHub screen's variables as one JSON object. Reads local files only (< 1 s) and
# never prints the token. Without the binary the screen keeps its defaults.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)

if [ ! -x "$APP_DIR/bin/ghsync" ]; then
    echo '{}'
    exit 0
fi
exec "$APP_DIR/bin/ghsync" state --app-dir "$APP_DIR"

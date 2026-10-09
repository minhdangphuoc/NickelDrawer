#!/bin/sh
# Variables for ui.json, printed as one JSON object.
. "$(dirname "$0")/lib.sh"
[ -x "$APP_DIR/bin/library" ] || exit 1
run_library state

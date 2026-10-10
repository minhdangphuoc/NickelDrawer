#!/bin/sh
# Prints the screen variables as one JSON object. Reads one small file (well under 1 s).
. "$(dirname "$0")/lib.sh"
require_binary
state=$(run_todo state) || exit 1
# Loading the list sets a damaged file aside and leaves the marker; only then is the notice text
# looked up, so a normal read does not start appsctl.
if [ -f "$DATA_DIR/recovered" ]; then
    state=$(run_todo state --notice "$(msg "$APP_ID" msg.recovered)") || exit 1
fi
printf '%s\n' "$state"

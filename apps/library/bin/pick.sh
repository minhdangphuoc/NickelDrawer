#!/bin/sh
# The tapped list row (NICKELDRAWER_VALUE) becomes the selected duplicate.
. "$(dirname "$0")/lib.sh"
require_binary
run_quick "" pick "${NICKELDRAWER_VALUE:-}" || exit 1
# The screen shows the selection from state.sh; the command line prints a confirmation.
if ! is_drawer; then
    msg "$APP_ID" msg.picked
fi

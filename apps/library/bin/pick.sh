#!/bin/sh
# The tapped list row (NICKELDRAWER_VALUE) becomes the selected duplicate.
. "$(dirname "$0")/lib.sh"
require_binary
run_library pick "${NICKELDRAWER_VALUE:-}" && msg "$APP_ID" msg.picked

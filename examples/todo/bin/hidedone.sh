#!/bin/sh
# The "Hide done items" switch sends true or false.

. "$(dirname "$0")/lib.sh"
mkdir -p "$DATA_DIR"
if [ "${NICKELDRAWER_VALUE:-}" = true ]; then
    : >"$HIDE_DONE_FILE"
else
    rm -f "$HIDE_DONE_FILE"
fi

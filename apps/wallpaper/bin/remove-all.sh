#!/bin/sh
# Removes the picture in NICKELDRAWER_VALUE and also deletes its original in Pictures. The
# binary deletes the original only when a file there is provably identical (same SHA-256);
# otherwise it keeps it, and the toast says so.
. "$(dirname "$0")/lib.sh"
require_binary
require_value
if is_working; then
    report busy "$APP_ID" msg.busy
    exit 0
fi
if is_drawer; then
    run_wallpaper remove --confirm --delete-original --signal "$NICKELDRAWER_VALUE" || fail_with "$APP_DIR" "remove-all failed"
else
    run_wallpaper remove --confirm --delete-original "$NICKELDRAWER_VALUE"
fi

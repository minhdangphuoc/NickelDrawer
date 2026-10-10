#!/bin/sh
# Removes the picture in NICKELDRAWER_VALUE from Wallpaper. The screen already asked the user, so
# --confirm is passed; --delete-original is never passed: the original in Pictures stays.
. "$(dirname "$0")/lib.sh"
require_binary
require_value
# A running import or re-fit holds the picture list in memory and would write it back over the
# removal; ask the user to wait instead.
if is_working; then
    if is_drawer; then
        toast busy
        exit 0
    fi
    msg "$APP_ID" msg.busy
    exit 0
fi
run_quick removed remove --confirm "$NICKELDRAWER_VALUE"

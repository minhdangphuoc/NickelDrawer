#!/bin/sh
# Shows the running or last conversion, then the current settings.
. "$(dirname "$0")/lib.sh"
require_binary

if is_working; then
    msg "$APP_ID" msg.working
else
    cat "$OUTPUT_FILE" 2>/dev/null
fi
run_wallpaper status

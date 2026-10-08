#!/bin/sh
# Shows whether a sync is running and the last result. Its manifest entry
# rescans the library, so tap it after a sync finishes to see new books.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=ghsync
PID_FILE=/tmp/ghsync.pid
TMP_RUN_FILE=/tmp/ghsync-last-run.txt

if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    msg "$APP_ID" msg.status_running
    cat "$TMP_RUN_FILE" 2>/dev/null
else
    msg "$APP_ID" msg.status_idle
    cat "$APP_DIR/data/last-run.txt" 2>/dev/null
fi

"$APP_DIR/bin/ghsync" status --app-dir "$APP_DIR" 2>&1 || exit 1

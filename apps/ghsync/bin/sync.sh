#!/bin/sh
# Starts a GitHub sync in the background. NickelMenu's cmd_output must return
# in < 10 s, a sync can take minutes, so this only launches it; the status action
# reports the result and rescans the library.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=ghsync
PID_FILE=/tmp/ghsync.pid
# The run writes to /tmp so nothing stays open on /mnt/onboard; the result is copied to data/ at the end.
TMP_RUN_FILE=/tmp/ghsync-last-run.txt
# Exit code of the finished run, so this script never has to parse translated output.
TMP_EXIT_FILE=/tmp/ghsync-last-exit.txt
LAST_RUN_FILE="$APP_DIR/data/last-run.txt"

is_sync_running() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

start_background_sync() {
    # All fds are redirected so NickelMenu does not wait for the child.
    rm -f "$TMP_EXIT_FILE"
    sh -c 'echo $$ >"$1"; "$2" run --app-dir "$3" >"$4" 2>&1; echo $? >"$6"; cp "$4" "$5"; rm -f "$1"' \
        sh "$PID_FILE" "$APP_DIR/bin/ghsync" "$APP_DIR" "$TMP_RUN_FILE" "$LAST_RUN_FILE" "$TMP_EXIT_FILE" \
        </dev/null >/dev/null 2>&1 &
}

if ! wait_for_wifi 8 >/dev/null; then
    msg "$APP_ID" msg.wifi_missing
    exit 1
fi

if is_sync_running; then
    msg "$APP_ID" msg.already_running
    msg "$APP_ID" msg.see_status
    exit 0
fi

if [ ! -x "$APP_DIR/bin/ghsync" ]; then
    app_log "$APP_DIR" "bin/ghsync missing or not executable"
    msg "$APP_ID" msg.binary_missing
    exit 1
fi

mkdir -p "$APP_DIR/data"
app_log "$APP_DIR" "sync started from menu"
start_background_sync
sleep 1

if is_sync_running; then
    msg "$APP_ID" msg.syncing
    msg "$APP_ID" msg.see_status
    exit 0
fi

# Finished within a second: usually a config or token problem.
cat "$TMP_RUN_FILE"
[ "$(cat "$TMP_EXIT_FILE" 2>/dev/null)" = 0 ]

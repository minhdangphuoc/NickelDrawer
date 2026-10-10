#!/bin/sh
# Starts a GitHub sync in the background. A sync can take minutes, so this only launches it;
# the drawer answers with a toast and the screen shows the result from state.sh.

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
    # All fds are redirected so the caller does not wait for the child.
    rm -f "$TMP_EXIT_FILE"
    sh -c 'echo $$ >"$1"; "$2" run --app-dir "$3" >"$4" 2>&1; echo $? >"$6"; cp "$4" "$5"; rm -f "$1"' \
        sh "$PID_FILE" "$APP_DIR/bin/ghsync" "$APP_DIR" "$TMP_RUN_FILE" "$LAST_RUN_FILE" "$TMP_EXIT_FILE" \
        </dev/null >/dev/null 2>&1 &
}

if ! ensure_wifi; then
    report_failure wifi_missing "$APP_ID" msg.wifi_missing
fi

if is_sync_running; then
    report already_running "$APP_ID" msg.already_running
    is_drawer || msg "$APP_ID" msg.see_status
    exit 0
fi

if [ ! -x "$APP_DIR/bin/ghsync" ]; then
    fail_with "$APP_DIR" "$(msg "$APP_ID" msg.binary_missing)"
fi

mkdir -p "$APP_DIR/data"
app_log "$APP_DIR" "sync started"
start_background_sync
sleep 1

if is_sync_running; then
    report syncing "$APP_ID" msg.syncing
    is_drawer || msg "$APP_ID" msg.see_status
    exit 0
fi

# Finished within a second: usually a config or token problem.
if is_drawer; then
    app_log "$APP_DIR" "sync ended at once: $(cat "$TMP_RUN_FILE" 2>/dev/null)"
    if [ "$(cat "$TMP_EXIT_FILE" 2>/dev/null)" = 0 ]; then
        emit_toast sync_done
    else
        emit_toast sync_failed
    fi
    exit 0
fi
cat "$TMP_RUN_FILE"
[ "$(cat "$TMP_EXIT_FILE" 2>/dev/null)" = 0 ]

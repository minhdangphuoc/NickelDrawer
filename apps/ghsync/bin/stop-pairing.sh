#!/bin/sh
# Stops an open phone pairing. The drawer answers with a toast; the screen shows the outcome
# from state.sh a moment later.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=ghsync
PID_FILE=/tmp/ghsync-pair.pid

is_pairing_open() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

if [ ! -x "$APP_DIR/bin/ghsync" ]; then
    fail_with "$APP_DIR" "$(msg "$APP_ID" msg.binary_missing)"
fi

if ! is_pairing_open; then
    report pair_not_running "$APP_ID" msg.pair_not_running
    exit 0
fi
"$APP_DIR/bin/ghsync" pair --stop --app-dir "$APP_DIR" --pid-file "$PID_FILE" >/dev/null 2>&1
app_log "$APP_DIR" "pairing stopped by the user"
report pair_stopped "$APP_ID" msg.pair_stopping

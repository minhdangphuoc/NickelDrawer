#!/bin/sh
# Opens phone pairing: starts `ghsync pair` in the background. The drawer answers with a toast
# and shows the address and one-time code from state.sh; the command line prints what the binary printed.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=ghsync
PID_FILE=/tmp/ghsync-pair.pid
# Everything the binary prints goes to /tmp so nothing stays open on /mnt/onboard.
PAIR_FILE=/tmp/ghsync-pair.txt

is_pairing_open() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

if ! ensure_wifi; then
    report_failure wifi_missing "$APP_ID" msg.wifi_missing
fi

if [ ! -x "$APP_DIR/bin/ghsync" ]; then
    fail_with "$APP_DIR" "$(msg "$APP_ID" msg.binary_missing)"
fi

if ! is_pairing_open; then
    mkdir -p "$APP_DIR/data"
    app_log "$APP_DIR" "pairing started"
    # All fds are redirected so the caller does not wait for the child.
    "$APP_DIR/bin/ghsync" pair --app-dir "$APP_DIR" --pid-file "$PID_FILE" \
        </dev/null >"$PAIR_FILE" 2>&1 &
    sleep 1
fi

# A pairing that ended within a second (no IP, port busy) leaves its error here.
if is_drawer; then
    if ! is_pairing_open; then
        app_log "$APP_DIR" "pairing ended at once: $(cat "$PAIR_FILE" 2>/dev/null)"
        emit_toast pair_failed
        exit 0
    fi
    emit_toast pair_started
    exit 0
fi
cat "$PAIR_FILE"
is_pairing_open

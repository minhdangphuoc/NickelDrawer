#!/bin/sh
# Opens phone pairing: starts `ghsync pair` in the background (NickelMenu's cmd_output
# must return in < 10 s) and shows the address and one-time code the binary prints.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=ghsync
PID_FILE=/tmp/ghsync-pair.pid
# Everything the binary prints goes to /tmp so nothing stays open on /mnt/onboard.
PAIR_FILE=/tmp/ghsync-pair.txt

is_pairing_open() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

if ! wait_for_wifi 8 >/dev/null; then
    msg "$APP_ID" msg.wifi_missing
    exit 1
fi

if [ ! -x "$APP_DIR/bin/ghsync" ]; then
    app_log "$APP_DIR" "bin/ghsync missing or not executable"
    msg "$APP_ID" msg.binary_missing
    exit 1
fi

if ! is_pairing_open; then
    mkdir -p "$APP_DIR/data"
    app_log "$APP_DIR" "pairing started from menu"
    # All fds are redirected so NickelMenu does not wait for the child.
    "$APP_DIR/bin/ghsync" pair --app-dir "$APP_DIR" --pid-file "$PID_FILE" \
        </dev/null >"$PAIR_FILE" 2>&1 &
    sleep 1
fi

# A pairing that ended within a second (no IP, port busy) leaves its error here.
cat "$PAIR_FILE"
is_pairing_open

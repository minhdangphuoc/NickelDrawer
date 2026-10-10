#!/bin/sh
# Start the LocalSend receiver. Received files land in the inbox and are routed one by one.
# In the drawer it answers with a toast; the screen shows the PIN and IP from state.sh.
. "$(dirname "$0")/env.sh"

mkdir -p "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_STATE_HOME" "$XDG_CACHE_HOME" "$INBOX"

# The PIN is generated once and kept across sessions; mode 600 because it is a secret.
if [ ! -s "$DATA_DIR/pin.txt" ]; then
    (umask 077 && awk 'BEGIN { srand(); printf "%04d\n", int(rand() * 10000) }' >"$DATA_DIR/pin.txt")
fi
PIN=$(cat "$DATA_DIR/pin.txt")

if ! ensure_wifi; then
    report_failure wifi_missing "$APP_ID" msg.wifi_missing
fi
IP=$(current_ip)

DEVICE_NAME=$(device_name)

if ! is_running; then
    "$BINARY" serve --alias "$DEVICE_NAME" --dir "$INBOX" --auto-accept \
        --pin "$PIN" --no-clipboard --quiet --exec "$ROUTE_HOOK" \
        </dev/null >"$TMP_LOG" 2>&1 &
    echo $! >"$PIDFILE"
    rm -f "$AUTO_STOP_FLAG"
    start_detached "$APP_DIR/bin/watch.sh"
    sleep 1
    if ! is_running; then
        app_log "$APP_DIR" "start failed: $(tail -n 3 "$TMP_LOG" 2>/dev/null)"
        report_failure start_failed "$APP_ID" msg.start_failed
    fi
fi

report started "$APP_ID" msg.started device="$DEVICE_NAME" ip="$IP" pin="$PIN"

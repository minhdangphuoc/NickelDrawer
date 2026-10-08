#!/bin/sh
# Start the LocalSend receiver. Received files land in the inbox and are routed one by one. NickelMenu runs this after nickel_wifi:autoconnect.
. "$(dirname "$0")/env.sh"

mkdir -p "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_STATE_HOME" "$XDG_CACHE_HOME" "$INBOX"

# The PIN is generated once and kept across sessions; mode 600 because it is a secret.
if [ ! -s "$DATA_DIR/pin.txt" ]; then
    (umask 077 && awk 'BEGIN { srand(); printf "%04d\n", int(rand() * 10000) }' >"$DATA_DIR/pin.txt")
fi
PIN=$(cat "$DATA_DIR/pin.txt")

# NickelMenu's cmd_output limit is 10s, so wait at most 8s for Wi-Fi.
if ! IP=$(wait_for_wifi 8); then
    msg "$APP_ID" msg.wifi_missing
    exit 1
fi

DEVICE_NAME=$(device_name)

if ! is_running; then
    "$BINARY" serve --alias "$DEVICE_NAME" --dir "$INBOX" --auto-accept \
        --pin "$PIN" --no-clipboard --quiet --exec "$ROUTE_HOOK" \
        </dev/null >"$TMP_LOG" 2>&1 &
    echo $! >"$PIDFILE"
    sleep 1
    if ! is_running; then
        msg "$APP_ID" msg.start_failed
        tail -n 3 "$TMP_LOG" 2>/dev/null
        exit 1
    fi
fi

msg "$APP_ID" msg.started device="$DEVICE_NAME" ip="$IP" pin="$PIN"

#!/bin/sh
# Background watcher (not an action): stops the receiver after IDLE_STOP_MINUTES without a
# received file, so a forgotten receiver does not keep Wi-Fi on and the battery draining.
# A received file shows up as a newer inbox journal entry or inbox folder.
. "$(dirname "$0")/env.sh"

CHECK_SECONDS=30

echo $$ >"$WATCH_PIDFILE"
touch "$ACTIVITY_MARKER"

is_current_watcher() {
    [ "$(cat "$WATCH_PIDFILE" 2>/dev/null)" = "$$" ]
}

has_new_activity() {
    [ -n "$(find "$INBOX" "$INBOX_JOURNAL" -newer "$ACTIVITY_MARKER" 2>/dev/null | head -n 1)" ]
}

is_idle_too_long() {
    [ -n "$(find "$ACTIVITY_MARKER" -mmin +"$IDLE_STOP_MINUTES" 2>/dev/null)" ]
}

while is_running && is_current_watcher; do
    sleep "$CHECK_SECONDS"
    is_running || break
    if has_new_activity; then
        touch "$ACTIVITY_MARKER"
    elif is_idle_too_long; then
        app_log "$APP_DIR" "stopped after $IDLE_STOP_MINUTES minutes without a file"
        : >"$AUTO_STOP_FLAG"
        : >"$RESCAN_FLAG"
        "$APP_DIR/bin/stop.sh" >/dev/null 2>&1
        break
    fi
done
is_current_watcher && rm -f "$WATCH_PIDFILE"

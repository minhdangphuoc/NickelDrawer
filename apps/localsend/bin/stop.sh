#!/bin/sh
# Stop the LocalSend receiver. NickelMenu then rescans the library.
. "$(dirname "$0")/env.sh"

if is_running; then
    kill "$(cat "$PIDFILE")" 2>/dev/null
fi
rm -f "$PIDFILE"
# Fallback for a lost pid file. Anchored on our exact binary path so no other process can match.
pkill -f "^$BINARY serve" 2>/dev/null

# Keep the last MAX_LOG_LINES lines of history; write via a temp file so a failure never loses the log.
save_log() {
    [ -s "$TMP_LOG" ] || return 0
    mkdir -p "$DATA_DIR"
    cat "$TMP_LOG" >>"$DATA_DIR/log.txt"
    tail -n "$MAX_LOG_LINES" "$DATA_DIR/log.txt" >"$DATA_DIR/log.txt.tmp" &&
        mv "$DATA_DIR/log.txt.tmp" "$DATA_DIR/log.txt"
}
save_log
rm -f "$TMP_LOG"

COUNT=$(find "$INBOX" -type f 2>/dev/null | wc -l)
msg "$APP_ID" msg.stopped count="$COUNT"

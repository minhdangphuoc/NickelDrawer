#!/bin/sh
# Stop the LocalSend receiver and sort whatever the hooks did not get to. The drawer answers
# with a toast and rescans the library (manifest "rescan"); the screen shows the summary.
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

# Hooks started by localgo are not children of this script, so poll for them (at most
# HOOK_WAIT_SECONDS) instead of waiting on them.
wait_for_hooks() {
    seconds=$HOOK_WAIT_SECONDS
    while [ "$seconds" -gt 0 ] && pgrep -f "^$APPSCTL inbox route" >/dev/null 2>&1; do
        sleep 1
        seconds=$((seconds - 1))
    done
}
wait_for_hooks

# Safety net: anything a hook missed (a killed hook, a crash) is routed with the same rules.
mkdir -p "$DATA_DIR"
"$APPSCTL" inbox sort --source localsend >/dev/null 2>>"$DATA_DIR/log.txt"

# Files older versions left in the legacy folder are routed here, once, the same way sort.sh
# does it.
has_legacy_files() {
    [ -n "$(find "$LEGACY_FOLDER" -maxdepth 1 -type f 2>/dev/null | head -n 1)" ]
}
if has_legacy_files; then
    "$APPSCTL" inbox sort --source localsend --source-dir "$LEGACY_FOLDER" >/dev/null 2>>"$DATA_DIR/log.txt"
fi
report stopped "$APP_ID" msg.stopped
if ! is_drawer; then
    print_inbox_summary
fi

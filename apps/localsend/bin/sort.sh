#!/bin/sh
# Sort files that older versions received into /mnt/onboard/LocalSend. The work runs in the
# background; run it again to see the result.
. "$(dirname "$0")/env.sh"

SORT_PID_FILE=/tmp/localsend-sort.pid
SORT_DONE_FILE=/tmp/localsend-sort.done

is_sorting() {
    [ -f "$SORT_PID_FILE" ] && kill -0 "$(cat "$SORT_PID_FILE")" 2>/dev/null
}

has_old_files() {
    [ -n "$(find "$LEGACY_FOLDER" -maxdepth 1 -type f 2>/dev/null | head -n 1)" ]
}

start_background_sort() {
    rm -f "$SORT_DONE_FILE"
    # All fds are redirected so the caller does not wait for the child.
    sh -c 'echo $$ >"$1"; "$2" inbox sort --source localsend --source-dir "$3" >/dev/null 2>&1; touch "$4"; rm -f "$1"' \
        sh "$SORT_PID_FILE" "$APPSCTL" "$LEGACY_FOLDER" "$SORT_DONE_FILE" \
        </dev/null >/dev/null 2>&1 &
}

if is_sorting; then
    report sort_running "$APP_ID" msg.sort_running
    exit 0
fi

if [ -f "$SORT_DONE_FILE" ]; then
    rm -f "$SORT_DONE_FILE"
    if is_drawer; then
        emit_toast sort_done
    else
        print_inbox_summary
    fi
    exit 0
fi

if ! has_old_files; then
    report sort_nothing "$APP_ID" msg.sort_nothing
    exit 0
fi

start_background_sort
sleep 1
report sort_started "$APP_ID" msg.sort_started

#!/bin/sh
# Menu item "Check for updates": starts `appsctl update apply` in the background. NickelMenu's
# cmd_output must return in < 10 s and an update can take minutes, so this only launches it;
# "Update status" shows the result. Same pattern as the ghsync sync script.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"
PID_FILE=/tmp/nickeldrawer-update.pid
# Output goes to /tmp so nothing stays open on /mnt/onboard while the update runs.
TMP_RUN_FILE=/tmp/nickeldrawer-update-last-run.txt
TMP_ERROR_FILE=/tmp/nickeldrawer-update-errors.txt
# Exit code of the finished run, so this script never has to parse translated output.
TMP_EXIT_FILE=/tmp/nickeldrawer-update-last-exit.txt

say() {
    "$APPSCTL" host-msg "$@"
}

is_update_running() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

start_background_update() {
    # All fds are redirected so NickelMenu does not wait for the child.
    rm -f "$TMP_EXIT_FILE"
    sh -c 'echo $$ >"$1"; "$2" update apply >"$3" 2>"$4"; echo $? >"$5"; rm -f "$1"' \
        sh "$PID_FILE" "$APPSCTL" "$TMP_RUN_FILE" "$TMP_ERROR_FILE" "$TMP_EXIT_FILE" \
        </dev/null >/dev/null 2>&1 &
}

# Without appsctl there is no localization to ask, so this one line stays English.
if [ ! -x "$APPSCTL" ]; then
    echo "appsctl is missing. Reinstall from the computer."
    exit 1
fi

if ! wait_for_wifi 7 >/dev/null; then
    say update.wifi_missing
    exit 1
fi

if is_update_running; then
    say update.already_running
    exit 0
fi

start_background_update
sleep 1

if is_update_running; then
    say update.started
    exit 0
fi

# Finished within a second: no update.json, no key, up to date, or a network problem.
cat "$TMP_RUN_FILE"
[ "$(cat "$TMP_EXIT_FILE" 2>/dev/null)" = 0 ]

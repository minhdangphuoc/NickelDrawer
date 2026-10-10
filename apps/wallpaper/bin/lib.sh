# Shared by the wallpaper action scripts. Source it; do not execute.
# Fast commands run in the foreground; picture conversion can take longer than NickelMenu's
# 10 s limit on the Kobo's CPU, so it runs detached and Status shows the result.
#
# Two callers: the app drawer (NICKELDRAWER=1) and the NickelMenu fallback. In the drawer a script
# never prints prose: it returns at once and prints at most one declared {"toast":"id"}; details go
# to data/log.txt, and the screen shows progress from the state action.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=wallpaper
PID_FILE=/tmp/wallpaper.pid
# Output goes to /tmp so no file stays open on /mnt/onboard.
OUTPUT_FILE=/tmp/wallpaper-last.txt
PROGRESS_FILE=/tmp/wallpaper-progress.json
RESULT_WAIT_SECONDS=7

is_drawer() {
    [ "${NICKELDRAWER:-}" = 1 ]
}

# Usage: toast <id>   (the id must be declared in "toasts" of the screen)
toast() {
    printf '{"toast":"%s"}\n' "$1"
}

# Drawer: log the reason and show the generic failure toast. NickelMenu: print the reason, fail.
fail_with() {
    reason=$1
    app_log "$APP_DIR" "$reason"
    if is_drawer; then
        toast failed
        exit 0
    fi
    echo "$reason"
    exit 1
}

require_binary() {
    if [ ! -x "$APP_DIR/bin/wallpaper" ]; then
        fail_with "$(msg "$APP_ID" msg.binary_missing)"
    fi
}

# The picture id the screen sends in NICKELDRAWER_VALUE.
require_value() {
    if [ -z "${NICKELDRAWER_VALUE:-}" ]; then
        fail_with "$(msg "$APP_ID" msg.picture_missing)"
    fi
}

is_working() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

# Usage: run_wallpaper <subcommand> [args...]   (flags must precede positional arguments)
run_wallpaper() {
    subcommand=$1
    shift
    "$APP_DIR/bin/wallpaper" "$subcommand" --app-dir "$APP_DIR" "$@"
}

# Runs a fast subcommand in the foreground.
# Usage: run_quick <drawer-toast-or-empty> <subcommand> [args...]
# Drawer: prints the toast on success (nothing when empty). NickelMenu: prints the command output.
run_quick() {
    success_toast=$1
    shift
    output=$(run_wallpaper "$@" 2>&1)
    status=$?
    if ! is_drawer; then
        echo "$output"
        return "$status"
    fi
    if [ "$status" -ne 0 ]; then
        fail_with "$* failed: $output"
    fi
    if [ -n "$success_toast" ]; then
        toast "$success_toast"
    fi
}

# Runs a subcommand detached. All fds are redirected so the caller does not wait for the child.
# Usage: run_wallpaper_in_background <drawer-toast> <subcommand> [args...]
# Drawer: returns at once (toast "busy" when a job runs, else the given toast). NickelMenu: waits a
# few seconds and shows the result.
run_wallpaper_in_background() {
    started_toast=$1
    subcommand=$2
    shift 2
    if is_working; then
        if is_drawer; then
            toast busy
            exit 0
        fi
        msg "$APP_ID" msg.busy
        exit 0
    fi
    # The recorded pid is the wrapper shell, which ends after the status summary is written, so
    # the output always finishes with the current settings (the Status action is not in NickelMenu).
    sh -c 'echo $$ >"$1"; app=$2; shift 2; "$app/bin/wallpaper" "$@"; "$app/bin/wallpaper" status --app-dir "$app"' sh "$PID_FILE" "$APP_DIR" \
        "$subcommand" --app-dir "$APP_DIR" --progress-file "$PROGRESS_FILE" "$@" </dev/null >"$OUTPUT_FILE" 2>&1 &
    if is_drawer; then
        toast "$started_toast"
        return 0
    fi
    wait_for_result
}

# NickelMenu shows the dialog for at most 9.5 s: show the result if the work ends in time.
wait_for_result() {
    waited=0
    while [ "$waited" -lt "$RESULT_WAIT_SECONDS" ]; do
        sleep 1
        waited=$((waited + 1))
        if ! is_working; then
            cat "$OUTPUT_FILE"
            return 0
        fi
    done
    msg "$APP_ID" msg.started
}

# shellcheck shell=sh
# Shared by the wallpaper action scripts. Source it; do not execute.
# Fast commands run in the foreground; picture conversion can take longer than NickelMenu's
# 10 s limit on the Kobo's CPU, so it runs detached and Status shows the result.
#
# Two callers: the app drawer (NICKELDRAWER=1) and the command line (tests, debugging). In the drawer a script
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

require_binary() {
    if [ ! -x "$APP_DIR/bin/wallpaper" ]; then
        fail_with "$APP_DIR" "$(msg "$APP_ID" msg.binary_missing)"
    fi
}

# The picture id the screen sends in NICKELDRAWER_VALUE.
require_value() {
    if [ -z "${NICKELDRAWER_VALUE:-}" ]; then
        fail_with "$APP_DIR" "$(msg "$APP_ID" msg.picture_missing)"
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

# Runs a fast subcommand in the foreground (see run_quick_command in common.sh).
# Usage: run_quick <drawer-toast-or-empty> <subcommand> [args...]
run_quick() {
    success_toast=$1
    shift
    run_quick_command "$APP_DIR" "$success_toast" run_wallpaper "$@"
}

# Starts a subcommand detached and records its pid. All fds are redirected so the caller does not wait.
# The pid is the wrapper shell, which ends after the status summary is written, so the output always
# finishes with the current settings (the Status action is not in NickelMenu). The parent writes the
# pid itself, so is_working is true as soon as this returns.
# Usage: launch_in_background <subcommand> [args...]
launch_in_background() {
    subcommand=$1
    shift
    sh -c 'app=$1; shift; "$app/bin/wallpaper" "$@"; "$app/bin/wallpaper" status --app-dir "$app"' sh "$APP_DIR" \
        "$subcommand" --app-dir "$APP_DIR" --progress-file "$PROGRESS_FILE" "$@" </dev/null >"$OUTPUT_FILE" 2>&1 &
    echo $! >"$PID_FILE"
}

# Runs a subcommand detached.
# Usage: run_wallpaper_in_background <drawer-toast> <subcommand> [args...]
# Drawer: returns at once (toast "busy" when a job runs, else the given toast). Command line: waits a
# few seconds and shows the result.
run_wallpaper_in_background() {
    started_toast=$1
    subcommand=$2
    shift 2
    if is_working; then
        report busy "$APP_ID" msg.busy
        exit 0
    fi
    launch_in_background "$subcommand" "$@"
    if is_drawer; then
        emit_toast "$started_toast"
        return 0
    fi
    wait_for_result
}

# The first time the gallery is empty and Pictures has pictures, imports them without being asked.
# Silent (no toast, no output; failures go to data/log.txt) so a caller that prints JSON stays clean;
# progress shows in the state.
start_first_import_if_due() {
    if is_working; then
        return 0
    fi
    if run_wallpaper first-import-due >/dev/null 2>&1; then
        launch_in_background import
    fi
}

# The command line waits a few seconds and shows the result if the work ends in time.
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

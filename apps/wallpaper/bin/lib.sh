# Shared by the wallpaper action scripts. Source it; do not execute.
# Fast commands run in the foreground; picture conversion can take longer than NickelMenu's
# 10 s limit on the Kobo's CPU, so it runs detached and Status shows the result.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=wallpaper
PID_FILE=/tmp/wallpaper.pid
# Output goes to /tmp so no file stays open on /mnt/onboard.
OUTPUT_FILE=/tmp/wallpaper-last.txt
RESULT_WAIT_SECONDS=7

require_binary() {
    if [ ! -x "$APP_DIR/bin/wallpaper" ]; then
        app_log "$APP_DIR" "bin/wallpaper missing or not executable"
        msg "$APP_ID" msg.binary_missing
        exit 1
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

# Runs a subcommand detached and tells the user where to look. All fds are redirected so
# NickelMenu does not wait for the child.
run_wallpaper_in_background() {
    subcommand=$1
    shift
    if is_working; then
        msg "$APP_ID" msg.busy
        exit 0
    fi
    # The recorded pid is the wrapper shell, which ends after the status summary is written, so
    # the output always finishes with the current settings (the Status action is not in NickelMenu).
    sh -c 'echo $$ >"$1"; app=$2; shift 2; "$app/bin/wallpaper" "$@"; "$app/bin/wallpaper" status --app-dir "$app"' sh "$PID_FILE" "$APP_DIR" \
        "$subcommand" --app-dir "$APP_DIR" "$@" </dev/null >"$OUTPUT_FILE" 2>&1 &
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

# Shared by the wallpaper action scripts. Source it; do not execute.
# Fast commands run in the foreground; picture conversion can take longer than NickelMenu's
# 10 s limit on the Kobo's CPU, so it runs detached and Status shows the result.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=wallpaper
PID_FILE=/tmp/wallpaper.pid
# Output goes to /tmp so no file stays open on /mnt/onboard.
OUTPUT_FILE=/tmp/wallpaper-last.txt

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
    # exec makes the recorded pid the converter itself, so is_working sees it end.
    sh -c 'echo $$ >"$1"; shift; exec "$@"' sh "$PID_FILE" \
        "$APP_DIR/bin/wallpaper" "$subcommand" --app-dir "$APP_DIR" "$@" </dev/null >"$OUTPUT_FILE" 2>&1 &
    sleep 1
    msg "$APP_ID" msg.started
}

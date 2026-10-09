# Shared by the library action scripts. Source it; do not execute.
# Scanning hashes and opens many books, which can take longer than NickelMenu's 10 s limit on
# the Kobo's CPU, so those commands run detached and the screen shows the progress.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=library
PID_FILE=/tmp/library.pid
# Output goes to /tmp so no file stays open on /mnt/onboard.
OUTPUT_FILE=/tmp/library-last.txt
RESULT_WAIT_SECONDS=7

require_binary() {
    if [ ! -x "$APP_DIR/bin/library" ]; then
        app_log "$APP_DIR" "bin/library missing or not executable"
        msg "$APP_ID" msg.binary_missing
        exit 1
    fi
}

is_working() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

# Usage: run_library <subcommand> [args...]   (flags must precede positional arguments)
run_library() {
    subcommand=$1
    shift
    "$APP_DIR/bin/library" "$subcommand" --app-dir "$APP_DIR" "$@"
}

# Runs a subcommand detached and shows the result if it ends in time. All fds are redirected
# so NickelMenu does not wait for the child.
run_library_in_background() {
    subcommand=$1
    shift
    if is_working; then
        msg "$APP_ID" msg.busy
        exit 0
    fi
    sh -c 'echo $$ >"$1"; app=$2; shift 2; "$app/bin/library" "$@"' sh "$PID_FILE" "$APP_DIR" \
        "$subcommand" --app-dir "$APP_DIR" "$@" </dev/null >"$OUTPUT_FILE" 2>&1 &
    wait_for_result
}

# NickelMenu shows the dialog for at most 9.5 s.
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

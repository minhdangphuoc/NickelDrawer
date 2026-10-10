# shellcheck shell=sh
# Shared by the library action scripts. Source it; do not execute.
# Scanning hashes and opens many books, which can take a long time on the Kobo's CPU, so those
# commands run detached and the screen shows the progress.
#
# Two callers: the app drawer (NICKELDRAWER=1), which never gets prose and only a declared
# {"toast":"id"}, and the command line (tests, debugging), which gets the localized text.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=library
PID_FILE=/tmp/library.pid
# Output goes to /tmp so no file stays open on /mnt/onboard.
OUTPUT_FILE=/tmp/library-last.txt
RESULT_WAIT_SECONDS=7

require_binary() {
    if [ ! -x "$APP_DIR/bin/library" ]; then
        fail_with "$APP_DIR" "$(msg "$APP_ID" msg.binary_missing)"
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

# Runs a fast subcommand in the foreground (see run_quick_command in common.sh).
# Usage: run_quick <drawer-toast-or-empty> <subcommand> [args...]
run_quick() {
    success_toast=$1
    shift
    run_quick_command "$APP_DIR" "$success_toast" run_library "$@"
}

# Runs a subcommand detached. All fds are redirected so the caller does not wait for the child.
# Drawer: returns at once with the "started" toast. Command line: shows the result if it ends in time.
run_library_in_background() {
    subcommand=$1
    shift
    if is_working; then
        report busy "$APP_ID" msg.busy
        exit 0
    fi
    sh -c 'echo $$ >"$1"; app=$2; shift 2; "$app/bin/library" "$@"' sh "$PID_FILE" "$APP_DIR" \
        "$subcommand" --app-dir "$APP_DIR" "$@" </dev/null >"$OUTPUT_FILE" 2>&1 &
    if is_drawer; then
        emit_toast started
        return 0
    fi
    wait_for_result
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

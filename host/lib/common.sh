# shellcheck shell=sh
# shellcheck disable=SC2034  # variables used by the scripts that source this file
# Shared helpers for koboapps app scripts. Source it; do not execute.
# Installed on the Kobo at /mnt/onboard/.adds/koboapps/lib/common.sh

KOBOAPPS_ROOT=/mnt/onboard/.adds/koboapps
APPS_ROOT=/mnt/onboard/.adds/apps

# Prints an app message in the device language.
# Usage: msg <app-id> <key> [name=value...]   (strings live in <app>/locales/<lang>.json)
# An unknown key prints the key itself, so a missing string never breaks a script.
msg() {
    "$KOBOAPPS_ROOT/bin/appsctl" msg --app "$@"
}

# Prints the Kobo model name (from .kobo/version), "Kobo" when unknown.
device_name() {
    "$KOBOAPPS_ROOT/bin/appsctl" device-name
}

# Prints the first non-loopback IPv4 address, or nothing.
current_ip() {
    /sbin/ifconfig 2>/dev/null |
        awk '/inet addr/ { ip = substr($2, 6); if (ip != "127.0.0.1") { print ip; exit } }'
}

# Waits up to $1 seconds for an IP. Prints it and returns 0, or returns 1.
wait_for_wifi() {
    seconds=${1:-8}
    while [ "$seconds" -gt 0 ]; do
        ip=$(current_ip)
        if [ -n "$ip" ]; then
            echo "$ip"
            return 0
        fi
        sleep 1
        seconds=$((seconds - 1))
    done
    return 1
}

# Checks that Wi-Fi is up. The drawer never waits (it already looked for an IP before running
# an action with "wifi": true); the command line waits up to $1 seconds (default 8).
ensure_wifi() {
    if is_drawer; then
        [ -n "$(current_ip)" ]
    else
        wait_for_wifi "${1:-8}" >/dev/null
    fi
}

# Appends a timestamped line to the calling app's data/log.txt.
# Usage: app_log <app_dir> <message...>
app_log() {
    app_dir=$1
    shift
    mkdir -p "$app_dir/data"
    echo "$(date '+%Y-%m-%d %H:%M:%S') $*" >>"$app_dir/data/log.txt"
}

# True when the script runs inside the native drawer (appsctl sets NICKELDRAWER=1). There it
# must return at once and answer only with JSON signals, never with prose for a dialog.
is_drawer() {
    [ "${NICKELDRAWER:-}" = 1 ]
}

# Prints the signal that makes the drawer show a toast declared in the screen's ui.json "toasts".
# Usage: emit_toast <toast-id>
emit_toast() {
    printf '{"toast":"%s"}\n' "$1"
}

# Prints the signal that makes the drawer open a dialog declared in the screen's ui.json "dialogs".
# Usage: emit_dialog <dialog-id>
emit_dialog() {
    printf '{"dialog":"%s"}\n' "$1"
}

# Prints the request that asks the drawer's host for a service the script cannot reach itself.
# The app must list it in app.json "requests". Usage: emit_request rescan [quick|full] | emit_request wifi
emit_request() {
    if [ "$1" = rescan ]; then
        printf '{"request":"rescan","mode":"%s"}\n' "${2:-quick}"
    else
        printf '{"request":"%s"}\n' "$1"
    fi
}

# Starts a command detached, with every fd closed, so the drawer does not wait for it.
# Usage: start_detached <command> [args...]
start_detached() {
    "$@" </dev/null >/dev/null 2>&1 &
}

# Tells the user an outcome, the one way every app does it.
# Usage: report <toast-id> <app-id> <msg-key> [name=value...]
# Drawer: the toast signal. Command line (tests, debugging): the localized message.
report() {
    toast_id=$1
    shift
    if is_drawer; then
        emit_toast "$toast_id"
    else
        msg "$@"
    fi
}

# Logs a failure and ends the script. The drawer gets the declared "failed" toast and exit 0
# (the toast is the answer); the command line gets the reason on stdout and exit 1.
# Usage: fail_with <app_dir> <reason>
fail_with() {
    app_dir=$1
    reason=$2
    app_log "$app_dir" "$reason"
    if is_drawer; then
        emit_toast failed
        exit 0
    fi
    echo "$reason"
    exit 1
}

# Like report, then ends the script as a failure: exit 0 in the drawer, exit 1 on the command line.
# Usage: report_failure <toast-id> <app-id> <msg-key> [name=value...]
report_failure() {
    report "$@"
    if is_drawer; then
        exit 0
    fi
    exit 1
}

# Runs a quick command for an action and tells the user how it went.
# Usage: run_quick_command <app_dir> <success-toast-or-empty> <command> [args...]
# Drawer: the toast on success (nothing when empty), the failed toast on failure. Command line:
# the command's output and exit status.
run_quick_command() {
    app_dir=$1
    success_toast=$2
    shift 2
    output=$("$@" 2>&1)
    status=$?
    if ! is_drawer; then
        echo "$output"
        return "$status"
    fi
    if [ "$status" -ne 0 ]; then
        fail_with "$app_dir" "$* failed: $output"
    fi
    if [ -n "$success_toast" ]; then
        emit_toast "$success_toast"
    fi
}

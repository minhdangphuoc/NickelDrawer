# shellcheck shell=sh
# shellcheck disable=SC2034  # variables used by the scripts that source this file
# Shared by the UI Kit action scripts. Source it; do not execute.
#
# Test overrides (all optional; the defaults are the real Kobo paths):
#   UIKIT_DATA_DIR      folder for the stored control values   (default: <app>/data)
#   UIKIT_KOBO_DIR      the device's .kobo folder              (default: /mnt/onboard/.kobo)
#   UIKIT_PICTURES_DIR  the Pictures folder the gallery shows  (default: /mnt/onboard/Pictures)
#   UIKIT_COMMON        host helper library                    (default: /mnt/onboard/.adds/koboapps/lib/common.sh)
#   UIKIT_APPSCTL, UIKIT_APPS_DIR  appsctl binary and the folder with the app folders

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=../../../host/lib/common.sh
. "${UIKIT_COMMON:-/mnt/onboard/.adds/koboapps/lib/common.sh}"

APP_ID=uikit
DATA_DIR=${UIKIT_DATA_DIR:-$APP_DIR/data}
KOBO_DIR=${UIKIT_KOBO_DIR:-/mnt/onboard/.kobo}
PICTURES_DIR=${UIKIT_PICTURES_DIR:-/mnt/onboard/Pictures}
SAMPLES_DIR=$DATA_DIR/samples
JOB_FILE=$DATA_DIR/job
APPSCTL=${UIKIT_APPSCTL:-/mnt/onboard/.adds/koboapps/bin/appsctl}

# Replaces common.sh's msg() only to honour the test overrides above.
msg() {
    app=$1
    shift
    if [ -n "${UIKIT_APPS_DIR:-}" ]; then
        "$APPSCTL" msg --app "$app" --apps-dir "$UIKIT_APPS_DIR" "$@"
    else
        "$APPSCTL" msg --app "$app" "$@"
    fi
}

# The toast with the given id on the screen, or its sentence on the command line.
# Usage: echo_toast <toast-id>
echo_toast() {
    report "$1" "$APP_ID" "msg.$1"
}

# The value a control stored; $2 is the default when nothing was stored.
# Usage: stored_value <name> <default>
stored_value() {
    if [ -s "$DATA_DIR/$1" ]; then
        head -n 1 "$DATA_DIR/$1"
    else
        printf '%s' "$2"
    fi
}

# A switch's stored value as a JSON boolean: anything but "true" reads as false.
# Usage: stored_flag <name> <default>
stored_flag() {
    if [ "$(stored_value "$1" "$2")" = true ]; then
        printf true
    else
        printf false
    fi
}

# Writes a small file by temp file and rename, so a power cut never leaves half a value.
# Usage: write_file <name> <text>
write_file() {
    mkdir -p "$DATA_DIR" || fail_with "$APP_DIR" "cannot create $DATA_DIR"
    printf '%s\n' "$2" | tr -d '\r' >"$DATA_DIR/.$1.tmp" || fail_with "$APP_DIR" "cannot write $DATA_DIR/$1"
    mv "$DATA_DIR/.$1.tmp" "$DATA_DIR/$1" || fail_with "$APP_DIR" "cannot replace $DATA_DIR/$1"
}

# Remembers the last event and its value for the readout of every component page.
# Usage: record_event <action|toggle|input|tap|menu> <value>
record_event() {
    write_file lastEvent "$1"
    write_file lastValue "$2"
}

# Stores NICKELDRAWER_VALUE under the control's name, records the event for the readout and
# echoes the "saved" toast.
# Usage: remember_value <name> <event>
remember_value() {
    write_file "$1" "${NICKELDRAWER_VALUE:-}"
    record_event "$2" "${NICKELDRAWER_VALUE:-}"
    echo_toast saved
}

# Escapes text for a JSON string: backslash, quote, and control characters become spaces.
json_escape() {
    printf '%s' "$1" | tr '\r\n\t' '   ' | sed 's/\\/\\\\/g; s/"/\\"/g'
}

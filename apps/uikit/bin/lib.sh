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
# Usage: stored_flag <name>
stored_flag() {
    if [ "$(stored_value "$1" false)" = true ]; then
        printf true
    else
        printf false
    fi
}

# Stores NICKELDRAWER_VALUE under the control's name (temp file and rename, so a power cut never
# leaves half a value) and echoes the control's toast.
# Usage: remember_value <name>
remember_value() {
    name=$1
    mkdir -p "$DATA_DIR" || fail_with "$APP_DIR" "cannot create $DATA_DIR"
    printf '%s\n' "${NICKELDRAWER_VALUE:-}" | tr -d '\r' >"$DATA_DIR/.$name.tmp" ||
        fail_with "$APP_DIR" "cannot write $DATA_DIR/$name"
    mv "$DATA_DIR/.$name.tmp" "$DATA_DIR/$name" || fail_with "$APP_DIR" "cannot replace $DATA_DIR/$name"
    echo_toast "$name"
}

# Escapes text for a JSON string: backslash, quote, and control characters become spaces.
json_escape() {
    printf '%s' "$1" | tr '\r\n\t' '   ' | sed 's/\\/\\\\/g; s/"/\\"/g'
}

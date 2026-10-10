# Shared by the imageview scripts. Source it; do not execute.
#
# Test overrides (all optional; defaults are the real Kobo paths):
#   IMAGEVIEW_IMAGES_DIR  photos folder           (default: /mnt/onboard/Images)
#   IMAGEVIEW_STATE_DIR   pid and run files       (default: /tmp)
#   IMAGEVIEW_MKCBZ       album packer            (default: <app>/bin/mkcbz)
#   IMAGEVIEW_COMMON      host helper library     (default: /mnt/onboard/.adds/koboapps/lib/common.sh)
#   IMAGEVIEW_APPSCTL     appsctl binary          (default: /mnt/onboard/.adds/koboapps/bin/appsctl)
#   IMAGEVIEW_APPS_DIR    folder containing app folders (default: appsctl's own default)

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. "${IMAGEVIEW_COMMON:-/mnt/onboard/.adds/koboapps/lib/common.sh}"

APP_ID=imageview
IMAGES_DIR=${IMAGEVIEW_IMAGES_DIR:-/mnt/onboard/Images}
ALBUMS_DIR=$IMAGES_DIR/Albums
STATE_DIR=${IMAGEVIEW_STATE_DIR:-/tmp}
MKCBZ=${IMAGEVIEW_MKCBZ:-$APP_DIR/bin/mkcbz}
APPSCTL=${IMAGEVIEW_APPSCTL:-/mnt/onboard/.adds/koboapps/bin/appsctl}
PID_FILE=$STATE_DIR/imageview.pid
# The worker writes "made=N skipped=N failed=N" here; the state action reads it.
RESULT_FILE=$STATE_DIR/imageview-result.txt
# Exists after a run that made albums, until the state action hands the rescan request over.
RESCAN_FILE=$STATE_DIR/imageview-rescan

# Replaces common.sh's msg() only to honour the test overrides above.
msg() {
    app=$1
    shift
    if [ -n "${IMAGEVIEW_APPS_DIR:-}" ]; then
        "$APPSCTL" msg --app "$app" --apps-dir "$IMAGEVIEW_APPS_DIR" "$@"
    else
        "$APPSCTL" msg --app "$app" "$@"
    fi
}

is_worker_running() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

# Prints the folders that may become albums, one per line: the Images folder itself and each
# direct subfolder except Albums. Names with newlines are not supported.
list_album_folders() {
    echo "$IMAGES_DIR"
    find "$IMAGES_DIR" -mindepth 1 -maxdepth 1 -type d ! -name Albums | sort
}

folder_has_images() {
    find "$1" -maxdepth 1 -type f ! -name '.*' \( -iname '*.png' -o -iname '*.jpg' -o -iname '*.jpeg' \) | grep -q .
}

album_name_for() {
    if [ "$1" = "$IMAGES_DIR" ]; then
        echo "Images"
    else
        basename "$1"
    fi
}

# Shared by the imageview scripts. Source it; do not execute.
#
# Test overrides (all optional; defaults are the real Kobo paths):
#   IMAGEVIEW_IMAGES_DIR  photos folder           (default: /mnt/onboard/Images)
#   IMAGEVIEW_STATE_DIR   pid and run files       (default: /tmp)
#   IMAGEVIEW_MKCBZ       album packer            (default: <app>/bin/mkcbz)
#   IMAGEVIEW_APPSCTL     appsctl binary          (default: /mnt/onboard/.adds/koboapps/bin/appsctl)
#   IMAGEVIEW_APPS_DIR    folder containing app folders (default: appsctl's own default)

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
IMAGES_DIR=${IMAGEVIEW_IMAGES_DIR:-/mnt/onboard/Images}
ALBUMS_DIR=$IMAGES_DIR/Albums
STATE_DIR=${IMAGEVIEW_STATE_DIR:-/tmp}
MKCBZ=${IMAGEVIEW_MKCBZ:-$APP_DIR/bin/mkcbz}
APPSCTL=${IMAGEVIEW_APPSCTL:-/mnt/onboard/.adds/koboapps/bin/appsctl}
PID_FILE=$STATE_DIR/imageview.pid
RESULT_FILE=$STATE_DIR/imageview-result.txt

say() {
    if [ -n "$IMAGEVIEW_APPS_DIR" ]; then
        "$APPSCTL" msg --app imageview --apps-dir "$IMAGEVIEW_APPS_DIR" "$@"
    else
        "$APPSCTL" msg --app imageview "$@"
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

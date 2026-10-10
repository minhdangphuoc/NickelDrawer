#!/bin/sh
# Starts the album worker in the background and returns at once: packing photos can take far
# longer than a script may run inside the drawer. The screen follows the worker through state.sh.

. "$(dirname "$0")/lib.sh"

if [ ! -x "$MKCBZ" ]; then
    report_failure helper_missing "$APP_ID" msg.helper_missing
fi
if is_worker_running; then
    report already_running "$APP_ID" msg.already_running
    exit 0
fi
if [ ! -d "$IMAGES_DIR" ]; then
    report_failure no_images "$APP_ID" msg.no_images path="$IMAGES_DIR"
fi

mkdir -p "$STATE_DIR"
start_detached "$APP_DIR/bin/build-albums.sh"
echo $! >"$PID_FILE"
report started "$APP_ID" msg.started

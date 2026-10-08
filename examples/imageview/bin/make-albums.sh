#!/bin/sh
# Starts the album worker in the background and returns at once: packing photos can take far
# longer than the 10 s a menu dialog allows. "Status" reports the result and rescans the library.

. "$(dirname "$0")/lib.sh"

if [ ! -x "$MKCBZ" ]; then
    say msg.helper_missing
    exit 1
fi
if is_worker_running; then
    say msg.already_running
    exit 0
fi
if [ ! -d "$IMAGES_DIR" ]; then
    say msg.no_images path="$IMAGES_DIR"
    exit 1
fi

mkdir -p "$STATE_DIR"
"$APP_DIR/bin/build-albums.sh" </dev/null >/dev/null 2>&1 &
echo $! >"$PID_FILE"
say msg.started

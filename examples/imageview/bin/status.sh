#!/bin/sh
# Reports the worker state and the last result. The manifest rescans the library after it exits
# 0, so tap it once after a run to make new albums appear.

. "$(dirname "$0")/lib.sh"

if is_worker_running; then
    say msg.status_running
    exit 0
fi

album_count=$(find "$ALBUMS_DIR" -maxdepth 1 -type f -name '*.cbz' 2>/dev/null | wc -l | tr -d ' ')
say msg.status_idle
cat "$RESULT_FILE" 2>/dev/null
if [ "$album_count" -gt 0 ]; then
    say msg.status_albums count="$album_count"
else
    say msg.status_none
fi

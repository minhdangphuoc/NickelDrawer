#!/bin/sh
# Prints the ui.json variables as one JSON object. When a run has just made albums it also
# hands the host one quick rescan request ("requests"), once.

. "$(dirname "$0")/lib.sh"

made=0
skipped=0
failed=0
if [ -f "$RESULT_FILE" ]; then
    . "$RESULT_FILE"
fi
albums=$(find "$ALBUMS_DIR" -maxdepth 1 -type f -name '*.cbz' 2>/dev/null | wc -l | tr -d ' ')
working=false
requests=
if is_worker_running; then
    working=true
elif [ -f "$RESCAN_FILE" ]; then
    rm -f "$RESCAN_FILE"
    requests=',"requests":[{"request":"rescan","mode":"quick"}]'
fi
printf '{"albums":"%s","made":"%s","skipped":"%s","failed":"%s","working":%s%s}\n' \
    "$albums" "$made" "$skipped" "$failed" "$working" "$requests"

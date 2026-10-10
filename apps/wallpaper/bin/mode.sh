#!/bin/sh
# Sets the picture size. The app screen passes fill or fit in NICKELDRAWER_VALUE; without it
# (NickelMenu) the mode toggles. The mode is stored at once; while a job runs, that job re-fits
# again when it ends, so no second job starts.
. "$(dirname "$0")/lib.sh"
require_binary
mode=${NICKELDRAWER_VALUE:-toggle}
if is_drawer && is_working; then
    run_quick mode_started mode --save-only "$mode"
    exit 0
fi
run_wallpaper_in_background mode_started mode "$mode"

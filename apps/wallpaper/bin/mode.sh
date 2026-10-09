#!/bin/sh
# Sets the picture size. The app screen passes fill or fit in NICKELDRAWER_VALUE; without it
# (NickelMenu) the mode toggles.
. "$(dirname "$0")/lib.sh"
require_binary
run_wallpaper_in_background mode "${NICKELDRAWER_VALUE:-toggle}"

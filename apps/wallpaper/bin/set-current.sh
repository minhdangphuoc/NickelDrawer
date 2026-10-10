#!/bin/sh
# Makes the picture in NICKELDRAWER_VALUE (its id, e.g. 0003) the wallpaper.
. "$(dirname "$0")/lib.sh"
require_binary
require_value
run_quick set set-current "$NICKELDRAWER_VALUE"

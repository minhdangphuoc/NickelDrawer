#!/bin/sh
# Sets which picture is shown. The app screen passes fixed or random in NICKELDRAWER_VALUE;
# without it (NickelMenu) the rotation toggles.
. "$(dirname "$0")/lib.sh"
require_binary
run_wallpaper rotation "${NICKELDRAWER_VALUE:-toggle}"

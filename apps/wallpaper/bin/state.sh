#!/bin/sh
# Prints the Wallpaper screen's variables as one JSON object. Reads local files only (< 1 s).
. "$(dirname "$0")/lib.sh"
require_binary
run_wallpaper state --pid-file "$PID_FILE" --last-file "$OUTPUT_FILE"

#!/bin/sh
# Prints the Wallpaper screen's variables as one JSON object. Reads local files only (< 1 s).
. "$(dirname "$0")/lib.sh"
require_binary
start_first_import_if_due
run_wallpaper state --pid-file "$PID_FILE" --last-file "$OUTPUT_FILE" --progress-file "$PROGRESS_FILE"

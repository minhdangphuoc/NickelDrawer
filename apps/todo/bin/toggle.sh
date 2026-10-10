#!/bin/sh
# Tap on a row: an open task becomes done and a done task opens again. The row change is the
# feedback, so no toast on success.
. "$(dirname "$0")/lib.sh"
change_and_report "toggled" toggle --text "$(screen_text)"

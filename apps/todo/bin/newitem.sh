#!/bin/sh
# Adds the text typed in the field as an open task.
. "$(dirname "$0")/lib.sh"
change_and_report "" add --text "$(screen_text)"

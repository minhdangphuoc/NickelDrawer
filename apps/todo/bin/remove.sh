#!/bin/sh
# Row menu "Delete": removes the task with the pressed row's text. The screen already asked.
. "$(dirname "$0")/lib.sh"
change_and_report "" remove --text "$(screen_text)"

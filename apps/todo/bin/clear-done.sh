#!/bin/sh
# Removes every done task. The button's confirm dialog already asked.
. "$(dirname "$0")/lib.sh"
change_and_report "" clear-done

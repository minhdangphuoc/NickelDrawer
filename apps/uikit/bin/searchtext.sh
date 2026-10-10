#!/bin/sh
# Stores the control value so the next screen read shows it, then echoes the toast.
. "$(dirname "$0")/lib.sh"
remember_value searchtext

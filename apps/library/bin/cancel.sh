#!/bin/sh
# Asks the running scan or fix to stop after the book it is working on.
. "$(dirname "$0")/lib.sh"
require_binary
run_quick stopping cancel

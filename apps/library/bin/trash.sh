#!/bin/sh
# Moves the duplicate picked on the screen to data/trash (restorable for 30 days).
. "$(dirname "$0")/lib.sh"
require_binary
run_library remove-duplicate --selected --confirm

#!/bin/sh
# Puts back the duplicates trashed on the latest day; nothing that exists again is overwritten.
. "$(dirname "$0")/lib.sh"
require_binary
run_quick restored restore

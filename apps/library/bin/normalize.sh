#!/bin/sh
. "$(dirname "$0")/lib.sh"
require_binary
run_library_in_background normalize-nfc

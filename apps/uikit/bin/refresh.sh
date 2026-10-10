#!/bin/sh
# Header icon: forget the copied sample pictures so the next screen read looks at Pictures again.
. "$(dirname "$0")/lib.sh"
rm -rf "$SAMPLES_DIR"
echo_toast refresh

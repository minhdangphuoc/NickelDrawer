#!/bin/sh
# Action "check" of the Updates screen: fetches and verifies the signed release index and
# records whether an update is available. It installs nothing.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"

if [ ! -x "$APPSCTL" ]; then
    emit_toast failed
    exit 0
fi

# Run from a shell, show the result. In the drawer, return at once: the screen reads the
# result from the state, and the check may take longer than the drawer waits for a script.
if ! is_drawer; then
    exec "$APPSCTL" update check
fi
start_detached "$APPSCTL" update check
emit_toast check_started

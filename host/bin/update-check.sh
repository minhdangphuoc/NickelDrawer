#!/bin/sh
# Action "check" of the settings screen: fetches and verifies the signed release index and the
# signed app catalog, and records whether updates are available. It installs nothing. The two
# checks are independent: one failing does not stop the other.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"

if [ ! -x "$APPSCTL" ]; then
    emit_toast failed
    exit 0
fi

# Run from a shell, show the results. In the drawer, return at once: the screen reads the
# results from the state, and the checks may take longer than the drawer waits for a script.
if ! is_drawer; then
    "$APPSCTL" update check
    exec "$APPSCTL" update apps
fi
start_detached sh -c '"$0" update check; "$0" update apps' "$APPSCTL"
emit_toast check_started

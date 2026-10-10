#!/bin/sh
# Action "apply" of the settings screen: downloads, verifies and installs the latest signed
# release. An update can take minutes, so in the drawer it runs detached and the screen
# follows it through the state.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"

if [ ! -x "$APPSCTL" ]; then
    emit_toast failed
    exit 0
fi

if ! is_drawer; then
    exec "$APPSCTL" update apply
fi
start_detached "$APPSCTL" update apply
emit_toast apply_started

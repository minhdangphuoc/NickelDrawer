#!/bin/sh
# Action "apps-update-all" of the App updates screen: installs every available app update. It
# can take minutes, so in the drawer it runs detached and the screen follows it through the
# state.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"

if [ ! -x "$APPSCTL" ]; then
    emit_toast failed
    exit 0
fi

if ! is_drawer; then
    exec "$APPSCTL" update apps --all
fi
start_detached "$APPSCTL" update apps --all
emit_toast update_started

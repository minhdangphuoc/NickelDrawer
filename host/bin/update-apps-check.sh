#!/bin/sh
# Action "apps-check" of the App updates screen: fetches and verifies the signed app catalog
# and records which apps have updates. It installs nothing.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"

if [ ! -x "$APPSCTL" ]; then
    emit_toast failed
    exit 0
fi

if ! is_drawer; then
    exec "$APPSCTL" update apps
fi
start_detached "$APPSCTL" update apps
emit_toast check_started

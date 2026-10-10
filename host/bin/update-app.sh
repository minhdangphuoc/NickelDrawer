#!/bin/sh
# Row action of the App updates screen: updates or installs one app. The drawer passes the app
# id in NICKELDRAWER_VALUE; appsctl checks it against the signed catalog. Detached, like the
# other update scripts.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"
APP_ID=${NICKELDRAWER_VALUE:-${1:-}}

if [ ! -x "$APPSCTL" ] || [ -z "$APP_ID" ]; then
    emit_toast failed
    exit 0
fi

if ! is_drawer; then
    exec "$APPSCTL" update app "$APP_ID"
fi
start_detached "$APPSCTL" update app "$APP_ID"
emit_toast update_started

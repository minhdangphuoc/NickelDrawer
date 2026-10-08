#!/bin/sh
# Menu item "Update status": installed version and the result of the last update run.

. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL="$KOBOAPPS_ROOT/bin/appsctl"

if [ ! -x "$APPSCTL" ]; then
    # Without appsctl there is no localization to ask, so this one line stays English.
    echo "appsctl is missing. Reinstall from the computer."
    exit 1
fi

"$APPSCTL" update status

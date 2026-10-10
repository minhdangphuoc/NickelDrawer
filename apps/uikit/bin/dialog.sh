#!/bin/sh
# Asks the drawer to open the screen's declared info dialog "info"; on the command line there is
# no dialog, so the echo sentence stands in.
. "$(dirname "$0")/lib.sh"
if is_drawer; then
    emit_dialog info
else
    msg "$APP_ID" msg.echo
fi

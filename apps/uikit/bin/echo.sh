#!/bin/sh
# Every plain event of the kit (button, row, header link or icon, list tap, menu item, gallery
# long-press, viewer button) runs this: it tells the readout what happened and echoes the toast.
# The event kind is read from the environment the drawer gives an action:
#   NICKELDRAWER_LIST set  -> a list row menu item ("menu")
#   NICKELDRAWER_VALUE set -> a tap that carries a value: a list row or a picture ("tap")
#   neither                -> a button, row or header action ("action")
. "$(dirname "$0")/lib.sh"
if [ -n "${NICKELDRAWER_LIST:-}" ]; then
    record_event menu "${NICKELDRAWER_VALUE:-}"
elif [ -n "${NICKELDRAWER_VALUE:-}" ]; then
    record_event tap "$NICKELDRAWER_VALUE"
else
    record_event action ""
fi
echo_toast echo

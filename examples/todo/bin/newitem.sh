#!/bin/sh
# Adds the text typed in the field as an open item.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

item=$(single_line_value)
if [ -z "$item" ]; then
    report_failure empty "$APP_ID" msg.empty_item
fi
echo "[ ] $item" >>"$TODO_FILE" || fail_with "$APP_DIR" "could not append to $TODO_FILE"
report added "$APP_ID" msg.added count=1

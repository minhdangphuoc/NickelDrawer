#!/bin/sh
# Row menu "Remove": deletes the first line with exactly this text (open or done).

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

item=$(single_line_value)
temp_file=$DATA_DIR/todo.txt.tmp
awk -v item="$item" '!removed && substr($0, 5) == item && /^\[[ x]\] / { removed = 1; next } { print }' \
    "$TODO_FILE" >"$temp_file" || {
    rm -f "$temp_file"
    fail_with "$APP_DIR" "could not rewrite $TODO_FILE"
}
replace_todo_file "$temp_file"
report removed "$APP_ID" msg.removed

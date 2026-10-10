#!/bin/sh
# Tap on a row: an open item becomes done and a done item opens again.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

item=$(single_line_value)
temp_file=$DATA_DIR/todo.txt.tmp
awk -v item="$item" '
    !changed && substr($0, 1, 4) == "[ ] " && substr($0, 5) == item { $0 = "[x] " item; changed = 1 }
    !changed && substr($0, 1, 4) == "[x] " && substr($0, 5) == item { $0 = "[ ] " item; changed = 1 }
    { print }' "$TODO_FILE" >"$temp_file" || {
    rm -f "$temp_file"
    fail_with "$APP_DIR" "could not rewrite $TODO_FILE"
}
replace_todo_file "$temp_file"

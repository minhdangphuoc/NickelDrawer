#!/bin/sh
# Marks the first open item as done.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

next_item=$(grep -m 1 '^\[ \] ' "$TODO_FILE" | sed 's/^\[ \] //')
if [ -z "$next_item" ]; then
    say msg.nothing_open
    exit 0
fi

temp_file=$DATA_DIR/todo.txt.tmp
awk '!changed && /^\[ \] / { sub(/^\[ \] /, "[x] "); changed = 1 } { print }' "$TODO_FILE" >"$temp_file" || {
    rm -f "$temp_file"
    say msg.write_failed
    exit 1
}
replace_todo_file "$temp_file"
say msg.marked_done item="$next_item"

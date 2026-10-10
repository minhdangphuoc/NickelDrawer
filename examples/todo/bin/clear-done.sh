#!/bin/sh
# Removes every "[x] " line. Only items the user finished are deleted.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

done_count=$(count_lines_starting '\[x\] ')
if [ "$done_count" -eq 0 ]; then
    report nothing_done "$APP_ID" msg.nothing_done
    exit 0
fi

temp_file=$DATA_DIR/todo.txt.tmp
grep -v '^\[x\] ' "$TODO_FILE" >"$temp_file"
# grep exits 1 when nothing is left; that is a valid empty list, so only status 2 is an error.
[ $? -le 1 ] || {
    rm -f "$temp_file"
    fail_with "$APP_DIR" "could not rewrite $TODO_FILE"
}
replace_todo_file "$temp_file"
report cleared "$APP_ID" msg.cleared count="$done_count"

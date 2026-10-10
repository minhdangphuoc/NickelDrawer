#!/bin/sh
# Appends the lines of the first non-empty inbox file as open items, then empties that file.
# The inbox is emptied (not deleted) so a later USB copy or LocalSend transfer can refill it.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

first_filled_inbox() {
    for inbox in $INBOXES; do
        if [ -s "$inbox" ]; then
            echo "$inbox"
            return 0
        fi
    done
    return 1
}

# Drops CR (files written on Windows), blank lines and surrounding spaces; adds the open prefix.
format_inbox_lines() {
    tr -d '\r' <"$1" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//' | grep -v '^$' | sed 's/^/[ ] /'
}

inbox=$(first_filled_inbox) || {
    report inbox_empty "$APP_ID" msg.inbox_empty path="${INBOXES%% *}"
    exit 0
}

temp_file=$DATA_DIR/todo.txt.tmp
{
    cat "$TODO_FILE"
    format_inbox_lines "$inbox"
} >"$temp_file" || {
    rm -f "$temp_file"
    fail_with "$APP_DIR" "could not write $temp_file"
}
added_count=$(format_inbox_lines "$inbox" | wc -l | tr -d ' ')
replace_todo_file "$temp_file"
: >"$inbox"
report added "$APP_ID" msg.added count="$added_count"

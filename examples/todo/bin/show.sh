#!/bin/sh
# Prints the counts and up to 10 open items. Stays within 12 lines for the dialog.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

MAX_SHOWN=10
open_count=$(count_lines_starting '\[ \] ')
done_count=$(count_lines_starting '\[x\] ')

if [ "$open_count" -eq 0 ] && [ "$done_count" -eq 0 ]; then
    say msg.list_empty
    exit 0
fi

say msg.list_counts open="$open_count" done="$done_count"
grep '^\[ \] ' "$TODO_FILE" | head -n "$MAX_SHOWN" | sed 's/^\[ \] //' | awk '{ print NR ". " $0 }'
if [ "$open_count" -gt "$MAX_SHOWN" ]; then
    say msg.list_more count=$((open_count - MAX_SHOWN))
fi

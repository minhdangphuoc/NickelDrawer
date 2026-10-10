#!/bin/sh
# Prints the ui.json variables as one JSON object: counts, the rows and the "hide done" switch.
# Each row is [text, "✓" for a done item or ""]; the text is escaped for JSON by awk.

. "$(dirname "$0")/lib.sh"
ensure_todo_file || exit 1

open_count=$(count_lines_starting '\[ \] ')
done_count=$(count_lines_starting '\[x\] ')
hide_done=false
[ -f "$HIDE_DONE_FILE" ] && hide_done=true

awk -v open="$open_count" -v done="$done_count" -v hide="$hide_done" -v doneword="✓" '
function escape(text) {
    gsub(/\\/, "\\\\", text)
    gsub(/"/, "\\\"", text)
    return text
}
BEGIN { rows = "" }
/^\[ \] / || /^\[x\] / {
    isDone = (substr($0, 2, 1) == "x")
    if (isDone && hide == "true") next
    row = "[\"" escape(substr($0, 5)) "\",\"" (isDone ? escape(doneword) : "") "\"]"
    rows = rows (rows == "" ? "" : ",") row
}
END {
    printf "{\"open\":\"%s\",\"done\":\"%s\",\"hidedone\":%s,\"newitem\":\"\",\"items\":[%s]}\n", open, done, hide, rows
}' "$TODO_FILE"

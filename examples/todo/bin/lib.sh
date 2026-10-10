# Shared by the todo actions. Source it; do not execute.
#
# Test overrides (all optional; the defaults are the real Kobo paths):
#   TODO_DATA_DIR  folder holding todo.txt        (default: <app>/data)
#   TODO_INBOXES   space-separated inbox files    (default: /mnt/onboard/todo-inbox.txt
#                                                  and /mnt/onboard/LocalSend/todo-inbox.txt)
#   TODO_COMMON    host helper library            (default: /mnt/onboard/.adds/koboapps/lib/common.sh)
#   TODO_APPSCTL   appsctl binary                 (default: /mnt/onboard/.adds/koboapps/bin/appsctl)
#   TODO_APPS_DIR  folder containing app folders  (default: appsctl's own default)

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. "${TODO_COMMON:-/mnt/onboard/.adds/koboapps/lib/common.sh}"

APP_ID=todo
DATA_DIR=${TODO_DATA_DIR:-$APP_DIR/data}
TODO_FILE=$DATA_DIR/todo.txt
HIDE_DONE_FILE=$DATA_DIR/hide-done
INBOXES=${TODO_INBOXES:-/mnt/onboard/todo-inbox.txt /mnt/onboard/LocalSend/todo-inbox.txt}
APPSCTL=${TODO_APPSCTL:-/mnt/onboard/.adds/koboapps/bin/appsctl}

# Replaces common.sh's msg() only to honour the test overrides above.
msg() {
    app=$1
    shift
    if [ -n "${TODO_APPS_DIR:-}" ]; then
        "$APPSCTL" msg --app "$app" --apps-dir "$TODO_APPS_DIR" "$@"
    else
        "$APPSCTL" msg --app "$app" "$@"
    fi
}

ensure_todo_file() {
    mkdir -p "$DATA_DIR" && touch "$TODO_FILE"
}

# Replaces todo.txt with the file $1. The temp file sits in data/ so mv is a rename on the same
# filesystem and the list is never half written.
replace_todo_file() {
    mv "$1" "$TODO_FILE" || {
        rm -f "$1"
        fail_with "$APP_DIR" "could not replace $TODO_FILE"
    }
}

count_lines_starting() {
    grep -c "^$1" "$TODO_FILE" || true
}

# The item text the screen sends (the tapped row, or the typed field), one line, trimmed.
single_line_value() {
    printf '%s' "${NICKELDRAWER_VALUE:-}" | tr '\r\n\t' '   ' | sed 's/^ *//; s/ *$//'
}

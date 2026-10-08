# Shared by the todo actions. Source it; do not execute.
#
# Test overrides (all optional; the defaults are the real Kobo paths):
#   TODO_DATA_DIR  folder holding todo.txt        (default: <app>/data)
#   TODO_INBOXES   space-separated inbox files    (default: /mnt/onboard/todo-inbox.txt
#                                                  and /mnt/onboard/LocalSend/todo-inbox.txt)
#   TODO_APPSCTL   appsctl binary                 (default: /mnt/onboard/.adds/koboapps/bin/appsctl)
#   TODO_APPS_DIR  folder containing app folders  (default: appsctl's own default)

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
DATA_DIR=${TODO_DATA_DIR:-$APP_DIR/data}
TODO_FILE=$DATA_DIR/todo.txt
INBOXES=${TODO_INBOXES:-/mnt/onboard/todo-inbox.txt /mnt/onboard/LocalSend/todo-inbox.txt}
APPSCTL=${TODO_APPSCTL:-/mnt/onboard/.adds/koboapps/bin/appsctl}

# Prints a translated message: say <key> [name=value...]
say() {
    if [ -n "$TODO_APPS_DIR" ]; then
        "$APPSCTL" msg --app todo --apps-dir "$TODO_APPS_DIR" "$@"
    else
        "$APPSCTL" msg --app todo "$@"
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
        say msg.write_failed
        exit 1
    }
}

count_lines_starting() {
    grep -c "^$1" "$TODO_FILE" || true
}

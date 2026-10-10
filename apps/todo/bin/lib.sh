# Shared by the To-do action scripts. Source it; do not execute.
#
# Test overrides (all optional; the defaults are the real Kobo paths):
#   TODO_DATA_DIR  folder holding todo.json   (default: <app>/data)
#   TODO_COMMON    host helper library        (default: /mnt/onboard/.adds/koboapps/lib/common.sh)
#   TODO_APPSCTL   appsctl binary             (default: /mnt/onboard/.adds/koboapps/bin/appsctl)
#   TODO_APPS_DIR  folder containing app folders (default: appsctl's own default)

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. "${TODO_COMMON:-/mnt/onboard/.adds/koboapps/lib/common.sh}"

APP_ID=todo
DATA_DIR=${TODO_DATA_DIR:-$APP_DIR/data}
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

require_binary() {
    if [ ! -x "$APP_DIR/bin/todo" ]; then
        fail_with "$APP_DIR" "$(msg "$APP_ID" msg.binary_missing)"
    fi
}

# Usage: run_todo <subcommand> [args...]
run_todo() {
    subcommand=$1
    shift
    "$APP_DIR/bin/todo" "$subcommand" --data-dir "$DATA_DIR" "$@"
}

# The text the screen sends (the typed field or the pressed row), one line; the helper cleans it.
screen_text() {
    printf '%s' "${NICKELDRAWER_VALUE:-}" | tr '\r\n\t' '   '
}

# Runs a changing subcommand and reports its outcome word as the declared toast of the same
# name. An outcome in $silent_outcomes (a space-separated list) shows no toast.
# Usage: change_and_report <silent_outcomes> <subcommand> [args...]
change_and_report() {
    silent_outcomes=$1
    shift
    require_binary
    if ! reply=$(run_todo "$@" 2>&1); then
        fail_with "$APP_DIR" "todo $1 failed: $reply"
    fi
    outcome=${reply%% *}
    count=${reply#"$outcome"}
    case " $silent_outcomes " in
    *" $outcome "*) return 0 ;;
    esac
    report "$outcome" "$APP_ID" "msg.$outcome" count="${count# }"
}

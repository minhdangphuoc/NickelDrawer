# Shared settings for the LocalSend scripts. Source it; do not execute.
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=localsend
APP_DIR=$APPS_ROOT/$APP_ID
DATA_DIR=$APP_DIR/data
BINARY=$APP_DIR/bin/localgo
INBOX=/mnt/onboard/.adds/koboapps/inbox
# Where earlier versions received files; "Sort old files" moves them into the library.
LEGACY_FOLDER=/mnt/onboard/LocalSend
APPSCTL=$KOBOAPPS_ROOT/bin/appsctl
# localgo runs this through `sh -c` after each received file, once per file and concurrently.
# SECURITY: it must never contain %f, %n, %a or any other %-placeholder. localgo pastes those
# straight into the shell string, and the file name and sender alias are chosen by whoever sends
# the file, so `%n` would let a sender run commands on the Kobo. The router reads the path from
# the LOCALGO_FILE environment variable instead, which the shell never parses.
ROUTE_HOOK="$APPSCTL inbox route --from-env --source localsend"
# Under /tmp so a stale pid never survives a reboot.
PIDFILE=/tmp/localsend.pid
# localgo logs to /tmp while running so nothing stays open on /mnt/onboard (USB mass storage);
# stop.sh appends it to data/log.txt.
TMP_LOG=/tmp/localsend.log
MAX_LOG_LINES=500
# How long stop.sh lets running hooks finish before it sorts what is left.
HOOK_WAIT_SECONDS=2

# localgo writes config and state under XDG_*; keep all of it inside data/.
export HOME=$DATA_DIR/home
export XDG_CONFIG_HOME=$HOME/config XDG_DATA_HOME=$HOME/data
export XDG_STATE_HOME=$HOME/state XDG_CACHE_HOME=$HOME/cache

is_running() {
    [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null
}

# Prints the localized "Books: 2 / Rejected: 1 ..." summary of the last receiving session.
print_inbox_summary() {
    "$APPSCTL" inbox status
}

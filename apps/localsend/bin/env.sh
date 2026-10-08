# Shared settings for the LocalSend scripts. Source it; do not execute.
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=localsend
APP_DIR=$APPS_ROOT/$APP_ID
DATA_DIR=$APP_DIR/data
BINARY=$APP_DIR/bin/localgo
INBOX=/mnt/onboard/LocalSend
# Under /tmp so a stale pid never survives a reboot.
PIDFILE=/tmp/localsend.pid
# localgo logs to /tmp while running so nothing stays open on /mnt/onboard (USB mass storage);
# stop.sh appends it to data/log.txt.
TMP_LOG=/tmp/localsend.log
MAX_LOG_LINES=500

# localgo writes config and state under XDG_*; keep all of it inside data/.
export HOME=$DATA_DIR/home
export XDG_CONFIG_HOME=$HOME/config XDG_DATA_HOME=$HOME/data
export XDG_STATE_HOME=$HOME/state XDG_CACHE_HOME=$HOME/cache

is_running() {
    [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null
}

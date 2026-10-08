# Shared helpers for koboapps app scripts. Source it; do not execute.
# Installed on the Kobo at /mnt/onboard/.adds/koboapps/lib/common.sh

KOBOAPPS_ROOT=/mnt/onboard/.adds/koboapps
APPS_ROOT=/mnt/onboard/.adds/apps

# Prints an app message in the device language.
# Usage: msg <app-id> <key> [name=value...]   (strings live in <app>/locales/<lang>.json)
# An unknown key prints the key itself, so a missing string never breaks a script.
msg() {
    "$KOBOAPPS_ROOT/bin/appsctl" msg --app "$@"
}

# Prints the Kobo model name (from .kobo/version), "Kobo" when unknown.
device_name() {
    "$KOBOAPPS_ROOT/bin/appsctl" device-name
}

# Prints the first non-loopback IPv4 address, or nothing.
current_ip() {
    /sbin/ifconfig 2>/dev/null |
        awk '/inet addr/ { ip = substr($2, 6); if (ip != "127.0.0.1") { print ip; exit } }'
}

# Waits up to $1 seconds for an IP. Prints it and returns 0, or returns 1.
wait_for_wifi() {
    seconds=${1:-8}
    while [ "$seconds" -gt 0 ]; do
        ip=$(current_ip)
        if [ -n "$ip" ]; then
            echo "$ip"
            return 0
        fi
        sleep 1
        seconds=$((seconds - 1))
    done
    return 1
}

# Appends a timestamped line to the calling app's data/log.txt.
# Usage: app_log <app_dir> <message...>
app_log() {
    app_dir=$1
    shift
    mkdir -p "$app_dir/data"
    echo "$(date '+%Y-%m-%d %H:%M:%S') $*" >>"$app_dir/data/log.txt"
}

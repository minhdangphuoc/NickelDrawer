#!/bin/sh
# Prints the LocalSend screen's variables as one JSON object. Reads local files only (< 1 s):
# no network, no waiting for Wi-Fi. The PIN is shown only while the receiver runs.
. "$(dirname "$0")/env.sh"

json_escape() {
    printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | tr -d '\n\r\t'
}

# "summary" and "received" come from the inbox journal; without them the screen shows its
# empty state.
inbox_fields() {
    fields=$("$APPSCTL" inbox state 2>/dev/null)
    case $fields in
    '{'*'}') printf '%s' "${fields#\{}" ;;
    *) printf '"summary":"","received":[]}' ;;
    esac
}

if is_running; then
    running=true
    status=$(msg "$APP_ID" ui.status_receiving)
    pin=$(cat "$DATA_DIR/pin.txt" 2>/dev/null)
    ip=$(current_ip)
else
    running=false
    status=$(msg "$APP_ID" ui.status_idle)
    pin=""
    ip=""
fi

printf '{"running":%s,"status":"%s","pin":"%s","device":"%s","ip":"%s",%s\n' \
    "$running" "$(json_escape "$status")" "$(json_escape "$pin")" "$(json_escape "$(device_name)")" \
    "$(json_escape "$ip")" "$(inbox_fields)"

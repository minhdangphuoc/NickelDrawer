#!/bin/bash
# Asks Kobo's upgrade API whether a Libra 2 firmware newer than the newest one in re/firmware.lock exists.
#   tools/fw-watch.sh [--fetch] [--current VERSION]
# Exit codes: 0 no update, 10 new firmware, 1 error.
# --fetch downloads the new package into ~/.cache/nickeldrawer (never the repo) via re/scripts/fetch-fw.sh
# and then runs symcheck. Without it, only the commands are printed. Uses a placeholder serial.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
LOCK_FILE=$REPO_ROOT/re/firmware.lock
LIBRA2_DEVICE_ID=00000000-0000-0000-0000-000000000388
PLACEHOLDER_SERIAL=N000000000000
EXIT_NEW_FIRMWARE=10

shouldFetch=false
currentVersion=

fail() {
    echo "fw-watch: $*" >&2
    exit 1
}

parse_flags() {
    while [ $# -gt 0 ]; do
        case $1 in
        --fetch) shouldFetch=true ;;
        --current)
            [ $# -ge 2 ] || fail "--current needs a version. Usage: $0 [--fetch] [--current VERSION]"
            currentVersion=$2
            shift
            ;;
        *) fail "unknown flag '$1'. Usage: $0 [--fetch] [--current VERSION]" ;;
        esac
        shift
    done
}

newest_locked_version() {
    grep -v '^#' "$LOCK_FILE" | grep -v '^version' | cut -f1 | sort -V | tail -n 1
}

query_upgrade_url() {
    local response
    response=$(curl -fsS -m 30 "https://api.kobobooks.com/1.0/UpgradeCheck/Device/$LIBRA2_DEVICE_ID/kobo/$currentVersion/$PLACEHOLDER_SERIAL") ||
        fail "upgrade API request failed. Check the network and retry."
    printf '%s' "$response" | python3 -I -c 'import json,sys; print(json.load(sys.stdin).get("UpgradeURL") or "")' ||
        fail "unexpected upgrade API response: $response"
}

is_newer() {
    [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -n 1)" = "$1" ]
}

print_new_firmware() {
    local url=$1 version=$2 month=$3
    echo "NEW FIRMWARE: $version (newest in firmware.lock: $currentVersion)"
    echo "URL: $url"
    case $version in
    5.*) echo "WARNING: 5.x is the Qt6 upgrade. NickelMenu and NickelHook mods do not support it; the drawer stays off. Report only, do not analyse (docs/DECISIONS.md)." ;;
    esac
    echo
    echo "Fetch, verify it is a Libra 2 image and dump exports (downloads to ~/.cache/nickeldrawer):"
    echo "  re/scripts/fetch-fw.sh $version $month"
    echo "Then add the 'lock:' line it prints to re/firmware.lock, add $version to builds: in the symbols list, and run:"
    echo "  python3 -I tools/symcheck.py"
}

fetch_and_check() {
    local version=$1 month=$2
    "$REPO_ROOT/re/scripts/fetch-fw.sh" "$version" "$month"
    echo "Add $version to builds: in the symbols list, then:"
    python3 -I "$REPO_ROOT/tools/symcheck.py" --builds "$(newest_locked_version),$version" || true
}

main() {
    parse_flags "$@"
    [ -f "$LOCK_FILE" ] || fail "missing $LOCK_FILE"
    : "${currentVersion:=$(newest_locked_version)}"
    local url version month
    url=$(query_upgrade_url)
    if [ -z "$url" ]; then
        echo "No update: API offers nothing newer than $currentVersion."
        exit 0
    fi
    version=$(basename "$url" .zip)
    version=${version#kobo-update-}
    month=$(basename "$(dirname "$url")")
    if ! is_newer "$version" "$(newest_locked_version)"; then
        echo "No update: API offers $version, already covered by firmware.lock."
        exit 0
    fi
    print_new_firmware "$url" "$version" "$month"
    if $shouldFetch; then
        fetch_and_check "$version" "$month"
    fi
    exit "$EXIT_NEW_FIRMWARE"
}

main "$@"

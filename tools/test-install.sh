#!/bin/bash
# Exercises tools/install-to-kobo.sh against a fake Kobo tree in a temp dir. Never touches a real volume.
# Needs `make build dist` first. The backups the installer makes are removed again at the end.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
INSTALLER=$REPO_ROOT/tools/install-to-kobo.sh
WORK=$(mktemp -d)
export KOBO_ROOT=$WORK/KOBOeReader
ADDS=$KOBO_ROOT/.adds
failures=0
backupsBefore=$(ls "$REPO_ROOT/backups" 2>/dev/null || true)

cleanup() {
    rm -rf "$WORK"
    local backup
    for backup in "$REPO_ROOT"/backups/*; do
        [ -e "$backup" ] || continue
        case " $(echo $backupsBefore) " in *" $(basename "$backup") "*) ;; *) rm -rf "$backup" ;; esac
    done
}
trap cleanup EXIT

check() {
    local description=$1
    shift
    if "$@"; then
        echo "ok   - $description"
    else
        echo "FAIL - $description"
        failures=$((failures + 1))
    fi
}

contains() { grep -qF -- "$2" "$1"; }
lacks() { ! grep -qF -- "$2" "$1"; }

write_kobo_conf() {
    mkdir -p "$KOBO_ROOT/.kobo/Kobo"
    printf '[ApplicationPreferences]\nCurrentLocale=%s\n' "$1" >"$KOBO_ROOT/.kobo/Kobo/Kobo eReader.conf"
}

build_fake_kobo() {
    mkdir -p "$KOBO_ROOT/.kobo" "$ADDS/nm" "$ADDS/localsend"
    write_kobo_conf vi
    cp "$REPO_ROOT/internal/nmconfig/testdata/legacy-items.txt" "$ADDS/nm/items"
    echo '#!/bin/sh' >"$ADDS/localsend/start.sh"
}

installed_backup_count() { ls "$REPO_ROOT/backups" | wc -l | tr -d ' '; }

build_fake_kobo
cp "$ADDS/nm/items" "$WORK/items.original"

echo "--- dry run"
"$INSTALLER" --dry-run >"$WORK/dry.out"
check "dry run creates no koboapps folder" test ! -e "$ADDS/koboapps"
check "dry run prints actions" contains "$WORK/dry.out" "[dry-run]"

echo "--- first install"
"$INSTALLER" >"$WORK/first.out"
check "appsctl installed and executable" test -x "$ADDS/koboapps/bin/appsctl"
check "localgo installed and executable" test -x "$ADDS/apps/localsend/bin/localgo"
check "ghsync installed" test -x "$ADDS/apps/ghsync/bin/ghsync"
check "config.example.json installed" test -f "$ADDS/apps/ghsync/data/config.example.json"
check "no other ghsync data files" test "$(ls "$ADDS/apps/ghsync/data" | wc -l | tr -d ' ')" = 1
check "menu has device paths" contains "$ADDS/nm/koboapps" ":/mnt/onboard/.adds/apps/localsend/bin/start.sh"
check "menu has refresh item" contains "$ADDS/nm/koboapps" "/mnt/onboard/.adds/koboapps/bin/appsctl refresh"
check "locales installed (host)" test -f "$ADDS/koboapps/locales/vi.json"
check "locales installed (app)" test -f "$ADDS/apps/localsend/locales/vi.json"
check "menu in Vietnamese (app labels)" contains "$ADDS/nm/koboapps" "LocalSend · Bắt đầu nhận"
check "menu in Vietnamese (refresh item)" contains "$ADDS/nm/koboapps" "Ứng dụng · Làm mới danh sách"
check "menu has update check item (vi)" contains "$ADDS/nm/koboapps" "Ứng dụng · Kiểm tra cập nhật"
check "menu has update status item (vi)" contains "$ADDS/nm/koboapps" "Ứng dụng · Trạng thái cập nhật"
check "menu runs the update scripts" contains "$ADDS/nm/koboapps" "/mnt/onboard/.adds/koboapps/bin/update-apply.sh"
check "update-apply.sh installed and executable" test -x "$ADDS/koboapps/bin/update-apply.sh"
check "update-status.sh installed and executable" test -x "$ADDS/koboapps/bin/update-status.sh"
check "menu has no English label under vi" lacks "$ADDS/nm/koboapps" "Start receiving"
check "menu has no volume path" lacks "$ADDS/nm/koboapps" "$KOBO_ROOT"
check "items untouched without --remove-legacy" cmp -s "$ADDS/nm/items" "$WORK/items.original"
check "legacy folder kept without --remove-legacy" test -d "$ADDS/localsend"
check "backup created" test "$(installed_backup_count)" -ge 1

echo "--- second install keeps data"
echo "secret-token" >"$ADDS/apps/ghsync/data/token"
echo '{"repo":"o/r"}' >"$ADDS/apps/ghsync/data/config.json"
echo "1234" >"$ADDS/apps/localsend/data/pin.txt" 2>/dev/null || { mkdir -p "$ADDS/apps/localsend/data"; echo 1234 >"$ADDS/apps/localsend/data/pin.txt"; }
echo "stale" >"$ADDS/apps/ghsync/bin/obsolete.sh"
cp "$ADDS/nm/koboapps" "$WORK/menu.first"
sleep 1
"$INSTALLER" >"$WORK/second.out"
check "token preserved" contains "$ADDS/apps/ghsync/data/token" "secret-token"
check "config preserved" contains "$ADDS/apps/ghsync/data/config.json" '"o/r"'
check "pin preserved" contains "$ADDS/apps/localsend/data/pin.txt" "1234"
check "bin/ replaced (stale file gone)" test ! -e "$ADDS/apps/ghsync/bin/obsolete.sh"
check "menu identical after reinstall" cmp -s "$ADDS/nm/koboapps" "$WORK/menu.first"
check "second backup holds the token" grep -rqF secret-token "$REPO_ROOT"/backups/*/.adds/apps/ghsync/data/token

echo "--- unsupported language falls back to English"
write_kobo_conf fr
sleep 1
"$INSTALLER" >"$WORK/french.out"
check "fr menu uses English app labels" contains "$ADDS/nm/koboapps" "LocalSend · Start receiving"
check "fr menu uses English refresh item" contains "$ADDS/nm/koboapps" "Apps · Refresh list"
check "fr menu has update check item" contains "$ADDS/nm/koboapps" "Apps · Check for updates"
check "fr menu has update status item" contains "$ADDS/nm/koboapps" "Apps · Update status"
write_kobo_conf vi

echo "--- host strings without a locales folder"
"$REPO_ROOT/build/appsctl-host" host-msg --locale vi --koboapps-dir "$WORK/no-such-koboapps" menu.update_check >"$WORK/fallback.out"
check "missing locales folder shows English, not the key" contains "$WORK/fallback.out" "Apps · Check for updates"
"$REPO_ROOT/build/appsctl-host" update status --onboard-root "$KOBO_ROOT" --locale vi >"$WORK/update-status.out"
check "update status is localized before any update" contains "$WORK/update-status.out" "Chưa cập nhật lần nào."

echo "--- remove legacy"
sleep 1
"$INSTALLER" --remove-legacy >"$WORK/legacy.out"
check "legacy folder removed" test ! -e "$ADDS/localsend"
check "legacy start item removed" lacks "$ADDS/nm/items" "Nhận sách LocalSend"
check "legacy stop item removed" lacks "$ADDS/nm/items" "Dừng LocalSend"
check "no legacy path left in items" lacks "$ADDS/nm/items" ".adds/localsend/"
check "other items intact" contains "$ADDS/nm/items" "Library Reload"
expectedLines=$(($(wc -l <"$WORK/items.original") - 4))
check "exactly 4 lines removed" test "$(wc -l <"$ADDS/nm/items" | tr -d ' ')" = "$expectedLines"
legacyBackup=$(ls -d "$REPO_ROOT"/backups/*/.adds/localsend 2>/dev/null | head -n 1 || true)
check "legacy folder backed up" test -n "$legacyBackup"
"$INSTALLER" --remove-legacy >/dev/null
check "remove-legacy is idempotent" lacks "$ADDS/nm/items" "LocalSend :"

echo "--- guards"
KOBO_ROOT=$WORK/nothing "$INSTALLER" >"$WORK/missing.out" 2>&1 && missingStatus=0 || missingStatus=$?
check "refuses a volume without .kobo" test "$missingStatus" -ne 0

if [ "$failures" -ne 0 ]; then
    echo "$failures check(s) failed"
    exit 1
fi
echo "all checks passed"

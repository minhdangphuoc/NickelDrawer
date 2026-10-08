# App format v1 (Phase 1)

An app is one folder on the Kobo at `/mnt/onboard/.adds/apps/<id>/`. Copy the folder in to install; delete it to uninstall; then tap **Apps · Refresh list** (shown in the device language).

Phase 1 exposes apps through NickelMenu only. Later phases (native drawer, UI kit) add optional fields; v1 manifests stay valid.

## Folder layout

```
.adds/apps/<id>/
├─ app.json        manifest (required)
├─ bin/            executables and scripts the actions run
├─ data/           app-owned state: config, tokens, logs (never shipped in a release zip)
└─ icon.svg        optional, unused until Phase 3
```

Host files (not an app, not scanned):

```
.adds/koboapps/
├─ bin/appsctl     host tool (Go, linux/armv7): validates manifests, generates the menu
└─ lib/common.sh   shared shell helpers for app scripts
.adds/nm/koboapps  generated NickelMenu config (do not edit by hand)
```

NickelMenu re-reads any config file under `.adds/nm/` whose mtime changed the next time a menu opens (verified in NickelMenu `src/config.c`, `nm_config_files_update`), so regenerating `.adds/nm/koboapps` needs no reboot.

## app.json

```json
{
  "schema": 1,
  "id": "localsend",
  "name": "LocalSend",
  "version": "1.0.0",
  "description": "Receive books from your phone with LocalSend",
  "actions": [
    {
      "id": "start",
      "label": "Start receiving",
      "run": "bin/start.sh",
      "output": "show",
      "timeout_ms": 9500,
      "wifi": true,
      "rescan": "none"
    },
    {
      "id": "stop",
      "label": "Stop receiving",
      "run": "bin/stop.sh",
      "output": "show",
      "timeout_ms": 3000,
      "rescan": "quick"
    }
  ]
}
```

| Field | Type | Rules |
|---|---|---|
| `schema` | int | must be `1` |
| `id` | string | `^[a-z0-9][a-z0-9-]{1,30}$`, must equal the folder name |
| `name` | string | 1–24 chars, no `:` |
| `version` | string | semver `MAJOR.MINOR.PATCH` |
| `description` | string | optional, ≤ 120 chars |
| `menu` | bool | optional, default `true`; `false` hides the app from the menu |
| `actions` | array | 1–8 items, `id` unique within the app |
| `actions[].id` | string | `^[a-z0-9][a-z0-9-]{0,30}$` |
| `actions[].label` | string | 1–32 chars, no `:` (NickelMenu field separator) |
| `actions[].run` | string | path relative to the app folder, no `..`, no leading `/`; must exist and be executable |
| `actions[].output` | enum | `show` = NickelMenu `cmd_output`, prints stdout in a dialog (keep under ~12 short lines); `none` = `cmd_spawn:quiet`, runs in background |
| `actions[].timeout_ms` | int | only with `output: show`; 500–9999 (NickelMenu limit is < 10000); default 5000 |
| `actions[].wifi` | bool | default `false`; `true` prepends `nickel_wifi:autoconnect` (does not wait; the script must wait for an IP itself, see `wait_for_wifi` in `common.sh`) |
| `actions[].rescan` | enum | `none` (default), `quick` = `nickel_misc:rescan_books`, `full` = `nickel_misc:rescan_books_full`; runs only if the action succeeded |

Unknown top-level or action fields are ignored with a warning (forward compatibility).

## Localization

`name`, `description` and `actions[].label` in `app.json` are the **English** source strings. Translations live next to the manifest:

```
.adds/apps/<id>/locales/vi.json
.adds/apps/<id>/locales/pt_BR.json
```

```json
{
  "name": "LocalSend",
  "description": "Nhận sách từ điện thoại qua LocalSend",
  "actions.start.label": "Bắt đầu nhận",
  "actions.stop.label": "Dừng nhận",
  "msg.wifi_missing": "Chưa kết nối Wi-Fi. Đợi rồi bấm lại.",
  "msg.started": "Đang nhận sách\nTên máy: {device}\nIP: {ip}\nPIN: {pin}"
}
```

- Keys: `name`, `description`, `actions.<id>.label` for manifest strings; `msg.<key>` for messages printed by the app's scripts or binaries. The English `msg.*` strings live in `locales/en.json` (required if the app prints messages).
- Placeholders are `{name}`; a translation must use the same set of placeholders as English (validated). `\n` allowed for multi-line `cmd_output` text.
- Same length and `:` rules as the English fields apply to translated labels.
- Locale resolution: read `CurrentLocale` from `/mnt/onboard/.kobo/Kobo/Kobo eReader.conf`; try the exact file (`pt_BR.json`), then the language (`pt.json`), then English. Missing keys fall back key by key.
- Scripts print messages with `appsctl msg --app <id> <key> [name=value ...]`, never literal text.
- Host strings (the refresh/error items, `appsctl` output) live in `/mnt/onboard/.adds/koboapps/locales/<lang>.json`.
- `appsctl gen-menu` takes `--locale CODE` (override), `--conf FILE` (default `Kobo eReader.conf`) and `--koboapps-dir DIR` (host `locales/` lives there); the installer passes the mounted Kobo's conf and folder.
- `appsctl validate` checks locale files: valid JSON, keys known to English (unknown keys warn), same placeholders, translated name/label length and `:` rules.
- `appsctl msg` prints the key and exits 0 when a string is missing, so a script never fails over a translation.
- `appsctl device-name` prints the model from `/mnt/onboard/.kobo/version` (apps use it instead of a hardcoded device name). See `docs/TRANSLATING.md` to add a language.
- The menu is rendered in the current locale; after changing the Kobo's language, tap the refresh item (its label is in the old language until then).

## Generated NickelMenu config

`appsctl gen-menu` writes `.adds/nm/koboapps` atomically (temp file + rename), in the device locale. Example below is rendered with `CurrentLocale=vi`; host labels come from `koboapps/locales/vi.json` (`menu.refresh` = "Ứng dụng · Làm mới danh sách", `menu.errors` = "Ứng dụng · Lỗi ({count})"). For each app (sorted by `name`) and action (manifest order):

```
# generated by appsctl — do not edit
menu_item :main :LocalSend · Bắt đầu nhận :nickel_wifi :autoconnect
  chain_success :cmd_output :9500 :/mnt/onboard/.adds/apps/localsend/bin/start.sh
menu_item :main :LocalSend · Dừng nhận :cmd_output :3000 :/mnt/onboard/.adds/apps/localsend/bin/stop.sh
  chain_success :nickel_misc :rescan_books
```

- `output: none` → `:cmd_spawn :quiet :<path>`.
- Paths are absolute on the device (`/mnt/onboard/.adds/apps/<id>/<run>`). Paths containing `:` are rejected at validation.
- Invalid apps are skipped and listed in one trailing item: `menu_item :main :Ứng dụng · Lỗi (N) :cmd_output :2000 :/mnt/onboard/.adds/koboapps/bin/appsctl errors`.
- Always appended system item: `menu_item :main :Ứng dụng · Làm mới danh sách :cmd_output :5000 :/mnt/onboard/.adds/koboapps/bin/appsctl refresh`.

## Script contract

- NickelMenu runs commands from `/` as root with a minimal environment: scripts must `cd "$(dirname "$0")/.."` (or use absolute paths) and source `/mnt/onboard/.adds/koboapps/lib/common.sh`.
- Exit code `0` = success (chains `rescan`), non-zero = failure (NickelMenu shows the error).
- `output: show` scripts print short status lines through `msg` (never literal text); they must finish before `timeout_ms`. Long work goes to the background with output redirected to `data/log.txt`.
- Never hold files open under `/mnt/onboard` after the script exits; never write outside the app's own `data/` except the app's declared destination folder (e.g. `/mnt/onboard/LocalSend`, `/mnt/onboard/GitHub`).
- Secrets live in `data/` only, file mode 600, never printed.

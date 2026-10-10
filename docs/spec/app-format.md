# App format v1 (Phase 1)

An app is one folder on the Kobo at `/mnt/onboard/.adds/apps/<id>/`. Copy the folder in to install; delete it to uninstall; then tap **Apps · Refresh list** (shown in the device language).

Phase 1 exposes apps through NickelMenu only. Later phases (native drawer, UI kit) add optional fields; v1 manifests stay valid.

## Folder layout

```
.adds/apps/<id>/
├─ app.json        manifest (required)
├─ bin/            executables and scripts the actions run
├─ data/           app-owned state: config, tokens, logs (never shipped in a release zip)
├─ icon.png        drawer icon (see below)
└─ ui.json         optional native screen, see ui-json.md
```

Host files (not an app, not scanned):

```
.adds/koboapps/
├─ bin/appsctl            host tool (Go, linux/armv7): validates manifests, generates the menu, applies signed updates
├─ bin/update-apply.sh    menu script: starts `appsctl update apply` in the background
├─ bin/update-status.sh   menu script: shows the installed version and the last update result
├─ lib/common.sh          shared shell helpers for app scripts
├─ locales/               host strings (en.json, vi.json, ...)
└─ data/                  host state: errors.txt, update.json, update/, rollback/ (never overwritten by updates)
.adds/nm/koboapps  generated NickelMenu config (do not edit by hand)
```

NickelMenu re-reads any config file under `.adds/nm/` whose mtime changed the next time a menu opens (verified in NickelMenu `src/config.c`, `nm_config_files_update`), so regenerating `.adds/nm/koboapps` needs no reboot.

## icon.png

Optional `icon.png` next to `app.json`: 96x96 px, black outline strokes on a transparent background. The drawer loads it with `QPixmap` and scales it smoothly (52 px in the list row, larger on a tile); SVG is not supported. Pick an icon from [Tabler Icons](https://tabler.io/icons) (MIT, outline style) or ship your own PNG; we never draw icons ourselves. Shipped icons are rasterized from the vendored Tabler SVGs in `third_party/tabler-icons/svg/` by `go run ./cmd/iconsgen -repo .` (add the SVG there, add a row to `cmd/iconsgen/icons.go`), and a test fails when a committed PNG is stale. Host item icons live in `.adds/koboapps/icons/` (`refresh.png`, `updates.png`); generic UI icons in `.adds/koboapps/icons/ui/<name>-<64|96>.png`. `apps.png` there is the "Apps" row icon in Nickel's More tab: 50x50 px with ~2 px strokes, the size of Nickel's own More icons; without it the row shows its title only (Nickel has no apps icon). The Tweaks row uses Nickel's own Settings gear (`:/images/reading/settings_gear.png`), else `ui/adjustments-horizontal-64.png`.

## app.json

```json
{
  "schema": 1,
  "id": "localsend",
  "name": "LocalSend",
  "version": "1.0.0",
  "description": "Receive books and files from your phone with LocalSend",
  "storage": { "reads": ["inbox"], "writes": ["inbox", "rejected", "books", "documents", "comics", "pictures"] },
  "network": true,
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
| `menu` | bool | optional, default `true`; `false` hides the whole app from the menu (see also `actions[].menu`) |
| `hidden` | bool | optional, default `false`; `true` keeps the app out of NickelMenu and the drawer list. It is still validated and `appsctl run` / `drawer run` still run it by id (use it for a postponed feature) |
| `storage` | object | optional: `{"reads": [...], "writes": [...]}` with logical folder names; see below |
| `network` | bool | optional, default `false`; `true` when the app talks to the network |
| `actions` | array | 1–8 items, `id` unique within the app |
| `actions[].id` | string | `^[a-z0-9][a-z0-9-]{0,30}$` |
| `actions[].label` | string | 1–32 chars, no `:` (NickelMenu field separator) |
| `actions[].run` | string | path relative to the app folder, no `..`, no leading `/`; must exist and be executable |
| `actions[].output` | enum | `show` = NickelMenu `cmd_output`, prints stdout in a dialog (keep under ~12 short lines); `none` = `cmd_spawn:quiet`, runs in background |
| `actions[].timeout_ms` | int | only with `output: show`; 500–9999 (NickelMenu limit is < 10000); default 5000 |
| `actions[].wifi` | bool | default `false`; `true` prepends `nickel_wifi:autoconnect` (does not wait; the script must wait for an IP itself, see `wait_for_wifi` in `common.sh`) |
| `actions[].menu` | bool | optional, default `true`; `false` keeps the action out of NickelMenu (it stays valid and runnable; the native drawer shows it). Keep NickelMenu short: NickelMenu cannot scroll, so a popup taller than the screen cannot be closed |
| `actions[].rescan` | enum | `none` (default), `quick` = `nickel_misc:rescan_books`, `full` = `nickel_misc:rescan_books_full`; runs only if the action succeeded |

### storage and network

`storage` and `network` declare what the app touches. Nothing is sandboxed (everything runs as root on a Kobo), so the declaration is what reviews and `appsctl validate` hold the code to. Names are **logical**, resolved by `internal/storage`; app code never hardcodes the paths.

| Name | Folder |
|---|---|
| `books` | `/mnt/onboard/Books` |
| `documents` | `/mnt/onboard/Documents` |
| `comics` | `/mnt/onboard/Comics` |
| `pictures` | `/mnt/onboard/Pictures` |
| `inbox` | `/mnt/onboard/.adds/koboapps/inbox` (staging for received files) |
| `rejected` | `/mnt/onboard/.adds/koboapps/inbox/rejected` |
| `screensaver` | `/mnt/onboard/.kobo/screensaver` (sleep screen; wallpaper app only) |

An unknown name in `reads` or `writes` makes the manifest invalid (`unknown folder "x"; use one of ...`). The app's own `data/` folder needs no declaration. Files received from the network go through the inbox router, see [inbox.md](inbox.md).

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
- Host strings (drawer, tweak and update texts, `appsctl` output) live in `/mnt/onboard/.adds/koboapps/locales/<lang>.json`. `appsctl` also embeds `en.json`, so a missing or broken `locales/` folder shows English text, never raw keys. Host scripts print them with `appsctl host-msg <key> [name=value ...]`.
- `appsctl msg`, `host-msg` and `drawer` take `--locale CODE` (override), `--conf FILE` (default `Kobo eReader.conf`) and `--koboapps-dir DIR` (host `locales/` lives there).
- `appsctl validate` checks locale files: valid JSON, keys known to English (unknown keys warn), same placeholders, translated name/label length and `:` rules.
- `appsctl msg` prints the key and exits 0 when a string is missing, so a script never fails over a translation.
- `appsctl device-name` prints the model from `/mnt/onboard/.kobo/version` (apps use it instead of a hardcoded device name). See `docs/TRANSLATING.md` to add a language.

## NickelMenu (retired)

NickelDrawer no longer generates NickelMenu items for apps: `appsctl gen-menu`, `refresh` and `errors` are gone and nothing writes `.adds/nm/koboapps`. Apps are reached only through the drawer and its API ([drawer-protocol.md](drawer-protocol.md)). `tools/install-to-kobo.sh` removes the old generated file (after backing up `.adds/nm`) and sets `experimental:menu_main_15505_enabled:0` in the user's `.adds/nm` config so NickelMenu's nav-bar tab is off. Reader-menu items such as Dark Mode are not touched; device tweaks now live in the native "Tinh chỉnh" modal ([tweaks.md](tweaks.md)).

## Script contract

- The drawer runs commands from `/` as root with a minimal environment: scripts must `cd "$(dirname "$0")/.."` (or use absolute paths) and source `/mnt/onboard/.adds/koboapps/lib/common.sh`.
- Exit code `0` = success (the manifest `rescan` applies), non-zero = failure (the drawer shows a short "did not work" message; details belong in `data/log.txt`).
- Inside the drawer (`NICKELDRAWER=1`) a script prints nothing for people: only a JSON signal `{"toast":"id"}` or `{"dialog":"id"}` declared on the screen. Any other output is logged and dropped. `common.sh` has the helpers `report`, `report_failure`, `fail_with`, `run_quick_command` and `emit_toast`, which print the signal in the drawer and the localized text on the command line.
- `output: show` scripts print short status lines through `msg` (never literal text); they must finish before `timeout_ms`. Long work goes to the background with output redirected to `data/log.txt`.
- Never hold files open under `/mnt/onboard` after the script exits; never write outside the app's own `data/` except the app's declared destination folder (a folder named in `storage.writes`, e.g. `inbox` or `books`; GitHub sync writes to `/mnt/onboard/Books/GitHub`).
- Secrets live in `data/` only, file mode 600, never printed.

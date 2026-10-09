# Drawer protocol v1 (mod ⇄ appsctl)

The native drawer (`libnickeldrawer.so`, C++/Qt inside Nickel) only renders and handles input. All app logic stays in `appsctl` (Go, tested on the Mac). The mod talks to `appsctl` by running it as a child process (`QProcess`) and reading one JSON document from stdout. Nothing else: no sockets, no shared files held open.

Why: less code inside Nickel means fewer ways to crash it (P6 fail soft); logic stays testable without a Kobo (P10); the same model can later feed other front ends.

## Rules

- Binary: `/mnt/onboard/.adds/koboapps/bin/appsctl`. Every call has a hard timeout in the mod: `model` 3 s, `info` 3 s, `screen` 5 s, `run` and `event` 35 s for `show` actions (appsctl caps scripts at 30 s) and 3 s for `background`, `remove` 10 s. On timeout or bad JSON the mod shows a localized error row and logs; it never blocks the UI thread (QProcess async signals only).
- Output is UTF-8 JSON on stdout, exit code 0. Errors: exit 2 and `{"error":{"code":"…","message":"…"}}` (message already localized).
- All user-facing text in the JSON is already localized by `appsctl` (device `CurrentLocale`). The mod never translates app strings; it only has a handful of built-in fallback strings (e.g. "Apps", "Error") for when `appsctl` itself fails.
- `"v": 1` in every response. The mod refuses a higher major version with an "update NickelDrawer" row.

## `appsctl drawer model`

Returns everything the drawer list/grid shows. Status lines come from each app's optional `status` action run with a 1 s budget per app in parallel (results cached in /tmp for 30 s); a slow or failing status just omits the line.

```json
{
  "v": 1,
  "title": "Ứng dụng",
  "hint": "Giữ một ứng dụng để xem thông tin hoặc gỡ.",
  "sections": [
    {
      "id": "installed",
      "title": "ĐÃ CÀI",
      "items": [
        {
          "id": "localsend",
          "name": "LocalSend",
          "status": "Đang nhận · PIN 4821",
          "icon": "/mnt/onboard/.adds/apps/localsend/icon.svg",
          "badge": false,
          "running": true,
          "actions": [
            {"id": "start", "label": "Bắt đầu nhận", "kind": "show", "primary": true},
            {"id": "stop", "label": "Dừng nhận", "kind": "show"}
          ]
        }
      ]
    },
    {
      "id": "system",
      "title": "HỆ THỐNG",
      "items": [
        {"id": "_refresh", "name": "Làm mới danh sách", "status": "", "badge": false, "actions": [{"id": "refresh", "label": "Làm mới", "kind": "show", "primary": true}]}
      ]
    }
  ],
  "errors": [ {"app": "broken-app", "message": "app.json: thiếu trường name"} ]
}
```

- `kind`: `show` (run, then show the printed result in a native dialog) or `background` (start, show a toast).
- `primary`: the action run when the row is tapped. Other actions appear in the long-press menu (with Info and Remove).
- Items with app-level `menu: false` still appear in the drawer (the menu flag only affects NickelMenu); `hidden: true` in the manifest (future) hides them everywhere.
- `screen: true` (omitted otherwise) when the app folder has a `ui.json`: with the screens flag on, a tap opens the app screen (`drawer screen`) instead of running the primary action.
- Ids starting with `_` are host items: `_refresh` (action `refresh`) and `_updates` (action `check`, only when `data/update.json` exists). `drawer run` accepts both. Host items have no `running` field; `icon` is omitted when the app has no `icon.svg`.
- `primary` is set on the first action in manifest order (no manifest field). The `status` action stays in `actions`.
- `running`/`status`: the app's action with id `status` is run (1 s budget). Its first line may be `{"running":true,"line":"…"}`; otherwise the first non-empty line is the text. Non-zero exit or timeout leaves both empty. The cache lives in the temp dir for 30 s and is ignored when the language changes.
- `errors[].message` is the first validation problem and is English (manifest messages are not localized).

## `appsctl drawer run <app-id> <action-id>`

Runs the declared action exactly like the NickelMenu item would (same script, same `wifi`/`rescan` semantics), but returns JSON:

```json
{"v": 1, "ok": true, "text": "LocalSend đang nhận sách\nPIN: 4821", "rescan": "quick", "wifi": false}
```

- `wifi: true` in the manifest → `appsctl` returns `{"v":1,"ok":false,"needs_wifi":true,…}` without running when there is no IP; the mod then calls Nickel's Wi-Fi connect and retries once.
- `rescan`: the mod triggers Nickel's library rescan (quick/full) after showing the result.
- A script that exits non-zero still returns exit 0 with `"ok": false` and its stdout (stderr if stdout is empty) as `text`; `rescan` is then `none`. `kind: show` is capped at 30 s (then `ok:false` with a timeout text). `kind: background` starts the script detached and returns `ok:true` at once. Scripts run from `/` with only `PATH` and `HOME=/`.
- Only actions declared in `app.json` can run (same validation as the menu). Unknown app/action → error.

## `appsctl drawer screen <app-id> [--screen <id>]`

Returns one screen of the app's `ui.json` (default: `start`), validated exactly like `appsctl validate`, with every text localized (`"@ui.key"` and `{en, vi}` maps, device language, English fallback) and every binding filled in. The mod only lays it out (`docs/spec/ui-kit.md`); it never evaluates bindings.

Variables: the app's optional action with id `state` is run with the status budget (1 s, no cache). Its stdout must be one JSON object of variable values. A declared variable (`vars` in `ui.json`) takes the value from that object, else its `vars` default; a missing state action, a slow or failing script, or bad JSON leaves the defaults. Undeclared names are ignored.

```json
{
  "v": 1, "app": "localsend", "screen": "main", "title": "LocalSend", "status": "Receiving",
  "action": {"label": "Settings", "push": "settings"},
  "body": [
    {"type": "bigcode", "text": "4821", "digits": 4},
    {"type": "kv", "rows": [["Device", "Kobo Libra 2"], ["IP", "192.168.1.23"]]},
    {"type": "list", "title": "Received", "bind": "received", "empty_text": "Nothing yet", "items": [["book.epub", ""]]}
  ],
  "footer": [{"type": "button", "text": "Stop and update library", "style": "primary", "action": "stop", "enabled": true}],
  "dialogs": {"ask": {"title": "…", "text": "…", "ok": "…", "cancel": "…", "action": "…"}},
  "toasts": {"saved": "…"},
  "live": true
}
```

- `action` (header, optional): `label` and either `push` (screen id) or `action` (action id). `dialogs`, `toasts` are omitted when the screen declares none.
- `live`: the app has a `state` action; the mod re-reads the screen every 5 s while it is visible and idle.
- Components keep the `ui.json` order; `type` is the component key. Fields per type (empty strings and `false` may be omitted):

| `type` | Fields |
|---|---|
| `text` | `text`, `style` (`body`/`caption`/`heading`, missing = body), `align` (`start`/`center`) |
| `section` | `text` |
| `kv` | `rows`: `[[key, value], …]` |
| `row` | `text`, `detail`, `action` |
| `toggle` | `text`, `bind`, `value` (bool), `enabled` (bool), `action` |
| `radio` | `text`, `bind`, `value` (selected option id), `options` `[[id, label], …]`, `enabled`, `action` |
| `field` | `text`, `bind`, `value`, `placeholder`, `keyboard` (`text`/`number`/`url`), `secret`, `enabled`, `action` |
| `bigcode` | `text` (empty: show `digits` dashes), `digits` |
| `qr` | `text` (the encoded value), `caption` |
| `pair` | `url`, `code`, `title`, `hint`, `expires`, `digits` |
| `steps` | `label`, `done`, `total`, `detail` |
| `notice` | `text`, `title` |
| `button` | `text`, `action`, `style` (`primary`/`secondary`), `enabled`, `confirm` `{title, text, ok, cancel}`, `async` |
| `list` | `bind`, `title`, `empty_text`, `tap`, `items` `[[primary, secondary], …]` (already cut to the last `max`) |
| `empty` | `text` (heading), `caption`, `button`, `action` |

- Input controls (`toggle`, `radio`, `field`) run the app action whose id equals their `bind` variable; `action` names it. Without such an action the control is read-only (`enabled: false`), because nothing could store the change.
- A `secret` field's value is sent as the same number of `•`; the secret never reaches the mod.
- List row menus (`menu`) are not sent yet (no backend to receive them in v1).
- Errors (exit 2): unknown app, host item, no `ui.json` or invalid `ui.json` (`invalid_app` with the first validation problem as detail), unknown screen (`invalid_app`).

## `appsctl drawer event <app-id> <action-id> [--value <v>] [--screen <id>]`

A tap on an app screen. Runs the declared action exactly like `drawer run` (same reply, Wi-Fi and rescan rules) with one addition: `--value` is passed to the script as the environment variable `NICKELDRAWER_VALUE` (toggle `true`/`false`, the radio option id, the field text, the list row's primary text). Without `--value` the variable is not set; `--value ''` sets it empty.

When `--screen` is given and the script's stdout is one JSON object `{"toast":"<id>"}` or `{"dialog":"<id>"}` naming a toast or dialog declared on that screen, the reply carries it resolved and `text` is empty:

```json
{"v": 1, "ok": true, "text": "", "rescan": "none", "wifi": false, "toast": "Saved"}
{"v": 1, "ok": true, "text": "", "rescan": "none", "wifi": false, "dialog": {"title": "…", "text": "…", "ok": "…", "cancel": "…", "action": "…"}}
```

The mod shows the toast, or the dialog and on OK runs its `action` as another event. An undeclared id stays plain text.

## `appsctl drawer info <app-id>`

`{"v":1,"name":"…","version":"…","description":"…","storage":{"reads":[…],"writes":[…]},"network":true,"path":"/mnt/onboard/.adds/apps/<id>"}` for the long-press Info dialog.

## `appsctl drawer remove <app-id> --confirm`

Removes an app folder (without `--confirm` it fails with `need_confirm`; a symlink, `..`, a host id or anything resolving outside the apps root is refused with `remove_refused`; an app with a broken manifest can be removed; never `data/` of other apps, never host items), regenerates the NickelMenu file, returns `{"v":1,"ok":true,"text":"…"}`. The mod always shows a confirm dialog first.

## Versioning

Additive fields are allowed within v1. Removing or changing meaning of a field → v2; the mod supports v1 until v1.x of NickelDrawer is end-of-life.

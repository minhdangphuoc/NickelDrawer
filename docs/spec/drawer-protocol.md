# Drawer protocol v1 (mod ⇄ appsctl)

The native drawer (`libnddrawer.so`, C++/Qt inside Nickel) only renders and handles input. All app logic stays in `appsctl` (Go, tested on the Mac). The mod talks to `appsctl` by running it as a child process (`QProcess`) and reading one JSON document from stdout. Nothing else: no sockets, no shared files held open.

Why: less code inside Nickel means fewer ways to crash it (P6 fail soft); logic stays testable without a Kobo (P10); the same model can later feed other front ends.

## Rules

- Binary: `/mnt/onboard/.adds/koboapps/bin/appsctl`. Every call has a hard timeout in the mod (model 3 s, action start 3 s). On timeout or bad JSON the mod shows a localized error row and logs; it never blocks the UI thread (QProcess async signals only).
- Output is UTF-8 JSON on stdout, exit code 0. Errors: exit ≠ 0 and `{"error":{"code":"…","message":"…"}}` (message already localized).
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
- Ids starting with `_` are host items.

## `appsctl drawer run <app-id> <action-id>`

Runs the declared action exactly like the NickelMenu item would (same script, same `wifi`/`rescan` semantics), but returns JSON:

```json
{"v": 1, "ok": true, "text": "LocalSend đang nhận sách\nPIN: 4821", "rescan": "quick", "wifi": false}
```

- `wifi: true` in the manifest → `appsctl` returns `{"needs_wifi": true}` without running when there is no IP; the mod then calls Nickel's Wi-Fi connect and retries once.
- `rescan`: the mod triggers Nickel's library rescan (quick/full) after showing the result.
- Only actions declared in `app.json` can run (same validation as the menu). Unknown app/action → error.

## `appsctl drawer info <app-id>`

`{"v":1,"name":"…","version":"…","description":"…","storage":{"reads":[…],"writes":[…]},"network":true,"path":"/mnt/onboard/.adds/apps/<id>"}` for the long-press Info dialog.

## `appsctl drawer remove <app-id> --confirm`

Removes an app folder (never `data/` of other apps, never host items), regenerates the NickelMenu file, returns `{"v":1,"ok":true,"text":"…"}`. The mod always shows a confirm dialog first.

## Versioning

Additive fields are allowed within v1. Removing or changing meaning of a field → v2; the mod supports v1 until v1.x of NickelDrawer is end-of-life.

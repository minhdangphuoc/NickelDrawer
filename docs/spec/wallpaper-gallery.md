# Wallpaper gallery (design and contract)

Status: approved design (10 Oct 2026). The Wallpaper app's native screens look like a phone gallery:
a grid of picture thumbnails, a picture viewer like Kobo's reader, and an options screen. This file is
the design and the exact contract between `ui.json`, `appsctl drawer`, `ndpreview`, the drawer mod and
the `wallpaper` backend. Component rules are also in [ui-json.md](ui-json.md) (normative keys and
error codes), [ui-kit.md](ui-kit.md) (looks) and [drawer-protocol.md](drawer-protocol.md) (wire format).

## 1. Flow

Declared up front in `apps/wallpaper/ui.json`, depth 1 below main:

```
main (gallery) ──tap tile / long-press "View"──▶ viewer
main ──header gear──▶ options
```

`←` pops one level. A bottom-tab tap clears the stack (drawer rule). The viewer is opened **with a
value** (the picture id); see §6.

## 2. Screens

### 2.1 main: Gallery

- Header: `← Hình nền` (title left). Right side: two **icon buttons**, Import (tray-down icon, runs
  `import`) and Options (gear icon, pushes `options`). 120 x 120 px tap targets; press feedback is
  Nickel's black circle with the icon in white.
- Under the divider: a summary line, caption style (small, muted): `12 ảnh · Đang dùng: Núi trắng · Lấp đầy`.
  Hidden when empty (no pictures).
- The sleep-screen notice (notice box) only while the Kobo settings are not ready.
- While importing: header status `Đang nhập 3/7` and a notice `Đang chuyển đổi 3/7 · bien-xanh.jpg`.
- Grid: **4 columns x 3 rows = 12 per page**. Side padding 56 px (content width 1152), gutter 24 px,
  tiles **270 x 360** (3:4, the screen's aspect), 1 px black border, the thumbnail fitted inside on
  white. No captions. The 3 rows are 1128 px high.
- Current wallpaper tile: 4 px black border and a black circular badge (56 px) with a white ✓ in its
  top-right corner (12 px inset).
- Importing: after the existing tiles, one **dashed placeholder tile** (2 px dashed, grey 140, no
  fill) per pending picture.
- A tile whose thumbnail is missing (not generated yet): the 1 px border and a small outline picture
  glyph in dim grey, centred.
- Pager `‹ 1 / 2 ›` at the bottom when there is more than one page; page-turn keys flip grid pages.
  Components above the grid (summary, notices) repeat on every grid page.
- Tap a tile: push `viewer` at that picture. Long-press: Nickel's popup menu (title row: the picture's
  label) with **Đặt làm hình nền**, **Xem**, **Thông tin** (name, size, date added; a plain dialog,
  tap outside closes), **Xoá khỏi Hình nền…** (confirm first, as in §2.2).
- Live: refreshed every 5 s while visible and idle; changed tiles repaint with a partial update only.

### 2.2 viewer: Picture viewer (like Kobo's reader)

- The picture as large as possible between the status bar and the bottom, fitted, on white.
  **Every picture change is a full e-ink refresh (GC16).**
- Chrome (shown by default):
  - header `← 3 / 12` (the mod builds the title from the position; no localization needed), status at
    the right `Đang dùng ✓` only when the shown picture is the current wallpaper;
  - bottom bar, one row 120 px high: secondary `Xoá…` (360 px) at the left, primary
    `Đặt làm hình nền` filling the rest (24 px gap). When the picture is already current the primary
    is disabled and reads `Đang dùng`.
- Tap zones: left third = previous picture, right third = next picture, centre = toggle the chrome.
  Chrome hidden: only the picture, filling the whole area below the status bar. The header and the
  bottom bar take their own taps while shown.
- Page-turn keys: next / previous picture. No wrap-around; at the ends nothing happens.
- Back returns to the gallery on the page that holds the last viewed picture.
- `Đặt làm hình nền`: event `set-current` with the picture id; toast `Đã đặt làm hình nền`; the
  screen reloads, so the status and the disabled primary update. No dialog.
- `Xoá…`: confirm dialog, title `Xoá ảnh này?`, text `Ảnh sẽ không còn trong Hình nền. Ảnh gốc trong
  thư mục Pictures vẫn được giữ.`, buttons [Huỷ] [Xoá] (destructive: only the buttons answer). On OK,
  event `remove` with the id; afterwards the next picture is shown (the previous one when it was the
  last), or the viewer pops back to the gallery when none is left.
- Memory: one full-size picture at a time (`data/images/NNNN.png`, decoded scaled to the target rect);
  the previous one is released before the next is loaded.

### 2.3 options

- Header `← Tùy chọn`.
- Section `CÁCH HIỂN THỊ`, radio (bind `mode`): `Lấp đầy màn hình` / sub-line `Cắt bớt mép ảnh`;
  `Vừa khung` / `Giữ cả ảnh, có viền trắng`.
- Section `ĐỔI ẢNH`, radio (bind `rotation`): `Cố định` / `Luôn dùng ảnh đã chọn`; `Ngẫu nhiên` /
  `Ảnh khác mỗi lần máy ngủ`.
- Section `MÀN HÌNH NGỦ`: one text line, `Sẵn sàng ✓`, or the hint with Nickel's setting names.

### 2.4 Empty state (no pictures, nothing importing)

Shown by the gallery itself: picture icon, heading `Chưa có ảnh`, text `Chép ảnh PNG hoặc JPEG vào
thư mục Pictures qua USB, hoặc gửi bằng LocalSend.`, primary button `Nhập ảnh` (action `import`).

## 3. ui.json additions (kit 1, additive)

### 3.1 Header icon actions

Screen key `actions`: 1 to 2 objects `{ "icon", "label", "push" | "action" }`; `icon` is one of
`import`, `settings`, `refresh`, `add`; `label` (text 24, no `:`) is the accessible name (shown by
ndpreview's sidebar and the log, not on the header). Exactly one of `push`/`action`. A screen has
either `action` (text link) or `actions` (icons), not both (`ui.needs_exactly_one`).

### 3.2 `gallery`

```json
{
  "gallery": "$pictures",
  "columns": 4, "rows": 3,
  "on_tap": { "push": "viewer" },
  "on_long_press": [
    { "label": "@ui.menu_set", "action": "set-current" },
    { "label": "@ui.menu_view", "push": "viewer" },
    { "label": "@ui.menu_info", "info": true },
    { "label": "@ui.menu_remove", "action": "remove", "confirm": { "title": "@ui.remove_title", "text": "@ui.remove_text", "ok": "@ui.remove_ok", "cancel": "@ui.cancel" } }
  ],
  "pending": "$pending",
  "empty": { "icon": "picture", "title": "@ui.empty_title", "text": "@ui.empty_text", "button": "@ui.import", "action": "import" }
}
```

| Key | Rule |
|---|---|
| `gallery` | one binding to an array variable (`ui.list_not_array` otherwise) |
| `columns`, `rows` | 1 to 6, defaults 4 and 3 |
| `on_tap` | object, exactly one of `push` (screen id) or `action` (action id). The item id is the value. |
| `on_long_press` | 1 to 6 items `{label (menu text 32), action | push | info: true, confirm?}`, exactly one of the three |
| `pending` | binding or literal (8): how many dashed placeholder tiles follow the items |
| `empty` | `title` (text 40, required), `text` (120), `icon` (`picture`), and together `button` (menu text 32) with `action`; the button is the screen's primary |

Item shape (data from the state action, checked by `appsctl drawer screen`, not by the validator):
`{ "id": "0003", "thumb": "/abs/…/data/thumbs/0003.png", "image": "/abs/…/data/images/0003.png",
"label": "nui-trang", "info": "nui-trang.jpg\n4000 × 3000\n08/10 20:15", "badge": "current" }`.
`id` matches `^[A-Za-z0-9_-]{1,32}$`; `badge` is `current` or absent.

At most one `gallery` and at most one `viewer` per screen. A screen with a gallery pages by grid
pages (components before the grid repeat on each page).

### 3.3 `viewer`

```json
{
  "viewer": "$pictures",
  "selected": "selected",
  "current_status": "@ui.viewer_current",
  "primary": { "label": "@ui.set_wallpaper", "label_current": "@ui.in_use", "action": "set-current" },
  "secondary": { "label": "@ui.remove", "action": "remove", "confirm": { … } }
}
```

`viewer` binds the same array as the gallery (items need `image`); `selected` names a declared
variable that receives the screen value (§6). `primary` counts as the screen's primary button. A
viewer must be the only component of its screen, with no footer (`ui.viewer_not_alone`).

### 3.4 Radio sub-lines

A radio option may have a third entry, the sub-line: `["fill", "@ui.mode_fill", "@ui.mode_fill_detail"]`
(text 60). Rows with a sub-line are 128 px.

### 3.5 Hidden empty text

A `text` component whose text resolves to empty is left out (like `notice`), so `"$summary"` hides.

### 3.6 Flow

The navigation graph (`ui.flow_*` rules) takes pushes from the header `action`, header `actions[]`,
`gallery.on_tap` and `gallery.on_long_press[]`.

## 4. Wire format (`appsctl drawer screen`)

New: `appsctl drawer screen <app> [--screen <id>] [--value <v>]`. With `--value` the state action runs
with `NICKELDRAWER_VALUE=<v>` and the variable `selected` (when declared) is set to `<v>` after the
state overlay. The reply echoes `"value": "<v>"` (omitted without `--value`).

```json
"actions": [ {"icon": "import", "label": "Nhập ảnh", "action": "import"},
             {"icon": "settings", "label": "Tùy chọn", "push": "options"} ]
```

```json
{"type": "gallery", "bind": "pictures", "columns": 4, "rows": 3,
 "tiles": [{"id": "0003", "thumb": "/mnt/…/data/thumbs/0003.png", "label": "nui-trang", "info": "…", "current": true}],
 "tap": {"push": "viewer"},
 "menu": [{"label": "Đặt làm hình nền", "action": "set-current"}, {"label": "Xem", "push": "viewer"},
          {"label": "Thông tin", "info": true},
          {"label": "Xoá khỏi Hình nền…", "action": "remove", "confirm": {"title": "…", "text": "…", "ok": "Xoá", "cancel": "Huỷ"}}],
 "pending": 0,
 "empty": {"icon": "picture", "title": "Chưa có ảnh", "text": "…", "button": "Nhập ảnh", "action": "import"}}
```

```json
{"type": "viewer", "bind": "pictures",
 "tiles": [{"id": "0003", "image": "/mnt/…/data/images/0003.png", "label": "nui-trang", "current": true}],
 "value": "0003", "current_status": "Đang dùng ✓",
 "primary": {"label": "Đặt làm hình nền", "label_current": "Đang dùng", "action": "set-current"},
 "secondary": {"label": "Xoá…", "action": "remove", "confirm": {"title": "…", "text": "…", "ok": "Xoá", "cancel": "Huỷ"}}}
```

- Radio: `options` stays `[[id, label], …]`; new parallel `option_details` `["…", "…"]` when any
  option has a sub-line.
- **Path safety:** a `thumb` or `image` is sent only when it is absolute, already clean, ends in
  `.png` and lies inside `<app dir>/data/` (no symlink resolution outside it: the cleaned path must
  have that prefix); otherwise it is sent as `""` (the tile shows the missing-thumbnail glyph). Items
  without a valid id are dropped; at most 500 tiles.
- `badge: "current"` becomes `current: true`. Viewer `value`: the screen value when it names a tile,
  else the first tile's id, else `""`.

Events (`appsctl drawer event <app> <action> --value <id> --screen <screen>`): gallery `on_tap.action`
and menu `action` items send the tile id; viewer primary sends the shown id; viewer secondary (after
its confirm) sends the shown id.

## 5. Mod behaviour (mod/drawer)

- Gallery and viewer are native: thumbnails are `QPixmap`s decoded from bytes read with
  open-read-close (`QFile` read whole, closed, then `QImageReader` on a `QBuffer` with
  `setScaledSize` to the tile or picture rect; no file stays open on `/mnt/onboard`). Files over 8 MB
  are not decoded. No QtSvg: icons, badge, dashed tiles and glyphs are painted with `QPainter`.
- Thumbnail cache: the current grid page only; changing page drops the old page's pixmaps; pop,
  close and a tab-tap clear drop all.
- Each tile is a tap target with press feedback (inverted rect) through the canvas `PressArea`
  (real tap point, `pressPosition()`); a long press opens the popup at the tile.
- The navigation stack entry of a pushed screen keeps its value (the picture id), so the live reload
  and back restore it. The viewer updates the value as the user flips pictures.
- Refresh: gallery value changes (a new thumbnail, a badge) are partial; page changes and every
  viewer picture change are full (GC16).

## 6. Wallpaper backend

- Import writes `data/thumbs/NNNN.png` (300 x 400, fitted, 16 grey levels) next to `data/images/`.
  `state` generates at most 2 missing thumbnails per call (lazy, keeps it under the 1 s budget).
  A mode change that re-fits a picture deletes its thumbnail (regenerated lazily).
- Import progress: `/tmp/wallpaper-progress.json` `{"done": 3, "total": 7, "name": "bien-xanh.jpg"}`
  while running (temp file + rename), removed at the end.
- `state` prints: `status`, `summary`, `pictures` (§3.2 items, ids `NNNN`), `current_id`, `count`,
  `mode`, `rotation`, `busy`, `pending`, `import_step`, `sleep_screen_ready`, `sleep_screen_hint`,
  `sleep_screen_line`, `last_import`.
- Index entries gain `width`, `height` (original pixels) and `added` (RFC 3339); older entries omit
  them and Info shows what is known.
- `wallpaper set-current <id>`: sets the current picture and applies the sleep screen.
- `wallpaper remove [--confirm] [--delete-original] <id>`: needs `--confirm`. Removes our converted
  copy, its thumbnail, its index entry and our `nd-wallpaper-*` file in `.kobo/screensaver`; records
  the original's hash in `data/removed.json` so a later import skips it. If it was current, the next
  picture (else the previous) becomes current. `--delete-original` (never sent by the screens) also
  deletes the original in Pictures, only when it is a regular direct child whose SHA-256 equals the
  entry's hash. The user's original is never touched otherwise.
- Actions (max 8): `import`, `mode`, `rotation`, `off`, `status`, `state`, `set-current`, `remove`.
  `next` stays a CLI command only.

## 7. Previews

`go run ./tools/wallpaper-samples --out build/preview/wallpaper-samples` generates 7 greyscale sample
pictures and runs the real import into that folder (`data/images`, `data/thumbs`). The committed
`apps/wallpaper/preview-state*.json` refer to them by paths relative to the state file; `ndpreview`
resolves relative image paths against the state file's folder and draws the missing-thumbnail glyph
when a file is absent.

```sh
go run ./cmd/ndpreview render apps/wallpaper --locale vi --state apps/wallpaper/preview-state.json --out build/preview/wallpaper-main.png
go run ./cmd/ndpreview render apps/wallpaper --locale vi --state apps/wallpaper/preview-state.json --screen viewer --value 0003 --out build/preview/wallpaper-viewer.png
go run ./cmd/ndpreview render apps/wallpaper --locale vi --state apps/wallpaper/preview-state.json --screen viewer --value 0003 --chrome hidden --out build/preview/wallpaper-viewer-full.png
go run ./cmd/ndpreview render apps/wallpaper --locale vi --state apps/wallpaper/preview-state.json --screen options --out build/preview/wallpaper-options.png
go run ./cmd/ndpreview render apps/wallpaper --locale vi --state apps/wallpaper/preview-state-empty.json --out build/preview/wallpaper-empty.png
go run ./cmd/ndpreview render apps/wallpaper --locale vi --state apps/wallpaper/preview-state-importing.json --out build/preview/wallpaper-importing.png
```

## 8. Device findings folded in (10 Oct 2026, `backups/2026-10-10-074602/debug.log`)

- **Lost radio event (use-after-free).** A radio tap logged `tap on radio 4, action "rotation"`, the
  optimistic re-render rebuilt the layout, then `tap has no action`: the session held a reference into
  the old layout's component list. Fix: the tapped component is copied before any visual change;
  re-renders are deferred to the event loop and guarded with `QPointer`.
- **Frozen screen.** `mode` re-fits every picture; the script waited up to 7 s for it and the drawer
  blocked input. Fix: inside the drawer (`NICKELDRAWER=1`) long work starts detached and the script
  returns at once; progress is step text from `state`. Controls are not disabled while busy; a mode
  change during a job is saved and the job re-fits again when it ends.
- **Dialog from script text.** Screen events show only declared JSON toasts and dialogs; prose is
  logged, and a failed event shows a short host toast.
- **Dropped second tap.** Radio and toggle events are applied optimistically and queued in order (a
  newer tap on the same control replaces its queued event), so two quick taps both apply.
- **Defaults after a failed state read.** `drawer screen` marks the reply `"stale": true` when the
  state action failed; the mod keeps the previous model of that screen.

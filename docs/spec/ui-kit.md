# UI kit v1 (NickelDrawer)

Status: approved screen designs (design canvas, 8 Oct 2026). This file is the written design contract; the exact `ui.json` keys, limits and error codes are in [ui-json.md](ui-json.md), which wins where they differ. It refines plan §7 (`docs/plan-detailed.md`); where they differ, this file wins until the plan is updated. The `ui.json` schema is `kit: 1`; the protocol is JSON-lines `proto: 1` (plan §7.7) and does not change here.

All sizes are device pixels on the Kobo Libra 2: 300 dpi, 1264 x 1680, 1 mm = 11.8 px.

## 1. Principles

1. **Look like Nickel.** Same fonts, row style and dialogs. Page headers use Nickel's own header (UI sans, §3.1); Nickel's serif only for large in-page titles; Nickel's UI sans for body. Black on white. Colours come from Nickel's stylesheet, so dark mode works.
2. **No grey fill under text.** The only grey box is the notice box.
3. **No pills, no badges, no moving bars.** Status is plain text.
4. **One primary action per screen.** One full-width button at the bottom.
5. **Few taps, big targets.** Minimum tap target 120 x 120 px, spacing 24 px.
6. **Calm e-ink.** Few, small, slow updates. One full refresh at the end of a step.
7. **Secrets never show on the Kobo.** Long or secret input goes through phone pairing.
8. **Fail soft.** A broken screen shows a clear error and the NickelMenu fallback stays.

## 2. Grid and sizes

| Item | Size |
|---|---|
| Screen | 1264 x 1680 |
| Side gutter | 72 px (content width 1120 px) |
| Vertical rhythm | 24 px |
| Drawer row | 152 px |
| Key-value row | 104 px |
| Primary button | 120 px high, full content width, at the bottom |
| Secondary button | 120 px high, 2 px outline |
| Row icon | 52 px, outline style |
| BigCode | monospace, 144 px |
| Separator | 2 px, Nickel's row colour |
| Grid cell (drawer grid view) | 280 x 240 px, 4 columns, no border |

Rules: no shadows, no gradients, no animation. Title font about 1.25 x body. Text follows Nickel's font and size settings; layouts must hold at the largest size (KIT-12).

## 3. Components

Every component below keeps the `ui.json` keys of plan §7.2 unless marked **new**. Extra components of `ui-json.md`: `gallery` and `viewer` (§3.19, §3.20), `section` (small-caps group label), `row` (128 px tappable row with detail and chevron; `action` runs an app action, `push` opens a screen, a row bound to an empty variable is hidden), `qr` (QR of one value), `empty` (§3.17); dialogs and toasts are declared per screen (`dialogs`, `toasts`). Strings may be plain or `{ "en": "...", "vi": "..." }`. Bindings use `$name` (plan §7.3).

Common to all components:
- **E-ink refresh:** a bound-value change repaints only that widget (partial). Push or pop does one full refresh. After 8 partial updates, the next change forces a full refresh. At most 2 updates per second per screen.
- **Press feedback:** every tappable widget shows its pressed state on finger down, as Nickel does: rows, tiles and buttons invert; icon buttons (back, toggles, page arrows) show a black circle with the icon in white. It is pushed to the panel at once (Kobo's "repaint now" hint); the action runs on release, then the normal look returns. A long press keeps the pressed state until the menu is open; a drag out cancels it. Back to normal is one plain update of the pressed rect (Nickel's `defaultColors()`), so no ghost stays. The page header shows the press on the arrow only (`TextHeaderBase::setInvertImageOnly`), never on the title.
- **Icons are never drawn by us.** On the device every icon is Nickel's own image from its Qt resources (back arrow, close, gear, download, reload, plus, the gallery check badge; list in `re/notes/system-icons.md`), drawn at Nickel's size with Nickel's `_hit` image as the pressed state. Where Nickel has none (list/grid toggles, picture placeholder, generic app icon, the Apps and Tweaks More-row fallbacks) or its image fails to load, the mod uses the Tabler Icons PNG (MIT) installed in `.adds/koboapps/icons/ui/<name>-<64|96>.png`; with neither, the control shows its text label or is hidden, and the log says which (`icon <name>: Nickel|Tabler|none WxH`). `ndpreview` cannot ship Kobo images, so its icons are approximations: on the device they are Nickel's.
- **Accessibility:** every tappable row or button is at least 120 px high. State is never shown by colour alone (use text, underline or a check mark). Text follows Nickel's size setting. Dark mode inverts through the stylesheet.
- **i18n:** all text is a string or `{en, vi}` map. Host strings come from `res/strings_<lang>.json`. Layout never depends on string length (see §8).

### 3.1 Header

Purpose: title, back, one status text, one optional action.

```json
"main": { "title": "LocalSend", "status": "$status", "action": { "label": "Settings", "push": "settings" } }
```

- **Same header as the drawer (§5.1):** Nickel's own page header (`TextHeader`, as on Wishlist) through the shared `mod/common/page_header`: back arrow "←" (64 px icon), title left-aligned next to it in Nickel's UI sans at Nickel's header size (34 px on the Libra 2, regular, never the serif), a 120 px row and a 1 px #666666 divider under the whole content width. Pressed, only the arrow shows Nickel's black circle with a white arrow. Without the symbols the mod draws the same header itself.
- Parts: back arrow (left), title, then at the right of the same row the status text (sans 32 px, dark grey, at most 340 px, elided) and the optional action (underlined, 120 px tap target, inverts while pressed). **Status is plain text, never a pill.** (`badge` from plan v2 is renamed `status`; `badge` stays accepted as an alias.)
- Size: 121 px (row + divider); the body starts 24 px below it. `cmd/ndpreview` draws the same boxes ("←" stands for Nickel's arrow image).
- Events: `action` (when `action.action` is set); back pops the screen (a pushed screen returns to the one under it, the start screen returns to the drawer). Back is ignored while an action is pending.
- **Icon actions** (`actions`, 1 to 2, instead of the text `action`): `{ "icon": "import", "label": "Import", "action": "import" }`. Icons (Nickel's own, see §3 "Icons"): `import` (download arrow in a circle), `settings` (gear), `refresh` (reload arrow), `add` (plus). Each is a 120 x 120 px tap target at the right end of the header row (the last one rightmost, the status text left of them); pressed, it shows Nickel's black circle with the icon in white. `label` is the accessible name (ndpreview's sidebar, the log); it is drawn only when no icon image is available. Exactly one of `push` / `action` per icon.
- States: status empty (hidden), status long (elided at the end, one line).
- Refresh: status change repaints the header text only.
- Drawer header differs: see §5.1.

### 3.2 Text

`{ "text": "...", "style": "body" | "caption" | "heading", "align": "start" | "center" }`

Wraps, never scrolls on its own. Caption uses the muted text colour (still black enough to pass contrast; no grey fill). No events. `align: "end"` is reserved for RTL (§8).

### 3.3 KeyValue row

`{ "kv": [["Device", "$device"], ["IP", "$ip"]] }`

- 104 px per row. Key left, value right, value elided in the middle for file names and at the end otherwise.
- No events. Value changes repaint the row only.
- Accessibility: a long-press on a row does nothing; if a value must be read in full, use Text.

### 3.4 BigCode

`{ "bigcode": "$pin", "digits": 4 }`

- Monospace, 144 px, centred, for PINs and one-time codes. Letters in a code use an alphabet without 0/O and 1/I/l.
- No events. Repaints only the code when it changes. (App screens are on by default; see §5.1.)
- Accessibility: also exposed as spaced digits to Nickel's accessibility text ("4 8 2 1").

### 3.5 Notice box

**New.** `{ "notice": "Keep the Kobo awake while files arrive.", "title": "Tip" }`

- Light grey fill with 2 px border, 24 px padding, body text. The only component allowed to have a grey fill. Never used for errors that block the screen (use §3.13).
- No events. Static.

### 3.6 Step progress

Replaces `progress` for v1 apps. Plan §7.2's bar is dropped.

```json
{ "steps": { "label": "Downloading", "done": "$done", "total": "$total", "detail": "$file" } }
```

- Renders two lines: `Downloading 2 / 3` and the file name (elided in the middle). No bar, nothing moving.
- Update rule: at most once every 5 s, plus always on step change and on finish. Partial refresh for updates; one full refresh when the task ends.
- States: running, done (label becomes the result, e.g. "Synced 3 books"), failed (label becomes the error).
- Backend sends `set` for `done`, `total`, `file`; the host coalesces.
- Accessibility: text only, so it reads well in any size.

### 3.7 Buttons

```json
{ "button": "Start receiving", "action": "start", "style": "primary", "enabled": "$canStart", "confirm": { "text": "...", "ok": "Remove", "cancel": "Cancel" } }
```

- **Primary:** 120 px, full width, black fill, white text, sits in `footer`. One per screen; the validator rejects two.
- **Secondary:** 120 px, 2 px outline, in the body or footer above the primary.
- Event: `{"event":"action","name":…}`. Debounced 500 ms. Disabled while an action is pending unless `async: true`.
- States: enabled, disabled, pending (label may switch to "Working…"). `enabled` is `true`, `false` or bound from state (`"$running"`, `"!$running"`); the variable must be declared in `vars`. Disabled: no fill, outline and label dimmed (grey 140), no press feedback, taps ignored; add the reason in a Text line when the cause is not obvious. Example (LocalSend): Start `"enabled": "!$running"`, Stop `"enabled": "$running"`.
- Refresh: press feedback is a partial refresh of the button; a result repaints the button label only.

### 3.8 Toggle

`{ "toggle": "Auto-update library", "bind": "autoRescan" }`

- Use only for a real two-way choice. Do not use a toggle to run an action (use a button).
- The **whole row** (104 px or more) is tappable, not just the switch. State is shown by switch position and the words On / Off at the right.
- Event: `{"event":"toggle","bind":…,"value":bool}`. Host updates the var at once; the backend may set it back.

### 3.9 Radio list

**New.** `{ "radio": "Language", "bind": "lang", "options": [["en", "English"], ["vi", "Tiếng Việt"]] }`

- One row per option, whole row tappable, filled ring on the chosen one. Up to 8 options; more goes to a List screen.
- An option may have a **sub-line** (third entry, text 60): caption style under the label; the row is 128 px instead of 104.
- Event: `{"event":"input","bind":"lang","value":"vi"}`.

### 3.10 Text field

`{ "field": "Repo", "bind": "repo", "keyboard": "text", "placeholder": "owner/name" }`

- Short values only (about 40 characters). Opens Nickel's keyboard; Go/Enter or blur commits.
- `secret: true` is **not allowed in kit 1** for tokens or long secrets: the validator warns and the app should use QR pairing (§3.11). A short secret (a 4-digit PIN to set) may use `secret: true`; it shows dots and is never logged (KIT-05).
- Event: `{"event":"input","bind":…,"value":…}`.

### 3.11 QR pairing

**New.** The Kobo shows a local address and a one-time code. The phone opens the page, fills the form and submits. The Kobo never shows the secret.

```json
{ "pair": { "url": "$pairUrl", "code": "$pairCode", "title": "Connect with phone", "hint": "Scan with your phone's camera", "expires": "$pairExpires" } }
```

- Layout: QR code (square, about 480 px, black on white, quiet zone 4 modules), the URL as text under it (mono), then a BigCode with the one-time code, then a caption with the time left.
- Backend flow: start a small local HTTP server on the Wi-Fi IP, `set` `pairUrl` and `pairCode`, wait for the form post, validate the code, store the secret in the app's `data/` (mode 600), `set` `pairUrl` to `""`, send `done`.
- States: waiting, expired (code rotates after 5 min, or "Try again"), done (QR replaced by "Connected" and the primary button), wrong code (3 tries, then new code).
- Security: HTTP on the LAN only; the server stops on success, on back, and after 5 min; the code is single-use; secrets are never in `vars`, logs, or the screen.
- Refresh: QR and code are drawn once; only the countdown caption updates, at most once per 30 s.
- Accessibility: the URL is always shown as text so it can be typed on a phone without a camera.

### 3.12 List

`{ "list": "$received", "title": "Received", "empty_text": "Nothing yet", "max": 50, "tap": "open" }`

As plan §7.2. `tap` names an app.json action; a tap runs it and also sends the `tap` event with the row to a running backend. A list keeps the last `max` items. Rows are 104 px, primary text left, secondary right. Appends redraw only the new rows. Page-turn keys flip pages (§5.3).

### 3.13 Notices and states

Error and info inside a screen use Text and the notice box. Blocking errors use §6.5. Toasts and dialogs below.

### 3.14 Toast

Nickel's own toast (`MainWindowController::toast`). Text up to about 60 characters, 3 s. Max 1 per 2 s per app. Not for anything the user must act on.

### 3.15 Confirm dialog

Nickel's `N3Dialog` when available, else `ConfirmationDialog`. Title, text, two buttons (cancel left, confirm right). Destructive confirm labels name the action ("Remove", never "OK"). Timeout 120 s counts as cancel. Page-turn keys are ignored while it is open.

### 3.16 Long-press row menu

On a drawer row or a List row with `menu` set. Items: **Info**, **Update**, **Open folder**, **Remove…** (more can be added by the app). Shown with Nickel's own popup menu (`NickelTouchMenu`, as NickelMenu builds it): bordered box, bold title row (the item name), plain text rows with thin dividers, the pressed row inverted, **Remove…** last. No Cancel row, as in Nickel's menus: a tap outside closes it. Fallback without Nickel's menu: the same look drawn by the mod (120 px rows). Remove asks for confirmation (§3.15). Items that do not apply are hidden, not greyed. A page-turn key, a page change, hiding or leaving the page closes it (never left orphaned).

```json
{ "list": "$items", "menu": [ { "id": "info", "label": "Info" }, { "id": "remove", "label": "Remove…", "confirm": true } ] }
```

Event: the mod runs `appsctl drawer event <app> <id> --list <bind> --index <n> --value <row primary text>`; `id` is the app.json action to run, and the script gets `NICKELDRAWER_LIST`, `NICKELDRAWER_INDEX` and `NICKELDRAWER_VALUE`. `confirm: true` makes the mod ask first (§3.15).

### 3.17 Empty state

Centered Text (heading + caption) and one primary button. Drawer empty state: "No apps yet" with **Find apps**. Never a blank page.

### 3.18 No-Wi-Fi dialog

Shown by the host when an action needs Wi-Fi and it is off. Text "Wi-Fi is off. Turn it on to continue." Buttons: **Cancel** / **Turn on Wi-Fi**. Turn on sends `{"event":"wifi",…}` after the connect flow. Cancel returns to the screen unchanged.

### 3.19 Gallery

**New.** `{ "gallery": "$pictures", "on_tap": { "push": "viewer" }, "on_long_press": [ … ], "pending": "$pending", "empty": { … } }`. Contract: [wallpaper-gallery.md](wallpaper-gallery.md) §2.1, §3.2.

- A grid of picture tiles, 4 columns x 3 rows by default (`columns`, `rows` 1 to 6). The grid spans the content width (same gutters as the header), gutter 24 px; with 4 columns a tile is 262 x 349 px (3:4, the screen's aspect, a leftover pixel split between the sides), 1 px black border, the thumbnail fitted inside on white, no caption. When the rows do not fit the free height, fewer rows are drawn.
- The current picture (`badge: current`): 4 px border and a black 56 px circle with a white check at the top-right corner (12 px inset).
- `pending` adds that many dashed placeholder tiles (2 px dashes, grey 140, no fill) after the items. A tile whose thumbnail is missing keeps the border and shows a small outline picture glyph in dim grey.
- More than one page: pager `‹ 1 / 2 ›` at the bottom; page-turn keys flip grid pages. Components before the gallery repeat on every grid page.
- Tap: `on_tap` (push a screen or run an action; the tile id is the value). Long press: Nickel's popup menu from `on_long_press` (title row: the tile label); `info: true` opens a plain dialog with the tile's `info` text; an item with `confirm` asks first.
- Empty (no tiles and no pending): the `empty` object draws a picture icon, heading, text and a primary button.
- Tiles repaint partially when a thumbnail or badge changes; a page change is a full refresh.

### 3.20 Viewer

**New.** `{ "viewer": "$pictures", "selected": "selected", "current_status": "In use", "primary": { … }, "secondary": { … } }`. The only component of its screen, no footer.

- The picture fitted as large as possible below the header and above the bottom bar, on white; every picture change is a full refresh.
- Chrome shown: header title `n / N` (built from the position), `current_status` at the right while the shown picture is current; bottom bar of 120 px: `secondary` (360 px) at the left, `primary` filling the rest, 24 px gap. When the shown picture is current the primary is disabled and reads `label_current`.
- Tap zones: left third previous, right third next, centre toggles the chrome. Chrome hidden (full view): the picture fills the whole area, with a back button at the top left (white disc, thin outline, `←`, 120 px target) that shows the chrome again. Page-turn keys: next / previous picture, no wrap.
- Events: `primary` and `secondary` (after its `confirm`) send the shown id as the value. The variable `selected` holds the screen value (the shown id) so bindings can read it.

## 4. i18n and layout notes

- Source language is English; Vietnamese ships with it. Missing key falls back to English.
- Vietnamese strings are often 20 to 40 % longer and have tall diacritics. Rules: row text wraps to two lines at most, then elides; buttons wrap to two lines instead of clipping (the button grows to 168 px, still one per screen); titles elide at the end; never fix widths in pixels for text; test with the longest vi string and the largest font (KIT-12, KIT-13).
- Always compose text with placeholders (`{name}`), never by joining pieces.
- Dates `dd/MM HH:mm`. Numbers use the locale separator.
- **RTL is not supported in v1.** The schema reserves `start` / `end` instead of `left` / `right` so RTL can come later (decision: RTL deferred, see `docs/DECISIONS.md`). Do not ship an RTL language file yet; the validator warns.

## 5. Drawer behaviour

### 5.1 Entry point

- One row in Nickel's **More** tab (vi "Thêm"): **Apps** (vi "Ứng dụng"). **No extra bottom tab.**
- The row has a second line (status, e.g. "LocalSend receiving · 1 update") and a small black update dot at the right when an update exists.
- Status text is the join of live app status lines and the update count; empty when nothing to say (row shows only "Apps").
- If the drawer cannot start (unknown firmware), the row is not added, the mod shows one toast and logs the reason. NickelMenu is not a fallback (DECISIONS.md).
- App screens (this kit) are on by default; the flag `.adds/nickeldrawer/no-screens` turns them off (a tap then runs the app's primary action). The old `screens` flag is ignored.

### 5.2 Views

Default is **list**. The choice is remembered (written on change, small file in `.adds/koboapps/data/`).

**Header:** Nickel's own page header (its `TextHeader`, as on Wishlist: back arrow "←", title in Nickel's UI sans at Nickel's header size, 1 px divider; pressed arrow = black circle, white arrow), then two small view-toggle icons (list, grid). Without the symbol the drawer draws the same header itself. The active one is underlined. Tap a toggle to switch; no animation.

**List view**
- Sections **INSTALLED** (vi "ĐÃ CÀI") and **SYSTEM** (vi "HỆ THỐNG"), small caps label.
- Row: 152 px. Outline icon (52 px), app name, status line under it, chevron at the right, optional black update dot.
- Hint line at the bottom: "Long-press an app for info or remove" (vi "Nhấn giữ một ứng dụng để xem thông tin hoặc gỡ").
- Page indicator "1 / N" (centred, below the hint).
- 7 rows per page.

**Grid view (optional)**, a phone-launcher grid
- 4 columns spanning exactly the content gutters (same left and right edges as the header title and divider): identical cells of 280 x 240 px. Cell width is `contentWidth / 4`; any leftover pixels are split evenly at the two sides (`DrawerLogic::gridCellRect`, unit-tested).
- Cell: no border. The app's icon (96 px, the generic apps icon without one) centred; under it the name centred in Nickel's sans at 32 px (one step under the list's 36), at most 2 lines, the last one elided. No status line (list view keeps it). Icon and name are one group centred in the cell with the icon at a fixed offset, so icons line up across a row.
- Rows start right under the header; as many rows as fit above the hint and page bar (5 on the Libra 2), so 20 apps per page. The page bar and hint stay as in the list.
- Pressed: the whole cell is inverted, as on Nickel's pressed rows. Long press and the update dot (at the icon's top-right) work as in the list.
- Host items are not apps and are never cells: **refresh** is a header icon button (Nickel's reload icon) left of the list/grid toggles; **check for updates** (only when the host offers it) is a second header button (Nickel's download icon) with the black update dot when an update is waiting. The list view still shows both as rows.
- `ndpreview drawer-grid --out build/preview/drawer-grid.png --locale vi` renders the grid with sample apps (icons approximated).

### 5.3 Paging and hard keys

- Page-turn buttons flip pages in the drawer (list 7 rows, grid 4 columns x the rows that fit) and on long app screens (one screen height minus the header).
- While a menu or dialog is open, a page key closes it instead of paging; keys are ignored while an action or the keyboard is busy.
- Every dialog has a visible OK or Cancel. A tap outside closes info, result and non-destructive dialogs (Nickel's `ConfirmationDialog::setRejectOnOutsideTap`); a destructive question (Remove, a screen button's confirm) closes only by its buttons.
- Direction follows Nickel's own page-turn direction setting (Reading settings, "swap page-turn buttons"). Read it at drawer open.
- Paging is a full-region redraw with one partial refresh; a full refresh every 6th page.
- Touch paging also works (swipe-less: tap the page indicator zones or use Nickel's scroll bar).
- Key-event delivery to a pushed view needs verification in the G2 reverse-engineering spike. If keys cannot be received, the drawer keeps touch paging only and logs it.

### 5.4 Long-press

Hold a row (about 600 ms) to open the row menu (§3.16). Short tap opens the app. The hint line teaches this.

## 6. Screen patterns

### 6.1 List app (e.g. library tools)

Header, optional Text, List with row menu, primary button at the bottom. Empty list uses §3.17.

### 6.2 Task with progress (e.g. GitHub sync)

1. Idle: header status "Last: 08/10 20:15", KeyValue rows (Repo, Folder), primary **Sync now**.
2. Running: header status "Syncing", step progress ("Downloading 2 / 3" + file name), primary button disabled or replaced by **Stop**.
3. Finished: one full refresh; step progress shows the result ("Synced 3 books"); primary **Done**. Library rescan is automatic.

### 6.3 Receive-and-wait (LocalSend)

Header status "Receiving". KeyValue rows (Device, IP). Notice box "Keep the Kobo awake while files arrive." — the device does stay awake while receiving (`sleep_inhibit`). List of received files. Primary **Stop and update library**.

Behaviour:
- The device stays awake while receiving.
- Receiving **stops itself after 15 minutes without files**; a toast says so.
- **Stop and update library always rescans**, even with zero new files.

### 6.4 Pairing

Header, QR pairing component (§3.11), caption "Open the address on your phone, enter the code, fill in the form." Secondary **Cancel**. On success the screen switches to "Connected" and a primary **Done**. First-run screens of apps that need a token start here instead of showing a token path (replaces plan GH-01).

### 6.5 Empty and error

- **Empty:** §3.17.
- **No Wi-Fi:** §3.18.
- **Blocking error:** heading ("Can't open the app"), the reason in one sentence, the fix in one sentence, secondary **Back**, primary **Try again**. An invalid `ui.json` adds the JSON path and **Open fallback menu** (plan §7.6, KIT-14).
- **Backend crash:** top Text line "App stopped (code n)" and **Try again** (plan §7.6 limits apply).

## 7. Validation

The `appsctl validate` step checks the rules of [ui-json.md](ui-json.md) and `ndpreview` shows the result (`ndpreview serve <app-dir>`, `docs/APPS.md` "Design a native screen"). In short: one primary button per screen; no `secret: true` on tokens; row icons are outline images; strings have `en`; vi strings are within 150 % of English length (warning); step progress interval; at most 8 radio options. Test IDs: KIT-01..14 (plan §9.9) plus the new cases to add in G4: KIT-15 pairing flow, KIT-16 radio list, KIT-17 step progress cadence (at most 1 update per 5 s), KIT-18 page-turn keys (swap setting, ignored in dialogs), KIT-19 notice box and no other grey fill.

## 8. What changed vs plan v2 wireframes

| Plan v2 wireframe | UI kit v1 (approved) | Why |
|---|---|---|
| Bottom tab "Ứng dụng" (DRW-01) | One **Apps** row in the More tab, with status line and update dot | No extra tab; fewer Nickel changes; fits the stock look |
| 2-column tiles as the drawer (DRW-02) | **List** by default (sections Installed / System), grid optional (4-column launcher grid) | Lists read better, show status, and match Nickel's settings pages |
| Tile height set by content | **Nickel-sized rows**: 152 px drawer rows, 104 px key-value rows, 120 px buttons | Same tap size and rhythm as stock screens |
| Header badge pill (`badge`) | **Plain status text** at the right of the header | No pills; less ink and flicker |
| Progress bar with 5 % steps | **Step text** ("Downloading 2 / 3" + file), at most every 5 s | No moving bars on e-ink |
| Token typed in fields or put in a file | **QR pairing**: phone fills the form, Kobo never shows the secret | Kobo keyboard is poor for long secrets; safer |
| Many buttons per screen | **One primary** full-width button at the bottom | Clear next step |
| Switches used for actions | Toggles only for real choices; whole row tappable; **radio lists** | Fewer mistakes |
| Manual stop and rescan | **Auto behaviours**: stay awake while receiving, stop after 15 min idle, "Stop and update library" always rescans | Fewer steps, no forgotten receiver |
| Page turn not specified | **Hard keys** flip pages; ignored in dialogs; follow Nickel's direction setting | Natural on a device with buttons |
| No long-press | **Long-press** row menu (Info, Update, Open folder, Remove…) | Per-app actions without extra screens |
| Header `badge`, `Progress` component | `status`, `steps`; new `notice`, `radio`, `pair`, row `menu` | Matches the above |

Plan §7.2 and §9.9 (DRW-01, DRW-02, GH-01, KIT list) should be updated to match when the task owner next edits the plan.

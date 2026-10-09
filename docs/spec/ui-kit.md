# UI kit v1 (NickelDrawer)

Status: approved screen designs (design canvas, 8 Oct 2026). This file is the written design contract; the exact `ui.json` keys, limits and error codes are in [ui-json.md](ui-json.md), which wins where they differ. It refines plan §7 (`docs/plan-detailed.md`); where they differ, this file wins until the plan is updated. The `ui.json` schema is `kit: 1`; the protocol is JSON-lines `proto: 1` (plan §7.7) and does not change here.

All sizes are device pixels on the Kobo Libra 2: 300 dpi, 1264 x 1680, 1 mm = 11.8 px.

## 1. Principles

1. **Look like Nickel.** Same fonts, row style and dialogs. Serif (Nickel's) for titles, Nickel's UI sans for body. Black on white. Colours come from Nickel's stylesheet, so dark mode works.
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
| Grid tile (drawer grid view) | square, 3 columns |

Rules: no shadows, no gradients, no animation. Title font about 1.25 x body. Text follows Nickel's font and size settings; layouts must hold at the largest size (KIT-12).

## 3. Components

Every component below keeps the `ui.json` keys of plan §7.2 unless marked **new**. Extra components of `ui-json.md`: `section` (small-caps group label), `row` (128 px tappable row with detail and chevron), `qr` (QR of one value), `empty` (§3.17); dialogs and toasts are declared per screen (`dialogs`, `toasts`). Strings may be plain or `{ "en": "...", "vi": "..." }`. Bindings use `$name` (plan §7.3).

Common to all components:
- **E-ink refresh:** a bound-value change repaints only that widget (partial). Push or pop does one full refresh. After 8 partial updates, the next change forces a full refresh. At most 2 updates per second per screen.
- **Accessibility:** every tappable row or button is at least 120 px high. State is never shown by colour alone (use text, underline or a check mark). Text follows Nickel's size setting. Dark mode inverts through the stylesheet.
- **i18n:** all text is a string or `{en, vi}` map. Host strings come from `res/strings_<lang>.json`. Layout never depends on string length (see §8).

### 3.1 Header

Purpose: title, back, one status text, one optional action.

```json
"main": { "title": "LocalSend", "status": "$status", "action": { "label": "Settings", "push": "settings" } }
```

- Parts: back chevron (left), serif title, status text at the right. **Status is plain text, never a pill.** (`badge` from plan v2 is renamed `status`; `badge` stays accepted as an alias.)
- Events: `action` (when `action.action` is set); back pops locally.
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
- No events. Repaints only the code when it changes.
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
- States: enabled, disabled (outline and text dimmed, plus the reason in a Text line when the cause is not obvious), pending (label may switch to "Working…").
- Refresh: press feedback is a partial refresh of the button; a result repaints the button label only.

### 3.8 Toggle

`{ "toggle": "Auto-update library", "bind": "autoRescan" }`

- Use only for a real two-way choice. Do not use a toggle to run an action (use a button).
- The **whole row** (104 px or more) is tappable, not just the switch. State is shown by switch position and the words On / Off at the right.
- Event: `{"event":"toggle","bind":…,"value":bool}`. Host updates the var at once; the backend may set it back.

### 3.9 Radio list

**New.** `{ "radio": "Language", "bind": "lang", "options": [["en", "English"], ["vi", "Tiếng Việt"]] }`

- One row per option, whole row tappable, filled ring on the chosen one. Up to 8 options; more goes to a List screen.
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

On a drawer row or a List row with `menu` set. Items: **Info**, **Update**, **Open folder**, **Remove…** (more can be added by the app). Shown in Nickel's own menu style. Remove asks for confirmation (§3.15). Items that do not apply are hidden, not greyed. Page-turn keys are ignored while it is open.

```json
{ "list": "$items", "menu": [ { "id": "info", "label": "Info" }, { "id": "remove", "label": "Remove…", "confirm": true } ] }
```

Event: `{"event":"menu","list":…,"index":n,"id":"remove"}`.

### 3.17 Empty state

Centered Text (heading + caption) and one primary button. Drawer empty state: "No apps yet" with **Find apps**. Never a blank page.

### 3.18 No-Wi-Fi dialog

Shown by the host when an action needs Wi-Fi and it is off. Text "Wi-Fi is off. Turn it on to continue." Buttons: **Cancel** / **Turn on Wi-Fi**. Turn on sends `{"event":"wifi",…}` after the connect flow. Cancel returns to the screen unchanged.

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
- If the drawer cannot start (unknown firmware, missing symbol), the row is not added and NickelMenu items remain.

### 5.2 Views

Default is **list**. The choice is remembered (written on change, small file in `.adds/koboapps/data/`).

**Header:** back chevron, serif title ("Apps" / "Ứng dụng"), then two small view-toggle icons (list, grid). The active one is underlined. Tap a toggle to switch; no animation.

**List view**
- Sections **INSTALLED** (vi "ĐÃ CÀI") and **SYSTEM** (vi "HỆ THỐNG"), small caps label.
- Row: 152 px. Outline icon (52 px), app name, status line under it, chevron at the right, optional black update dot.
- Hint line at the bottom: "Long-press an app for info or remove" (vi "Nhấn giữ một ứng dụng để xem thông tin hoặc gỡ").
- Page indicator "1 / N" (centred, below the hint).
- 7 rows per page.

**Grid view (optional)**
- 3 columns of square outlined tiles. Each tile: mark (icon or initial) at the top, status line, name. 3 x 2 per page.
- Same long-press menu and update dot (top-right corner of the tile).

### 5.3 Paging and hard keys

- Page-turn buttons flip pages in the drawer (list 7 rows, grid 3 x 2) and on long app screens (one screen height minus the header).
- Keys are ignored while a dialog, menu, toast action or keyboard is open.
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
| 2-column tiles as the drawer (DRW-02) | **List** by default (sections Installed / System), grid optional (3 columns) | Lists read better, show status, and match Nickel's settings pages |
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

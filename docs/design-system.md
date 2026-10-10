# CopperUI design system

CopperUI is the design system of NickelDrawer and of software built on it: this document (tokens, states, component specs), the token file `copperui.tokens.json` and the Qt 5.2 Widgets library in `lib/copperui/` (see `docs/DECISIONS.md`).

Status: approved on the design canvas, 10 Oct 2026. It replaces the sizes in `docs/spec/ui-kit.md` §2. The numbers live in `lib/copperui/tokens/copperui.tokens.json`; `make tokens` generates `lib/copperui/include/copperui/tokens.h` (used by `mod/drawer/screen_logic.h`), `internal/copperui/tokens.go` (used by `cmd/ndpreview/metrics.go`) and the table `docs/copperui-tokens.md`, and `make check` fails when one of them is stale. The `ui.json` contract (`docs/spec/ui-json.md`) does not change except where §6 says "schema".

Every number is a device pixel on the Kobo Libra 2 panel: 1264 × 1680, 300 ppi, 1 mm = 11.8 px, 16 grey levels. Other models with the same panel (Libra H2O, Libra Colour) get the same numbers; the drawer refuses any other panel (`mod/common/device_gate.cc`).

## 1. Principles

1. **Native first.** An app screen must be mistaken for a Nickel page. Nickel's own header (`TextHeader`), fonts, dialogs, toasts, popup menus and icons are used whenever a symbol or resource exists; the mod draws the same look only as a fallback and logs it. Nothing we draw may look "designed": no shadows, gradients, pills, badges, radii, tints or icons of our own (`docs/DECISIONS.md`, 2026-10-10).
2. **E-ink first.** Black on paper white. A change is either a partial repaint of one rectangle or one full refresh; nothing animates, nothing moves, nothing updates faster than twice a second. Grey is a tool for hierarchy, never a decoration, and never a background under text (one exception: the notice box).
3. **Pages, not scrolling.** Content is cut into whole pages; blocks never split. The pager and the hardware page keys turn pages. A screen is read like a book page: header, body, one footer action.
4. **One primary action per screen,** full width, at the bottom. Everything else is secondary or a row.
5. **Big targets, calm density.** Every target is at least 120 px tall (10 mm) and rows span the full content width. Fewer, larger things per page beat a dense page that needs zoom.
6. **State is shown by shape, not colour.** On / selected = filled or bold; pressed = inverted; disabled = dimmed outline and no reaction. Never grey alone, never colour.
7. **Text does the talking.** Status, progress and results are plain text ("Downloading 2 / 3"), never bars, spinners, counters in pills or icons without a label.
8. **Short, localisable copy.** Sentence case, verbs first, placeholders, English source and Vietnamese shipped. Layouts hold the longest Vietnamese string at Nickel's largest font setting.
9. **Fail soft.** A missing symbol, icon or font never crashes: the control falls back (Nickel image → Tabler PNG → text label), the screen shows a clear error, and the drawer disables itself on unknown firmware.
10. **Secrets stay off the Kobo.** Long input and tokens go through phone pairing; the Kobo shows a QR, an address and a short code only.

## 2. Tokens

The token file is the source of the numbers. Where the tables below and the "Resolved" table at the end differ (greys, line heights, drawer type sizes), the token file follows "Resolved"; the generated table `docs/copperui-tokens.md` lists every value with its C++ and Go constant.

### 2.1 Page metrics

| Token | px | Notes |
|---|---|---|
| `screen.width` / `screen.height` | 1264 / 1680 | The panel. |
| `inset.statusBar` | 128 | Nickel's status bar overlays the view; every page leaves this margin (Nickel's own `topMargin: 128`; `mod/common/status_bar_inset.h`). |
| `header.row` | 120 | Nickel `TextHeader`: back arrow target 120 × 120 at the left, title 34 px sans, status / actions at the right. |
| `header.divider` | 1 | Nickel's own 1 px line, grey level 6 (102, A1). |
| `gutter` | 72 | Left and right. Same edge as Nickel's header title and divider. |
| `content.width` | 1120 | 1264 − 2 × 72. Every block spans it. |
| `body.top` | 273 | 128 + 121 + 24 (one rhythm below the divider). |
| `margin.bottom` | 72 | Below the footer or the last block. |
| `footer.primary` | 120 | The primary button sits at `viewHeight − 72 − 120`. |
| `pager.height` | 56 | "1 / 3" line, caption size, 24 px above the footer; only when the body has more than one page. |
| `tabBar` | 126 px (`tabBar.height`) | Nickel's `MainNavView` stays below the view (G2 findings Q2). Measured on 4.38.23828: the view is 1264 × 1554 (`view.height`, tab bar 126), the status bar covers 126 px and we inset 128, so an app screen is laid out in a 1264 × 1426 canvas (`page.height`). The view height is what Nickel gives us; `copperui::v1::layoutBody` pages against it, never against 1680. `ndpreview` uses the same canvas inside the 1680 px picture, between a status strip and a tab strip drawn as context (gap G-14, done). |
| `dialog.width` | 1040 | Centred box, 3 px black border (Nickel `N3Dialog` when available). |
| `menu.width` | 760 | Nickel `NickelTouchMenu` look: 3 px border, 120 px rows, 1 px level-6 dividers. |
| `toast.bottom` | 260 | Nickel's own toast; position is Nickel's. |

### 2.2 Type scale

Two families, both Nickel's own and both Vietnamese-capable once kobo-tieng-viet replaced the files (`mod/common/nickel_fonts.cc`): **UI sans** (Avenir Next / Rakuten Sans) for everything interactive, **serif** (Georgia / Rakuten Serif) for large titles, and a **mono** (Courier / DejaVu Sans Mono, else the sans) for codes and addresses. Generic family names are never used.

| Token | Family | Size / line | Weight | Used by |
|---|---|---|---|---|
| `type.headerTitle` | sans | 34 / 120 row | regular | Page header title (Nickel's size on this class). |
| `type.headerStatus` | sans | 32 | regular | Header status text, header text link (underlined). |
| `type.heading` | **serif** | 52 / 72 | bold | `text` style `heading`, empty-state heading, pairing title. The only serif on an app screen, like Nickel's Settings and library titles. |
| `type.dialogTitle` | serif | 42 / 56 | bold | Dialog title. |
| `type.body` | sans | 40 / 56 | regular | `text` body, notice text, button labels (bold), steps headline. |
| `type.row` | sans | 36 / 56 | regular (bold = selected / value) | `row`, `kv`, `list`, `toggle`, `radio`, `field` value, menu rows, dialog text and dialog buttons. |
| `type.caption` | sans | 32 / 44 | regular | `text` style `caption`, row detail, radio sub-line, field label, steps detail, pager, hints, "Expires in 5 min". |
| `type.section` | sans | 28 / 40 | bold, UPPERCASE, letter-spacing 0 | `section` labels. The only uppercase text. |
| `type.gridName` | sans | 32 / 40 | regular | Drawer grid cell names (2 lines max). |
| `type.bigCode` | mono | 144 / 200 | regular | `bigcode`, pairing code. Alphabet without 0/O/1/I/l. |
| `type.mono` | mono | 32 / 44 | regular | Addresses under a QR, file paths. |

Rule: a size is never set in a string; `text` picks a style, components pick their role. Nickel's font-size setting scales reading only, not UI; the drawer's sizes are fixed per panel, which is also what Nickel does for its own chrome.

### 2.3 Spacing

| Token | px | Notes |
|---|---|---|
| `space.unit` | 8 | Everything is a multiple of 8. |
| `rhythm` | 24 | Gap after a block, padding inside boxes, gap between icon and text, gap between footer buttons. |
| `space.rowText` | 24 | Top inset of text in a 104 px row; 36 in a 128 px row. |
| `space.section` | 16 | Top inset of a section label; the 2 px rule sits at the label's bottom (64 px block, no gap after: the first row touches the rule). |
| `space.group` | 48 | Between two unrelated groups when no section label separates them (two rhythms). |
| `space.column` | 24 | Gallery gutter, sample-matrix gutter. |
| `space.keyColumn` | 440 | `kv` key column; the value takes the rest, right-aligned. |

### 2.4 Ink: the greys

The panel shows 16 levels (0, 17, 34 … 255). A value off that grid is quantised by the controller and can change between a partial and a full refresh, which shows as a slightly different grey after a page turn. So every grey we use is **on the grid**.

| Token | Level | Value | Used for | Why |
|---|---|---|---|---|
| `ink` | 0 | 0 | Text, borders, fills, the primary button, selected switch track, radio dot. | Maximum contrast; e-ink has no bright white, so black carries everything. |
| `ink.muted` | 5 | 85 | Captions, row detail, section labels, pager, hints, header status. | Darkest grey that still reads as "secondary" next to black at 300 ppi; keeps roughly 4.5:1 against paper. Never for a word the user must act on. |
| `ink.dim` | 8 | 136 | Disabled labels and outlines, placeholders, dashed placeholder tiles, the missing-picture glyph. | Reads as "not available" at a glance and is still legible. Only for things that are not currently usable. |
| `line.row` | 6 | 102 | Row and list dividers (2 px), menu dividers (1 px). | Nickel's own `#666666` separator from `NickelTouchMenu.qss`; lighter than black so rows do not become a table. |
| `line.header` | 6 | 102 | Header divider (1 px). | What Nickel's `TextHeader` draws, `#666666` (A1). |
| `paper.notice` | 14 | 238 | The notice box fill. | The only grey fill under text; light enough that the 40 px black text keeps full contrast. |
| `paper` | 15 | 255 | Background, pressed text, knob. | |

Dark mode: Nickel's own stylesheet inverts its widgets; our painted screens invert the same way (`defaultColors()` is Nickel's). The tokens above are "ink on paper" and are inverted as a whole; nothing is tuned per mode.

### 2.5 Lines, radii, fills

| Token | px | Where |
|---|---|---|
| `stroke.rule` | 2 | Section rule (black), row dividers (`line.row`), notice border, field border, QR frame, pressed-icon disc outline in the viewer. |
| `stroke.button` | 3 | Secondary button outline, dialog box, menu box, switch track. |
| `stroke.select` | 4 | Radio ring, current-tile border. |
| `stroke.tile` | 1 | Gallery tile border. |
| `radius` | 0 | Buttons, fields, notice, tiles, dialogs, menus. Nickel's chrome is square. |
| Circles | — | The only round shapes: radio ring (52), switch knob (40), pressed-icon disc (88 inside a 120 target), gallery badge (56), update dot (16). The switch track is Nickel's pill (26 px radius) because Kobo's own switches are pills. |
| Fills | — | Black (primary button, pressed areas, switch on, radio dot, badge disc) and `paper.notice`. No other fill. |

### 2.6 Icons

Never drawn by us (`docs/DECISIONS.md`). Order: Nickel's resource → Tabler Icons PNG (MIT, installed in `.adds/koboapps/icons/ui/`) → text label or hidden, logged.

| Role | Size | Source |
|---|---|---|
| Header back `←` | 56 | Nickel `:/images/reading/back_arrow.png`, pressed `_hit` (white on black disc). |
| Header close `✕` | 56 | Nickel `global_close.png`. |
| Header actions: refresh, settings, import, add | 70 / 70–80 / 70 / 56 | Nickel's `browser_reload`, `settings_gear`, `library_download_lg_off`, `plus`; drawn 1:1, centred in a 120 × 120 target. |
| List / grid toggles, apps, picture, tweaks | 64 | Tabler `layout-list`, `layout-grid`, `apps`, `photo`, `adjustments-horizontal` at 64, 1:1. |
| Drawer row icon | 64 | App icon or Tabler `apps-64`. **Not 52**: raster icons are never scaled (gap G-07). |
| Grid cell icon, empty-state icon | 96 | App icon or Tabler at 96. |
| Gallery current badge | 56 | Nickel `checkmark_hit.png` (white check on black disc). |
| Row chevron `›` | 36 text glyph | Nickel's own rows use a text chevron in the row font; allowed as text, not as a drawing. |
| Update dot | 16 | A filled circle is a shape, not an icon. |

Pressed look of an icon button: Nickel's `_hit` image, or the Tabler icon in white on an 88 px black disc (`PressArea::HIGHLIGHT_CIRCLE`).

### 2.7 Touch targets and hard keys

| Rule | Value |
|---|---|
| Minimum target | 120 × 120 px (10 mm). Icon buttons are 120 × 120 with the icon centred. |
| Rows | Full content width (1120); 104 px for `kv`, `list`, `toggle`; 128 px for `row`, radio with sub-line; 152 px drawer rows. A 104 px row is below 120 by design: Nickel's own list rows are 104 and the full-width row makes up for it. Never put two 104 px targets side by side. |
| Buttons | 120 px tall, 1120 wide (or 360 + rest in a viewer bar), 24 px apart. Grows to 168 when the label wraps. |
| Long press | 600 ms; the pressed look stays until the menu is open. |
| Debounce | 400 ms per target; an action is disabled while pending unless `async`. |
| Page keys | Forward / back = next / previous page, following Nickel's swap setting. Ignored while a dialog, menu or the keyboard is open; a key closes an open menu instead. |

### 2.8 Text fit

Widths are measured with the real font, never estimated (the mod with `QFontMetrics` of Nickel's family, `ndpreview` with the font it renders with). One algorithm, implemented in `mod/drawer/screen_logic.cc` and `cmd/ndpreview/textfit.go` and pinned by the vectors in `lib/copperui/tests/text-vectors.json`:

| Text | Rule |
|---|---|
| Paragraph (text, notice, dialog text) | Wraps to any number of lines. Paragraphs split at line breaks, words at Unicode spaces; a word wider than the line is cut into the longest pieces that fit. |
| Button label | At most `button.maxLines` (2) centred lines; the rest joins the second line and is elided at the end. The button grows from 120 to 168 px for two lines. |
| Fixed control (row, key, value, header, viewer button, grid name second line) | One line: the longest prefix that fits together with "…" (file names keep both ends). |
| Section label | Uppercased first, then elided (capitals are wider). |

### 2.9 Refresh budget

| Event | Refresh |
|---|---|
| Finger down | Pressed rect repainted at once ("repaint now" hint). |
| Release | Pressed rect back to normal, plain update. The action runs. |
| Bound value change | Partial repaint of that block only; at most 2 per second per screen; after 8 partial updates the next one is full. |
| Steps progress | At most one update every 5 s, plus on step change and on finish (full refresh on finish). |
| Page turn, push, pop | Full-region redraw with a partial refresh; a full refresh every 6th page and on every push/pop. |
| Picture change (viewer) | Full refresh. |
| Toast | Nickel's own. Max 1 per 2 s per app. |

## 3. States

Every interactive component has exactly these states. "Focus" does not exist: there is no keyboard navigation; page keys only turn pages.

| State | Shown by | Target | Notes |
|---|---|---|---|
| **Normal** | Black text, outline or divider on paper. | Active | |
| **Pressed** | The whole target rectangle inverted (white text on black) from finger down to release; icon buttons show the white glyph on a black disc; the header shows the press on the arrow only. | — | Pushed to the panel immediately. A drag out of the rect cancels. A long press keeps it until the menu opens. |
| **Selected / on** | Switch: track filled black, knob at the right, word "On" bold. Radio: 24 px black dot in the ring, label bold. Header toggle: 4 px underline. Tile: 4 px border + badge. Menu row: inverted while pressed only. | Active | Bold + shape, never grey alone. |
| **Disabled** | Outline and label `ink.dim` (136); switch and radio dimmed the same way; no press feedback; tap ignored. | None | Add the reason as a caption when it is not obvious ("Turn on the switch above"). A disabled primary button keeps its place in the footer. |
| **Pending** | Label replaced ("Working…"), target removed, look stays normal (not dimmed). | None | Only buttons and rows with `async: false`. Cleared when the action returns. |
| **Hidden** | A `text`, `notice`, `kv` row or `row` bound to an empty variable is left out and the page re-packs. | — | Prefer hiding over greying for things that do not apply. |
| **Error** | Not a component state in kit 1: the screen shows a `notice` or a blocking error screen; a field has no inline error. | | |

## 4. Components

Sizes are device px. "Use" and "Don't" are rules for app authors; the validator enforces the ones marked (V).

### 4.1 Header
Anatomy: Nickel `TextHeader`: `←` (56 icon in a 120 target, pressed = `_hit` disc) · title 34 sans, elided at the end · right side (≤ 340 px): status text 32 muted, **or** one underlined text link, **or** 1–2 icon actions (120 targets). 120 + 1 px divider. Below the 128 px status-bar inset.
Use: every screen. Status for one short fact ("Receiving", "Kit 1 · 4.38"). Icon actions for refresh / settings / import / add only (the 4 Nickel icons). Text link for one named action ("Settings").
Don't: two text links; a status longer than 40 chars; a status that changes faster than every 2 s.

### 4.2 Text
Styles: heading (serif 52/72 bold), body (40/56), caption (32/44 muted). Wraps to any number of lines, never scrolls; 24 px after. `align: center` for empty states and pairing only.
Use: headings for one in-page title; body for ≤ 3 sentences; caption for hints under a control.
Don't: a heading on a screen whose header already says the same; caption text that is required to act.

### 4.3 Section label
64 px block: label 28 bold uppercase muted at y+16, 2 px black rule at the bottom; no gap after (the first row sits on the rule). Elided at 40 chars.
Use: to group rows. Don't: a section over a single paragraph; two sections in a row with nothing between; lowercase labels (the host uppercases, diacritics kept).

### 4.4 Row (tappable)
128 px: label 36 at the left (width 760), optional detail 36 muted right-aligned (280 px, right inset 320), chevron `›` at the right edge, 2 px `line.row` divider. Pressed: inverted. Tapping runs `action` or pushes `push`. Hidden when bound to an empty variable.
Variants: one-line; with value (detail = the current value, e.g. "Balanced"); with chevron (push) – chevron only when the row leads somewhere or runs something.
Don't: a row that does nothing (use `kv`); more than 8 rows without a section.

### 4.5 Key–value
104 px: key 36 regular in a 440 px column, value 36 **bold** right-aligned, elided in the middle for file names, at the end otherwise; 2 px `line.row` divider. Not tappable. A row with empty key and value is hidden.
Use: facts (Device, IP, Version). Don't: more than 12 rows; a value the user must read in full (use `text`).

### 4.6 Big code
200 px block, 144 mono centred, letter-spaced by the mono font itself; placeholders `----` while empty. Not tappable.
Use: PIN and one-time codes. Don't: anything but 3–12 characters.

### 4.7 Notice
Box: 2 px black border, `paper.notice` fill, 24 px padding, optional bold 40 title line, 40/56 body. 24 px after. Hidden when the text resolves to empty.
Use: one tip or non-blocking warning per screen. Don't: errors that block the screen (use the error pattern); more than one per page; a notice as a heading.

### 4.8 Toast
Nickel's own (`MainWindowController::toast`), 3 s, ≤ 60 chars, bottom of the screen. The app names a toast id, the host shows the localised text.
Use: a result that needs no action ("Pictures rescanned"). Don't: anything the user must read before continuing; more than one per 2 s.

### 4.9 Dialog (confirm, info)
Nickel `N3Dialog` when available (serif 42 title, 36 text, two 120 px buttons, cancel left, confirm right), else the mod's `PlainDialog` with the same metrics: 1040 wide, 3 px border, 48 px padding, 24 px spacing. Timeout 120 s = cancel. Page keys ignored. Destructive labels name the action ("Remove"); the trigger label carries the ellipsis ("Remove…"). A tap outside closes info dialogs only.
Don't: three buttons; "OK / Cancel" for a destructive choice; text over 200 chars.

### 4.10 Buttons
Primary: 120 × 1120, black fill, white 40 bold label, in the footer, **one per screen** (V). Secondary: 3 px black outline, 40 bold black label, in the body or above the primary. Destructive: a secondary with a `confirm`; the label carries "…". Disabled: outline and label `ink.dim`, no target. Pending: label "Working…". **Labels are centred horizontally and vertically and never overflow the box**: a long label wraps to at most two lines, each line centred, and the button grows to 168; past two lines it is elided at the end. Fixed-size controls (state samples, tiles, cells) keep one centred line and elide. Vietnamese diacritics must never be clipped (line height ≥ 1.4 × size). Footer bar (viewer): secondary 360 px at the left + primary filling the rest, 24 px apart.
Don't: two primaries (V); a button to flip a setting (use a toggle); icons in buttons; ALL CAPS.

### 4.11 Toggle (switch)
104 px row: label 36 at the left, Nickel-style switch (96 × 52 pill, 3 px track, 40 px knob; on = filled track, knob right) then the word On / Off (140 px column, bold when on). The whole row is the target. Disabled: label, track and word `ink.dim`.
Use: a real two-way setting that takes effect at once. Don't: a toggle that runs an action; a toggle that needs a Save button.

### 4.12 Radio group
One 104 px row per option (128 with a sub-line): 52 px ring (4 px stroke) at the left, 24 px dot when chosen, label 36 (bold when chosen), optional 32 muted sub-line, divider. 1–8 options (V); whole row is the target.
Use: 2–8 exclusive choices. Don't: one option; options that need an explanation longer than the sub-line.

### 4.13 Field
Caption label 32 muted (8 px gap), then a 120 px box with a 2 px border and the value 36 (placeholder `ink.dim`). Tap opens Nickel's keyboard (text / number / url); Go / Enter commits. `secret` only for a short PIN (shows dots).
Use: short values (≤ 40 chars). Don't: tokens, URLs with secrets, paragraphs — use `pair`.

### 4.14 QR and pairing
`qr`: a 480 px code centred, 4-module quiet zone, mono caption under it. `pair`: serif heading (centred), the QR, the address in mono 32, a `bigcode`, a 32 muted countdown caption. Only the countdown repaints (≤ once per 30 s).
Don't: a QR narrower than 480; a URL over 200 chars; a secret in the QR.

### 4.15 Steps (progress)
Two lines: headline 40 ("Downloading 2 / 3"), detail 32 muted elided in the middle. No bar. ≤ 1 update per 5 s. Done: headline becomes the result; failed: the error.
Don't: percentages; two steps blocks on one screen; updating the detail every file when files arrive faster than 5 s.

### 4.16 List
Title 40 (optional), 104 px rows: primary 36 left, secondary 36 muted right; empty text centred in one 104 row. Tap runs `tap` with the row text; long press opens Nickel's popup menu (`menu`, 1–6 items, "Remove…" last, no Cancel row). Keeps the last `max` items; pages with the screen.
Don't: rows with more than two texts; a list as navigation (use `row`).

### 4.17 Gallery and viewer
Gallery: 4 × 3 tiles of 270 × 360 (3:4) on a 24 px gutter spanning the content width, 1 px border, thumbnail fitted on white, current = 4 px border + 56 px Nickel check badge; dashed `ink.dim` placeholders for pending; pager `‹ 1 / 2 ›`. Viewer: picture fitted below the header, bottom bar 120 (secondary 360 + primary), tap zones thirds, full view with a back disc at the top left.
Don't: captions under tiles; more than one gallery per screen (V).

### 4.18 Empty state
Centred: optional 96 Tabler icon, serif 52 heading, 32 muted caption, one primary button. Never a blank page.

### 4.19 Pager
56 px caption line "1 / 3" (muted, centred) 24 px above the footer, **with** `‹` and `›` on every multi-page screen (today only the gallery shows them: gap G-11); the left and right halves of the line are the touch targets (560 × 80 each). Page keys do the same.

### 4.20 Long-press menu
Nickel `NickelTouchMenu` look: 760 wide, 3 px border, bold 36 title row (the item name), 120 px rows 36, 1 px `line.row` dividers, pressed row inverted, "Remove…" last, no Cancel (tap outside closes). Items that do not apply are hidden.

## 5. Content style and i18n

- **Sentence case** everywhere: titles, buttons, rows, dialogs. **Uppercase only in section labels** (the host does it; authors write "Installed", never "INSTALLED").
- **Verb first** on buttons and menu items: "Start receiving", "Remove…", "Turn on Wi-Fi". No "OK" for anything that changes data; confirm buttons repeat the verb ("Remove").
- **Lengths** (chars, English): header title 40, status 40, button 32 (aim ≤ 20), row 60, kv key 24 / value 60, section 40, toast 60, dialog text 200, notice 200. Vietnamese within 150 % (V, warning).
- **Numbers**: locale separators; dates `dd/MM HH:mm`; counts as "2 / 3" with spaces; units after a space ("1 MB"). Never a bare number without its noun ("3 books", not "3").
- **Punctuation**: the ellipsis character "…" (U+2026) on triggers that ask first; the middle dot " · " to join status facts; no trailing period on labels and captions of one line; periods on body sentences.
- **Vietnamese**: NFC always; diacritics kept in uppercase section labels (the fonts have them; `nickel_fonts.cc` self-checks U+1EE8 / U+1EE5); no ALL CAPS elsewhere because stacked diacritics on capitals clip at 28 px; prefer "Bật / Tắt" for On / Off, "Xoá…" for Remove….
- **Placeholders** `{count}`, `{name}`: strings are never joined; a template in `ui.json` uses `$var`.
- **No jargon** on screen: "Wi-Fi is off", not "no network"; "Can't open the app", not "ui.json invalid" (the path goes in the second line for developers).
- **RTL** is deferred: `start` / `end`, never `left` / `right`, in `align`.

## 6. Gap map: current implementation vs this spec

Priority: **P1** visible on every screen or breaks a rule; **P2** visible on some screens; **P3** polish.

| # | P | Gap | Where | Fix |
|---|---|---|---|---|
| G-01 | P1 | Row and section dividers are 2 px **black**; every list reads as a table and competes with buttons. | `mod/drawer/screen_layout.cc` `separatorItem` (`INK_BLACK`); `cmd/ndpreview/components.go`. | Row dividers 2 px at `line.row` (102); keep black only for the section rule, notice, field and buttons. |
| G-02 | P1 | `heading` style is sans bold; the spec and Nickel's pages use the serif for in-page titles. | `screen_layout.cc` `paragraph(..., STYLE_HEADING, FAMILY_SANS, true, ...)` (text, empty, pair, gallery empty); ndpreview `components.go`. | `STYLE_HEADING` → `FAMILY_SERIF`, 52/72 bold. |
| G-03 | P1 | Greys off the 16-level grid: `INK_MUTED` 64, `INK_DIM` 140, `PAPER_GREY` 232. | `mod/drawer/screen_logic.h`, `cmd/ndpreview/metrics.go`. | 85 / 136 / 238 (levels 5 / 8 / 14); add `LINE_ROW` 102 and `LINE_HEADER` 187. |
| G-04 | P1 | Self-drawn glyphs still exist: gallery picture glyph (`SHAPE_PICTURE_GLYPH`, 4 px stroke), gallery check (`GALLERY_CHECK_STROKE` 6), empty-state picture icon (`KIND_PICTURE_ICON`, 160 px). | `screen_layout.cc` `pictureGlyph`, `GALLERY_EMPTY_ICON_SIDE`; `mod/drawer/painted_marks.cc`. | Nickel `checkmark_hit.png` for the badge; Tabler `photo-96` / `photo-off-96` for placeholder and empty state; delete the painters. |
| G-05 | P1 | Dialog metrics drift: `PlainDialog` is 1000 wide, 32 px text and 32 px buttons with 2 px borders; the kit says 1040, row size 36, 3 px. | `mod/drawer/plain_dialog.cc` `DIALOG_STYLE`, `DIALOG_WIDTH`; `screen_logic.h` `DIALOG_WIDTH`. | One token set (§2.1, §4.9); `PlainDialog` reads `ScreenLogic`. |
| G-06 | P1 | Disabled state is drawn for buttons only; `toggle` and `radio` accept `enabled` (schema) but are painted normal and stay tappable when disabled. | `screen_layout.cc` `drawToggle`, `drawRadioOption`; `internal/uikit/spec.go`. | Dim label, control and word to `ink.dim`; no target. |
| G-07 | P1 | Drawer row icon is 52 px, scaled down from the 64 px Tabler PNG (blurry at 300 ppi). | `mod/drawer/item_widgets.cc` `ROW_ICON_SIDE`; `docs/spec/ui-kit.md` §2. | 64, drawn 1:1; row stays 152. |
| G-08 | P2 | Switch is a square 96 × 52 box; Nickel's switches are pills. | `screen_layout.cc` `drawSwitch` (`SWITCH_BORDER` 3, square `rectItem`); ndpreview. | Pill track (radius 26) or Nickel's switch image if `re/notes/system-icons.md` finds one. |
| G-09 | P2 | Menu rows use 34 px text; rows elsewhere are 36. | `plain_dialog.cc` `#plainMenuTitle`, `#plainMenuRowLabel`. | 36. |
| G-10 | P2 | No way to show a state statically (pressed / selected / disabled side by side), so the UI Kit cannot teach states and `ndpreview` cannot render them. | `internal/uikit/spec.go`, `screen_layout.cc`, `cmd/ndpreview`. | **Schema:** add the `samples` component (`docs/uikit-plan.md` §5). |
| G-11 | P2 | Pager shows `‹ ›` only on gallery screens; other multi-page screens show "1 / 3" with invisible tap halves. | `screen_layout.cc` `drawPageBar` (`hasGrid`). | Arrows on every multi-page screen. |
| G-12 | P2 | Readout latency: after an event the `state` action is re-read only on the 5 s tick, so a value the user just changed can show late. | `mod/drawer/drawer_session.cc` (`LIVE_REFRESH_MILLISECONDS`). | Re-read `state` once right after an event script returns (one partial repaint), then keep the 5 s tick. Verify in the session code. |
| G-13 | P2 | Field label sits 8 px above the box in caption style; Nickel's fields put the label inside the box as placeholder or beside it. Acceptable, but the label is not dimmed when the field is disabled (no `enabled` on fields). | `screen_layout.cc` `FIELD_CAPTION_GAP`; schema. | Keep the caption label; add `enabled` to `field` for parity. |
| G-14 | P2 | `ndpreview` renders a 1680 px canvas with the header at y = 0: no 128 px status-bar inset and no tab bar, so page counts and footer position differ from the device. | `cmd/ndpreview/metrics.go`, `layout.go`. | Render the view height Nickel gives (1680 − tab bar) with the inset; draw a grey status strip and tab bar as context. |
| G-15 | P3 | `kv` values are always bold; Nickel's Settings rows show values regular. Keep bold (it is the scan column) but document it. | `screen_layout.cc` `drawValueRow`. | None; spec §4.5. |
| G-16 | P3 | The 2 px section rule touches the first row's top edge, which then draws its own divider 104 px lower: fine, but the section label top inset (16) and the row text inset (24) are not on the 8-grid with each other visually. | `screen_layout.cc` `drawSection`. | Keep; tokens §2.3 document it. |
| G-17 | P3 | Header status and header text link share the same 340 px slot; a long Vietnamese link ("Liên kết tiêu đề") underlined at 32 px reads like a title. | `mod/common/page_header.cc`. | Links ≤ 16 chars; prefer icon actions. Validator warning. |
| G-18 | P3 | `ui-kit.md` §2 still says row icon 52, separator "Nickel's row colour", grid cell 280 × 240 while `drawer_logic` computes 280 × 240 from the content width; one table should be the source. | `docs/spec/ui-kit.md`. | Replace §2 by a pointer to this file after approval. |

## 7. Token sheet for implementers

Names map to constants in `mod/drawer/screen_logic.h` (and `cmd/ndpreview/metrics.go`). New or changed constants are marked.

| Token | Constant | Value |
|---|---|---|
| gutter / content.width | `GUTTER` / `CONTENT_WIDTH` | 72 / 1120 |
| rhythm | `RHYTHM` | 24 |
| inset.statusBar | `StatusBarInset::FALLBACK_TOP_INSET` | 128 |
| header.row / divider | `HEADER_ROW_HEIGHT` / `HEADER_DIVIDER` | 120 / 1 |
| type.headerTitle / headerStatus | `HEADER_TITLE_SIZE` / `HEADER_STATUS_SIZE` | 34 / 32 |
| type.heading | `HEADING_SIZE` / `HEADING_LINE_HEIGHT` (+ `FAMILY_SERIF`, changed) | 52 / 72 |
| type.body | `BODY_SIZE` / `BODY_LINE_HEIGHT` | 40 / 56 |
| type.row | `ROW_TEXT_SIZE` | 36 |
| type.caption | `CAPTION_SIZE` / `CAPTION_LINE_HEIGHT` | 32 / 44 |
| type.section | `SECTION_TEXT_SIZE` / `SECTION_HEIGHT` | 28 / 64 |
| type.bigCode | `BIG_CODE_SIZE` / `BIG_CODE_HEIGHT` | 144 / 200 |
| type.dialogTitle (new) | `DIALOG_TITLE_SIZE` | 42 |
| row / tapRow / drawerRow | `ROW_HEIGHT` / `TAP_ROW_HEIGHT` / drawer `ROW_HEIGHT` | 104 / 128 / 152 |
| button / button.tall | `BUTTON_HEIGHT` / `BUTTON_TALL_HEIGHT` | 120 / 168 |
| field | `FIELD_HEIGHT` | 120 |
| pager | `PAGE_BAR_HEIGHT` | 56 |
| ink / muted / dim (changed) | `INK_BLACK` / `INK_MUTED` / `INK_DIM` | 0 / 85 / 136 |
| line.row (new) / line.header (new) | `LINE_ROW` / `LINE_HEADER` | 102 / 102 |
| paper.notice (changed) / paper | `PAPER_GREY` / `PAPER_WHITE` | 238 / 255 |
| stroke.rule / button / select | `SEPARATOR` / `BUTTON_BORDER` / `RADIO_RING_BORDER` | 2 / 3 / 4 |
| switch | `SWITCH_WIDTH` × `SWITCH_HEIGHT`, knob | 96 × 52, 40 (+ radius 26, changed) |
| radio ring / dot | `RADIO_RING_SIDE` / `RADIO_DOT` | 52 / 24 |
| key column | `KEY_COLUMN_WIDTH` | 440 |
| qr | `QR_SIDE` | 480 |
| dialog.width (changed in PlainDialog) | `DIALOG_WIDTH` | 1040 |
| icon sizes | `SystemIcon` sizes; `ROW_ICON_SIDE` (changed) | Nickel 56 / 70 / 80; Tabler 64 / 96; row 64 |
| refresh | `LIVE_REFRESH_MILLISECONDS` | 5000 |

## Resolved (10 Oct 2026, after the conformance audit)

Native-first wins every tie: where Nickel's real metrics differ from this document, Nickel's are used.

| # | Question | Decision |
|---|---|---|
| A1 | Header divider colour | What Nickel's `TextHeader` really draws (`#666666`, level 6 = 102). `line.header` = 102. |
| A2 | Line height ≥ 1.4 × size | Applies to every text token. Captions 32/46, headings 52/74, dialog titles 42/60, mono 32/46, grid names 32/46; uppercase section labels 28/44. |
| A3 | Switch | Nickel's `ToggleSwitch` images (94 × 56 on the Libra 2) with its regular uppercase 29 px state word; §4.11's pill is the fallback drawing only for previews. |
| A4 | Full-view back button | A white disc with a 2 px outline is allowed in the normal state: it is the only control over a picture and must stay visible on any image. |
| A5 | Vietnamese tone marks | New style everywhere: Xóa, Hủy, Tùy (the mark on the main vowel). |
| A6 | Tinh chỉnh modal | Uses the dialog tokens (1040 wide, 3 px border, 120 targets) and list-row text (sans 36); cells: one measured line, or two centred lines, then elide. No serif italic. |
| A7 | Drawer surfaces | Row status = caption (32 muted), section = section token, hint = caption, empty state = empty-state component with icon and primary, Qt6 warning = notice. |
| A8 | Main action impossible | Promote the possible action to primary (LocalSend idle: "Bắt đầu nhận" primary); never a disabled primary when another action is possible. |
| A9 | Pending vs detached jobs | Quick actions (< 2 s): the button shows "Đang …" while pending. Detached jobs: the button is disabled and progress shows as `steps`. |
| A10 | Header status length | Measured, ≤ 340 px at 32 px; the validator checks the longest locale string. |
| A11 | Menu row text | Nickel's 34 px. |
| A12 | Parity test | The generator writes both the C++ header and the Go constants from `copperui.tokens.json`; a test fails if either is stale. |
| A13 | ✓ in lists | Tabler `check` icon, not a text glyph. |

## Implemented in Batch 2 (components)

| Item | Where it lives |
|---|---|
| Dividers: row 2 px `line.row`, section rule black, drawer rows and section labels | `lib/copperui` `divider.h`; `screen_layout.cc`, `drawer_view.cc` |
| Heading in the serif (52 / 74 bold) on `text`, `empty`, `pair`, gallery empty | `screen_layout.cc`, `cmd/ndpreview/components.go` |
| Button: ≤ 2 centred lines, 120 → 168, pending label "Working…" for quick actions (A9), rows and empty buttons too | `button.h`; `screen_session.cc` (`showPendingIfActionStarted`), `screen_view.cc` (`showPending`), host key `mod.pending` |
| Disabled toggle, radio and field (dim, no target); `field.enabled` | `screen_layout.cc`, `internal/uikit/spec.go`, schema |
| List row component (primary full width, muted caption, middle-elided file names, `✓` as the check icon) | `list_row.h`, `screen_layout.cc` |
| Pager with Tabler chevrons, hidden at one page (drawer too; the drawer footer keeps its height) | `pager.h`, `drawer_view.cc`, `screen_layout.cc` |
| Keep-with-next packing: section label + first row, field caption + box, pair title + QR + address, steps headline + detail, radio caption + all options | `pack.h` (vectors `pack-vectors.json`), `cmd/ndpreview/layout.go` |
| PlainDialog and fallback menu on tokens (1040, 3 px border, 36 text, serif 42 title, 120 buttons, cancel left, A11 menu rows) | `plain_dialog.cc` |
| Icons: empty-state icon 96, drawer row icon 64 drawn 1:1, row and pager chevrons and the list check as Tabler icons | `system_icon.cc`, `painted_marks.cc`, `item_widgets.cc` |
| Big code regular, radio title as a caption | `screen_layout.cc` |
| `ndpreview` parity: real page area, Tabler PNGs, drawer list and grid, Tinh chỉnh modal, English fixtures | `cmd/ndpreview` |

Still open after Batch 2 was the drawer's own surfaces (3.1) and the Tinh chỉnh modal (3.2); Batch 3 closes them. The `✓` in the Settings key-value rows stays a text value (A13 covers lists).

## Implemented in Batch 3 (surfaces)

| Item | Where it lives |
|---|---|
| Drawer rows: name 36, status caption 32 muted, both **one measured line, elided** (a problem row too: its full message is in the Info dialog); section labels uppercased by the renderer and elided; hint one caption line | `drawer_view.cc`, `item_widgets.cc`, `DrawerLogic::rowTextWidth` |
| Drawer empty state = the CopperUI empty-state component (96 px apps icon, serif heading, caption, primary). **Decision:** the primary "Find apps" / "Tìm ứng dụng" exists only when there is something to do: the model is loaded and the host offers `_refresh` (it rescans `.adds/apps`). While loading, only the heading "Loading…" shows | `lib/copperui` `empty_state.h`, `drawer_surfaces.cc` |
| Qt6 upgrade warning is a notice (2 px border, grey fill, body text, 24 px padding); it costs list rows, so the list shows the rows that fit (`DrawerLogic::listRowsThatFit`) | `drawer_surfaces.cc`, `drawer_logic.cc` |
| Tinh chỉnh modal on the dialog tokens: 1040 wide, 3 px border, 120 px title row (serif title, ✕ in a 120 target), cells sans 36, painted from measured text: **action** cells centred (one line or two centred lines, then elided), **toggle** cells with the label at the left and Nickel's switch image with the state word under it (a 516 px cell has no room for the word beside the switch), **info** cells with the value as a caption | `tweaks_overlay.cc`, `tweaks_panel.cc`, `tweaks_logic.cc`; tokens `tweaks.*` |
| All "Legacy" tokens removed | `copperui.tokens.json` |
| `samples` component (states matrix): `look` normal / pressed (the cell inverted, `255 - level`, icons white) / selected / disabled (`ink.dim`), one block that never splits, caption line reserved under each cell | `internal/uikit`, `internal/drawer/screen_samples.go`, `screen_layout.cc`, `cmd/ndpreview/samples.go` |
| `button` with `push`; `row` with `confirm`; `show` flag on `button` and `row` (hidden while false); several primaries allowed when each has its own `show` flag (A8) | `internal/uikit`, `screen_session.cc`, `ndpreview` |
| `steps` replace notices as progress (Wallpaper importing, Settings check and update); hidden while the label is empty; no count = label alone | `cmd/wallpaper`, `internal/drawer/settings_state.go`, `ndpreview` |
| A row's `detail` that does not fit the 280 px value column goes under the label as a caption (never cut short); a run of rows has no gap between them (each draws its own divider), the last keeps its gap except before a section | `screen_layout.cc`, `components.go` |
| A section label stays with all the blocks under it when the group fits a page: a group never starts at the bottom of one page and goes on label-less on the next | `keepSectionsTogether` in `screen_layout.cc` and `components.go` |
| Content-style validator rules (warnings): typed capitals, header status > 340 px at 32 px (estimated, calibrated against the preview font), one-line caption with a full stop, section over a single paragraph, a disabled primary next to its opposite action (A8) | `internal/uikit/validate_style.go`, `textstyle.go` |
| Dates and times from the locale (`format.date`, `format.datetime`, `format.time`), default `dd/MM/yyyy HH:mm` | `internal/i18n/datetime.go` |
| UI Kit rebuilt per `docs/uikit-plan.md`: grouped home (6 sections, 19 rows, 3 pages on group boundaries; the Pager page documents §4.19), one page per component, live sample with readout | `apps/uikit` |

Deviations from `uikit-plan.md`, all forced by a rule above: the "Use for / Not for" pair is one section "Guidance" with two captions (two sections over one paragraph each would warn); the app has 12 actions, so `primary`, `confirm` and `refresh`-style variants share `echo` and the bound controls are named like their actions; the Menu and List pages do not clear their list; the Toggle page has no "disabled on" sample (a cell has one look).

# UI Kit app: redesign plan

Status: proposal for approval (design canvas boards K1–K5, 10 Oct 2026). Applies the catalogue-app research (WinUI Gallery, Compose Material Catalog, UIKitCatalog, Qt Quick Controls Gallery, Carbon) to our constraints: e-ink, no scrolling, at most 2 screens below `main`, no animation. Design tokens and component rules are in `docs/design-system.md`; the `ui.json` contract is `docs/spec/ui-json.md`.

## 1. What is wrong today

`apps/uikit` has three screens (`main`, `more`, `pictures`) that stack every component under section labels, 3 to 4 pages each. It shows that components render; it does not show what a component is for, how it looks in each state, or what happens when you use it. There is no description, no state matrix, no result readout, and the buttons page mixes a disabled sample with live buttons that all say "Tapped". The app is a smoke test, not a kit.

## 2. Shape (from the research)

- **Home = one grouped list** (UIKitCatalog, Qt gallery, WinUI index): sections by function, one 128 px row per component with a one-line detail, chevron. No search (18 items), no theme toggle (Nickel's dark mode inverts us anyway).
- **Level 1 = one page per component** (Compose Material Catalog collapsed by one level): description → static **states matrix** → one **live sample** → a **readout** of the last event. No code tab: the JSON lives in `screens/<id>.json`, which is the code sample.
- **Level 2 only for real pushes**: the gallery page pushes the viewer. Dialogs, the popup menu, toasts and the keyboard are overlays, not screens, so they cost no depth.
- **Header on every page**: `←`, the component name, status "Actions · 1 / 3" (group and position). The pager "‹ 1 / 2 ›" at the bottom turns the page; page keys do the same.
- **Interaction feedback** is the pressed inversion itself; the readout row confirms the event in text.

Home order is chosen so that **pages break on group boundaries** with the real body height (1111 px per page with a pager): page 1 = Actions, Selection, Text input (1088 px); page 2 = Feedback, Navigation (896 px); page 3 = Containment (704 px).

## 3. Home (`main`)

Header: title "UI Kit" / "Bộ giao diện"; status "Kit 1 · 18" ("18 thành phần" would exceed the slot; keep the number). No header action. No footer.

| Page | Section (en / vi) | Row label (en / vi) | Detail (en / vi) | Pushes |
|---|---|---|---|---|
| 1 | Actions / Thao tác | Button / Nút | Primary, secondary, confirm / Chính, phụ, xác nhận | `button` |
| 1 | | Row and header actions / Dòng và nút tiêu đề | Tap, push, link, icons / Chạm, mở, liên kết, biểu tượng | `row` |
| 1 | | Long-press menu / Menu nhấn giữ | Popup on a list row / Menu trên dòng danh sách | `menu` |
| 1 | Selection / Lựa chọn | Toggle / Công tắc | On, off, disabled / Bật, tắt, khoá | `toggle` |
| 1 | | Radio group / Chọn một | Up to 8 options / Tối đa 8 lựa chọn | `radio` |
| 1 | Text input / Nhập liệu | Field / Ô nhập | Nickel keyboard / Bàn phím Nickel | `field` |
| 1 | | Phone pairing / Ghép nối điện thoại | QR, address, code / QR, địa chỉ, mã | `pair` |
| 2 | Feedback / Phản hồi | Toast / Thông báo nhanh | 3 seconds, no action / 3 giây, không cần bấm | `toast` |
| 2 | | Dialog / Hộp thoại | Confirm and info / Xác nhận và thông tin | `dialog` |
| 2 | | Notice / Ghi chú | The only grey box / Hộp xám duy nhất | `notice` |
| 2 | | Steps / Tiến trình | Text progress, no bar / Tiến trình bằng chữ | `steps` |
| 2 | | Empty state / Màn trống | Heading, hint, one button / Tiêu đề, gợi ý, một nút | `empty` |
| 2 | Navigation / Điều hướng | Header and pages / Tiêu đề và trang | Back, status, pager, keys / Quay lại, trạng thái, lật trang | `navigation` |
| 3 | Containment / Bố cục | Text / Văn bản | Heading, body, caption / Tiêu đề, chữ thường, chú thích | `text` |
| 3 | | Section and key–value / Mục và khoá–giá trị | Facts in rows / Thông tin theo dòng | `kv` |
| 3 | | Big code and QR / Mã lớn và QR | Codes you read or scan / Mã để đọc hoặc quét | `code` |
| 3 | | List / Danh sách | Rows from state, tap, menu / Dòng từ trạng thái, chạm, menu | `list` |
| 3 | | Gallery and viewer / Thư viện và xem ảnh | Tiles, badge, full view / Ô ảnh, dấu chọn, toàn màn | `gallery` |

18 rows, 6 sections, 3 pages. Row labels are ≤ 24 chars in both languages so the detail column (280 px) keeps one line.

## 4. Component page template (level 1)

Every `screens/<id>.json` follows the same order; blocks that do not apply are left out.

1. **Header**: `←`, component name, status "Actions · 1 / 3" (vi "Thao tác · 1 / 3").
2. **Description**: one `text` body paragraph, ≤ 2 lines (≤ 110 chars en): what it is for and the one rule that matters.
3. **Section "States" / "Trạng thái"** + a `samples` matrix (static, non-interactive): every variant × state that exists. Captions name the state in sentence case ("Pressed", "Disabled").
4. **Section "Try it" / "Thử"** + one live instance of the component, wired to an action.
5. **Readout**: a `kv` with "Last event" / "Sự kiện cuối" and "Value" / "Giá trị", bound to `$lastEvent` and `$lastValue`; hidden until the first event (both empty → row hidden).
6. **Section "Use for" / "Dùng khi"** + one caption line; **"Not for" / "Không dùng khi"** + one caption line. Two lines, not paragraphs.
7. **Footer**: only on the Button page (the live primary is the sample). No other page has a primary.

Page count per component page: 1 or 2 (the pager shows "‹ 1 / 2 ›"). The states matrix never splits across pages because a `samples` block is one block.

Readout vars: `lastEvent` (the event name: "action", "toggle", "input", "tap", "menu") and `lastValue` (the value as text). Scripts write `data/last.json`; the `state` action returns both plus the stored control values so the live controls show their last value when the page is reopened.

## 5. What the schema and the mod must add

### 5.1 `samples` component (schema, validator, ndpreview, mod)

```json
{ "samples": [
    { "look": "normal",   "caption": "@ui.st_normal",   "button": "@ui.btn_sync", "style": "primary" },
    { "look": "pressed",  "caption": "@ui.st_pressed",  "button": "@ui.btn_sync", "style": "primary" },
    { "look": "disabled", "caption": "@ui.st_disabled", "button": "@ui.btn_sync", "style": "primary" },
    { "look": "normal",   "caption": "@ui.st_normal",   "button": "@ui.btn_stop" },
    { "look": "pressed",  "caption": "@ui.st_pressed",  "button": "@ui.btn_stop" },
    { "look": "disabled", "caption": "@ui.st_disabled", "button": "@ui.btn_stop" }
  ], "columns": 3 }
```

| Rule | Value |
|---|---|
| Items | 1 to 8. Each item is one inner component (`button`, `toggle`, `radio` with exactly one option, `row`, `field`, `kv` with one row, `notice`, `steps`, `text`) plus `look` and `caption`. |
| `look` | `normal` (default), `pressed`, `selected`, `disabled`. `pressed` inverts the whole cell rect (what `PressArea` does); `selected` draws the on state (switch on, radio dot, bold label); `disabled` draws the `ink.dim` look. `selected` is an error on components without an on state (`ui.bad_enum`). |
| `caption` | text, 24 chars, caption style (32 muted) centred under the cell, 8 px gap. |
| `columns` | 1 to 3 (default 1). Cell width = (1120 − (columns − 1) × 24) / columns. Rows, fields and kv force `columns: 1` (a 548 px row is unreadable; validator `ui.out_of_range`). |
| Behaviour | No tap targets, no events, no bindings (`ui.bad_binding` if a `$` appears). Inner `action` / `bind` / `options` values are not required inside `samples` (the validator skips the required-props check for the inner component). |
| Height | rows of cells; a cell is the inner component height + 44 + 8; 24 px between rows. One block (never splits). |
| Primary rule | A `button` sample with `style: primary` does **not** count towards "one primary per screen" (it is a picture, not a button). |

Error codes: reuse `ui.bad_enum`, `ui.out_of_range`, `ui.bad_binding`; new `ui.sample_not_static` when an inner component is one that cannot be sampled (`list`, `gallery`, `viewer`, `pair`, `qr`, `bigcode`, `empty`, `section`).

The `look: pressed` rendering is the one piece the mod must add to `screen_layout.cc`: paint the cell with `INK_BLACK` fill and swap every ink in the inner block to `PAPER_WHITE` (the same inversion `PressArea::HIGHLIGHT_INVERT` does on screen).

### 5.2 Readout right after an event (mod)

After an event script returns, re-read `state` once (partial repaint of the bound blocks) instead of waiting for the 5 s tick (`docs/design-system.md` G-12). Without it the readout lags up to 5 s and the kit looks broken.

### 5.3 `enabled` on `toggle` and `radio` must paint (mod)

Design-system gap G-06. The Toggle and Radio pages show a disabled sample through `samples`, and the live sample has an "Enabled" switch that binds the sample's `enabled` to `$sampleEnabled`.

### 5.4 Nothing else

Dialogs and toasts are declared per screen (`dialogs`, `toasts`); the popup menu comes from `list.menu`; the keyboard from `field`; the viewer from the gallery's `on_tap.push`. No new navigation feature is needed: home → component page → viewer is the only depth-2 path.

### 5.5 Phase A without schema changes

If the boards are approved before `samples` ships, every page is built with existing components: the states matrix becomes real components preset through `state` (a `toggle` bound to `$alwaysOn`, a button with `"enabled": "$never"`), and "Pressed" is a caption line "Press and hold to see the inverted state". Phase B replaces those blocks with `samples`. The home, headers, readouts and copy do not change between phases.

## 6. Actions (≤ 12 per app)

| Action id | Run by | Script does |
|---|---|---|
| `state` | host, every 5 s and after an event | Prints vars: `lastEvent`, `lastValue`, stored control values, `books`, `pictures`, `progressDone`, `progressTotal`, `progressDetail`. |
| `echo` | buttons, rows, header link/icon, list tap, menu items, gallery long-press | Stores `lastEvent`/`lastValue` from `NICKELDRAWER_*`, prints a toast `echo`. |
| `confirm` | the confirm button (after the dialog) | Same as echo, toast `confirmed`. |
| `primary` | the Button page footer | Same as echo, toast `primary`. |
| `toggle`, `radio`, `field` | the live controls (`bind`) | Store the value in `data/`, update the readout. |
| `dialog` | Dialog page button | Prints `{"dialog":"demo"}`. |
| `toast` | Toast page button | Prints `{"toast":"demo"}`. |
| `steps` | Steps page button | Starts a detached 15 s fake job writing `progress.json` (3 steps, 5 s each); `state` reads it. |
| `refresh` | Gallery page header icon | Rescans `Pictures` into `data/samples/` (existing script). |

11 actions. `rowdelete`, `rowopen`, `tilelong`, `shownote`, `refreshmode`, `searchtext`, `pagenumber` are removed; menu items reuse `echo` (`{ "id": "echo", "label": "Remove…", "confirm": true }`).

## 7. Component pages

Status strings are "Group · n / N"; copy is given as en / vi. Readout and the Use/Not-for lines are on every page and omitted below.

### 7.1 Button (`button`) — Actions · 1 / 3
- Description: "One primary per screen, full width at the bottom. Secondary for the rest; a confirm for anything destructive." / "Mỗi màn một nút chính, rộng hết cỡ ở dưới. Nút phụ cho việc còn lại; xác nhận cho việc xoá."
- States: `samples`, 3 columns, 2 rows: primary normal / pressed / disabled; secondary normal / pressed / disabled. Labels "Sync now" / "Đồng bộ" and "Stop" / "Dừng".
- Try it: secondary "Secondary" / "Nút phụ" → `echo`; secondary "Remove…" / "Xoá…" with confirm (title "Remove 3 books?" / "Xoá 3 sách?", text "They stay on the phone." / "Sách vẫn còn trên điện thoại.", ok "Remove" / "Xoá", cancel "Cancel" / "Huỷ") → `confirm`; toggle "Enable the primary" / "Bật nút chính" bound to `primaryEnabled`.
- Footer: primary "Primary" / "Nút chính", `enabled: "$primaryEnabled"` → `primary`.
- Use for: "A verb the user does now." Not for: "Settings (use a toggle), navigation (use a row)."

### 7.2 Row and header actions (`row`) — Actions · 2 / 3
- Description: "A row runs an action or opens a screen. The header carries one text link or up to two icons." / "Dòng chạy một thao tác hoặc mở màn khác. Tiêu đề có một liên kết hoặc tối đa hai biểu tượng."
- Header: `actions` refresh (→ `echo`) and settings (→ `echo`), so the page shows icon actions live.
- States: `samples` 1 column: row normal with detail and chevron; row pressed; row with value ("Refresh" / "Làm mới" · "Balanced" / "Cân bằng"); kv row (non-tappable) for contrast.
- Try it: row "Tap me" / "Chạm vào đây" detail "Echo" → `echo`; row "Open a screen" / "Mở một màn" → `push: "navigation"` (counts as depth 1 from this page: allowed, the target is a level-1 page already reachable from main; validator sees a DAG).
- Use for: "Lists of actions and settings." Not for: "Facts (use key–value)."

### 7.3 Long-press menu (`menu`) — Actions · 3 / 3
- Description: "Hold a list row 600 ms for Nickel's popup. Items that do not apply are hidden. Remove… goes last and asks first." / "Giữ một dòng 600 ms để mở menu của Nickel. Mục không áp dụng thì ẩn. Xoá… ở cuối và hỏi trước."
- States: `samples` 1 column: text block that reproduces the menu as static rows is not possible (menu is Nickel's); instead the matrix shows a list row normal and pressed. The board K2–K5 show the menu itself.
- Try it: `list` of 3 books ("Truyện Kiều", "Số đỏ", "Dế Mèn phiêu lưu ký") with `tap: echo` and `menu`: Info / Thông tin (`echo`), Open / Mở (`echo`), Remove… / Xoá… (`echo`, confirm).
- Use for: "Per-item actions." Not for: "The main action of a screen."

### 7.4 Toggle (`toggle`) — Selection · 1 / 2
- Description: "A real on/off setting that takes effect at once. The whole row is the target." / "Cài đặt bật/tắt có hiệu lực ngay. Cả dòng là vùng chạm."
- States: `samples` 1 column: off; on (`selected`); pressed; disabled off; disabled on.
- Try it: toggle "Auto-update library" / "Tự cập nhật thư viện" bound `autoRescan`; toggle "Enabled" / "Cho phép" bound `sampleEnabled` that drives the first one's `enabled`.
- Use for: "Two-way settings." Not for: "Running an action, choices with three values (use radio)."

### 7.5 Radio group (`radio`) — Selection · 2 / 2
- Description: "Two to eight exclusive options; a sub-line explains one." / "Hai đến tám lựa chọn loại trừ nhau; dòng phụ để giải thích."
- States: `samples` 1 column: option normal; option chosen (`selected`); option with sub-line; pressed; disabled.
- Try it: radio "Page refresh" / "Làm mới trang" bound `refreshMode`: Fast / Nhanh, Balanced / Cân bằng ("Full refresh every 6 pages" / "Làm mới toàn bộ mỗi 6 trang"), Clean / Sạch ("Every page" / "Mỗi trang").
- Use for: "One of few." Not for: "More than 8 (use a list screen), yes/no (toggle)."

### 7.6 Field (`field`) — Text input · 1 / 2
- Description: "Short values on Nickel's keyboard. Go commits. Long or secret values go through pairing." / "Giá trị ngắn trên bàn phím Nickel. Go để lưu. Giá trị dài hoặc bí mật dùng ghép nối."
- States: `samples` 1 column: empty with placeholder ("owner/name"); filled ("Truyện Kiều"); disabled.
- Try it: field "Search" / "Tìm kiếm" placeholder "Type and confirm" / "Gõ rồi xác nhận" keyboard text; field "Go to page" / "Đến trang" keyboard number. The keyboard is the level-2 overlay (board K5 variant).
- Use for: "Names, numbers, short URLs." Not for: "Tokens, passwords, paragraphs."

### 7.7 Phone pairing (`pair`) — Text input · 2 / 2
- Description: "The phone fills the form; the Kobo shows only the address and a one-time code." / "Điện thoại điền biểu mẫu; Kobo chỉ hiện địa chỉ và mã một lần."
- States: none (static block). Try it: `pair` with `url` "http://192.0.2.10:8080", `code` "4821", title "Connect with phone" / "Kết nối với điện thoại", hint "Scan with your phone's camera" / "Quét bằng camera điện thoại", expires "Expires in 5 min" / "Hết hạn sau 5 phút". Fills the page (2 pages with the readout).

### 7.8 Toast (`toast`) — Feedback · 1 / 5
- Description: "Nickel's toast, 3 seconds, at most one per 2 seconds. For results that need no action." / "Thông báo của Nickel, 3 giây, tối đa một mỗi 2 giây. Cho kết quả không cần bấm."
- States: none. Try it: secondary "Show a toast" / "Hiện thông báo" → `toast` (text "Pictures rescanned" / "Đã quét lại ảnh").
- Use for: "Done, saved, rescanned." Not for: "Errors the user must read, questions."

### 7.9 Dialog (`dialog`) — Feedback · 2 / 5
- Description: "Nickel's dialog: serif title, text, Cancel left, the verb right. 120 s without an answer means Cancel." / "Hộp thoại của Nickel: tiêu đề serif, nội dung, Huỷ bên trái, động từ bên phải. 120 giây không trả lời là Huỷ."
- States: none static (board K5 shows the dialog). Try it: secondary "Ask a question" / "Hỏi một câu" → `dialog` (title "Remove 3 books?" / "Xoá 3 sách?", text "They stay on the phone." / "Sách vẫn còn trên điện thoại.", ok "Remove" / "Xoá", cancel "Cancel" / "Huỷ", action `echo`); secondary "Show info" / "Hiện thông tin" → `dialog` with an info dialog (ok only, outside tap closes).
- Use for: "Destructive or irreversible choices." Not for: "Confirming every tap."

### 7.10 Notice (`notice`) — Feedback · 3 / 5
- Description: "The only grey box. One tip per screen; hidden when its text is empty." / "Hộp xám duy nhất. Một mẹo mỗi màn; ẩn khi chữ trống."
- States: `samples` 1 column: with title; without title. Try it: toggle "Show the notice" / "Hiện ghi chú" bound `showNote` → a `notice` bound to `$noteText` appears under it.
- Use for: "Tips, non-blocking warnings." Not for: "Errors that block (error screen), headings."

### 7.11 Steps (`steps`) — Feedback · 4 / 5
- Description: "Progress as text, at most one update every 5 seconds. No bar, nothing moves." / "Tiến trình bằng chữ, cập nhật tối đa 5 giây một lần. Không có thanh, không gì chuyển động."
- States: `samples` 1 column: running ("Downloading 2 / 3" + "pride-and-prejudice.epub"); done ("Synced 3 books"); failed ("Download failed: no Wi-Fi").
- Try it: secondary "Run 3 steps" / "Chạy 3 bước" → `steps`; a live `steps` bound to `progressDone` / `progressTotal` / `progressDetail`.

### 7.12 Empty state (`empty`) — Feedback · 5 / 5
- Description: "Never a blank page: heading, one hint, one button." / "Không bao giờ để trang trắng: tiêu đề, một gợi ý, một nút."
- Try it: the page itself is an `empty` ("No books yet" / "Chưa có sách", "Add books on the phone, then sync." / "Thêm sách trên điện thoại rồi đồng bộ.", button "Sync now" / "Đồng bộ" → `echo`). The `empty` button is the page's primary. The readout is on page 2.

### 7.13 Header and pages (`navigation`) — Navigation · 1 / 1
- Description: "Back returns one level; a tab tap clears the stack. Pages turn with the pager or the page keys; dialogs ignore keys." / "Quay lại lùi một cấp; chạm tab xoá cả ngăn xếp. Lật trang bằng thanh trang hoặc phím; hộp thoại bỏ qua phím."
- Header: status "Navigation · 1 / 1"; text link "Link" / "Liên kết" → `echo`.
- Try it: three `text` body paragraphs and a `kv` of 12 rows so the page runs to 3 pages; the readout on page 1 shows "page" events are not events (nothing is sent on a page turn, by design: caption says so).

### 7.14 Text (`text`) — Containment · 1 / 5
- Description: "Heading in Nickel's serif, body and caption in the sans. Text wraps; it never scrolls." / "Tiêu đề bằng serif của Nickel, chữ thường và chú thích bằng sans. Chữ xuống dòng, không cuộn."
- States: `samples` 1 column: heading "Reading list" / "Danh sách đọc"; body (2 lines); caption; centred caption.
- Try it: none (static). Readout omitted.

### 7.15 Section and key–value (`kv`) — Containment · 2 / 5
- Description: "Sections group rows; key–value rows show facts, value bold at the right." / "Mục gom các dòng; dòng khoá–giá trị hiện thông tin, giá trị in đậm bên phải."
- Try it: section "Device" / "Thiết bị" + kv Firmware "$firmware", Language "$locale", Kit "1", File "pride-and-prejudice-annotated-edition.epub" (middle elision).

### 7.16 Big code and QR (`code`) — Containment · 3 / 5
- Description: "Codes you read aloud or scan. Big code 144 px mono; QR 480 px with a text address under it." / "Mã để đọc hoặc quét. Mã lớn 144 px đơn cách; QR 480 px với địa chỉ chữ bên dưới."
- Try it: `bigcode` "4821"; `qr` "https://example.org/nickeldrawer" caption "Scan with your phone" / "Quét bằng điện thoại". 2 pages.

### 7.17 List (`list`) — Containment · 4 / 5
- Description: "Rows from the app's state: primary left, secondary right. Tap runs an action; hold opens the menu. Keeps the last N." / "Dòng từ trạng thái của ứng dụng: chính bên trái, phụ bên phải. Chạm chạy thao tác; giữ mở menu. Giữ N dòng cuối."
- Try it: `list` "$books" (4 books, author right) title "Received" / "Đã nhận", empty text "Nothing yet" / "Chưa có gì", tap `echo`, menu Open / Remove…; secondary "Clear the list" / "Xoá danh sách" → `echo` (the script empties `books` so the empty text shows; the next `state` restores it after 30 s).

### 7.18 Gallery and viewer (`gallery`) — Containment · 5 / 5
- Description: "Tiles 3:4 with a 1 px border; the current one gets a 4 px border and Nickel's check. Tap opens the viewer; hold opens the menu." / "Ô ảnh 3:4 viền 1 px; ảnh đang dùng viền 4 px và dấu kiểm của Nickel. Chạm mở trình xem; giữ mở menu."
- Header: icon action refresh → `refresh`.
- Try it: `gallery` "$pictures" 4 × 2, `on_tap: { push: "viewer" }`, long press Info / Echo; `empty` "No PNG pictures" / "Chưa có ảnh PNG" with the existing hint. The **viewer** screen (`viewer`) is the only level-2 screen: `selected` "selectedPicture", `current_status` "In use" / "Đang dùng", primary "Use this" / "Dùng ảnh này" (`label_current` "In use" / "Đang dùng") → `echo`, secondary "Remove…" / "Xoá…" confirm → `echo`.

## 8. Navigation flow (validated by `appsctl validate`)

```
main ─push─▶ button, row, menu, toggle, radio, field, pair, toast, dialog, notice,
             steps, empty, navigation, text, kv, code, list, gallery      (depth 1)
row ─push─▶ navigation                                                     (depth 1, DAG)
gallery ─push─▶ viewer                                                     (depth 2)
```

No cycles, every screen reachable, nothing deeper than 2. Dialogs, toasts, the popup menu and the keyboard are overlays. Back from any component page returns to `main` on the page it was on; a tab tap clears everything.

## 9. Files

```
apps/uikit/
  app.json            11 actions (§6), hidden: true, storage reads pictures
  ui.json             kit, start, vars, screens: { main }  (the home)
  screens/<id>.json   18 component pages + viewer
  locales/en.json, vi.json   ui.* keys (≈ 180), msg.* toasts
  bin/state.sh, echo.sh, confirm.sh, primary.sh, toggle.sh, radio.sh, field.sh,
      dialog.sh, toast.sh, steps.sh, refresh.sh, lib.sh
  preview-state.json  firmware, locale, books, pictures, progress
```

Previews: `go run ./cmd/ndpreview render apps/uikit --locale vi --screen <id> --out build/preview/uikit-<id>.png` for every screen; the Makefile target `preview-uikit` renders all 20 in both languages (40 PNGs) so a reviewer can diff them.

## 10. Acceptance (device cases, checklist 5b)

| ID | Case |
|---|---|
| UIK-01 | Home shows 3 pages with the groups of §3; page keys and the pager turn pages; back returns to the same page. |
| UIK-02 | Every component page opens in under 1 s, shows its states matrix on page 1, and the pressed sample is an exact inversion of the normal one. |
| UIK-03 | Button page: primary disabled until the switch is on; the readout shows "action · primary" within 1 s of the tap. |
| UIK-04 | Dialog page: the confirm dialog is Nickel's (`N3Dialog` logged), Cancel left, Remove right; page keys do nothing while it is open; after 120 s it closes as Cancel. |
| UIK-05 | Field page: the keyboard is Nickel's with the Vietnamese layout; Go stores the value; the readout shows it. |
| UIK-06 | Toggle page: the "Enabled" switch off dims the first toggle and it ignores taps. |
| UIK-07 | Steps page: three updates 5 s apart, then "Done" with a full refresh; nothing repaints in between. |
| UIK-08 | Gallery page → viewer → full view → back three times lands on home; depth never exceeds 2 (log). |
| UIK-09 | Vietnamese: no clipped diacritics in section labels or 168 px buttons; no string over 150 % (validator). |
| UIK-10 | `appsctl validate apps/uikit` passes with zero warnings. |

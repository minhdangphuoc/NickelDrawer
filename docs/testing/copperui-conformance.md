# CopperUI conformance audit (10 Oct 2026)

Every UI surface inside the app drawer checked against `docs/design-system.md`: 64 surfaces × 12 criteria (type, spacing, greys, dividers, targets, states, icon source, content style, i18n, text fit, refresh, navigation). Evidence: code references, 79 ndpreview renders and 425 label strings measured with real Avenir Next / Georgia metrics.

| Verdict | Cells |
|---|---|
| ✅ conforms | 401 |
| ⚠️ partial, risk or device-only | 104 |
| ❌ fails | 102 |

| Group | ✅ | ⚠️ | ❌ |
|---|---|---|---|
| Drawer | 62 | 36 | 22 |
| Nickel integration | 32 | 14 | 6 |
| Host screens | 29 | 8 | 11 |
| App screens | 158 | 31 | 42 |
| Shared components | 120 | 15 | 21 |

F01 and F02 cause about 60 of the ❌ cells.

## Findings

### Tokens and global
- **F01 ❌ Greys off the 16-level grid.** `screen_logic.h:62-64`, `metrics.go:79-81` (64 / 140 / 232); header status `#404040` (`screen_view.cc:29`); drawer labels all black (`drawer_view.cc:36`). Fix: 85 / 136 / 238, add `LINE_ROW` 102 and `LINE_HEADER` 187.
- **F02 ❌ Row dividers are 2 px black.** `screen_layout.cc:105-108`, `drawer_view.cc:39,47`, ndpreview `components.go`. Only the section rule stays black.
- **F03 ❌ Header divider colour** (`page_header.cc:30`, `metrics.go:22`), see A1.
- **F04 ❌ Heading drawn in the sans, not the serif.** `screen_layout.cc:285,335,353,368`, `components.go:374,535`.
- **F05 ❌ No token source yet**; px/colour literals spread over `screen_layout.cc`, `item_widgets.cc`, `drawer_view.cc`, `tweaks_panel.cc`, `tweaks_overlay.cc`, `plain_dialog.cc`, `page_header.cc`, `more_row.cc`, `painted_marks.cc`; the promised parity test between `screen_logic.h` and `metrics.go` does not exist.
- **F06 ❌ Text width is estimated (0.56 × size per char), not measured** (`screen_logic.cc:8-30`, `metrics.go`). Regular text really runs ~0.49, bold 40 up to 0.60, bold uppercase 28 up to 0.675: paragraphs wrap early, bold labels get elided on the device, button height (120 vs 168) is decided by the guess.
- **F07 ⚠️ Line heights under 1.4 × size** (caption 44/32, heading 72/52, grid name 42/32): stacked Vietnamese diacritics may clip 1–2 px; device check.
- **F08 ❌ Pager**: arrows only on gallery screens; the drawer pager shows at "1 / 1"; drawer hint 28 black instead of 32 muted.
- **F09 ❌ Groups split across pages**: section label alone at a page bottom (settings p1), field caption split from its box, pair title split from its QR (UI Kit).
- **F10 ❌ No disabled look for toggle, radio and field.** `screen_layout.cc:252-280,644-710`.
- **F11 ❌ No pending "Working…" state**; input is only blocked (`screen_session.cc:226-236`).
- **F12 ❌ List secondary text is bold black** (`screen_layout.cc:599`).
- **F13 ❌ List primary cut at 440 px and end-elided** (list rows reuse kv geometry, `screen_layout.cc:316,593`).

### Drawer, Nickel integration and dialogs
- **F14 ⚠️ Shapes and glyphs**: full-view back disc drawn in the normal state, empty-state icon box 160 (token 96), drawer ‹ › and chevron are text glyphs, To-do ✓ is a glyph, ndpreview draws its own icons.
- **F15 ❌ Drawer row icon 52 px scaled** (`item_widgets.cc:16`).
- **F16 ❌ Drawer type sizes off the scale** (`drawer_view.cc:35-50`: status 28, section 26, hint 28, empty 40/30, Qt6 row 28 bold with a black rule).
- **F17 ❌ Drawer empty state has no icon and no primary** (`drawer_view.cc:295-306`).
- **F18 ❌ PlainDialog/menu metrics** (`plain_dialog.cc:15-36`: 1000 wide, 32 px text, 2 px button borders, 34 px menu rows).
- **F19 ❌ Tinh chỉnh modal**: 1140 wide, 1 px border, 30 px title, ✕ target 110 (`tweaks_overlay.cc:15-22`); cells serif italic 34, left-aligned, word-wrapped in a fixed 128 px cell with no elide (`tweaks_panel.cc:17,30,65`); toggles show a word only.
- **F20 ❌ 49 user-facing strings hardcoded in C++ (en/vi only)**, some with jargon ("appsctl failed (code {n})") (`fallback_strings.cc:16-62`).
- **F21 ⚠️ Mixed Vietnamese tone-mark style** ("Tùy/Xóa/Hủy" next to "Xoá…"), see A5.
- **F22 ⚠️ ALL CAPS typed by authors** in locale strings.
- **F23 ⚠️ Header status overflows its 340 px slot**: "Đang ở màn hình chờ: {count}" 438 px, "On the sleep screen: {count}" 413, "Luân phiên: Ngẫu nhiên" 352; "5/7" should be "5 / 7".
- **F24 ❌ ndpreview differs from the device**: no status-bar inset, no ⚙ in drawer-grid, Vietnamese-only fixtures, no list-view or Tinh chỉnh renderer.
- **F25 ⚠️ Content style**: captions ending with ".", ISO dates, commas instead of " · ", To-do recovered shows a file path, "● 1.0.0" text bullet, a section over a single paragraph.
- **F26 ⚠️ One-primary rule**: LocalSend idle (Start secondary, Stop disabled primary), GitHub unpaired (primary disabled), Library (7 secondaries acting as a list).
- **F28/F29 ⚠️ Switch spec** (§4.11 pill) vs Nickel's 94×56 image and regular uppercase "ON", see A3.
- **F30 ⚠️ Radio title drawn as a section label.**
- **F31 ❌ Big code drawn bold** (`screen_layout.cc:827`).
- **F33 ⚠️ UI Kit gallery 3×2.** **F36 ⚠️ grid name line height 42 vs 40.** **F37 ⚠️ drawer problem status wraps in a fixed 152 px row** (`item_widgets.cc:75`). **F38 ⚠️ underlined header link.** **F40 ⚠️ notice used as progress** (Wallpaper importing, update check).
- **Text that can wrap left-aligned or overflow** (the canvas "Đang chạy…" case): tweak cells (`tweaks_panel.cc:65`), fallback dialog labels (`plain_dialog.cc:43`), drawer problem status (`item_widgets.cc:75`), drawer hint and empty text. The app-screen canvas itself centres one line and elides.

## Fix plan (S < ½ day, M ½–2 days, L > 2 days)

**Batch 1 · tokens and generated constants**
1.1 `copperui.tokens.json` + generator (C++ header, Go constants, doc table) M · 1.2 generated constants replace `screen_logic.h`/`metrics.go`, parity test S · 1.3 greys and line tokens (F01, F03) S · 1.4 QSS literals to tokens (F05, F16) M · 1.5 line heights (A2) S · 1.6 measured text with `QFontMetrics` and real metrics in ndpreview, one wrap/elide algorithm (F06) L · 1.7 fallback strings into `host/locales`, no jargon (F20) M.

**Batch 2 · components (lib/copperui and the canvas)**
2.1 divider (F02) S · 2.2 serif heading (F04) S · 2.3 button: measured ≤ 2 centred lines, 168, pending "Working…" (F06, F11) M · 2.4 disabled toggle/radio/field + `field.enabled` (F10) M · 2.5 list row component (F12, F13) S · 2.6 pager (F08) S · 2.7 keep-with-next packing (F09) M · 2.8 PlainDialog/menu on tokens (F18) S · 2.9 icons (F14, F15) S · 2.10 big code regular (F31) S · 2.11 radio title (F30) S · 2.12 ndpreview parity (F24) L.

**Batch 3 · surfaces**
3.1 drawer rows, sections, hint, empty, problem and Qt6 rows (F16, F17, F37) M · 3.2 Tinh chỉnh modal (F19) M · 3.3 Settings/About S · 3.4 locale clean-up (F21–F23) S · 3.5 app flows (F25, F26) M · 3.6 notice → steps (F40) S · 3.7 UI Kit rebuild per `uikit-plan.md` L · 3.8 validator rules (ALL CAPS, status width in px, caption period, section over one paragraph) M.

## Status after Batch 3 (10 Oct 2026)

"Fixed" means implemented, covered by host tests and checked in `ndpreview` renders; items that Nickel draws on the device are listed in "Device-only checks" below and stay open until the next device session.

| Finding | Status | Where |
|---|---|---|
| F01 greys, F02 dividers, F03 header divider | fixed (Batch 1, 2) | tokens, `divider.h` |
| F04 serif heading, F05 token source, F06 measured text | fixed (Batch 1, 2) | tokens, `text_meter.cc`, `textfit.go` |
| F07 line heights ≥ 1.4 × size | fixed in the tokens; Vietnamese clipping still needs the device | tokens |
| F08 pager, F09 groups split across pages | fixed (Batch 2); F09 also for whole section groups in Batch 3 (`keepSectionsTogether`) | `pager.h`, `pack.h` |
| F10 disabled controls, F11 pending, F12 list secondary, F13 list primary | fixed (Batch 2) | `screen_layout.cc`, `list_row.h` |
| F14 shapes and glyphs, F15 row icon, F31 big code, F30 radio title | fixed (Batch 2) | `system_icon.cc`, `screen_layout.cc` |
| F16 drawer type sizes | **fixed (3.1)**: status 32, section 28 (uppercased by the renderer), hint 32, no legacy sizes | `drawer_view.cc` |
| F17 drawer empty state | **fixed (3.1)**: icon, serif heading, caption; primary "Find apps" only when the host offers `_refresh` | `drawer_surfaces.cc` |
| F18 PlainDialog and menu metrics | fixed (Batch 2) | `plain_dialog.cc` |
| F19 Tinh chỉnh modal | **fixed (3.2)**: dialog tokens, sans cells, switch image and state word, measured one or two lines | `tweaks_*.cc` |
| F20 hardcoded mod strings | fixed (Batch 1); new keys `mod.find_apps`, `mod.empty_hint`, `mod.problems_section` | `host/locales` |
| F21 tone marks | fixed (Xóa, Hủy, Tùy everywhere, checked by a test) | locales |
| F22 typed capitals | **fixed (3.4, 3.8)**: sentence-case keys, validator warning `ui.uppercase_text`, host locale test | locales |
| F23 header status width | **fixed (3.4)**: Wallpaper "Màn hình chờ: {count}" / "Sleep screen: {count}"; validator warning `ui.status_too_wide`; "5 / 7" spacing | locales |
| F24 ndpreview parity | fixed (Batch 2); Batch 3 adds samples, drawer empty / loading / warning / problem, the new modal | `cmd/ndpreview` |
| F25 content style | **fixed (3.3, 3.4, 3.5)**: no trailing periods on captions, dates from the locale (`dd/MM/yyyy HH:mm`), " · " separators, To-do recovered notice without the path, "Bản mới" row without "●", no section over a lone paragraph | apps, host |
| F26 one-primary rule | **fixed (3.5, 3.8)**: LocalSend Start / Stop, GitHub Pair / Sync and Start / Stop pairing, Library Scan / Stop and a list of rows, Settings Check / Update; `show` flags, validator A8 warning | apps |
| F28 / F29 switch | fixed (Batch 2: Nickel's image, the pill is the preview fallback) | `screen_layout.cc` |
| F33 UI Kit gallery 3×2 | fixed: 4×2 | `apps/uikit` |
| F36 grid name line height | fixed (A2) | tokens |
| F37 drawer problem status wraps | **fixed (3.1)**: one measured line, details in the Info dialog | `item_widgets.cc`, `action_runner.cc` |
| F38 underlined header link | open: Nickel draws its own; device check | |
| F40 notice used as progress | **fixed (3.6)**: Wallpaper importing and Settings check / install are `steps` | apps, host |
| Text that can wrap left-aligned or overflow | **fixed**: tweak cells, drawer problem rows, hint and empty text are measured and elided or centred | mod |

Remaining: F38 and the device-only checks below. New in Batch 3, device only: the Qt6 notice (it costs list rows, so a 5-row page shows) and the painted Tinh chỉnh cells.

## Device-only checks
Nickel TextHeader (title, arrow press, divider) · More-tab rows (icon, dot, press) · N3Dialog (metrics, button order, timeout, page keys) · NickelTouchMenu · Nickel toast and the rate limit · Nickel keyboard (Vietnamese, Go) · switch images 94×56 and the BẬT/TẮT alignment · pressed inversion on every target, no ghost · Vietnamese diacritic clipping (F07) · a long bold-40 label and the 168 px case · drawer list view and Tinh chỉnh modal · refresh counts · dark mode greys · `debug.log` `icon …: Tabler|none` lines.

## Spec questions resolved
See the "Resolved" section of `docs/design-system.md` (A1–A13).

## Re-validation (2026-10-10)

Re-walk of every ⚠️ and ❌ finding after Batches 1–3 (commit `cf4d7e0`), against the code (`mod/drawer`, `mod/common`, `lib/copperui`, `host/screens`, `apps/*`, locales, `cmd/ndpreview`) and 106 fresh `ndpreview` renders per locale (`sh tools/preview-all.sh`, vi and en: drawer list / grid / empty / loading / warning, Tinh chỉnh, Settings main / about / checking / failed, every app screen and state, all UI Kit pages). Greys were measured on the PNGs: row dividers 102, header divider 1 px 102, section rule 2 px black, notice fill 238; the only off-grid pixels are antialiasing on icons, glyphs and sample pictures. `make check` and `appsctl validate` pass.

The audit kept no per-cell matrix, so the counts below are per finding, not per cell.

| Verdict | Before | Now |
|---|---|---|
| ❌ fails | 20 findings (F01–F06, F08–F13, F15–F20, F24, F31) | 0 |
| ⚠️ partial or risk | 15 findings (F07, F14, F21–F23, F25, F26, F28/29, F30, F33, F36–F38, F40, text wrap) | 0 in code (F38 is Nickel's); 4 low new (N4, N5, N7, N8) |
| ✅ fixed in code and renders | — | 34 of 35 (all but F38) |
| Device check still needed | — | 6: F07 diacritic clipping, F16/F17 drawer list view, F18 N3Dialog, F19 painted Tinh chỉnh cells, F28/29 switch images, F38 header link |
| Accepted deviation (design-system "Resolved") | — | F03 (A1), F14 back disc (A4), F21 (A5), F28/29 (A3), F26 (A8) |

### Fixed during re-validation
Preview fixtures had fallen behind the host and app code, so the renders showed defects the device does not have:
- **N1** `tools/preview/settings-state-*.json`: "● 1.0.0" bullet (F25), no `has_update` (Settings showed Check as primary with an update waiting, F26) and the check in progress as a notice instead of `steps` (F40). Now as `internal/drawer/settings_state.go` writes them.
- **N2** `apps/ghsync/preview-state*.json`: ISO "2026-10-10 08:14" (F25) → `format.datetime` "10/10/2026 08:14"; summary with commas → `msg.state_summary` " · "; trailing period on the failed caption.
- **N3** `tools/preview-settings.sh`: About has 3 pages, page 3 (credits) was never rendered.
- **N6** `docs/design-system.md` §2.1, §2.4, §7 still gave `line.header` 187; aligned to A1 (102), as the token file already is.

### Still open (none blocks a device install)
- **N4** Empty drawer still shows the long-press hint ("Long-press an app for info or remove") though there is no app: `mod/drawer/drawer_view.cc:217`, mirrored in `cmd/ndpreview/drawer_empty.go:42`. Hide the hint while the empty state shows. Needs a mod build and docker test.
- **N5** LocalSend idle shows the IP row with an empty value where GitHub shows "—": `apps/localsend/bin/state.sh:30`.
- **N7** Library has no `preview-state*.json`, so its render shows ui.json defaults (blank "Last scan" where the app writes "never"); add fixtures for idle / scanning / duplicates.
- **N8** UI Kit header status "Kit 1 · 18" is cryptic (`apps/uikit/locales/en.json:264`, `vi.json:264`); e.g. "18 components", within 340 px.
- **F38** Header text link underlined: Nickel draws it (`mod/drawer/screen_view.cc:32`); device check.
- Notes, not defects: the pressed primary in the UI Kit button matrix inverts to white with no outline, which is PressArea's `HIGHLIGHT_INVERT` (device check "pressed inversion"); Settings firmware value "4.38.23828 ✓" is a text mark by design (`settings_state.go:125`), A13 covers list rows only.

### Needs the device
Unchanged from "Device-only checks" above, plus the F16/F17 drawer list view and empty state as Nickel lays them out, and the F19 painted Tinh chỉnh cells. Of these, check first: Vietnamese diacritic clipping (F07) on the 46 px caption line, switch images and BẬT/TẮT alignment, pressed inversion without ghosting, and dark-mode greys.

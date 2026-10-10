# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0-beta.4] - 2026-10-11

### Added

- UI Kit 2.1.0: a Pager page ("Chỉ báo trang") that documents the page indicator: when it shows, what the arrows do at the first and last page, the pressed circle, guidance, and a long page so the real pager can be tried. The home now lists 19 components.

### Fixed

- The pressed pager arrow no longer cuts a white strip into the button below it.

## [1.0.0-beta.3] - 2026-10-10

### Fixed

- Pressing a pager arrow on an app screen shows a black circle behind the arrow, like the drawer footer, instead of inverting half the pager line over the page number.
- The UI Kit header says "18 components" instead of "Kit 1 · 18".

## [1.0.0-beta.2] - 2026-10-10

### Added

- **CopperUI tokens** (`lib/copperui/`): one token file for every size, grey, line height and type setting; `make tokens` generates the C++ header, the Go constants and `docs/copperui-tokens.md`, and `make check` fails when one is stale. `mod/drawer` and `ndpreview` read the same numbers.
- The drawer's own texts are now `mod.*` keys in `host/locales/<lang>.json`, read when the drawer opens (exact locale, language, English): a new language needs no rebuild. The compiled English and Vietnamese table remains as the last resort.

### Added (components)

- Pager arrows are Tabler chevron icons on every paged screen, gallery included, and the pager is hidden when there is one page (drawer too).
- Keep-with-next paging: a section label, a field caption, a pair title and QR, steps and a radio group are never split across pages.
- Buttons show "Working…" ("Đang xử lý…") while a quick action runs; disabled toggles, radios and fields are dimmed; `field` accepts `enabled`.
- List rows are their own component (full-width primary text, muted secondary caption, `✓` as the check icon).
- `lib/copperui` is built into the mod as a static library; `ndpreview` renders in the real page area (status bar, tab bar), uses the Tabler icons, and has drawer list and Tinh chỉnh renderers and English fixtures.

### Added (surfaces, batch 3)

- **UI Kit rebuilt**: a grouped home (6 sections, 18 rows, 3 pages on group boundaries), one page per component with a description, a states matrix, a working sample, a readout of the last event and guidance, and a picture viewer one level down.
- `samples` component (a states matrix: normal, pressed, selected, disabled), `push` on `button`, `confirm` and `show` on `row`, `show` on `button`; several primary buttons are allowed when each has its own `show` flag.
- Drawer: empty state with an icon and a "Find apps" button (only when there is something to do), the Qt6 warning as a notice, problem rows on one line with the full message in Info, section labels in sentence case.
- Tinh chỉnh modal on the dialog tokens: sans cells, the Nickel switch image with its state word, info cells with a caption value.
- `steps` show progress of Wallpaper importing and of the update check and install (they were notices); a `steps` block is hidden while its label is empty.
- Validator warnings for content style: typed capitals, a header status wider than 340 px, a one-line caption with a full stop, a section over a single paragraph, a disabled primary next to its opposite action.
- `format.date`, `format.datetime` and `format.time` locale keys; GitHub, Library, Wallpaper and the update status write dates as `dd/MM/yyyy HH:mm` by default.
- `ndpreview`: samples, the new Tweaks modal, `drawer-empty`, `drawer-list --problems --qt6`.

### Changed (surfaces, batch 3)

- LocalSend shows **Start receiving** while idle and **Stop** while receiving (one primary at a time); GitHub shows **Pair with your phone** until paired and then **Sync now**, and **Start pairing** / **Stop pairing**; Library has **Scan** / **Stop** and its actions are rows (shown only when they apply); Settings shows **Check** or **Update** as the primary.
- A row's detail too long for its column goes under the label; rows follow each other without a gap; a section label stays with its rows when they fit a page.
- Section labels in locale files are sentence case (the renderer uppercases them); header status strings fit 340 px ("Màn hình chờ: {count}"); "5 / 7" with spaces; " · " between facts; "Bản mới" row without a text bullet; To-do's recovered notice no longer shows a file path; captions have no trailing full stop; Vietnamese wording "màn hình chờ" everywhere.
- The "Legacy" CopperUI tokens are gone (the modal and the drawer use real tokens): CopperUI 0.3.0.

### Changed

- Headings use the serif, the big code is regular weight, radio titles are captions, dialogs and menus follow the dialog tokens (1040 wide, 3 px border, 36 px text), the drawer row icon is drawn 1:1 at 64 px (apps may ship `icon-64.png`), chevrons are icons.
- Greys sit on the panel's 16-level grid (muted 85, dim 136, notice fill 238); row and menu dividers are Nickel's row grey instead of black, the header divider is Nickel's own, and the drawer's status, section label, hint and pager are muted at the scale sizes (32 / 28 / 32 / 32).
- Line heights are at least 1.4 times the size (caption 46, heading 74, dialog title 60, grid name 46, section 44) so stacked Vietnamese diacritics do not clip.
- Text wraps and elides by measured width in the real font (drawer, app screens and `ndpreview`) instead of an average-width guess: bold labels are no longer cut early, buttons keep at most two centred lines.
- Drawer error messages no longer show tool names or exit codes ("The app drawer hit a problem. Try again."); the code goes to the log.
- Vietnamese uses the new tone-mark style everywhere (Xóa, Hủy, Tùy).
- The native drawer now runs only on models with the tested 1264×1680 screen (Libra 2, Libra H2O, Libra Colour). It reads the model from the device id in `.kobo/version`, checks the live screen size, and on any other model stays off with one "does not support this model yet" toast; the firmware check still applies. New test flag `.adds/nickeldrawer/test-unknown-model`.

## [1.0.0-beta.1] - 2026-10-10

First public beta. Tested on one Libra 2 running 4.38.23828; the other four supported builds are checked statically (symbols) only.

Licensed under PolyForm Noncommercial 1.0.0: free for personal and non-profit use, no commercial use.

### Added

- **Native drawer** (`libnickeldrawer`, a NickelHook mod): an Apps row in the More tab that opens a list or grid of apps, with Nickel's own fonts, icons, dialogs, header and press feedback. Failsafe (the mod removes itself if Nickel crashes right after boot), an `uninstall` flag, a `disable` kill switch and a firmware check that turns the drawer off on an unknown build or a missing symbol.
- **Tweaks** (Tinh chỉnh in Vietnamese): a settings-style modal in the More tab for sleep screen, screenshots, auto USB connect, text legibility, IP address, reboot and library rescan ([docs/spec/tweaks.md](docs/spec/tweaks.md)).
- **UI kit**: `ui.json` screens (lists, toggles, buttons, fields, QR codes, live status, paged lists), validator, JSON schema, nested screens with a native back stack, and `ndpreview` to preview and render screens on a computer.
- **Apps**, all using only the NickelDrawer API (toasts, confirmations and state signals, never raw script output):
  - LocalSend: receive files from a phone; the inbox router sorts them into Books, Documents, Comics and Pictures.
  - GitHub sync: add-only book sync from a private repo, paired from your phone (token never typed on the Kobo).
  - GitHub sync details: its screen shows the repo, last sync, books added and any error; a pairing check names a missing repository, branch or folder instead of a bare 404 (an empty repository pairs); files are checked against the inbox allowlist and magic bytes, sorted into Books, Documents or Comics under `GitHub`, and never overwritten or deleted.
  - Wallpaper: a phone-style gallery with a full-screen viewer for your sleep screen pictures; tells you which Nickel setting to enable.
  - Library: dedupe, language tags and Unicode normalisation.
  - To-do ("Việc cần làm"): add tasks with Nickel's keyboard (Vietnamese NFC-normalised), tap to mark done, long-press to delete, clear done tasks; saved atomically in `data/todo.json`, a damaged file is kept aside and the list starts empty.
  - UI Kit ("Bộ giao diện", hidden by default): every `ui.json` component with sample content on three screens, the target of the KIT-* device cases; show it with ⚙ → Hidden apps (no file edit).
- **Settings** (⚙ in the drawer header, after refresh and the list/grid buttons; it replaces the separate Updates button and carries the update dot): switches for hidden apps (kept in the host's `data/settings.json`, so updates keep them), check and install updates, a debug-log switch, the firmware check (✓ or !), and an About screen with a QR code to the project page, credits and licence. Updates now come from the project repo by default; the project's name, repo, maintainer and supported firmware are set in one file, `host/config/project.json`.
- **Developer kit**: `appsctl new` scaffolds an app, `appsctl validate` checks manifests, locales and `ui.json`; guide in [docs/APPS.md](docs/APPS.md), examples in [examples/](examples/README.md).
- **Signed over-the-air updates** from GitHub Releases (ed25519) with backup, self-check and rollback; `ndrelease` for maintainers. Pre-release versions such as `1.0.0-beta.1` are supported and sort before the final release; only the `beta` channel installs them.
- **Languages**: English source and Vietnamese, chosen from the device language; a new language is one JSON file ([docs/TRANSLATING.md](docs/TRANSLATING.md)).
- **CI**: gofmt, vet and tests; installer test against a fake Kobo tree; app validation; mod build, host unit tests and static checks (bridge symbols against the five supported libnickel builds, ARM hard-float, no C++ standard library, vtable typeinfo); weekly firmware watcher. Tags build the unsigned release inputs.
- Icons come from Nickel's own resources, with [Tabler Icons](https://github.com/tabler/tabler-icons) (MIT) as the fallback.

### Changed

- NickelMenu items are no longer generated; the installer removes the old generated file and switches the NickelMenu tab off. NickelMenu stays optional.
- Mods are built with RTTI so Nickel's `dynamic_cast` cannot crash.
- Wallpaper reads pictures only from `Pictures` and imports them on first open.

### Known limits

- Firmware 5.x is not supported; other Kobo models are untested.
- App screens (`ui.json`) open only when `.adds/nickeldrawer/screens` exists.
- Right-to-left languages are not supported yet.

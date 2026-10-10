# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

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

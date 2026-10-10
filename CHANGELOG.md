# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Apps answer only through the NickelDrawer API: script text never becomes a dialog (toasts and `set` signals only). LocalSend, Library and GitHub sync follow Wallpaper.
- NickelMenu items are no longer generated (`gen-menu`, `refresh`, `errors` removed). The installer removes the old generated file and switches the NickelMenu tab off.

### Added

- `appsctl drawer tweaks` and `drawer tweak <id>`: backend of the native "Tinh chỉnh" modal ([docs/spec/tweaks.md](docs/spec/tweaks.md)).

## [0.1.0] - unreleased

### Added

- App folders with a manifest (`app.json`), per-app locales and a NickelMenu generator; `appsctl validate`, `gen-menu`, `refresh`, `errors`, `msg`.
- Included apps: LocalSend (receive files from a phone), GitHub sync (pairing from the phone), Wallpaper (your pictures as the sleep screen).
- Inbox router that sorts received files into Books, Documents, Comics and Pictures.
- Signed over-the-air updates from GitHub Releases (ed25519), with backup, self-check and rollback; `ndrelease` for maintainers.
- `ui.json` format, validator, UI kit component registry and JSON schema; `ndpreview` to preview and render screens on a computer.
- Native drawer host (`libnickeldrawer`) and `appsctl drawer` API, build only; native app screens from `ui.json` behind a flag.
- `appsctl new` to scaffold an app (optionally with a `ui.json`), and the guide in `docs/APPS.md`.
- Test mod toolchain and install/uninstall drill (gate G1 passed on a Libra 2 running 4.38.23828).
- CI for firmware symbol checks and a firmware watcher.

### Changed

- NickelMenu stays short (Apps · Refresh list and friends) until the native drawer ships.
- Wallpaper reads pictures only from `Pictures`.
- Mods are built with RTTI so Nickel's `dynamic_cast` cannot crash.

### Known limits

- The native drawer is not yet validated on a device (gate G2); menu apps are.
- Firmware 5.x is not supported.

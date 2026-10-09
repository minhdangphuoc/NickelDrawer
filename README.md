# NickelDrawer

NickelDrawer is an app drawer and a set of small tools for Kobo e-readers running the stock firmware (Nickel). Apps are plain folders with a manifest and a few scripts; the menu is generated from them, and a native "Apps" screen is being built on top. It keeps Kobo's own look, works without a cloud account, and sends nothing anywhere unless you start a feature that needs the network. English and Vietnamese ship first; any other language is a single JSON file.

| LocalSend | Wallpaper |
|---|---|
| ![LocalSend screen preview](docs/img/localsend.png) | ![Wallpaper screen preview](docs/img/wallpaper.png) |

Previews are rendered on a computer with `ndpreview` (layout check at the e-ink size, not a photo of a device). The native drawer is not released yet.

## Features

- **App folders.** Drop a folder in `.adds/apps/` and it appears in the menu; no code changes anywhere else.
- **Native screens (`ui.json`).** Lists, toggles, buttons and live status for apps that want more than a dialog (in progress, see [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md)).
- **Included apps.** LocalSend (receive files from a phone, sorted into your library folders), GitHub sync (pull books from a repo) and Wallpaper (your pictures as the sleep screen).
- **Any language.** English source, translations in `locales/<lang>.json`, picked from the device language.
- **Signed updates.** Apps and the host tool update over Wi-Fi from GitHub Releases, verified with an ed25519 key ([docs/UPDATES.md](docs/UPDATES.md)).
- **Safe by design.** NickelMenu stays as a fallback; the native part disables itself on an unknown firmware.

## Compatibility

Target: Kobo Libra 2, firmware 4.38.x. Firmware 5.x is not supported. Per-build status is in [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md).

## Install

USB installer (Kobo mounted on a Mac or Linux computer):

```sh
make check build dist                  # tests, ARMv7 binaries, dist/ tree and zip
tools/install-to-kobo.sh --dry-run     # show what would happen
tools/install-to-kobo.sh               # install (set KOBO_ROOT if the volume has another name)
```

The installer backs up everything it replaces to `backups/<date-time>/` first and never touches an app's `data/` folder. Eject the Kobo afterwards and tap **Apps · Refresh list**. Over-the-air installation from a GitHub Release is planned; once installed, updating already works that way ([docs/UPDATES.md](docs/UPDATES.md)).

## Safety model

Apps run as root on a device that holds your library, so the rules are strict: backups before any write, no telemetry, signed releases, no Kobo binaries in this repository, and a documented uninstall and failsafe path. Read [SECURITY.md](SECURITY.md) before installing or reporting a problem.

## Building

Go 1.24 or newer, `make`, `zip`. See [CONTRIBUTING.md](CONTRIBUTING.md).

```sh
make check      # gofmt, go vet, go test
make build      # ARMv7 binaries for the Kobo plus host tools in build/
make dist       # dist/ tree and dist/koboapps-<version>.zip
```

Maintainers cut releases with the steps in [docs/RELEASING.md](docs/RELEASING.md).

## Developing apps

```sh
go run ./cmd/appsctl new my-app --with-ui     # scaffold ./my-app (add --dir to choose where)
go run ./cmd/appsctl validate my-app          # checks app.json, locales and ui.json
go run ./cmd/ndpreview serve my-app           # preview ui.json at Kobo size on 127.0.0.1
```

The guide is [docs/APPS.md](docs/APPS.md); the formats are in [docs/spec/](docs/spec/). Worked examples are in [examples/](examples/README.md).

Other helpers: `tools/vncheck.py FONT...` (missing Vietnamese glyphs), `tools/nfc.py DIR` (non-NFC text in EPUB HTML), `make schema` (rewrite `spec/ui.schema.json`).

## License

MIT, see [LICENSE](LICENSE). Third-party code keeps its own license, listed in [third_party/README.md](third_party/README.md).

## Credits

- [NickelMenu](https://github.com/pgaskin/NickelMenu) and [NickelHook](https://github.com/pgaskin/NickelHook) by Patrick Gaskin: the menu fallback and the hook framework the native drawer builds on.
- [NickelTC](https://github.com/pgaskin/NickelTC) and the wider Kobo reverse-engineering community for the knowledge of how Nickel works.
- [localgo](https://github.com/bethropolis/localgo): the LocalSend protocol implementation used by the LocalSend app (MIT).
- [kobo-tieng-viet](https://github.com/redphx/kobo-tieng-viet): Vietnamese UI and font fixes for Kobo; NickelDrawer is designed to coexist with it (not a dependency).

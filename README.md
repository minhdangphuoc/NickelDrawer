# NickelDrawer

An app drawer for Kobo e-readers that run the stock firmware (Nickel). It adds an **Apps** row to the More tab, and gives small apps a native Kobo look. It works without a cloud account and sends nothing anywhere unless you start a feature that needs the network.

**Status: 1.0.0 beta** ([releases](../../releases)). Verified on one device and one firmware build (see [Supported firmware](#supported-firmware)). Read [Safety](#safety) before you install.

Tiếng Việt: [README.vi.md](README.vi.md).

| List view | Grid view |
|---|---|
| ![The drawer's list view on a Kobo Libra 2](docs/img/device-list.png) | ![The drawer's grid view on a Kobo Libra 2](docs/img/device-grid.png) |

Screenshots from a Kobo Libra 2 set to Vietnamese. The drawer shows your apps as a list (icon, name, live status) or as a grid; the buttons in the header switch between them, refresh the list and open the drawer settings.

## What you get

- **A native drawer.** An Apps row in Nickel's More tab opens a list or grid of your apps. It uses Nickel's own fonts, icons and dialogs.
- **Apps are folders.** Copy a folder with a manifest and a few scripts to `.adds/apps/` and it appears. No C++ needed ([docs/APPS.md](docs/APPS.md)).
- **Native app screens.** Apps describe lists, toggles, buttons, QR codes and live status in a `ui.json` file; the drawer draws them.
- **Signed updates.** The drawer (host tool and mod) updates over Wi-Fi from GitHub Releases. Only releases signed with the maintainer's ed25519 key are applied, with backup, self-check and rollback ([docs/UPDATES.md](docs/UPDATES.md)).
- **Any language.** English is the source; Vietnamese ships first. The language follows the Kobo's own setting ([docs/TRANSLATING.md](docs/TRANSLATING.md)).

## Supported firmware

Kobo Libra 2 (Mark 9, firmware family `kobo9`), the five newest 4.38 builds. The native drawer checks every libnickel symbol it needs at boot and turns itself off, with a log line, on a build or model it does not know or a missing symbol.

| Firmware | Native drawer | Notes |
|---|---|---|
| 4.38.23828 | tested on a device | |
| 4.38.23697 | symbols checked in CI | not run on a device yet |
| 4.38.23684 | symbols checked in CI | not run on a device yet |
| 4.38.23648 | symbols checked in CI | not run on a device yet |
| 4.38.23552 | symbols checked in CI | not run on a device yet |

Firmware 5.x is **not** supported. Details: [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md).

### Supported devices

Of the 27 Kobo models ever sold, **1 is tested** (Libra 2, firmware 4.38.23828), **2 are expected to work but untested** (Libra H2O and Libra Colour, the other models with the same 1264×1680 screen), **19 run firmware 4.x but the drawer stays off** (screen size not supported yet), and **5 are not supported** (Kobo eReader and Wireless, Mini, the Vox and Arc tablets). Libra Colour and Clara units converted to 5.x are not supported either. The screens are laid out for the 1264×1680 panel, so the mod checks the model (device id in `.kobo/version`) and the live screen size, and on any other model it shows a "does not support this model yet" toast and stays off. It also loads only on the exact builds listed above. The full table, with years, platforms, device ids and how to help add a model, is in [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md#supported-devices).

## Install

You need a Kobo Libra 2 on one of the builds above, a computer, a USB cable and the files from the latest [release](../../releases): `koboapps-<version>.zip` (the drawer's host tool) and `KoboRoot-nickeldrawer.tgz` (the native drawer).

1. **Back up first.** Connect the Kobo, copy the whole `.kobo` and `.adds` folders to your computer. Keep them until you are happy.
2. Unzip `koboapps-<version>.zip`. Copy the `.adds` folder inside `onboard/` into the root of the Kobo drive and merge it with the existing `.adds` (hidden folders: Cmd+Shift+. in macOS Finder, "show hidden files" on Windows and Linux). The zip holds no apps and no `data/` folders. **macOS Finder: if it asks whether to Replace a folder that already exists, do not.** Replace deletes the folder's existing contents (app settings, tokens, other mods' files). Hold Option and choose **Merge**, or use the installer below.
3. Copy `KoboRoot-nickeldrawer.tgz` to `.kobo/KoboRoot.tgz` on the drive (rename it while copying).
4. Eject the Kobo from the computer. It installs the drawer and reboots. After the boot, the More tab has an **Apps** row.

The drawer is empty at first. **Apps are separate from the drawer:** each is its own `app-<id>-<version>.zip` in the release, installed one by one; see [apps/README.md](apps/README.md). A drawer install or update never adds, changes or removes an app.

On macOS or Linux with a checkout of this repository there is a safer installer that backs up everything it replaces to `backups/<date-time>/` and never touches `data/` folders:

```sh
make check build dist
tools/install-to-kobo.sh --dry-run     # show what would happen
tools/install-to-kobo.sh               # installs the drawer only; finds /Volumes/KOBOeReader, /media/$USER/KOBOeReader or /run/media/$USER/KOBOeReader; else set KOBO_ROOT=/path
```

[NickelMenu](https://github.com/pgaskin/NickelMenu) is optional. NickelDrawer does not need it, and it coexists with it.

### Switches

Empty files you create over USB in `.adds/nickeldrawer/` (create the folder if it is missing). They are read at boot.

| File | Effect |
|---|---|
| `uninstall` | Removes the native drawer on the next boot, then deletes itself |
| `disable` | Loads the drawer but keeps it inactive (kill switch) |
| `no-screens` | Opt out of app screens: a tap runs the app's main action instead of opening its `ui.json` screen. Screens are on by default (an old `screens` file is ignored) |
| `no-keyboard` | Text fields show "edit on your phone" instead of Nickel's keyboard |
| `debug` | Writes a step log to `.adds/nickeldrawer/debug.log` |
| `test-unknown-fw` | Test hook: pretends the firmware is unknown, so the drawer disables itself with one toast. Delete it and reboot to return to normal |

## Safety

Apps run as root on a device that holds your library. Install only apps you trust. The rules:

- The native drawer is a [NickelHook](https://github.com/pgaskin/NickelHook) mod with a **failsafe**: if Nickel crashes (or you power off once) in the first 15 seconds after the mod loads, the mod is left out at the next boot and Nickel starts as stock.
- **Uninstall:** create `.adds/nickeldrawer/uninstall` (empty), eject, wait for the reboot. Then delete the `.adds/koboapps` and `.adds/apps` folders if you want everything gone. Your books are never touched.
- No telemetry. The network is used only for what you start (an app's network feature, the update check).
- Releases are signed (ed25519). The private key stays on the maintainer's computer.
- No Kobo binaries or Kobo assets are in this repository or in a release.

Read [SECURITY.md](SECURITY.md) before you install or report a problem.

## Building

Go 1.24 or newer, `make`, `zip`. The native drawer needs Docker (the pinned [NickelTC](https://github.com/pgaskin/NickelTC) toolchain image, see [mod/TOOLCHAIN.md](mod/TOOLCHAIN.md)).

```sh
make check                          # gofmt, go vet, go test
make build dist                     # ARMv7 binaries, dist/ tree, dist/koboapps-<version>.zip
bash tools/test-install.sh          # installer against a fake Kobo tree

make -C mod/drawer docker-build     # build/mod/KoboRoot-nickeldrawer.tgz
make -C mod/drawer docker-test      # host unit tests of the Qt-free units
mod/drawer/check-symbols.sh         # static checks of the built library
```

CI and release steps: [docs/CI.md](docs/CI.md), [docs/RELEASING.md](docs/RELEASING.md). Contributing: [CONTRIBUTING.md](CONTRIBUTING.md).

## Writing an app

```sh
go run ./cmd/appsctl new my-app --with-ui     # scaffold ./my-app
go run ./cmd/appsctl validate my-app          # checks app.json, locales and ui.json
go run ./cmd/ndpreview serve my-app           # preview ui.json at Kobo size on 127.0.0.1
```

The guide is [docs/APPS.md](docs/APPS.md); the formats are in [docs/spec/](docs/spec/); worked examples are in [examples/](examples/README.md).

## Languages

English and Vietnamese are maintained here. A new language is one JSON file per app and one for the host, no code change. See [docs/TRANSLATING.md](docs/TRANSLATING.md). Right-to-left languages are not supported yet.

Tiếng Việt: xem [README.vi.md](README.vi.md).

## License

[PolyForm Noncommercial 1.0.0](LICENSE): free for personal use, study, hobby projects and non-profit organisations; **no commercial use**. Forks and changes are welcome under the same terms. NickelDrawer is source-available, not OSI open source. Third-party code keeps its own license, listed in [third_party/README.md](third_party/README.md).

**Apps you write are yours.** The license does not reach apps that only use the drawer's app format, protocol and tools: you choose their license, commercial included, and the app template and examples are 0BSD. Details: [LICENSE-APPS.md](LICENSE-APPS.md).

## Credits

- [NickelHook](https://github.com/pgaskin/NickelHook) and [NickelMenu](https://github.com/pgaskin/NickelMenu) by Patrick Gaskin (MIT): the hook framework the native drawer is built on, and the menu it coexists with.
- [NickelTC](https://github.com/pgaskin/NickelTC): the cross toolchain, and the wider Kobo reverse-engineering community for the knowledge of how Nickel works.
- [localgo](https://github.com/bethropolis/localgo) (MIT): the LocalSend protocol implementation.
- [Tabler Icons](https://github.com/tabler/tabler-icons) (MIT): fallback icons when Nickel's own are not available.

NickelDrawer is an independent project. It is not made by, affiliated with or endorsed by Rakuten Kobo. "Kobo" and "Nickel" are trademarks of their owners.

## Extra tools and setups

Optional, separate from the drawer. Install only what you want.

- **Apps** ([apps/README.md](apps/README.md)): LocalSend, GitHub sync, Library, Wallpaper and To-do (plus the developer-only UI Kit). Each is its own zip in the release and is installed, updated and removed one by one; the drawer and its updates never touch them.

- **Extras** ([helpers/README.md](helpers/README.md)): a small separate mod with a screenshot button in Kobo's top status bar and a **Tweaks** panel in the More tab (sleep screen, auto USB connect, text legibility, IP address, reboot, library rescan). It works with or without NickelDrawer. Built and installed from the source; there is no release file (it is not part of NickelDrawer's releases). Install and uninstall steps are in that README.

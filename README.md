# NickelDrawer

An app drawer for Kobo e-readers that run the stock firmware (Nickel). It adds an **Apps** row to the More tab, and gives small apps a native Kobo look. It works without a cloud account and sends nothing anywhere unless you start a feature that needs the network.

**Status: 1.0.0 beta** ([releases](../../releases)). Verified on one device and one firmware build (see [Supported firmware](#supported-firmware)). Read [Safety](#safety) before you install.

Tiếng Việt: [README.vi.md](README.vi.md).

| List view | Grid view |
|---|---|
| ![The drawer's list view on a Kobo Libra 2](docs/img/device-list.png) | ![The drawer's grid view on a Kobo Libra 2](docs/img/device-grid.png) |

Screenshots from a Kobo Libra 2 set to Vietnamese. The drawer shows your apps as a list (icon, name, live status) or as a grid; the buttons in the header switch between them, refresh the list and open the drawer settings.

> **Built on [NickelHook](https://github.com/pgaskin/NickelHook).** Patrick Gaskin's NickelHook is what makes a safe Nickel mod possible at all: it loads our library into Nickel, resolves every libnickel symbol by name instead of by address, and brings the 15-second failsafe and the `uninstall` flag that let you back out of a bad build without a computer. NickelDrawer, and the optional extras mod, are NickelHook plugins. Thank you, pgaskin, for NickelHook, [NickelMenu](https://github.com/pgaskin/NickelMenu) and [NickelTC](https://github.com/pgaskin/NickelTC), and for years of documenting how Nickel works.

## Features

### The drawer

- **An Apps row in the More tab** that opens your apps as a **list** (icon, name, live status line) or a **grid** of icons. The choice is remembered.
- **Native look.** Nickel's own header, fonts, icons, dialogs, toasts and keyboard; pages instead of scrolling, turned with the ‹ › pager or the page keys.
- **Long-press an app** for its info (version, storage, folder) or to remove it, with a confirmation.
- **Refresh** in the header rescans `.adds/apps/` at once.
- **Drawer settings** (the gear): show hidden apps, check for and install updates, debug log, firmware check, and a project page with a QR code.

### Apps

- **An app is a folder** with `app.json`, a few scripts and optional `ui.json` screens: no C++ and no rebuild of the drawer.
- **Native app screens** described in `ui.json`: lists, rows, toggles, radio choices, text fields with Nickel's keyboard, buttons, notices, progress steps, QR codes, galleries and a picture viewer, all drawn by the drawer in the CopperUI style.
- **A small, safe protocol.** Apps talk to the drawer only through JSON events (state, toast, confirm, rescan, Wi-Fi); they never draw their own dialogs.
- **Background jobs** with live progress, and requests for a library rescan or Wi-Fi when the user starts something that needs it.
- **Ready-made apps** to install one by one: GitHub sync, Library, LocalSend, To-do and Wallpaper (list at the end of this page).

### Updates

- **Drawer updates over Wi-Fi** from GitHub Releases: check, install, restart. Signed with ed25519; the device verifies the signature, every file's checksum and size, keeps a backup and rolls back if the self-check fails.
- **App updates over Wi-Fi**, separate from the drawer: **Settings › Updates › Apps** lists updates, installed apps and apps you have not installed yet. Update one or all, or install a new one. Each app's `data/` is moved into the new copy before the swap, so a power cut cannot lose it.
- **Nothing runs in the background.** The network is used only when you tap Check or start a feature that needs it. No telemetry.

### Safety

- Every libnickel symbol is checked at load time; an unknown firmware, an unsupported model or a missing symbol turns the drawer off with one message instead of crashing Nickel.
- Failsafe and uninstall switches from NickelHook, plus a kill switch (`disable`).
- No Kobo binaries or Kobo assets in the repository or in a release.

### For developers

- **`appsctl new`** scaffolds an app; **`appsctl validate`** checks its manifest, screens and translations against the design rules.
- **`ndpreview`** renders any screen at the Kobo's exact size on your computer, live in a browser or as a PNG.
- **CopperUI** (`lib/copperui/`): the design system the drawer draws with: one token file for sizes, greys and type, generated into C++ and Go, plus a small Qt Widgets library.
- **The UI Kit app**, a catalogue of every component with sample content (hidden by default).
- **Your app, your license**: apps that use only the public interfaces may use any license, commercial included ([LICENSE-APPS.md](LICENSE-APPS.md)).
- Guides: [developer guide](docs/guide/developer-guide.md), [publishing apps](docs/guide/publishing-apps.md), formats in [docs/spec/](docs/spec/), examples in [examples/](examples/README.md).

### Any language

English is the source; Vietnamese is maintained here. The language follows the Kobo's own setting, and a new language is one JSON file per app and one for the host, no code change ([docs/TRANSLATING.md](docs/TRANSLATING.md)). Right-to-left languages are not supported yet.

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

The drawer is empty at first. **Apps are separate from the drawer:** install them from the Kobo itself in **Apps › gear › Updates › Apps** (Wi-Fi), or copy them over USB as described in [apps/README.md](apps/README.md). A drawer install or update never adds, changes or removes an app.

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

## License

[PolyForm Noncommercial 1.0.0](LICENSE): free for personal use, study, hobby projects and non-profit organisations; **no commercial use**. Forks and changes are welcome under the same terms. NickelDrawer is source-available, not OSI open source. Third-party code keeps its own license, listed in [third_party/README.md](third_party/README.md).

**Apps you write are yours.** The license does not reach apps that only use the drawer's app format, protocol and tools: you choose their license, commercial included, and the app template and examples are 0BSD. Details: [LICENSE-APPS.md](LICENSE-APPS.md).

## Credits

- **[NickelHook](https://github.com/pgaskin/NickelHook)** by Patrick Gaskin (MIT): the hook framework both mods are built on, with its symbol checks, failsafe and uninstall flag. Without it there would be no safe way to put a drawer inside Nickel.
- [NickelMenu](https://github.com/pgaskin/NickelMenu) by Patrick Gaskin (MIT): the menu NickelDrawer coexists with, and the reference for many of the Nickel calls used here.
- [NickelTC](https://github.com/pgaskin/NickelTC): the cross toolchain, and the wider Kobo reverse-engineering community for the knowledge of how Nickel works.
- [localgo](https://github.com/bethropolis/localgo) (MIT): the LocalSend protocol implementation.
- [Tabler Icons](https://github.com/tabler/tabler-icons) (MIT): fallback icons when Nickel's own are not available.

NickelDrawer is an independent project. It is not made by, affiliated with or endorsed by Rakuten Kobo. "Kobo" and "Nickel" are trademarks of their owners.

## Extra tools and setups

Optional, separate from the drawer. Install only what you want.

### Apps

Each is its own zip in the release; install, update and remove them one by one ([apps/README.md](apps/README.md)).

| App | What it does |
|---|---|
| [GitHub](apps/ghsync/README.md) | Sync books from a private GitHub repo, paired from your phone (add-only, never deletes) |
| [Library](apps/library/README.md) | Fix the language of Vietnamese books, normalise names, find duplicate books |
| [LocalSend](apps/localsend/README.md) | Receive books and files from your phone, sorted into Books, Documents, Comics and Pictures |
| [To-do](apps/todo/README.md) | A small to-do list that stays on your Kobo |
| [Wallpaper](apps/wallpaper/README.md) | Use your own pictures as the sleep screen, fixed or random |
| [UI Kit](apps/uikit/README.md) | Every component with sample content, for developers (hidden by default) |

### Extras

[helpers/README.md](helpers/README.md): a small separate NickelHook mod with a screenshot button in Kobo's top status bar and a **Tweaks** panel in the More tab (sleep screen, auto USB connect, text legibility, IP address, reboot, library rescan). It works with or without NickelDrawer. Built and installed from the source; there is no release file (it is not part of NickelDrawer's releases). Install and uninstall steps are in that README.

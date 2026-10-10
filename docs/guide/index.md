# NickelDrawer documentation

NickelDrawer adds an **Apps** drawer and a **Tweaks** window to the More tab of a Kobo e-reader running Kobo's own software, with small apps that look like the rest of the Kobo. It is an independent project, not made by or affiliated with Rakuten Kobo.

**Status: 1.0.0-beta.1.** Supported on the Kobo Libra 2 with five 4.38 firmware builds; tried on one device so far. Each guide says what has run on a real device.

## Guides

| Guide | For | Language |
|---|---|---|
| [Hướng dẫn sử dụng](user-guide.vi.md) | Người dùng Kobo: cài, dùng các ứng dụng, xử lý sự cố, gỡ cài đặt | Tiếng Việt |
| [User guide](user-guide.en.md) | Kobo owners: install, use the apps, fix problems, uninstall | English |
| [Developer guide](developer-guide.md) | People who build apps for the drawer: a 15-minute tutorial and the reference | English |

Rendered pages: [user-guide.vi.html](user-guide.vi.html), [user-guide.en.html](user-guide.en.html), [developer-guide.html](developer-guide.html).

## Start here

- **Install it:** [check your firmware](user-guide.en.md#check-that-your-kobo-is-supported), [back up](user-guide.en.md#back-up), then [install](user-guide.en.md#install).
- **Something is wrong:** [troubleshooting](user-guide.en.md#troubleshooting) and [safety and uninstall](user-guide.en.md#safety-and-uninstall).
- **Build an app:** [the 15-minute tutorial](developer-guide.md#build-an-app-in-15-minutes).
- **Translate:** [languages in the user guide](user-guide.en.md#languages) and [TRANSLATING.md](../TRANSLATING.md).

## Specifications

The exact formats and contracts. When a guide and a spec disagree, the spec wins.

| Spec | What it defines |
|---|---|
| [app-format.md](../spec/app-format.md) | An app folder and its `app.json` |
| [ui-json.md](../spec/ui-json.md) | The `ui.json` screen schema and validator error codes |
| [ui-kit.md](../spec/ui-kit.md) | How each component looks, sizes, screen patterns |
| [drawer-protocol.md](../spec/drawer-protocol.md) | The JSON protocol between the Kobo mod and `appsctl` |
| [tweaks.md](../spec/tweaks.md) | The Tweaks window and the settings it changes |
| [inbox.md](../spec/inbox.md) | How received files are checked and sorted |
| [ghsync-pairing.md](../spec/ghsync-pairing.md) | Phone pairing and token safety for GitHub sync |
| [library.md](../spec/library.md) | The Library app |
| [wallpaper.md](../spec/wallpaper.md), [wallpaper-gallery.md](../spec/wallpaper-gallery.md) | The Wallpaper app and its gallery |

## More background

- [README.md](../../README.md): overview, build commands, credits.
- [COMPATIBILITY.md](../COMPATIBILITY.md): firmware and device support.
- [UPDATES.md](../UPDATES.md): how signed updates work.
- [ARCHITECTURE.md](../ARCHITECTURE.md) and [DECISIONS.md](../DECISIONS.md): design and the decisions behind it.
- [SECURITY.md](../../SECURITY.md): reporting a problem.
- [LICENSE](../../LICENSE): PolyForm Noncommercial 1.0.0. Free for personal, study, hobby and non-profit use; no commercial use. Source-available, not open source.

The screen pictures in these guides are drawn on a computer with `ndpreview` at the Kobo's screen size, not photographed on a device.

# Helpers

Optional mods that are **not part of NickelDrawer** and not in its releases. You build them from the source and install them yourself. Today there is one: **Extras**.

## Extras

Extras is a small Nickel mod (`libextras.so`, NickelHook key `nh.extras`) with two features:

- **Screenshot button.** A button in Kobo's top status bar, just left of the brightness icon. A tap saves the screen as `screen_NNN.png` at the root of the Kobo drive and shows a toast.
- **Tweaks.** A row **Tweaks** (Vietnamese: **Tinh chỉnh**) in the More tab that opens a settings-style window: sleep screen, screenshot button on or off, auto USB connect, text legibility, IP address, invert colours and reboot, sleep, reboot, library rescan.

It lives in `/mnt/onboard/.adds/helpers/extras/` and has its own flag files and failsafe.

**Standalone and side by side.** Extras works with no NickelDrawer on the Kobo (no `.adds/koboapps`, no `appsctl`), and NickelDrawer works with no Extras. Installed together they do not conflict: each adds its own row to the More tab, hooks a different Nickel function, keeps its own folder and flags, and is uninstalled on its own. Both stage a `KoboRoot.tgz` and Kobo installs only one per reboot, so install one, reboot, then install the other.

### Install

Needs Docker, Go and a Kobo Libra 2 on firmware 4.38 (one of the builds in `docs/COMPATIBILITY.md`). Do the first run with the failsafe and uninstall drill done and yourself present (plan gate G1 in `docs/plan-detailed.md`).

```sh
make build                                   # Go tool: build/helpers/extras-tweaks-linux-armv7
make -C helpers/extras docker-build          # build/helpers/libextras.so and KoboRoot-extras.tgz
helpers/extras/check-symbols.sh              # symbols on all five firmware builds, vtables, plugin key
```

Plug the Kobo in, tap Connect, then:

```sh
helpers/extras/install.sh --dry-run          # shows what would change
helpers/extras/install.sh                    # backs up, copies files, stages the library
```

Eject and reboot the Kobo. The installer backs up `.adds/helpers/extras` to `backups/<date>/`, refuses when another `KoboRoot.tgz` is waiting, and never touches `.adds/koboapps` or `.adds/nickeldrawer`.

If the old status-bar helper (`libstatusbarshot.so`, folder `.adds/helpers/screenshot`) was installed, the installer removes its files and writes its uninstall flag, so the old library goes at the same reboot. If you had deleted that folder by hand but the library is still on the Kobo, create an empty file `.adds/helpers/screenshot/uninstall` over USB before you reboot.

### Uninstall

```sh
helpers/extras/install.sh --uninstall
```

or create an empty file `.adds/helpers/extras/uninstall` on the Kobo over USB. Eject and reboot: NickelHook deletes `libextras.so` at boot. Afterwards delete the folder `.adds/helpers/extras` if you want the files gone. NickelDrawer is not affected.

If Nickel crashes right after boot, NickelHook's failsafe sets each mod aside (one flag per library), so a faulty Extras never leaves you without the Kobo. The file `.adds/helpers/extras/disable` turns Extras off without removing it (read at boot).

### Files and flags (`/mnt/onboard/.adds/helpers/extras/`)

| File | Effect |
|---|---|
| `screenshot-off` | The status-bar button is hidden. Absent = shown (the default). The Tweaks window creates and removes it; the bar updates the next time it is shown. |
| `disable` | Kill switch: both features stay off (read at boot). |
| `debug` | Verbose lines in `debug.log` in the same folder (read at boot). |
| `uninstall` | NickelHook deletes the library at the next boot. |
| `bin/`, `locales/`, `icons/`, `LICENSES/` | Installed files; `locales/<lang>.json` holds all texts (English and Vietnamese; add a language by adding a file). |

More: [helpers/extras/README.md](extras/README.md) (build and layout), [the Tweaks contract](extras/docs/tweaks.md), [the status-bar design](extras/docs/statusbar-screenshot.md), [the device checklist](extras/docs/device-checklist.md) (Vietnamese).

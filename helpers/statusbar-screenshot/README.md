# Status-bar screenshot helper

A personal helper for the maintainer's own Kobo, not a NickelDrawer feature. It is a separate NickelHook
mod (`libstatusbarshot.so`, plugin key `nh.statusbarshot`) with its own flags and failsafe. It is not part
of `make dist`, `dist/`, the release zips, the README, the guide or the changelog. The design, symbols and
the device checklist are in `re/notes/statusbar-screenshot.md`.

It adds a Tabler "screenshot" button to Nickel's status bar, just left of the brightness icon. A tap blanks
the button for about 250 ms (`CAPTURE_DELAY_MILLISECONDS` in `screenshot_button.cc`), calls Nickel's
`ScreenshotMaker::saveCurrentScreen()` (a file `/mnt/onboard/screen_NNN.png`), restores the button, checks
that a new file appeared and shows a toast. Texts: `locales/<lang>.json`, chosen by `CurrentLocale`
(exact locale, language, `en`), with a compiled English fallback.

## Build and check

From the repo root (needs Docker):

```sh
make -C helpers/statusbar-screenshot docker-build   # build/helpers/libstatusbarshot.so + KoboRoot-statusbarshot.tgz
make -C helpers/statusbar-screenshot docker-test    # host tests of the Qt-free units
helpers/statusbar-screenshot/check-symbols.sh       # symbols on all five firmware builds, ELF, plugin key, KoboRoot, vtables
helpers/statusbar-screenshot/test-install.sh        # install.sh against a fake Kobo tree
```

`docker-build` first deletes the plugin manifest NickelHook generates inside the shared
`third_party/NickelHook/` folder; a stale one from the drawer build would give this library the drawer's
key. `check-symbols.sh` fails when the key `nh.statusbarshot` is not in the library.

## Install (maintainer, G1 only with the user present)

```sh
helpers/statusbar-screenshot/install.sh --dry-run
helpers/statusbar-screenshot/install.sh             # backs up, copies locales and icon, stages KoboRoot.tgz
helpers/statusbar-screenshot/install.sh --uninstall # writes the uninstall flag; the library goes at the next boot
```

It backs up `.adds/helpers/screenshot` to `backups/<date>/`, refuses when another `KoboRoot.tgz` is waiting,
and never touches `.adds/nickeldrawer`. Do the uninstall and failsafe drills first (the note, section 7).

## Flag files (`/mnt/onboard/.adds/helpers/screenshot/`)

| File | Effect |
|---|---|
| `off` | No button. Absent = button (the default). NickelDrawer's hidden Tweaks row creates and removes it. |
| `disable` | Kill switch: the hook passes straight through (read at boot). |
| `debug` | Verbose lines in the syslog under `(statusbarshot)` (read at boot). |
| `uninstall` | NickelHook deletes the library at the next boot. |

The helper reads `off` with one `stat()` when a status bar is built, when it is shown and on each tap. The
drawer also sets the dynamic property `nhScreenshotButton` on the `StatusBarView` for an immediate update;
there is no link between the two libraries.

## Layout

| File | Job |
|---|---|
| `statusbarshot.cc` | `nh_info` and the tiny `init()` |
| `nickel_bridge.{h,cc}` | the only file naming libnickel symbols: hook, dlsym table, model and firmware gate, wrappers |
| `screenshot_button.{h,cc}` | the button, the flag checks and the capture sequence |
| `helper_flags`, `capture_check`, `helper_log` | flag files, the "did a new file appear" rule, debug lines |
| `helper_strings`, `icon_pixmaps` | texts (JSON locale files), the three images |
| `../../mod/common/{device_gate,locale_chain}.cc` | model, firmware and locale-chain rules shared with the drawer (each library compiles its own copy) |

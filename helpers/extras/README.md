# Extras helper

A separate NickelHook mod (`libextras.so`, plugin key `nh.extras`) with the status-bar screenshot button and the Tweaks window. It is not part of NickelDrawer: not in `make dist`, `dist/` or the release zips. Install and uninstall steps for users are in [../README.md](../README.md). It needs nothing from NickelDrawer and shares no file or hook with it.

## Build and check

From the repo root (needs Docker):

```sh
make build                                  # build/helpers/extras-tweaks-linux-armv7 (Go)
make -C helpers/extras docker-build         # build/helpers/libextras.so + KoboRoot-extras.tgz
make -C helpers/extras docker-test          # host tests of the Qt-free units
helpers/extras/check-symbols.sh             # all five firmware builds, ELF, plugin key, KoboRoot, vtables
helpers/extras/test-install.sh              # install.sh against fake Kobo trees (helper only, drawer only, both)
```

`docker-build` first deletes the plugin manifest NickelHook generates inside the shared `third_party/NickelHook/` folder: a stale one from another mod would give this library that mod's key.

## How it works

- **Screenshot button.** Hook on `StatusBarView::StatusBarView(QWidget*)`; a Nickel `StatusBarIcon` is added before the brightness icon. A tap blanks the button for about 250 ms, calls `ScreenshotMaker::saveCurrentScreen()` (`/mnt/onboard/screen_NNN.png`), restores it, checks a new file appeared and shows a toast. Design, symbols and risks: [docs/statusbar-screenshot.md](docs/statusbar-screenshot.md).
- **Tweaks row.** Hook on `Ui_MoreView::setupUi(QWidget*)` adds a Nickel `IconLeftButton`; a tap opens Nickel's own `N3Dialog` with the cells. Each tweak runs the small Go program `bin/extras-tweaks list|apply <id>` (JSON). Contract: [docs/tweaks.md](docs/tweaks.md); window choice: [docs/tweaks-modal.md](docs/tweaks-modal.md).
- **One hook per symbol.** NickelHook's `nh_dlhook` returns the original function from `dlsym`, not the previous hook, so two mods hooking the same symbol silently drop the first. NickelDrawer hooks `MoreView::MoreView`; Extras hooks `Ui_MoreView::setupUi` and `StatusBarView::StatusBarView` (all three have a PLT slot in the five target builds).
- Unknown model or firmware, or a missing libnickel symbol: the affected feature is disabled, logged, and one toast says so. Nothing stays open on `/mnt/onboard`.
- Texts: `locales/<lang>.json`, chosen by `CurrentLocale` (exact locale, language, `en`), with a compiled English fallback in the library.

## Device folder

| Path | What |
|---|---|
| `bin/extras-tweaks` | the Go tweaks program |
| `locales/en.json`, `vi.json` | all texts of the mod and the program |
| `icons/screenshot.png`, `tweaks.png` | Tabler glyphs made by `cmd/iconsgen` (MIT notice in `LICENSES/tabler.txt`) |
| `screenshot-off`, `disable`, `debug`, `uninstall` | flag files, see [../README.md](../README.md) |
| `debug.log` | written only while `debug` exists |

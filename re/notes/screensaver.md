# Sleep-screen picture folder (libnickel 4.38.23828)

Static analysis with Ghidra headless (`re/ghidra-scripts/run-headless.sh 4.38.23828 -postScript DecompileByName.java ...`). The decompiled output stays in `~/.cache/nickeldrawer/decomp/ss-power.c` and `ss-settings.c`. Below are our own notes only.

## Call chain

- `N3PowerWorkflowManager::showSleepView` creates `PowerViewController(true)` and calls `show(controller)`.
- `N3PowerWorkflowManager::show(AbstractController*)`: if `PowerSettings::getShowCover()` (`ShowBookCover`, `[PowerOptions]`) is true, it calls `getLatestVolume()` (the first entry of `RecentOpenContentDataProvider` over `AllVolumesLibraryDataProvider`) and then `PowerViewController::setVolume`. Otherwise the volume stays empty.
- `PowerViewController::loadView` (0x00f5a96c):
  - empty volume (it matches a static default `Volume`, by d-pointer or id): builds `IconPowerView`, the stock logo screen with the retailer (FNAC) branding. It does not call `updateCover`, so **the folder is never read**;
  - otherwise: sets `this+0x1d = 1`. If `PowerSettings::getFullscreenCoverStretch()` (`FullscreenCoverStretch`, default false) is true and `!Volume::isInstapaper()`, it sets `this+0x1e = 1` and builds `FullScreenDragonPowerView` with the info panel set from `getFullscreenCoverInfoPanel()` (default true). Otherwise it sets `this+0x1e = 0` and builds `FramedDragonPowerView`. Then it calls `updateCover()`.
- The constructor at 0x00f58c78 zeroes `+0x1d` and `+0x1e`; `+0x1c` is the constructor's bool argument.
- `loadView` is the only writer of `+0x1e`.

## updateCover (0x00f5a0a4)

- `QDir("/mnt/onboard/.kobo/screensaver")`. The folder is used only if `exists()` and `this[0x1e]` are both true.
- Then: `setInfoPanelVisible(false)`; builds `"*.%1"` filters from `QImageReader::supportedImageFormats()`; calls `entryList(filters, default filter, default sort)`; then a loop of at most 5 tries: `qrand() % count`, `QImageReader(absoluteFilePath).read()`, until the image is non-null.
- Fallback when the image is null: a non-external book uses `getCoverImage("N3_FULL")`, then `getCoverImage("N3_LIBRARY_FULL")` (not for Instapaper); an external (SD) book uses `getSdCoverPowerImage("N3_FULL")`.
- No name prefix, size limit or settings key is checked. The filters are lowercase and `QDir` is case-sensitive on Linux, so `.PNG` is likely skipped.

## Settings UI

`N3SettingsPowerController::loadView/saveSettings` maps `ShowBookCover`, `FullscreenCoverStretch` and `FullscreenCoverInfoPanel` to three checkboxes in `N3SettingsPowerView`. `setStretchCovers(b)` enables the info-panel checkbox only when stretch is on.

No `FeatureSettings`, `DeveloperSettings` or `ApplicationSettings` key, affiliate check or sleep-reason check exists on this path. `ApplicationSettings::sideloadedMode` in `showSleepView` only gates usage-data updates.

## Why the device showed the FNAC logo

`ShowBookCover=false` → empty volume → `IconPowerView`. With `ShowBookCover=true` but `FullscreenCoverStretch=false` (the current conf) → framed cover with info, so the folder is also skipped. Both settings must be true.

## qrand seeding and the wake signal (random rotation)

Ghidra headless on 4.38.23828 (`DecompileByName`/`XrefsTo`; Ghidra addresses, image base 0x10000 above the ELF `st_value`):

- `qsrand` has two callers. `ApplicationBase<QApplication>::ApplicationBase(int&, char**)` (0x00eca660, call at 0x00eca69c) seeds with `QDateTime::currentDateTimeUtc().toTime_t()` right after constructing `QApplication`, on the GUI thread, so `qrand` is seeded once per boot (`Nickel3Application` derives from it). The other caller, `Sparling::Sparling` (0x00b14f80, via `startSparlingDevice`), is a test device path. `PowerViewController::updateCover` (call at 0x00f5a2ac) is the only sleep-screen `qrand` user.
- So Nickel's pick is not a fixed sequence, but each sleep draws independently: with few pictures a repeat is frequent (1 in 2 with two). The app now publishes one picture and chooses it itself.
- `N3PowerWorkflowManager::resumedFromSleep()` is a signal (`QMetaObject::activate` index 2, 0x00f54d68). It is emitted by `pinCodeAccepted` (0x00f554c8) right after `hideSleepView()`, unless `handleResume` started the front-light fade (`this+0xc` set), in which case the fade's functor slot (0x00f54e74) emits it and clears the flag. `resumeFromSleep` (0x00f56bb4) calls `pinCodeAccepted` when no PIN dialog is needed. Nickel's own `Nickel3Application::finishInitialization`, `LibrarySyncProxy`, `BedtimeManager` and `AudiobookPlayer` connect to it.
- `resumingFromSleep()` (index 1, 0x00f54d48) fires at the start of `handleResume`, while the sleep view is still up; not used.

ELF addresses (`nm -D`) of `_ZN22N3PowerWorkflowManager16resumedFromSleepEv`: 23552 0x00f3d448, 23648 0x00f3d708, 23684 0x00f3dd88, 23697 0x00f3dd88, 23828 0x00f44d68. Exported on all five builds (`re/symbols/*/exports.txt`).

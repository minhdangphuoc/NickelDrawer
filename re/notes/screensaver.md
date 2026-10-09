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

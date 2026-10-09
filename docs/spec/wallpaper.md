# Wallpaper app (sleep screen)

The user puts PNG or JPEG pictures in `/mnt/onboard/Pictures`. **Wallpaper · Import pictures** converts them for the e-ink screen and places them where Nickel looks for sleep-screen images. Press the power button to see the result.

## Actions

| Action | Effect |
|---|---|
| Import pictures | converts every new picture (background job; Status shows the result), shows the newest one |
| Next picture | selects the following picture in library order (wraps) and applies it |
| Fill / Fit | toggles the mode, re-fits every picture whose original is still in a source folder, applies |
| Fixed / Random | toggles the rotation and applies |
| Turn off | removes our files from `.kobo/screensaver` (library in `data/images` stays) |
| Status | running job or last result, picture count, mode, rotation, files on the sleep screen |

CLI: `wallpaper import|apply|next|mode fill|fit|toggle|rotation fixed|random|toggle|off|status [--app-dir DIR] [--locale CODE] [--version-file FILE]` (flags before positional arguments).

## Safety: what the app reads and writes

Read (read-only, never modified or deleted):

| Path | How |
|---|---|
| `/mnt/onboard/Pictures/` | direct children only (sub-folders are not entered) |
| `/mnt/onboard/.kobo/version` | the device id, to pick the screen size |
| `/mnt/onboard/.kobo/Kobo/Kobo eReader.conf` | `CurrentLocale` |
| `/mnt/onboard/.adds/apps/wallpaper/` | the app's own files |

Write:

| Path | What |
|---|---|
| `/mnt/onboard/.kobo/screensaver/nd-wallpaper-NNNN.png` | our converted pictures only (temp files `.nd-wallpaper-*.tmp` while copying) |
| `/mnt/onboard/.adds/apps/wallpaper/data/` | `config.json`, `images/` (converted PNGs + `index.json`), `log.txt` |
| `/mnt/onboard/Pictures/` | created if missing (empty folder only) |

Rules, enforced in code (`internal/storage`, `internal/wallpaper`) and covered by tests:

- The source list is a hardcoded allowlist built from the `internal/storage` constants. No config value, flag or environment variable can add a path.
- Only direct children are read: no recursion, names starting with `.` or `_` are skipped, sub-folders are ignored.
- Every entry is resolved with `filepath.EvalSymlinks`; it must be a direct child of the real source folder. A link that points elsewhere (another folder, `/etc/...`, a file in a sub-folder) is refused and logged.
- Only regular files (no devices, pipes, sockets).
- PNG/JPEG is decided by magic bytes, not by name. Other files are ignored silently (the Pictures folder may hold anything). Pictures over 20 MB are skipped with a logged reason; decoded size is capped at 48 megapixels.
- Pictures with identical content (SHA-256) are imported once.
- Nothing is held open on `/mnt/onboard`: each file is read whole (capped) and closed; all writes are temp file + fsync + rename.
- In `.kobo/screensaver` only names starting with `nd-wallpaper-` (or `.nd-wallpaper-`) are ever replaced or deleted, regular files only (symlinks are never followed or removed). Other files there are never read, changed or deleted.
- Nothing is read under `/`, `/etc`, `/usr`, other `.kobo` or `.adds` folders.

## Conversion

1. Decode PNG/JPEG with the standard library (EXIF orientation is not applied: the picture is used as stored). Alpha is flattened onto white.
2. Luma to an 8-bit gray plane; scale to the device resolution: area average when shrinking, bilinear when enlarging.
3. `fill` (default): cover the screen, center-crop the overflow. `fit`: whole picture centered on white.
4. Floyd-Steinberg dithering (serpentine) to 16 gray levels (multiples of 17), saved as a gray PNG.

Resolution table keyed by the last three digits of the device id (`internal/wallpaper/resolution.go`): `388` Libra 2 = 1264 x 1680. An unknown model is refused with a message; add a model only after checking its resolution.

## State

- `data/images/NNNN.png` converted pictures, `data/images/index.json` (`id`, `hash` of the original, `mode` the file was fitted for, original `name`).
- `data/config.json`: `{"mode": "fill|fit", "rotation": "fixed|random", "current": <id>}`.
- Fixed rotation puts exactly the current picture into `.kobo/screensaver`; random puts all of them (Nickel chooses).
- Import runs detached because converting a large photo on the Kobo can exceed NickelMenu's 10 s limit; output goes to `/tmp/wallpaper-last.txt`.

## What Nickel does (libnickel 4.38.23828, Ghidra headless)

Details and addresses: `re/notes/screensaver.md`.

**The folder is read only on the full-screen book-cover sleep screen.** It is not a separate "custom screensaver" feature, and no hidden key turns it on. All three must hold:

1. **Show current read** is on (`[PowerOptions] ShowBookCover=true`). `N3PowerWorkflowManager::show` only hands the most recently opened book (`getLatestVolume`) to `PowerViewController::setVolume` when `PowerSettings::getShowCover` is true.
2. **A recently opened book exists** and it is not an Instapaper article. With no book, `PowerViewController::loadView` builds an `IconPowerView`: the stock sleep screen with the retailer logo (FNAC) and "Sleeping". The folder is never read on that path.
3. **Full-screen covers** is on (`[PowerOptions] FullscreenCoverStretch=true`, default false; the checkbox under Show current read in Settings → Energy saving and privacy). `loadView` sets the flag at `this+0x1e` to true only when this setting is on and the book is not Instapaper, and then builds a `FullScreenDragonPowerView`. When the setting is off it builds a `FramedDragonPowerView` (cover with info) and clears the flag.

When the flag is set, `PowerViewController::updateCover`:

- opens `/mnt/onboard/.kobo/screensaver`; if the folder is missing, it shows the book cover;
- hides the info panel (`FullScreenDragonPowerView::setInfoPanelVisible(false)`), which overrides `FullscreenCoverInfoPanel`;
- lists `*.<fmt>` for every format in `QImageReader::supportedImageFormats()` (lowercase; case-sensitive on Linux; hidden files skipped). Any name works, no prefix needed, no size check;
- picks `qrand() % count`; up to 5 random tries while the decoded image is null;
- if no image loads, falls back to the book cover (`VolumePixmapView::getCoverImage` "N3_FULL", then "N3_LIBRARY_FULL"; SD books use `getSdCoverPowerImage`).

The retailer (FNAC) logo never overrides the folder: it only appears on the no-book path. The sleep reason (power button, timer or SleepCover) makes no difference: every path goes through `N3PowerWorkflowManager::show`. The power-off screen uses the same controller, so the same rules apply there.

Other notes:

- The regex `(\.kobo|koboExtStorage)/screensaver` in `SyncFileSystemCommand::findFilesToParse` appears to keep the folder out of the book import (verify on device).
- `FullScreenDragonPowerView` scales the image when stretching. A 1264 x 1680 picture matches the Libra 2 panel, so the scaling does nothing visible.

### Recipe to show our picture

1. Settings → Energy saving and privacy: tick **Show current read** and the **full-screen cover** checkbox under it (writes `ShowBookCover=true` and `FullscreenCoverStretch=true`).
2. Open any book once, so a recent book exists.
3. Keep `.kobo/screensaver` in place (NickelMenu Screensaver toggle on), with at least one `nd-wallpaper-*.png` in it.
4. Sleep the device.

No hidden key is needed. Editing the conf by hand is possible, but only with Nickel stopped (Nickel rewrites the conf), so the Settings UI is the safe route.

## Verify on device

- The recipe above shows our picture and no info panel (the panel stays hidden even with `FullscreenCoverInfoPanel=true`).
- With Full-screen covers off, the cover with info is shown, not our picture (expected).
- That a 1264 x 1680 gray PNG is shown without extra scaling or margins, and that the dither looks right after the sleep-screen update.
- That the pictures do not appear in the library after a rescan.
- That Nickel does not delete or rewrite files in `.kobo/screensaver`.
- The NickelMenu "Screensaver" toggle that moves `.kobo/screensaver` away and back must be ON for pictures to show; Turn off here does not rename the folder.

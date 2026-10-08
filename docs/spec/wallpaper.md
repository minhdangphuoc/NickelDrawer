# Wallpaper app (sleep screen)

The user puts PNG or JPEG pictures in `/mnt/onboard/Pictures/Wallpapers` (or `/mnt/onboard/Pictures`). **Wallpaper · Import pictures** converts them for the e-ink screen and places them where Nickel looks for sleep-screen images. Press the power button to see the result.

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
| `/mnt/onboard/Pictures/Wallpapers/` | direct children only |
| `/mnt/onboard/Pictures/` | direct children only (sub-folders, including `Wallpapers`, are not entered) |
| `/mnt/onboard/.kobo/version` | the device id, to pick the screen size |
| `/mnt/onboard/.kobo/Kobo/Kobo eReader.conf` | `CurrentLocale` |
| `/mnt/onboard/.adds/apps/wallpaper/` | the app's own files |

Write:

| Path | What |
|---|---|
| `/mnt/onboard/.kobo/screensaver/nd-wallpaper-NNNN.png` | our converted pictures only (temp files `.nd-wallpaper-*.tmp` while copying) |
| `/mnt/onboard/.adds/apps/wallpaper/data/` | `config.json`, `images/` (converted PNGs + `index.json`), `log.txt` |
| `/mnt/onboard/Pictures/Wallpapers/` | created if missing (empty folder only) |

Rules, enforced in code (`internal/storage`, `internal/wallpaper`) and covered by tests:

- The source list is a hardcoded allowlist built from the `internal/storage` constants. No config value, flag or environment variable can add a path.
- Only direct children are read: no recursion, names starting with `.` or `_` are skipped, sub-folders are ignored.
- Every entry is resolved with `filepath.EvalSymlinks`; it must be a direct child of the real source folder. A link that points elsewhere (another folder, `/etc/...`, a file in a sub-folder) is refused and logged.
- Only regular files (no devices, pipes, sockets).
- PNG/JPEG is decided by magic bytes, not by name. Other files are ignored silently (the Pictures folder may hold anything). Pictures over 20 MB are skipped with a logged reason; decoded size is capped at 48 megapixels.
- Pictures with identical content (SHA-256) are imported once, even when they sit in both folders; `Wallpapers/` is read first.
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

- The string `/mnt/onboard/.kobo/screensaver` (30 chars) is used by `PowerViewController::updateCover`. Logic seen in the decompilation:
  1. open `QDir` on that folder; if it exists (and an internal flag at `this+0x1e` is set) the info panel of the full-screen power view is hidden;
  2. build name filters `*.<ext>` from `QImageReader::supportedImageFormats()` (so every format Qt can read is listed, not only PNG/JPEG);
  3. `QDir::entryList` with those filters, then pick a random entry (`qrand() % count`) and read it with `QImageReader::read()`; up to 5 attempts while the image is null (a broken file is retried with another random pick);
  4. if no image was loaded, fall back to the cover of the current book (`VolumePixmapView::getCoverImage`) or the SD-card cover.
  So the folder has priority over the book cover whenever step 1 holds, selection is random (no ordering by name), and several files are chosen from at random.
- The regex `(\.kobo|koboExtStorage)/screensaver` is used in `SyncFileSystemCommand::findFilesToParse`, the file scan; it appears to keep the screensaver folder out of book import, so our pictures do not become library entries (verify on device).
- `PowerSettings::getShowCover` is read by `N3PowerWorkflowManager::show` (decides whether the latest book is passed to the power view) and by `N3SettingsPowerController::loadView` (the settings toggle for showing the current book cover).

## Verify on device

- Whether a Settings toggle (sleep screen / "show current book cover") must be on or off for the folder to be used, and what the flag at `this+0x1e` means.
- Whether a book cover ever wins over a picture in the folder.
- That a 1264 x 1680 gray PNG is shown without extra scaling or margins, and that the dither looks right after the sleep-screen update.
- Case sensitivity of the `*.png` filter (we write lowercase `.png`).
- That the pictures do not appear in the library after a rescan.
- That Nickel does not delete or rewrite files in `.kobo/screensaver`.
- The NickelMenu "Screensaver" toggle that moves `.kobo/screensaver` away and back must be ON for pictures to show; Turn off here does not rename the folder.

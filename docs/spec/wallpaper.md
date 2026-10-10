# Wallpaper app (sleep screen)

The user puts PNG or JPEG pictures in `/mnt/onboard/Pictures`. **Wallpaper · Import pictures** converts them for the e-ink screen and places them where Nickel looks for sleep-screen images. Press the power button to see the result.

## Actions

The screens (gallery, viewer, options) are specified in [wallpaper-gallery.md](wallpaper-gallery.md). `app.json` declares 9 actions:

| Action | Effect |
|---|---|
| `import` | converts every new picture (background job with a progress file), writes thumbnails, shows the newest one; pictures the user removed (`data/removed.json`) are skipped |
| `mode` | fill or fit (value `fill`/`fit`; `toggle` from a shell). The mode is stored at once; the pictures are re-fitted in the background (each re-fitted picture gets its new thumbnail at once). While a job runs only the mode is stored and that job re-fits again when it ends |
| `rotation` | fixed or random, applied at once (foreground); see "Random rotation" |
| `off` | removes our files from `.kobo/screensaver` (library in `data/images` stays) |
| `status` | running job or last result, picture count, mode, rotation, files on the sleep screen |
| `state` | the screen variables as one JSON line (below); starts `thumbnails` detached when thumbnails are missing |
| `set-current` | value = picture id; makes it the wallpaper and applies; prints `{"toast":"set"}` |
| `remove` | value = picture id; removes it from Wallpaper (passes `--confirm`, never `--delete-original`); prints `{"toast":"removed"}` |
| `remove-all` | value = picture id; the long-press menu item "Remove with original..." (own confirm dialog): passes `--confirm --delete-original --signal` and prints `{"toast":"removed_original"}` (original deleted), `{"toast":"removed_original_kept"}` (no identical original found, nothing else deleted) |

Inside the drawer (`NICKELDRAWER=1`) scripts never print prose: background jobs start detached and return at once with a declared toast (`import_started`, `mode_started`, `busy`), errors give the toast `failed` and the reason goes to `data/log.txt`. `next` exists only as a CLI command (the NickelDrawer mod runs it after a wake, see "Random rotation"). `thumbnails` is CLI-only too: the background job `state` starts detached when thumbnails are missing; it makes them one by one, runs once at a time (`--thumbs-pid-file`, default `/tmp/wallpaper-thumbs.pid`) and stops when the import job in `--pid-file` runs.

**Requirement.** Pictures show only with a current book and both Kobo settings on: Settings → Energy saving and privacy → "Show current read" and "Show book cover full screen" (see "What Nickel does"). The app reads `[PowerOptions] ShowBookCover` and `FullscreenCoverStretch` from `Kobo eReader.conf` (read only, never written) and, when either is not true or the file is unreadable, adds a hint line to Status, the import result and the screen (state fields `sleep_screen_ready`, `sleep_screen_hint`; the notice is hidden when the hint is empty). It cannot check for a current book.

CLI: `wallpaper import|apply|next|mode fill|fit|toggle|rotation fixed|random|toggle|off|status|state [--app-dir DIR] [--locale CODE] [--version-file FILE] [--progress-file FILE]` (flags before positional arguments), plus:

- `wallpaper set-current <id>`: ids are `0003` or `3`.
- `wallpaper remove [--confirm] [--delete-original] [--signal] <id>` (`--signal` prints the drawer toast id of the outcome instead of a sentence): without `--confirm` nothing happens and the error says so. Removes our converted copy, its thumbnail, its index entry and our `nd-wallpaper-*` file; records the original's SHA-256 in `data/removed.json`; if it was current, the next picture (else the previous) becomes current. `--delete-original` also deletes the original in Pictures, only when it is a regular direct child (after `EvalSymlinks`, the name itself not a link) whose SHA-256 equals the entry's hash; otherwise it is kept and the message says so.
- `wallpaper mode --save-only <mode>`: only stores the mode (used while a job runs).

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
| `/mnt/onboard/.adds/apps/wallpaper/data/` | `config.json`, `images/` (converted PNGs + `index.json`), `thumbs/` (300 x 400 previews), `removed.json` (hashes of removed pictures), `rotation-history.json`, `rotate-on-wake` (empty flag), `log.txt` |
| `/tmp/wallpaper-progress.json`, `/tmp/wallpaper.pid`, `/tmp/wallpaper-last.txt` | job progress, pid, last output; temp storage, not on the user partition |
| `/mnt/onboard/Pictures/<name>` | deleted only by `remove --delete-original` (never sent by the screens), under the rule below |
| `/mnt/onboard/Pictures/` | created if missing (empty folder only) |

Rules, enforced in code (`internal/storage`, `internal/wallpaper`) and covered by tests:

- The source list is a hardcoded allowlist built from the `internal/storage` constants. No config value, flag or environment variable can add a path.
- Only direct children are read: no recursion, names starting with `.` or `_` are skipped, sub-folders are ignored.
- Every entry is resolved with `filepath.EvalSymlinks`; it must be a direct child of the real source folder. A link that points elsewhere (another folder, `/etc/...`, a file in a sub-folder) is refused and logged.
- Only regular files (no devices, pipes, sockets).
- PNG/JPEG is decided by magic bytes, not by name. Other files are ignored silently (the Pictures folder may hold anything). Pictures over 20 MB are skipped with a logged reason; decoded size is capped at 48 megapixels.
- Pictures with identical content (SHA-256) are imported once; a hash listed in `data/removed.json` is skipped.
- `remove --delete-original` deletes an original only if it is found by `storage.ListImages` in a source folder (direct child, links resolved), its path equals `<real folder>/<name>` (a link is refused), it is a regular file, its name equals the entry's `name` and its SHA-256 equals the entry's `hash`. Nothing else in Pictures is ever deleted.
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

- `data/images/NNNN.png` converted pictures, `data/images/index.json` (`id`, `hash` of the original, `mode` the file was fitted for, original `name`; newer entries also `width`, `height` of the original and `added`, RFC 3339 UTC).
- `data/thumbs/NNNN.png`: written at import (fit inside 300 x 400, 16 grey levels, atomic). A mode re-fit writes the new one too. `state` never makes one: it starts the detached `thumbnails` job (below) when some are missing.
- `data/removed.json`: JSON array of SHA-256 hashes of removed pictures.
- `data/config.json`: `{"mode": "fill|fit", "rotation": "fixed|random", "current": <id>}`. Every writer changes one field on a fresh copy, so a background job never overwrites a later choice. `current` is always the picture on the sleep screen, so the gallery's ✓ badge follows a random change.
- `data/rotation-history.json`: the last 2 pictures shown in random rotation, most recent first.
- `data/rotate-on-wake`: an empty file that exists exactly while rotation is random (every config write and `state` keep it in step).
- Progress file (`--progress-file`, default `/tmp/wallpaper-progress.json`): `{"done": 3, "total": 7, "name": "bien-xanh.jpg"}`, plus `"kind": "refit"` for a re-fit; written atomically after each picture, removed at the end. `state` believes it while the pid is alive or the file is under 2 minutes old.
- Both rotations put exactly the current picture into `.kobo/screensaver`. With random, we choose, not Nickel (see "Random rotation").
- Import runs detached because converting a large photo on the Kobo can take longer than a drawer script may; output goes to `/tmp/wallpaper-last.txt`.

After an import that added pictures the app queues one quick library rescan; `state` hands it over once as `"requests":[{"request":"rescan","mode":"quick"}]` when no job is running (`app.json`: `"requests": ["rescan"]`).

`state` prints one JSON line with the keys `status`, `summary`, `pictures`, `current_id`, `count`, `mode`, `rotation`, `busy`, `pending`, `step_label`, `step_done`, `step_total`, `step_detail`, `sleep_screen_ready`, `sleep_screen_hint`, `sleep_screen_line`, `last_import`. A picture is `{id, thumb, image, label, info, badge}`: absolute paths (`thumb` omitted while missing), `label` = file name without extension, `info` = name, original size and date added (`dd/MM/yyyy`) on separate lines, `badge` = `current` only for the current picture. It reads files only and decodes no picture, so it stays far under the drawer's 1 s state budget, also while a job runs.

## Random rotation

Random used to put every picture into `.kobo/screensaver` and let Nickel pick (`qrand() % count`, below). On the device the same picture kept coming back: Nickel's pick is independent each time, so with few pictures a repeat is likely (1 in 2 with two pictures), and nothing stops a run of repeats. Now we choose:

- `wallpaper next` in random rotation picks uniformly (`crypto/rand`) among the pictures that are neither the current one nor the one shown before it (`data/rotation-history.json`). With 2 pictures it alternates; with 1 it keeps it. It makes the pick current, publishes only that picture (identical bytes are not rewritten) and removes our previous file. In fixed rotation `next` takes the following picture in library order, as before.
- The NickelDrawer mod connects to Nickel's `N3PowerWorkflowManager::resumedFromSleep()` signal. After each wake, when `data/rotate-on-wake` exists (a stat, nothing is opened), it starts `bin/wallpaper next --app-dir <app>` detached (never waited for), at most once per 30 s, and logs it. Rotating after the wake, not before sleep, never races Nickel reading the folder while it builds the sleep screen; the new picture shows at the next sleep.
- Without the mod (NickelMenu only, or a firmware where the signal is missing) random keeps showing the current picture until `wallpaper next` runs.

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
- picks `qrand() % count`; up to 5 random tries while the decoded image is null (`qrand` is seeded once per boot: `ApplicationBase<QApplication>` calls `qsrand(QDateTime::currentDateTimeUtc().toTime_t())`);
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

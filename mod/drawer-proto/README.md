# nddrawer: gate-G2 on-device prototypes

One NickelHook mod, `libnddrawer.so`, with two prototypes that the user switches on one at a time
with flag files. It proves the two G2 assumptions that static analysis could not
(`re/notes/G2-findings.md`): an extra row in the More tab, and a pushed drawer page that keeps
the tab bar and receives the page-turn keys.

| Prototype | Flag file (on the Kobo) | What it does |
|---|---|---|
| A | `.adds/nickeldrawer/proto-a` | Hooks `MoreView::MoreView(QWidget*)`, adds one `IconLeftButton` row "Apps" (or "Ứng dụng" when Nickel's language is Vietnamese) right after "Beta Features". Tap: opens prototype C if enabled, else toast "NickelDrawer: row tapped". |
| C | `.adds/nickeldrawer/proto-c` | Empty drawer page: back chevron + title, 3 placeholder rows, page indicator "1 / 2". Page-turn keys flip between pages 1 and 2 (logged, full e-ink refresh). Back returns to More. Opened only from the prototype A row, so it needs `proto-a` too. |
| C, route B | `.adds/nickeldrawer/proto-c-pushview` | Optional. Opens C with `pushView`/`popView` instead of the controller push (use only if route A misbehaves). |

Flags are read once at boot: **reboot after adding or removing one.** No flag = the mod is
installed but does nothing except one log line.

Routes for C (details in `re/notes/G2-findings.md`, "Prototype build"):

- **Route A (default)**: a mod-side `AbstractController` subclass with `navSection() = 6`, opened with
  `MainWindowController::push` and closed with `pop`. Nickel treats it like its own More sub-pages
  (Wishlist, Dropbox): tab bar visible, "More" active; tab taps and Home close it.
- **Route B (fallback)**: the bare widget is pushed with `pushView`; the mod pops it itself when
  any other view becomes current.

Safety behaviour: every libnickel symbol is optional and checked before use; a missing one disables
only the prototype that needs it (logged, never a crash). Unknown firmware (not one of the five
4.38 builds) disables both. `init()` only reads `/mnt/onboard/.kobo/version` (closed right away)
and checks the flag files. Failsafe: a crash in the first 15 s of boot removes the mod.

## Build and static checks (Mac, nothing touches the Kobo)

```sh
make -C mod/drawer-proto docker-build      # build/mod/libnddrawer.so, build/mod/KoboRoot-nddrawer.tgz
mod/drawer-proto/check-symbols.sh          # symbols in all 5 builds, ARM hard-float, NEEDED, no STL
```

## On-device test script (with the user present)

### Preconditions

1. Gate G1 passed with `ndhello`: toast seen, uninstall drill and failsafe drill done.
   **`ndhello` must be uninstalled** (the G1 uninstall drill does this). Both mods use the same
   uninstall flag and the first mod that sees the flag deletes it, so with both installed one
   uninstall run removes only one of them.
2. Full backup on the Mac in `backups/<date>/`: `.kobo/KoboReader.sqlite`, `.kobo/Kobo/Kobo eReader.conf`,
   the whole `.adds/` folder.
3. Battery above 50 %. NickelMenu installed and working (it is the log viewer and the fallback).
4. Shell rule: use absolute paths and never `cd` into `/Volumes/KOBOeReader` (it blocks eject).

### Step 0: add the log items to NickelMenu (once)

Copy this file to `/Volumes/KOBOeReader/.adds/nm/nddrawer-log` (NickelMenu reads every file in
`.adds/nm/`). It is not installed by the build.

```
# NickelDrawer prototype logs (remove after gate G2)
menu_item:main:NickelDrawer log:cmd_output:5000:logread | grep nddrawer | tail -20
menu_item:main:NickelDrawer Nickel warnings:cmd_output:5000:logread | grep -E "MainWindowController|AbstractController|controller" | tail -20
menu_item:main:NickelDrawer save log:cmd_spawn:quiet:logread > /mnt/onboard/.adds/nickeldrawer/proto-log.txt
```

"save log" writes the whole syslog to `.adds/nickeldrawer/proto-log.txt`; copy it to the Mac over
USB afterwards (into `backups/<date>/`, it can contain personal data; do not commit it).
NickelHook's own lines also carry the `(nddrawer)` prefix, so "NickelDrawer log" shows them too.
A NickelHook failure also writes `.kobo/nddrawer_<date>.log` by itself.

### Step 1: install with prototype A only

1. Mount the Kobo. Create the folder and the flag:
   `mkdir -p /Volumes/KOBOeReader/.adds/nickeldrawer && touch /Volumes/KOBOeReader/.adds/nickeldrawer/proto-a`.
   Make sure `proto-c`, `proto-c-pushview` and `uninstall` do **not** exist there.
2. `cp build/mod/KoboRoot-nddrawer.tgz /Volumes/KOBOeReader/.kobo/KoboRoot.tgz`, eject, unplug.
   The Kobo installs it and reboots.
3. Wait on the home screen for at least 20 s (failsafe window), then open "NickelDrawer log".

Expected:
- Log: `(nddrawer) loaded on 4.38.<build>: prototype A on, prototype C off (route A controller)`.
- Open the **More** tab: a new row "Apps" (or "Ứng dụng") with the Beta Features icon, directly
  below "Beta Features" (or at the end of the list if that row is hidden on your device), styled like
  the other rows. Log: `more row: added at index N`.
- Tap the row: toast "NickelDrawer: row tapped". Log: `more row: tapped`.
- The other More rows (Settings, Help, ...) still open normally. Switch to Home and back to More
  twice: the row is there each time, only once.
- Optional: switch Nickel's language to Tiếng Việt, open More again: the row reads "Ứng dụng".
- Plug in USB and eject again: library still works, no crash.

Report: a photo of the More tab, the "NickelDrawer log" output, anything odd in spacing or font.

### Step 2: prototype C, route A (controller push)

1. Mount, `touch /Volumes/KOBOeReader/.adds/nickeldrawer/proto-c` (keep `proto-a`), eject, **reboot**
   (hold power → Restart, or Settings → Device information → Restart if available).
2. Wait 20 s, open More, tap "Apps".

Expected:
- A page with a back chevron "‹" and the title "Apps"/"Ứng dụng", three rows "Placeholder 1..3",
  and "1 / 2" at the bottom. **The bottom tab bar stays visible with More highlighted.**
  Log: `drawer: route A: push controller`, `drawer controller: loadView`, `viewWillAppear`,
  `drawer: route A: open`.
- Press the page-forward button: rows become "Placeholder 4..6", indicator "2 / 2", the screen
  flashes once. Log: `drawer: key 0x1000014 (next) page 1 -> 2` (or 0x1000015). Press it again:
  nothing changes, log says `(at edge)`. Page-back button: back to page 1.
  If Settings → Reading → "swap page-turn buttons" is on, the direction swaps too (Nickel applies it).
- Tap "‹": back to the More tab. Log: `drawer: back tapped`, `viewWillBeDestroyed`,
  `drawer controller destroyed`.
- Open the drawer again and tap the **Home** tab: Home opens; tap More: the normal More list, no
  leftover drawer. Log shows `drawer controller destroyed` once per close.
- Open the drawer, put the Kobo to sleep with the power button and wake it: drawer still shown, keys
  still work.
- Tap "Apps" twice quickly: only one drawer opens (log: `already open, ignoring tap`).

If Nickel freezes or restarts when you tap "Apps": wait for it to come back (it restarts by itself),
mount USB, delete `.adds/nickeldrawer/proto-c`, eject, reboot, and send the logs. Then try step 3.

### Step 3 (only if step 2 failed): prototype C, route B (pushView)

1. Mount, `touch /Volumes/KOBOeReader/.adds/nickeldrawer/proto-c-pushview` (keep `proto-a` and
   `proto-c`), eject, reboot. Repeat step 2.

Differences to expect: the tab bar keeps whatever state the More tab had (it is not re-evaluated);
tapping Home or another tab closes the drawer through the watcher
(log: `another view became current ... closing drawer`).

### Disable a prototype

Delete its flag file (`proto-a`, `proto-c`, `proto-c-pushview` in `.adds/nickeldrawer/`) and reboot.
Without `proto-a` the More tab is stock again, even with `proto-c` present.

### Uninstall

In order, stop at the first that works:

1. `touch /Volumes/KOBOeReader/.adds/nickeldrawer/uninstall-proto`, eject, reboot. The mod deletes
   `/usr/local/Kobo/imageformats/libnddrawer.so` and the flag. Check: "NickelDrawer log" shows no new
   `loaded on` line after the next reboot. Remove the flag files and `.adds/nm/nddrawer-log` too.
2. Power off within 15 s of boot (or let Nickel crash during that window); the failsafe removes the mod.
3. NickelMenu stays as the fallback and is not affected by any of the above.
4. Last resort: re-apply the stock firmware update (user's Kobo guide); restore
   `KoboReader.sqlite` from `backups/<date>/` only if the library was affected.

### What to send back after the session

- "NickelDrawer log" and "Nickel warnings" output (photo is fine) after each step, or the saved
  `proto-log.txt`.
- Photos: More tab with the row; drawer page 1 and 2; More after back.
- Which steps passed, and the exact action before any freeze or restart.

# ndhello: NickelDrawer hello-world mod

A minimal NickelHook mod (`libndhello.so`) used to prove the toolchain and the failsafe/uninstall
path before any real drawer code.

## What it does

- `init()` reads the firmware from `/mnt/onboard/.kobo/version`, logs `ndhello loaded on <firmware>`
  to syslog (prefix `(ndhello)`), and schedules one toast. Nothing else runs in `init()`.
- 8 s later, on the GUI thread (`QTimer::singleShot`), it shows the toast "NickelDrawer test mod loaded"
  via `MainWindowController::sharedInstance` + `MainWindowController::toast`
  (`_ZN20MainWindowController14sharedInstanceEv`, `_ZN20MainWindowController5toastERK7QStringS2_i`).
  Both symbols are optional: if either is missing it logs and does nothing.
- Failsafe: armed before `init()`, disarmed after 15 s. If Nickel crashes earlier, the mod removes itself on the next boot.
- Uninstall flag: `/mnt/onboard/.adds/nickeldrawer/uninstall`. If the file exists at boot, the mod deletes
  itself and the flag.
- Delete-to-uninstall (`uninstall_xflag`) is supported by NickelHook but deliberately unused: it
  uninstalls when the file is absent, which would remove the mod on the first boot.

## Build (does not install anything)

```sh
make -C mod/hello docker-build
```

Produces `build/mod/libndhello.so` and `build/mod/KoboRoot-ndhello.tgz`. See `mod/TOOLCHAIN.md`.

## First on-device test (gate G1; only with the user present)

Prerequisites: backup copied to `backups/<date>/` (including `.kobo/KoboReader.sqlite` and the
current `.kobo/KoboRoot.tgz` state if any), battery above 50 %, the uninstall drill below understood.

1. Mount the Kobo, copy `build/mod/KoboRoot-ndhello.tgz` to `/Volumes/KOBOeReader/.kobo/KoboRoot.tgz`
   (use absolute paths; never leave the shell's working directory on the volume), eject, and let it reboot.
2. Expect the toast "NickelDrawer test mod loaded" about 8 s after the home screen appears.
   After a USB connect/disconnect, the library must still work.
3. Uninstall drill: connect USB, create an empty file `.adds/nickeldrawer/uninstall`, eject, reboot.
   Expect no toast and `/usr/local/Kobo/imageformats/libndhello.so` gone (telnet or the next
   NickelHook log dump confirms; the flag file is consumed).
4. Failsafe drill (separate run, reinstall first): reboot, then power off within about 15 s of boot
   (or force a crash). On the next boot the mod must be gone and Nickel must start normally.

## Rollback

In order, stop at the first that works:

1. Create `.adds/nickeldrawer/uninstall` (above) and reboot.
2. Power off within 15 s of boot; the failsafe removes the mod on the next boot.
3. Delete `/usr/local/Kobo/imageformats/libndhello.so` via telnet/SSH if enabled.
4. Last resort: factory-safe recovery by re-applying the stock firmware update (user's Kobo guide);
   restore `KoboReader.sqlite` from `backups/<date>/` only if the library DB was affected.

# Prototype A crash: tapping the "Ứng dụng" row kills Nickel

Date: 2026-10-09. Firmware 4.38.23828, Kobo Libra 2. Analysis: static only (headless Ghidra on
`libnickel-4.38.23828`, decompiled output in `~/.cache/nickeldrawer/decomp/iconbutton.t.c`, not in the repo),
plus a dump of the built `.so` files.

## Symptom

- Flag `proto-a` only (no `proto-c`). The row appears in More under "Tính năng thử nghiệm" (Beta Features),
  looks right, More opens and works.
- Tapping the row crashes Nickel. Expected: toast "NickelDrawer: row tapped".
- No log: syslog is in RAM and lost with the restart.

## Root cause (confirmed statically)

**Our tap receiver was a child of the row, and its vtable had no typeinfo (`-fno-rtti`). Nickel's
`IconButtonBase::tapGesture` calls `__dynamic_cast` on every direct child of the button when the finger goes
down, and `__dynamic_cast` reads the typeinfo pointer from the object's vtable without checking it. Null typeinfo
means a null-pointer call: Nickel crashes on the press, before `tapped()` is emitted. Our slot and the toast never
ran.**

Evidence:

1. `IconButtonBase::tapGesture(GestureReceiver*, TapGesture*)` (Ghidra 0x10ef9f4):
   - state 1 (gesture started = finger down): renders the button into a pixmap, inverts it (the pressed look),
     then loops over `d_ptr->children`. For each child: `__dynamic_cast(child, &typeinfo(QObject),
     &typeinfo(QWidget), 0)`; for a widget, also `__dynamic_cast(child, QObject, ReversibleBase, -2)` and then
     either `ReversibleBase` vfunc +8 or `setUpdatesEnabled`. Then flash repaint (attribute 0x80).
   - states 3 (finished) and 4 (cancelled): the same loop to restore colours; state 3 then emits `tapped()`.
   - The typeinfo arguments were resolved from the literal pool + GOT: `R_ARM_GLOB_DAT typeinfo for QObject`,
     `typeinfo for QWidget`, `typeinfo for ReversibleBase`.
2. `__dynamic_cast` (libsupc++) takes `whole_type` from `vtable[-1]` and calls `whole_type->__do_dyncast(...)`
   through it. With `whole_type == NULL` that is a load from address 0: SIGSEGV.
3. The old `build/mod/libnddrawer.so`: `vtable for MoreRowTapReceiver` = `00000000 00000000 ...` (offset-to-top 0,
   **typeinfo 0**, and no relocation on that slot). The new check (below) on the old builds reported null typeinfo
   for `MoreRowTapReceiver`, `DrawerView`, `DrawerLauncher` (prototype) and 19 classes in `libnickeldrawer.so`,
   among them `EntryRow` and `UpdateDot`, both direct children of the drawer host's row.
4. Why `ndhello` worked: its `ToastOnce` had no typeinfo either, but nobody casts it. It has no parent and is never
   a child of a Nickel widget. Only `DrawerController` was built with `-frtti`, because we already knew that Nickel
   casts controllers.

The drawer host (`mod/drawer`) had the same bug: `EntryRow` (a QObject) and `UpdateDot` (a QWidget) are children
of the `IconLeftButton`. Its first tap would have crashed the same way.

## Ranked hypotheses

| # | Hypothesis | Verdict | Evidence |
|---|---|---|---|
| 1 | Null typeinfo in our child object's vtable; Nickel `__dynamic_cast`s button children on press | **Root cause** | Points 1–3 above; it explains why the row shows and only a tap crashes, and why it crashes even without our slot |
| 2 | Wrong signal (`tapped(bool)`, `clicked`, ...) | Ruled out | Only `IconButtonBase::tapped()` is exported (`_ZN14IconButtonBase6tappedEv`, no arguments). `MoreView::MoreView` connects all 9 built-in rows to that same signal (`connectImpl` with one pointer-to-member for every row) and re-emits its own `wishlist()`, `settings()`, ... signals. Our old-style `connect` returned true, otherwise the code would have removed the row |
| 3 | Toast call differs from `ndhello` | Ruled out as the cause | Same symbol, same `(window, QString, QStringLiteral(""), 3000)`. `MainWindowController::toast` only forwards to `N3ToasterController::queueMeSomeToast`, which copies the strings into a `QueuedToast` and shows it or queues it. The only difference was the context (inside the gesture handler, not on a timer), and the slot never ran anyway |
| 4 | Receiver lifetime / MoreView rebuilt / wrong thread | Unlikely | The receiver was a child of the row, so it was deleted with it. Gestures and slots both run on the GUI thread. The More view is not rebuilt while it handles a tap |
| 5 | Bad `IconLeftButton` construction (size, null pixmap used when painting the pressed state) | Unlikely | The constructor sets up every member (QPixmaps at +0x20/+0x30, Ui at +0x54) inside our 256-byte zeroed block; 0x58 is used. The pressed look draws the widget into a new pixmap. The row already painted normally. `setPixmap` is what Nickel's `setProperty("pixmap")` calls |

## Fix (both `mod/drawer-proto` and `mod/drawer`)

1. **`-frtti` for the whole mod** (Makefiles), not only `drawer_controller.cc`. Every vtable now has typeinfo,
   so Nickel can cast any object we give it. This also covers the drawer page (`DrawerView` and the other
   widgets in Nickel's window stack). C++ standard library imports: unchanged (only `__cxxabiv1` typeinfo
   vtables from the ABI runtime).
2. **New static check** `mod/common/check-vtable-typeinfo.sh` (run by both `check-symbols.sh`): fails if any
   vtable in the library has a 0 typeinfo slot and no relocation on it. Validation case: on the old builds it
   fails and lists the classes above; on the new builds it passes (proto: 6 vtables, drawer: 21).
3. **Receiver is no longer a child of the row** (`MoreRowTapReceiver`, `EntryRow`). It is deleted through
   `row destroyed() → deleteLater()`, and keeps a `QPointer` to the row (the drawer host also keeps one to its
   `UpdateDot`). The slots return early when the row is gone. This keeps Nickel's button child list close to
   what it expects. `UpdateDot` stays a child widget; it has typeinfo now.
4. **The tap is handled after Nickel's gesture handler returns**: `onTapped()` logs and starts a 0 ms
   `QTimer::singleShot` to `handleTap()`, the same event-loop context as `ndhello`'s toast. A second tap while
   one is pending is ignored. The toast call itself is unchanged (it already matched `ndhello`).
5. **Signal check on the live object**: `metaObject()->indexOfSignal("tapped()")` must be ≥ 0 before
   connecting; otherwise the row is removed and the reason logged.
6. **Debug log** `mod/common/debug_log.{h,cc}`, used by both mods. It is on only if
   `/mnt/onboard/.adds/nickeldrawer/debug` exists at boot. Each line is appended to
   `/mnt/onboard/.adds/nickeldrawer/debug.log` (open, append, fsync, close, so no file stays open and a
   line survives the crash that follows it) and also sent to `nh_log`. The file restarts at 64 KB.
   Lines logged: init; MoreView constructed; IconLeftButton memory, parent and className after construction; icon
   loaded and size; pixmap and text set; `tapped()` signal index; row, receiver and child count; tap received
   (receiver, sender, row alive); tap handled; before and after the toast (or opening the drawer); window pointer.

Build results (2026-10-09): `make -C mod/drawer-proto docker-build`, `make -C mod/drawer docker-build`: OK;
both `check-symbols.sh`: OK (10 and 25 symbols in 5/5 builds; vtables OK); `make -C mod/drawer docker-test`:
all tests passed. Nothing installed.

## On-device retest (user present; prototype only, `nickeldrawer` not installed)

1. Back up first (CONTRIBUTING.md): `backups/<date>/` ← `.kobo/KoboReader.sqlite`, `.kobo/Kobo/Kobo eReader.conf`,
   `.adds/`.
2. Mount the Kobo and use absolute paths only:
   ```sh
   mkdir -p /Volumes/KOBOeReader/.adds/nickeldrawer
   touch /Volumes/KOBOeReader/.adds/nickeldrawer/proto-a /Volumes/KOBOeReader/.adds/nickeldrawer/debug
   rm -f /Volumes/KOBOeReader/.adds/nickeldrawer/proto-c /Volumes/KOBOeReader/.adds/nickeldrawer/proto-c-pushview \
         /Volumes/KOBOeReader/.adds/nickeldrawer/debug.log
   cp build/mod/KoboRoot-nddrawer.tgz /Volumes/KOBOeReader/.kobo/KoboRoot.tgz
   ```
   Eject and unplug. The Kobo installs the update and reboots.
3. Wait at least 20 s on Home (failsafe window). Open **More**. Check that the row is there.
4. Tap the row once. Expected: toast "NickelDrawer: row tapped" and no crash. Tap it again, then switch Home → More
   twice and tap again.
5. Plug in USB, then copy the log off **into the backups folder** (it can contain paths; do not commit it):
   `cp /Volumes/KOBOeReader/.adds/nickeldrawer/debug.log backups/<date>/protoA-debug.log`. Eject.
6. Expected log tail for one tap:
   ```
   ... nddrawer: more row: tapped() received by 0x..., sender 0x..., row alive
   ... nddrawer: more row: handling tap, row alive
   ... nddrawer: more row: before toast
   ... nddrawer: toast: window 0x..., text "NickelDrawer: row tapped"
   ... nddrawer: more row: after toast
   ```
   If it still crashes, the last line in the log shows the step that died. If there is no `tapped() received` line
   after `row ... receiver ..., N children`, the crash is still inside Nickel's press handling. Send the log.
7. Afterwards remove `debug` (and `debug.log`) and reboot. The log costs one fsync per line on the FAT partition.

Recovery if it still crashes: Nickel restarts by itself. Then delete `proto-a` over USB and reboot, or use the
`uninstall-proto` flag (`mod/drawer-proto/README.md`).

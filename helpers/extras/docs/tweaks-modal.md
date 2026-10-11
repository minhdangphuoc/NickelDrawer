# Tweaks window: which Nickel window, and why (libnickel 4.38.23828)

> Part of the **Extras helper** ([../README.md](../README.md)), not of NickelDrawer. Not yet run on a device.

Date: 2026-10-10. Headless Ghidra 11.3.2, read-only project `libnickel-4.38.23828`; decompiled output only in
`~/.cache/nickeldrawer/decomp/` (`settings.c`, `n3dialog2.c`, `mwc-dialogs.c`, `statusbar.c`, `conf-keys*.c`).
Addresses are file addresses of 23828 (`nm -D`; Ghidra shows them + 0x10000). Every symbol below is exported by
all five target builds (`helpers/extras/check-symbols.sh`). Not yet run on a device.

## Kobo's own Settings window

- **Settings (top level)** is not a dialog: `MoreController::settings()` creates an `N3SettingsController` and
  `show()` pushes it (`MainWindowController::push`). Its view is `N3SettingsView` (a `N3SettingsPagedView`,
  `Ui_N3SettingsView`: `TouchLabel` rows in a `PagingContainer`, styled by `:/qss/N3SettingsView.qss`).
  `navSection()` is 0, so the tab bar is hidden.
- **Settings sub-pages** ("Reading settings", ...) are `N3SettingsControllerBase`/`N3DialogController`s.
  `N3DialogController::viewAsDialog()` wraps the controller's view with `N3DialogFactory::getDialog(view, true)`,
  sets the large title (`N3Dialog::setLargeTitle`), and `N3SettingsControllerBase::viewAsDialog()` sets the small
  title "Settings" (`N3Dialog::setTitle`). `SettingsMixin::demoModePasswordProtectedPush` pushes the controller.
- **`N3Dialog`** (`Ui_N3Dialog::setupUi`, 0x44 bytes): `#primaryLayoutContainer` (#4d4d4d) → `#mainFrame` (white,
  `border: 1px solid black`; none in `fullViewMode`) → top row: back container, `#title` (a
  `FontSizeAdjustingLabel`, centred, the small title), `#closeContainer` with an `N3BatteryStatusLabel` and the
  close `TouchLabel` (`:/images/search/global_close.png`, pressed `global_close_hit.png`); then
  `#largeTitleContainer` (`#titleLarge`, Georgia, word wrap; empty after the constructor) and
  `#contentLayoutWidget` where `setContent` puts the content (aligned top). `closeTapped()` is the close button.
- `N3DialogFactory::getDialog(content, bool)` (0xeb4708): `new N3Dialog(NULL, bool)` (the bool adds a
  `GoToPageMenuController` on the "pages" label), `setContent`, copies `previousViewTitle`, and connects
  `closeTapped()` to `MainWindowController::closeActiveN3Dialogs()`. That slot only pops while the top controller
  casts to Nickel's dialog-controller type; a plain controller of ours is left alone (our own slot pops it).

## Decision: Nickel's N3Dialog, pushed as a controller (Settings' recipe)

The modal is Nickel's `N3Dialog` (non-full-view: frame with a thin black border on Nickel's grey) with our
the tweaks panel as content, the small title "Tinh chỉnh", the close button top-right, the large title row hidden
(`setLargeTitleLayoutVisible(false)`). It is the view of a small controller class of ours with `navSection()` 0, pushed with
`MainWindowController::push`. So Nickel itself:

- hides the tab bar (`updateNavFooter`, section 0) and sets the status bar from the dialog's `showTitle`,
  `showIndicators`, `showStatusBottomLine` properties (`showStatusBarAsNecessary`); the mod sets all three false,
  so the bar is hidden as on Nickel's own N3Dialog pages (the dialog has its own battery icon). `pushView` alone
  would leave the bar over the dialog's top row and hide the close button under it;
- restores both for the More page on pop (re-prepares `MoreController`), and removes the page with the stack on
  USB connect or anything that clears the window stack;
- parents the controller to the dialog, so the dialog's deletion takes the controller with it.

Why not the alternatives: `N3SettingsView` is a fixed uic page (its rows are its own `TouchLabel`s, no API to add
ours); `ConfirmationDialog` is a question box (title, text, two buttons) with no content area; full-view mode
loses the modal frame the decision asked for.

**No fallback overlay.** When `N3DialogFactory::getDialog`, `N3Dialog::setTitle`, the controller push or the
`closeTapped()` signal is missing, the controller layout check fails, or the push did not put the dialog in the
window stack, the window is not offered: the More-tab row is not added (checked when the row is built) or, if it
fails later, one toast says Tweaks are unavailable. The screenshot button is unaffected. (An earlier design had a
full-screen frameless `QDialog` as a fallback; it was dropped to keep the mod small.)

### Close

- ✕: `N3Dialog::closeTapped()` → our slot (from Nickel's tap handler) → 0 ms timer → `pop(controller)` only if the
  dialog is still visible (Nickel's `popView` hides it first if it removed the page itself).
- Tap outside: the session filters the dialog's own touch events (`N3Dialog` sets `WA_AcceptTouchEvents` and
  accepts touches no child took). A touch that begins and ends outside `#mainFrame` closes. If Nickel's QSS gives
  the frame no margin there is no outside on the device: then ✕ is the only close (to be checked, see below).
- Page keys: `TactileButtonFilter` sends them to the current view (the dialog); the session's filter pages the
  panel when it has more than 7 rows (14 tweaks).

### Native actions (`"nickel"` of a `tweak` reply)

| Field | Nickel call (23828 file address) | Notes |
|---|---|---|
| `sleep` | `N3PowerWorkflowManager::sharedInstance()` (0xf466c0) + `requestSleep()` (0xf46b24) | NickelMenu's `power:sleep`; no-op while sleeping is not allowed (`sleepAllowed`) |
| `reboot` | `… ::reboot()` (0xf476ec) | NickelMenu's `power:reboot`; closes the DBs, on Libra 2 turns Wi-Fi/BT off and `sleep(3)` on the GUI thread, `sync()`, then the hardware reboot |
| `rescan-library` | `PlugWorkflowManager::sharedInstance()` + `sync()` (0xf3c3b4) | the quick rescan (NickelMenu `rescan_books`); the full one fakes a USB session and clears the window stack |

Sleep and reboot run 1.5 s after the reply from a self-deleting timer object, so the toast is painted first.
A missing symbol → toast "Not available on this firmware." instead.

Other symbols used: `N3DialogFactory::getDialog` (0xeb4708), `N3Dialog::setTitle` (0x10ee908),
`N3Dialog::setLargeTitleLayoutVisible` (0x10eed94), signal `N3Dialog::closeTapped()` (0x10ee6d4),
`MainWindowController::push` (0xeb1d1c) / `pop` (0xeb20b4), `toast` (0xeaf300), `ConfirmationDialogFactory::
getConfirmationDialog` (0x10d60e8, confirm before actions with `confirm`), `IconLeftButton` (the More row).

## Kobo eReader.conf keys behind the tweaks (for helpers/extras/internal/tweaks/settings.go)

Group = the `sectionName()` of Nickel's settings class (each class's keys are followed by its section string in
libnickel's string table; the device's own conf has these sections). Reads go through `Settings::getSetting`,
which builds a fresh `QSettings` on the file each time.

| Tweak | Group / key Nickel reads | Where Nickel reads it | Takes effect |
|---|---|---|---|
| screenshots (no longer offered by Extras; the status-bar button replaced it) | `[FeatureSettings] Screenshots` (bool, default false) | `PowerKeyFilter::filter`, on each power-key press | next press, no reboot |
| auto-usb | `[DeveloperSettings] AutoUsbGadget` (bool, `DevSettings::autoUsbGadget`) | `PlugWorkflowManager::doPluggedConnection`, each USB plug | next plug, no reboot |
| invert-reboot | `[FeatureSettings] InvertScreen` (bool, `FeatureSettings::invertScreen`) | `Nickel3Application` constructor (and the developer options view) | after reboot (as the tweak does) |
| legibility | **`[Reading] webkitTextRendering`** (string, default `auto`; `ReadingSettings::getWebkitTextRendering`) | `KepubBookReader::pageStyleCss`: `text-rendering: %3` in the page CSS | next kepub page style (reopen the book); no reboot; kepub only |

Correction: `webkitTextRendering` is in `[Reading]` (`ReadingSettings`), not `[ApplicationPreferences]`. Its
value goes verbatim into CSS `text-rendering`; the known legibility tweak is `optimizeLegibility` (kerning and
ligatures); `geometricPrecision` is valid CSS but not that tweak. Removing the key gives `auto`.

## Validation (next device session, `debug` flag on, user present, after the G1 drill)

1. More tab: "Tinh chỉnh" row in the More tab (below Beta features; with NickelDrawer also installed, each mod has its own row) with Nickel's Settings gear as its icon (re/notes/system-icons.md).
2. Tap it: Nickel's dialog: small centred "Tinh chỉnh", ✕ top-right, no status bar over it, tab bar hidden,
   our rule and the 2-column grid (italic serif labels, Bật/Tắt or the IP under them, thin dividers). The first cell pair is Sleep screen and Screenshot button ("Trên thanh trạng thái").
   Log: "bridge: settings dialog … (N3Dialog)", "tweaks: Nickel N3Dialog … pushed", "tweaks panel: 9 tweak(s)".
   If the large title row shows "[LargeTitle]" or the frame fills the page with no grey, note it.
3. Tap a toggle: only that cell inverts, then its value flips in place; Nickel's toast shows the text. The screenshot button toggle hides or shows the status-bar button as soon as the window is closed.
4. Tap "Khởi động lại": Nickel's confirmation; Cancel does nothing; OK → toast, reboot ~1.5 s later.
5. Tap outside the frame (if there is grey): closes. ✕: closes. Both: back on More with tab bar and status bar.
6. Sleep tweak: the sleep screen; wake: the modal is still there and works.
7. Rename `bin/extras-tweaks`: the window shows the localized "part of Extras is missing" text instead of the grid.
8. Copy `debug.log` to `backups/<date>/`.

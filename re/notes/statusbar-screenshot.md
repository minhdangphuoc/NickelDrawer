# Screenshot button in Nickel's status bar: feasibility and design

Date: 2026-10-11. Static analysis only, no device run. Sources: `re/symbols/<fw>/exports.txt` (all five builds),
headless Ghidra on the analysed 4.38.23828 project (`DecompileByName`, `XrefsTo`; output in
`~/.cache/nickeldrawer/decomp/ssb*.c`, not in the repo), `llvm-objdump` of libnickel (relocations, per-build
disassembly of `Ui_StatusBarView::setupUi` and `ScreenshotMaker::saveCurrentScreen`) and of `platforms/libkobo.so`,
the extracted QSS (`~/.cache/nickeldrawer/qss/`), and the rcc name tables in the `nickel` binary.

**Scope.** This is a personal helper for the owner's own Kobo, **not a NickelDrawer feature**. It is a separate
NickelHook mod with its own `.so`, its own KoboRoot, its own dlsym checks and failsafe. It lives tracked in the
repo under `helpers/statusbar-screenshot/`. It stays out of the main README, `docs/guide`, CHANGELOG feature lists,
`dist/` and releases. NickelDrawer has **no link to it at all** (decision changed, see "Decision: the toggle is a
flag file" in section 5).

Labels used below: **[confirmed]** means read from the binaries (all five builds unless a build is named).
**[assumed]** means inferred and still to be checked on the device.

## Summary

| Question | Answer |
|---|---|
| Hook point | `StatusBarView::StatusBarView(QWidget*)`, `_ZN13StatusBarViewC1EP7QWidget`. It has an `R_ARM_JUMP_SLOT` (PLT) relocation in libnickel in all five builds, so an `nh_hook` on it works like the `MoreView` hook **[confirmed]** |
| How many status bars | One at a time. It is built only by `StatusBarController::loadView`, and that controller only by `MainWindowController::createStatusBar`. Nickel rebuilds it after first-time setup, a "reset main window" and a language change, and deletes the old one **[confirmed]** |
| Layout | `StatusBarView` (QVBoxLayout) > `statusbarContainer` (QHBoxLayout `mainHorizontalLayout`: `timeLabel`, then `iconContainer`) > `iconLayout` (QHBoxLayout) holding, in order: `parentalControlIcon`, `rotateIcon`, `lightIcon`, `wifiIcon`, `bluetoothIcon`, `batteryIcon`, `notificationIcon`, `syncIcon`, `searchIcon` **[confirmed]**. The same widget sequence and allocation sizes appear in all five builds |
| Button | A Nickel `StatusBarIcon` (sizeof 0xC0 in all five builds), the same class as the search icon. Set its pixmaps with `TouchLabel::setDeselectedPixmap` / `setSelectedPixmap`, connect `tapped(bool)`, and insert it into `iconLayout` just before `lightIcon` |
| Nickel icon | None fits. No camera or screenshot image is among libnickel's `:/images/` paths or the `nickel` rcc name tables. Use Tabler `screenshot` (outline, MIT) |
| Capture | `ScreenshotMaker::saveCurrentScreen()` writes `/mnt/onboard/screen_NNN.png` (3 digits, zero-padded, first free number from 001). It calls `QScreen::grabWindow` on the main window. libkobo's `AbstractKoboScreen::grabWindow` copies Qt's in-memory screen image, **not** the e-ink panel **[confirmed]** |
| `[FeatureSettings] Screenshots` | Only gates the power-button path in `PowerKeyFilter::filter`. `saveCurrentScreen` itself does not read it, so the button works with the setting off **[confirmed]** |
| Own instance | Safe. `ScreenshotMaker` is a 4-byte object `{int counter}` and its constructor stores 1. Nickel's own copy lives inside `PowerKeyFilter`; there is no singleton or getter **[confirmed]** |
| Sequence | Blank the button (keep its size) → wait about 250 ms on a single-shot timer → `saveCurrentScreen` → restore the button → `MainWindowController::toast` |
| On/off | A flag file `/mnt/onboard/.adds/helpers/screenshot/off`; no file means on. `install.sh --off` / `--on` or the owner creates and removes it over USB. The helper reads it when a status bar is built, on its Show event and on every tap |

## 1. StatusBarView

### Construction (confirmed, 23828 decomp plus relocations in all builds)

- `MainWindowController::createStatusBar()` deletes any old controller (`deleteLater`), does
  `new StatusBarController(mainWindowWidget)` (0x18 bytes), then calls `hideIndicators`, `hideBottomLine`,
  `showStatusBarAsNecessary(currentView)` and `setRotateIconVisible`.
  Callers (XrefsTo): `MainWindowController::loadView` (boot), `N3PostFTEWorkflowManager::endWorkflow` (after first-time
  setup), `N3SettingsExtrasController::resetMainWindow`, and `N3SettingsLanguageSelectionController::localeSelected`.
- `StatusBarController::loadView()` does `operator new(0x7C)` + `StatusBarView::StatusBarView(window)`, connects a
  signal, stores the view, then calls `adjustWidth()` (fixed width = window width), `updateFrontLightStatus`,
  `updateWifi`, `setParentalControlVisibility` and `setNotificationsVisibility`.
- `StatusBarView::StatusBarView` is called only from `loadView`, through the PLT. The relocation
  `R_ARM_JUMP_SLOT _ZN13StatusBarViewC1EP7QWidget` is present in all five builds, so an `nh_hook` on the C1 symbol
  catches every construction. Its parent is the main window widget: the bar is an overlay, not part of the page
  stack (re/notes/G2-findings.md, "Header on device").
- The hook runs after Nickel's constructor has returned. At that point `setupUi` has finished, the pixmaps and
  connections exist, and `loadView` has not yet called `adjustWidth`. Adding a child widget there is safe.
- Timing: NickelHook installs PLT hooks from the plugin's init, before Nickel builds its main window. The existing
  `MoreView` hook depends on the same order. The first status bar therefore goes through our hook **[assumed, same
  evidence as the MoreView hook which works on device]**. As a fallback, a one-shot timer after init could also look
  up `findChild<QWidget*>("StatusBarView")` on the main window and add the button if the hook missed it. The drawer
  already finds the bar that way in `mod/common/status_bar_inset.cc`.

### Layout and object names (`Ui_StatusBarView::setupUi` + string table)

The classes and their order are confirmed. The names `statusbarContainer`, `timeLabel`, `iconContainer`,
`iconLayout`, `parentalControlIcon`, `wifiIcon`, `bluetoothIcon`, `notificationIcon` and `syncIcon` are confirmed:
they sit together in the string table next to `:/qss/StatusBarView.qss`. `searchIcon` is confirmed by
`StatusBarView.qss`. `rotateIcon`, `lightIcon` and `batteryIcon` are shared strings elsewhere in libnickel, so they
are **[assumed]**. The helper should therefore find the light icon by `objectName == "lightIcon"` first and fall
back to `QObject::inherits("StatusBarLightIcon")` (Nickel's moc data supplies the class name).

```
StatusBarView  (QVBoxLayout "mainLayout", margins 0, styleSheetFile ":/qss/StatusBarView.qss")
├── statusbarContainer  GenericContainerWidget, QHBoxLayout "mainHorizontalLayout"
│   ├── timeLabel       TimeLabel
│   └── iconContainer   GenericContainerWidget, QHBoxLayout "iconLayout" (margins 0)
│       ├── parentalControlIcon  StatusBarIcon (0xC0)       hidden unless parental controls are on
│       ├── rotateIcon           StatusBarIcon (0xC0)       shown only where rotation is allowed
│       ├── lightIcon            StatusBarLightIcon (0xC8)  <- insert our button before this one
│       ├── wifiIcon             StatusBarWiFiIcon (0xF0)
│       ├── bluetoothIcon        StatusBarBluetoothIcon (0x128)
│       ├── batteryIcon          N3BatteryStatusLabel (0xD8)
│       ├── notificationIcon     StatusBarIcon (0xC0)
│       ├── syncIcon             StatusBarSyncIcon (0x168)
│       └── searchIcon           StatusBarIcon (0xC0)       search.png / search_hit.png
└── (QLabel, "background-color: black;", the bottom line)
```

The allocation sizes passed to `operator new` in `Ui_StatusBarView::setupUi` are byte-identical in 23552, 23648,
23684, 23697 and 23828. That sequence is 0x10, 0x2C, 0x10, 0x58, 0x2C, 0x10, 0xC0, 0xC0, 0xC8, 0xF0, 0x128, 0xD8,
0xC0, 0x168, 0xC0, 0x18 (checked with `llvm-objdump` on each build).
In `StatusBarView` the `Ui_` struct starts at +0x24. The member pointers used by the slots are +0x3C parental, +0x40
rotate, +0x44 light, +0x48 Wi-Fi, +0x4C Bluetooth, +0x50 battery, +0x54 notification, +0x58 sync and +0x5C search
(`setCanCollideForIcons`). **We do not use offsets.** The helper finds widgets by object name with plain Qt.

How Nickel builds an icon button (the search icon in the `StatusBarView` constructor, confirmed):
`StatusBarIcon(iconContainer)` (a `TouchLabel` subclass, `QLabel::setAlignment(0x44)`, `WA_` attribute 0x97 =
"can collide"). Then `TouchLabel::setDeselectedPixmap(QPixmap(":/images/statusbar/search.png"))`,
`TouchLabel::setSelectedPixmap(QPixmap(":/images/statusbar/search_hit.png"))`, `setProperty("actionName", "Search")`,
and a connection from the icon's tap signal to a `StatusBarView` slot. `StatusBarView::iconTapped()` only fires an
analytics `EventEngine` event ("statusBarOption"/"Action") with the `actionName`. Our button sets no `actionName`
and does not connect to `iconTapped`, so no Kobo analytics event fires.

Look (QSS, Storm class = Libra 2, confirmed): bar height 126 px, `iconContainer` spacing 18 px, `searchIcon`
80 px wide with `padding-right: 67px` (the right margin of the whole row). The other icons are not sized in the
QSS, so they take their pixmap's size. The pressed look is Nickel's `*_hit.png`: the glyph in white on a black disc.
`StatusBarIcon::paintEvent` paints the label pixmap. Only icons with an animated SVG (sync) draw their own
ellipse.

Recommended button: `StatusBarIcon` + `TouchLabel` setters, `objectName "nhScreenshotIcon"`, fixed size equal to
`lightIcon`'s size hint (read at run time, so the row stays even). Pixmaps:
- deselected: Tabler `screenshot` (outline) rendered black at the light icon's height (about 56-64 px **[assumed]**,
  logged at first use);
- selected: the same glyph in white on a black disc of the same size. This copies Nickel's `_hit` style. It is
  drawn at run time with QPainter from the Tabler PNG, like `PressArea::HIGHLIGHT_CIRCLE` in the drawer, and is
  not a new icon design.

The helper ships its own copy of the PNG (`.adds/helpers/screenshot/icons/screenshot-64.png`) plus Tabler's MIT
notice. Adding Tabler `screenshot` to `host/icons/ui` is not enough: the helper must not depend on koboapps being
installed.

### Nickel image check (confirmed)

No Nickel image fits. Among libnickel's `:/images/statusbar/*` paths there is nothing for camera or capture. A
UTF-16 scan of the five rcc name tables in `nickel` finds no `camera`, `screenshot`, `capture` or `photo` names. The
only libnickel string with "camera" in it is `cameraMake` (EXIF). Tabler 3.49.0 (`downloads/`) has
`outline/screenshot.svg`, `outline/camera.svg` and `filled/camera.svg`. Use `screenshot`, because it shows "capture
the screen" and not "take a photo".

### Reading mode

`MainWindowController::showStatusBarAsNecessary(view)` reads the page's `showTitle` / `showIndicators` /
`showStatusBottomLine` properties and shows or hides the one `StatusBarView` (re/notes/G2-findings.md). The reader's
tap-to-open menu is `ReadingMenuView`, which has its own `lightIcon` / `rotateIcon` / `settingsIcon` in its QSS.
That suggests the status bar is hidden while reading and the reading menu is a different widget **[assumed;
check on device]**.
Recommendation: v1 adds the button only to `StatusBarView`. If the device shows the status bar above the reading
menu, the button appears there too at no extra cost, and the capture then includes the open reading menu. A reading
toolbar button (hooking `ReadingMenuView`) is a possible later addition behind its own flag.

## 2. ScreenshotMaker (confirmed, 23828 decomp; same call sequence in all five builds)

- `ScreenshotMaker::ScreenshotMaker()` stores `1` in the object's only field. **sizeof = 4** (it is embedded at
  `PowerKeyFilter + 8`, next to a `QTimer` at +0xC). No vtable and no QObject.
- `QString ScreenshotMaker::fileName()` returns `QString("/mnt/onboard/screen_%1.png").arg(counter, 3, 10, '0')`,
  for example `/mnt/onboard/screen_001.png`. The string is present in all five builds.
- `void ScreenshotMaker::saveCurrentScreen()`:
  1. returns without doing anything when `PlugManager::gadgetMode()` is true (USB mass storage active);
  2. counts the counter up while `QFile::exists(fileName())`, so it never overwrites a file;
  3. takes `MainWindowController::sharedInstance()`'s main view widget and its geometry, then calls
     `QGuiApplication::primaryScreen()->grabWindow(view->winId(), 0, 0, width, height)`;
  4. calls `QPixmap::save(fileName, NULL)`, so the format comes from the suffix (PNG). The return value is
     ignored and nothing tells the user.
- Nickel's only trigger is `PowerKeyFilter::filter(QObject*, QKeyEvent*)`, installed from `Nickel3Application::notify`.
  On a power key press it builds `FeatureSettings` and reads `Screenshots` (default false). If true, a press calls
  `saveCurrentScreen` on its embedded maker and does not go to sleep. If false, press and release run the normal
  sleep toggle. So `[FeatureSettings] Screenshots=true` is needed **only for the power button**. NickelDrawer's
  existing Tweaks row "Chụp màn hình" / "Screenshots" (`internal/tweaks`, id `screenshots`) toggles exactly that
  key.
- Nickel shows **no confirmation** of its own (no toast in `saveCurrentScreen` or `filter`).
- Our own instance: allocate 4 bytes (use 16 for headroom), call the constructor through dlsym, and keep it for the
  helper's lifetime. Its counter is separate from Nickel's. That is harmless, because both skip existing files.
  To check success afterwards, call `fileName()` again **after** `saveCurrentScreen`. The counter still points at
  the name just written, so `QFile::exists(name)` with a modification time from the last few seconds means it
  worked. In gadget mode nothing is written and the helper toasts nothing (USB is connected anyway).

## 3. Hide, capture, show on e-ink

`AbstractKoboScreen::grabWindow` (libkobo.so, confirmed by disassembly) builds a `QPixmap(w, h)` and
`QPainter::drawImage`s a part of the screen object's `QImage` member (screen + 0x60) into it. That image is the
screen image Qt composes the backing store into. It is **not** the e-ink panel state. Two things follow:

1. We do **not** have to wait for the e-ink waveform to finish.
2. We **do** have to let Qt repaint and compose. `hide()` / `update()` only schedule a paint, and the backing
   store is flushed into the screen image from the event loop. A capture in the same call stack would still
   contain the button, probably in its pressed `_hit` look.

Do not `hide()` the button. A hidden widget leaves the `QHBoxLayout`, and the icons to its right would move for
the captured frame. Keep the size and paint nothing instead: swap both pixmaps for a transparent `QPixmap` of the
same size. (`QSizePolicy::setRetainSizeWhenHidden` should exist in Qt 5.2 **[assumed]**, but the pixmap swap does
not depend on it.)

Recommended sequence (all on the GUI thread):

1. `tapped` → if the flag file says off, remove the button and stop. If a capture is already running, ignore the
   tap.
2. Set both pixmaps to the transparent pixmap and call `update()`.
3. `QTimer::singleShot(250 ms, …)`. 250 ms is a starting value; tune it on the device. Lowest safe value: one
   event-loop pass plus Nickel's flush, likely under 100 ms **[assumed]**.
4. Call `saveCurrentScreen()` through dlsym. It blocks for the PNG encode of 1264×1680 (a few hundred ms
   **[assumed]**).
5. Restore the pixmaps and call `update()`.
6. Verify the file (section 2), then `MainWindowController::toast(text, "", 2000)`. The toast is created only after
   the capture, so it is never in the image.

On-device check: open `screen_NNN.png`. It must show the status bar without the button and without a black press
disc, and with the other icons in their normal positions.

## 4. Feedback toast

Nickel gives no feedback. The helper shows a toast through `MainWindowController::sharedInstance()` +
`MainWindowController::toast(QString const&, QString const&, int)`, the symbols the drawer and `mod/hello` already
use, present in all five builds. Text:

- en `"Screenshot saved"`, with `{file}` in the secondary line (`screen_004.png`);
- vi `"Đã chụp màn hình"`, secondary `{file}`.

Strings come from `/mnt/onboard/.adds/helpers/screenshot/locales/<lang>.json`, chosen by `CurrentLocale` with the
usual fallback chain (exact → language → `en`). A built-in English fallback is used only when the file is missing,
following the precedent of the drawer's `fallback_strings.cc`. If the toast symbols are missing, the helper logs and
saves silently.

## 5. On/off switch

### Storage (user decision)

- Flag file: `/mnt/onboard/.adds/helpers/screenshot/off`. No file means the button is **on**. Installing the helper
  is the opt-in, so the button is on by default after install. This is not a Kobo conf key, so Nickel's settings
  stay untouched.
- Other helper files in the same folder: `uninstall` (NickelHook `uninstall_flag`), `disable` (kill switch: the hook
  passes straight through), `debug` (verbose log), `locales/`, `icons/`, `LICENSES/tabler.txt`.

### How the helper picks up a change

Never keep files open on `/mnt/onboard`, and do not use a `QFileSystemWatcher` there: inotify on the FAT/vfat user
partition is unreliable, and the partition is unmounted while USB is connected. The helper reads the flag with a
single `stat()` (`QFileInfo::exists`) at these points:

- when the `StatusBarView` hook runs (boot, language change, reset);
- on the status bar's `QEvent::Show` (an event filter on the bar, which Nickel shows and hides as pages change);
- on every tap. If the flag says off, the helper removes the button and does not capture.

If the stat fails (for example while USB is connected), the helper keeps the current state.

### Decision: the toggle is a flag file, not a Tweaks row

An earlier design had a hidden NickelDrawer Tweaks row (shown only while the helper was installed) that wrote the
`off` file and pinged the bar through a dynamic property (`nhScreenshotButton`). It was built and then removed.
Why: the releases bundle only the drawer and its dependencies, so the drawer must carry no code, strings or
detection for an extra mod. The switch is therefore only the helper's own flag file: `install.sh --off` / `--on`,
or create and delete `/mnt/onboard/.adds/helpers/screenshot/off` over USB. The helper still re-checks the file
on bar build, bar show and every tap, so the change shows up the next time the bar is shown, with no reboot.
The generic optional `caption` field of the Tweaks protocol stayed (the Screenshots cell says "With the power
button").

## 6. NickelHook coexistence and risks

Two NickelHook mods side by side (confirmed from `third_party/NickelHook`):

- **Plugin key**: `NickelHook.mk` derives `Keys` from the library name (`nh.<name>`). `libstatusbarshot.so` gets
  `nh.statusbarshot` and the drawer `nh.nickeldrawer`. Both are unique. Qt can skip loading a plugin whose key
  repeats (NickelHook issue #9). Trap: `nhplugin.json` is generated **inside the shared vendored folder**
  `third_party/NickelHook/`. The helper's `docker-build` must delete `nhplugin.json nhplugin.moc nhplugin.moc.o
  nh.o` first, as `mod/drawer` and `mod/hello` already do. Its `check-symbols.sh` must also check that the built
  `.so` contains the string `nh.statusbarshot`.
- **Failsafe**: per library. `nh_failsafe_create` renames *its own* path to `<path>.failsafe` and restores it later.
  Different `.so` names do not clash.
- **Hooks**: each `nh_hook` rewrites libnickel's GOT slot for one symbol. The drawer hooks
  `_ZN8MoreViewC1EP7QWidget` and the helper `_ZN13StatusBarViewC1EP7QWidget`: separate slots, no conflict. (If two
  mods ever hooked the same symbol, the second would chain to the first through its `.out`, which also works.)
  NickelMenu, NickelDBus and NickelHardcover do not hook `StatusBarView` (known-mods matrix; NickelMenu only
  dlsyms `StatusBarController::home` / `MainNavController::home`).
- **Exported hook target names** (`sym_new`) are looked up with `dlsym(self, …)`, so they are per library. Still use a
  unique prefix: `statusbarshot_StatusBarView_construct`.
- **Flag and log paths**: the helper uses `.adds/helpers/screenshot/*` and the syslog prefix `(statusbarshot)`, and
  never touches `.adds/nickeldrawer/*`.
- **kobo-tieng-viet** installs `imageformats/libtiengviet.so` (a Qt image plugin), `trans_vi.qm` and fonts. It does
  not touch the status bar. Its translation changes only text. The helper's icon has no text.

Risks:

| Risk | Mitigation |
|---|---|
| Status bar built before the hook is installed | Same order as the working `MoreView` hook. Fallback: a one-shot lookup by object name after init |
| Layout differs on a future firmware (no `iconLayout` or `lightIcon`) | Find by object name; if `iconLayout`'s parent is missing, log once and do nothing (no crash). If `lightIcon` is missing, insert at index 0 |
| `StatusBarIcon` size changes (0xC0 now) | Allocate 0x100 with zero fill. The `firmware-watch` skill re-runs the per-build `setupUi` size check in this note |
| Missing symbol | Every libnickel symbol is `optional` in the `nh_dlsym` table, and the helper checks them all in init. If any is missing, it does not add the button, logs it, and shows **one** toast about 10 s after boot ("Screenshot button disabled: firmware not supported") |
| Unknown firmware | `/mnt/onboard/.kobo/version` not in the five-build list → same graceful disable |
| Crash at boot | NickelHook failsafe (`failsafe_delay` 15 s, as for the drawer) removes the `.so` on the next boot |
| Accidental taps | A `TouchLabel` tap needs a full press-release. Ignore new taps while a capture is running |
| Memory/CPU | One widget, two pixmaps of about 64×64 (about 32 KB), one 4-byte `ScreenshotMaker`, a few `stat()`s on show and tap. No timers while idle |
| `/mnt/onboard` busy (USB) | `saveCurrentScreen` already returns in gadget mode; the helper checks the file afterwards and does not toast |
| Full partition / save fails | `QPixmap::save` result is invisible; the file check catches it → toast "Could not save screenshot" |
| Our pixmap swap fails to repaint in time | Tune the 250 ms delay on device; worst case the button appears in the image (cosmetic) |

## 7. Implementation plan (for a Sonnet coder)

New folder `helpers/statusbar-screenshot/` (tracked, outside `dist/` and release tooling):

| File | Content |
|---|---|
| `Makefile` | Copy of `mod/hello/Makefile` pattern: `LIBRARY_NAME := libstatusbarshot.so`, NickelHook from `../../third_party/NickelHook`, `-frtti`, `-fvisibility=hidden`, Qt5Widgets; `docker-build` cleans `GENERATED_IN_VENDOR` first; KoboRoot `build/helpers/KoboRoot-statusbarshot.tgz` with the `.so` plus `/mnt/onboard/.adds/helpers/screenshot/{locales/en.json,locales/vi.json,icons/screenshot-64.png,LICENSES/tabler.txt}` |
| `statusbarshot.cc` | `nh_info` (`name "statusbarshot"`, `uninstall_flag "/mnt/onboard/.adds/helpers/screenshot/uninstall"`, `failsafe_delay 15`), tiny `init()`: read the `disable` / `debug` flags and the firmware version; nothing else |
| `nickel_bridge.{h,cc}` | The only file naming libnickel symbols. Hook table + dlsym table (below), `isActive()`, typed wrappers: `makeStatusBarIcon(QWidget*)`, `setIconPixmaps(QWidget*, QPixmap, QPixmap)`, `saveScreen()` → `QString` path or empty, `toast(QString, QString)` |
| `screenshot_button.{h,cc}` | `ScreenshotButton::attachTo(QWidget *statusBarView)`: find `iconLayout` / `lightIcon` by object name, build the icon, insert, connect `tapped(bool)`; the event filter (Show); `capture()` sequence of section 3 |
| `helper_flags.{h,cc}` | Paths and `isButtonEnabled()` (one stat), `isKillSwitchOn()`, `isDebug()` |
| `helper_strings.{h,cc}` | Locale pick from `Kobo eReader.conf` `CurrentLocale` + JSON load (QJsonDocument), fallback English |
| `icon_pixmaps.{h,cc}` | Load the Tabler PNG, scale to the target height, make the `_hit` look (white glyph on black disc) and the transparent placeholder |
| `check-symbols.sh` | Adapted from `mod/drawer/check-symbols.sh`: all symbols in `nickel_bridge.cc` exported by all five `re/symbols/<fw>/exports.txt`, ARM hard-float, allowed NEEDED list, no libstdc++ beyond the ABI, KoboRoot contents exact, the `nh.statusbarshot` plugin key present, vtables with typeinfo (`mod/common/check-vtable-typeinfo.sh`) |
| `locales/en.json`, `locales/vi.json` | `toast.saved`, `toast.failed`, `toast.disabled` with `{file}` placeholder |
| `README.md` | Short: what it is, install/uninstall drill, flag files. Not linked from the main docs |

Symbols (all present in 23552, 23648, 23684, 23697 and 23828; all `optional`):

| Use | Mangled |
|---|---|
| hook | `_ZN13StatusBarViewC1EP7QWidget` → `statusbarshot_StatusBarView_construct` |
| dlsym | `_ZN13StatusBarIconC1EP7QWidget` |
| dlsym | `_ZN10TouchLabel19setDeselectedPixmapERK7QPixmap` |
| dlsym | `_ZN10TouchLabel17setSelectedPixmapERK7QPixmap` |
| dlsym | `_ZN15ScreenshotMakerC1Ev` |
| dlsym | `_ZN15ScreenshotMaker17saveCurrentScreenEv` |
| dlsym | `_ZN15ScreenshotMaker8fileNameEv` (returns `QString` by hidden pointer, as a member call) |
| dlsym | `_ZN20MainWindowController14sharedInstanceEv` |
| dlsym | `_ZN20MainWindowController5toastERK7QStringS2_i` |
| signal (metaobject, string connect) | `TouchLabel::tapped(bool)`. Note it as `signal-symbol: _ZN10TouchLabel6tappedEb` for the check script |

Order: (1) helper build + check script, (2) host-side desk test of `icon_pixmaps` / `helper_flags` / `helper_strings`
(Qt-free parts as plain functions, like the drawer's `docker-test`), (3) G1 device run, (4) `install.sh --off` / `--on`.

### Validation on device (G1: first run only with the user present)

Before: back up the device (`backups/<date>/`, including `KoboReader.sqlite` and `Kobo eReader.conf`). Battery
must be above 50 %. Do the uninstall drill on this helper first: create `.adds/helpers/screenshot/uninstall`,
reboot, and confirm `libstatusbarshot.so` is gone. Then do the failsafe drill: power off within 15 s of boot, and
the `.so` is removed.

1. Install the KoboRoot. After boot, the home status bar shows the new icon left of the brightness icon. The other
   icons have the same spacing and the row still fits.
2. Press and hold: Nickel's black-disc pressed look shows. Release: about 250 ms later the toast "Đã chụp màn hình"
   appears. `/mnt/onboard/screen_001.png` exists. On the Mac, it shows no button, no disc, no toast, and the icons in
   place.
3. Tap again → `screen_002.png`. With `screen_00N.png` files already there, the helper does not overwrite them.
4. More tab, a settings page, the store, a book (reading mode: does the bar or the button show? Note the result for
   section 1).
5. Change the language and come back: the button is still there (the status bar was rebuilt).
6. `[FeatureSettings] Screenshots=false` → the button still works, and the power button still puts the device to
   sleep.
7. `install.sh --off` (or create `off` over USB): after ejecting, the button is gone the next time the status bar
   is shown, and stays off after a reboot. `install.sh --on` brings it back.
8. NickelMenu, NickelDrawer and kobo-tieng-viet all still load (the drawer's More rows, NickelMenu's menu, and
   Vietnamese UI).
9. USB connect while on the home screen: no crash. Tap during USB (if the bar is reachable): no file, no toast.
10. `disable` flag → no button; the debug log says why.
11. Leave it running for a day: no extra battery drain compared with the day before (no idle timers).

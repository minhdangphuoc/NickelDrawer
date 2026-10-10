# re/ — reverse-engineering notes for NickelDrawer

Static analysis of `libnickel.so.1.0.0` across the five target Libra 2 firmware builds
(plan `docs/plan-detailed.md` §3 and §6). First pass done 8 Oct 2026 from exported symbols only;
no Ghidra yet.

## Legal note

This is interoperability work on software running on a device the user owns, the same approach
NickelMenu, NickelDBus, NickelHardcover and kobopatch took. **No Kobo binaries are in this repo**:
firmware zips, `KoboRoot.tgz`, `libnickel.so`, `nickel` and `libkobo.so` live only in the local cache
(below). The repo holds our scripts, symbol **names** (`exports.txt`), the public CDN URLs and hashes
(`firmware.lock`) and our own notes. Never commit anything from `~/.cache/nickeldrawer/`.

## Contents

| Path | What |
|---|---|
| `firmware.lock` | TSV: version, CDN month, URL, zip size + sha256, libnickel sha256 + ELF BuildID, verification result |
| `scripts/fetch-fw.sh` | download → verify Libra 2 image → extract libnickel/nickel/libkobo → dump exports |
| `scripts/symreport.py` | `diff` → `symbols/DIFF.md`; `matrix` → `symbols/known-mods-matrix.md` |
| `symbols/<fw>/exports.txt` | demangled, sorted, defined dynamic symbols of libnickel (`nm -D --defined-only`). ~4.6 MB each |
| `symbols/DIFF.md` | added/removed exports between consecutive builds, with full lists |
| `symbols/known-mods-matrix.md` | every libnickel symbol NickelMenu / NickelDBus / NickelHardcover dlsym, plus drawer candidates, × 5 builds |

`symbols/*/exports.txt` total ~23 MB. If that is too heavy for git, regenerate it instead of
committing it (decision for the repo owner; `.gitignore` was not touched by this pass).

## Cache layout (`~/.cache/nickeldrawer`, override with `NICKELDRAWER_CACHE`)

```
fw/<version>/kobo-update-<version>.zip      official package (~121 MB)
fw/<version>/unz/{KoboRoot.tgz,manifest.md5sum,upgrade/}
fw/<version>/root/usr/local/Kobo/{libnickel.so.1.0.0,nickel,platforms/libkobo.so}
fw/<version>/koboroot-list.txt              tar -tzv of KoboRoot.tgz
symbols/<version>/exports-mangled.txt       libnickel exports, mangled (input for symreport.py and future symcheck)
symbols/<version>/nickel-exports-mangled.txt
src/{NickelMenu,NickelDBus,NickelHardcover}/  upstream sources (tarballs of HEAD)
```

Mod sources used (HEAD on 8 Oct 2026): NickelMenu `bc799ebe95a4` (2026-06-12), NickelDBus `ab7ac0c7fee3`
(2024-04-28), NickelHardcover `a176fd52510a` (2026-09-21).

## Reproduce

Needs Xcode command line tools (`xcrun llvm-nm`, `llvm-cxxfilt`, `llvm-objdump`), curl, unzip, tar, python3.

```sh
for p in Nov2025/4.38.23552 Mar2026/4.38.23648 Apr2026/4.38.23684 May2026/4.38.23697 Sep2026/4.38.23828; do
  re/scripts/fetch-fw.sh "${p#*/}" "${p%/*}"      # prints a "lock:" line; compare with firmware.lock
done
S=~/.cache/nickeldrawer/src; mkdir -p "$S"
for r in pgaskin/NickelMenu shermp/NickelDBus RedHatter/NickelHardcover; do
  n=${r#*/}; mkdir -p "$S/$n"
  curl -fsSL "https://codeload.github.com/$r/tar.gz/HEAD" | tar -xz -C "$S/$n" --strip-components=1
done
python3 -I re/scripts/symreport.py diff
python3 -I re/scripts/symreport.py matrix
```

Useful one-offs: `xcrun llvm-objdump -p <libnickel>` (NEEDED, SONAME, RPATH),
`file <libnickel>` (ELF class/ABI/BuildID).

## Findings (8 Oct 2026)

### Firmware verification — all five are genuine Libra 2 (Freescale i.MX6SLL) packages

Evidence, identical for every build including **4.38.23697**:

- `upgrade/` has a single platform dir `mx6sll-ntx/` with `zImage-E70K10`, `u-boot-mx6sll-E70K10-LPDDR2-512MB.imx`,
  `mx6sll-E70K10*.dtb`, `ntxfw-E70K10*.bin` (E70 = 7" NTX board, i.MX6SLL; the bad 4.39.22801 package had `mt8113t-ntx`).
  `manifest.md5sum` has 20 `mx6sll` entries and 0 `mt8113`.
- The dtb strings contain `fsl,imx6sll`; u-boot contains `imx` driver strings. `zImage-E70K10` is byte-identical
  across all five (md5 `61a1525606fdb49f77090e465f3623be`) — the kernel did not change in this range.
- `KoboRoot.tgz` ships `etc/images/io-my_first_time.raw.gz` (`io` = Libra 2 codename).
- libnickel: `ELF 32-bit LSB shared object, ARM, EABI5`, SONAME `libnickel.so.1`, hard-float loader
  `ld-linux-armhf.so.3`. libnickel and `libkobo.so` are multi-platform (contain MTK / AllWinner / NXP screen
  classes, incl. `KoboScreenNXPDeviceConfigIo`), so their strings alone cannot prove the target; the `upgrade/`
  boot files can.
- 4.38.23697's libnickel differs from 23684's (different sha256/BuildID) but exports the **identical** symbol set.
- No `version` file in `KoboRoot.tgz`; the version string `4.38.NNNNN` appears in libnickel's strings.
  On device, `/mnt/onboard/.kobo/version` remains the runtime source (plan §3.2.3).

### Qt

libnickel NEEDS `libQt5{Core,Gui,Widgets,Network,Sql,DBus,Xml,Svg,WebKit,WebKitWidgets,Multimedia,Script,
Concurrent,Sensors,PrintSupport}.so.5` with RPATH `/chroot/lib:/chroot/usr/lib:/usr/local/Qt-5.2.1-arm/lib`
→ **Qt 5.2.1** (consistent with what NickelHook targets). The Qt libraries themselves are on the rootfs, not in
`KoboRoot.tgz` (KoboRoot only carries legacy symlinks under `Qt-5.2.1-arm/lib`). libnickel uses libstdc++
(`GLIBCXX_3.4.18` max, `CXXABI_ARM_1.3.3`); our mod still must not use the C++ stdlib (see CONTRIBUTING.md).

### Exports

| build | exports | vs previous |
|---|---|---|
| 4.38.23552 | 68,378 | – |
| 4.38.23648 | 68,379 | +5 / −4 (`SelectionMenuController::lookupWeb` ↔ `lookupGoogle`, `WebSearchMixinBase`) |
| 4.38.23684 | 68,392 | +13 / −0 (`N3WebStoreWebPage`, QWebFrame metatype) |
| 4.38.23697 | 68,392 | identical |
| 4.38.23828 | 68,499 | +201 / −94 |

4.38.23828 highlights (see `symbols/DIFF.md`):

- **New `Qt6UpgradeWorkflowManager`**, `ApplicationSettings::{availableQt6Upgrade, qt6UpgradeConfirmedByUser, …}`,
  `DialogSettings::qt6UpgradeNoteDialogShown`, new `HtmlNoteDialog`; `OptionalQT6UpgradeDialog` removed.
  Guess: 23828 is the build that offers the Libra 2 an (optional, user-confirmed) jump to a Qt 6 / 5.x firmware.
  5.x breaks NickelMenu, NickelHook mods and kobo-tieng-viet → the user should decline it; worth a line in the
  user docs and the risk register.
- `N3SettingsController::{wirelessConnection, readingSettings, …}` moved to a new **`SettingsMixin`** class
  (same method names). `N3SettingsAccessibilityController/View` removed.
- `WirelessWorkflowManager::showNetworksDialog(bool)` became `showNetworksDialog(bool, bool)` (+ `…Impl(bool)`).
- `HardwareInterface::{rootfs,recovery,aux,userdata}PartitionPath` added (A/B-ish partition awareness, guess).

### Known-mod symbols — all resolve on all five builds

167 distinct libnickel symbols are dlsym'd by the three mods. Every one that is missing in some build is a
deliberate alternative the mod already handles (legacy ctor overloads, non-const `currentView()`,
`StatusBarController::home` → `MainNavController::home`, `lookupWeb`/`lookupGoogle`, Pocket/Dropbox/Instapaper
library tabs, a Qt symbol that lives in libQt5Gui). So NickelMenu, NickelDBus and NickelHardcover should load
on all five — matching the user's NickelMenu working on 23828.

### Drawer (gate G2) — looks feasible from exports alone (to be confirmed in Ghidra and on device)

All plan §6.4 starting symbols for R1, R2, R4–R11 are exported unchanged in all five builds:

- **R1 view stack:** `MainWindowController::{sharedInstance, pushView(QWidget*), popView(QWidget*),
  currentView() const, viewWithObjectName}` — NickelHardcover already pushes full-screen views with
  `pushView`, which is strong evidence the plain-QWidget route works on 4.38. There is also a controller-level
  API: `push(AbstractController*, bool)`, `pop`, `popTo`, `topController`, `updateNavFooter(AbstractController*)`,
  `createNavFooter()`, `showStatusBarAsNecessary(QWidget*)`. *Guess:* the bottom tab bar is window-owned
  (`createNavFooter`) and shown/hidden per controller via `updateNavFooter`, so keeping the tab bar visible may
  require the controller route (`AbstractController` subclass = vtable work) rather than `pushView`. This is
  the main G2 question for Ghidra.
- **R2 tab button:** `MainNavView` ctor (NickelMenu hooks it), `MainNavButton::{ctor, setPixmap,
  setActivePixmap, setText, setActive, tapped}`, `MainNavView::{updateVisibleButtons, set*Active}`,
  `MainNavController::setActiveSection(MainNavSection)`.
- **R4 settings look:** `SettingContainer`, `TouchCheckBox`, `TouchLabel`, `N3ButtonLabel` (used by
  NickelHardcover's settings rows); also exported but unused by mods: `ToggleSwitch`, `SettingItemWithToggleSwitch`,
  `N3SettingsPagedView`, `PagingFooter`, `N3SettingsControllerBase`.
- **R3 header:** no `N3SettingsHeader`; candidates by name are `TextHeader*`, `HeaderBarLabel`,
  `PopupHeaderContainer`, `StoreNavTitleBar`, and `MainWindowController::previousViewTitle()`. Needs Ghidra.
- **R5/R6 dialogs, R7 toast, R8 keyboard, R9 Wi-Fi, R10 rescan, R11 dark mode:** all present
  (`N3DialogFactory::getDialog`, `ConfirmationDialogFactory::*`, `MainWindowController::toast`,
  `KeyboardFrame::createKeyboard`, `KeyboardReceiver`, `WirelessWorkflowManager::connectWireless(bool,bool)`,
  `PlugWorkflowManager::sync`, `N3FSSyncManager::sync(QStringList const&)`, `ReadingSettings::getDarkMode`).
- **Per-firmware aliases already needed:** only for optional extras (`SettingsMixin` vs `N3SettingsController`
  to open Wi-Fi settings; `showNetworksDialog` signature). Nothing on the drawer's core path changed in this range.

Not answerable from exports: object sizes (for `calloc` before ctors), signal/slot names, stylesheet objectNames,
how the nav footer reacts to `pushView`, refresh behaviour (R12). Those need Ghidra (plan §6.3) and the G1/G2
on-device runs.

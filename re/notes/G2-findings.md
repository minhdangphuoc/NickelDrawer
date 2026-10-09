# G2 reverse-engineering spike — findings (libnickel 4.38.23828)

Date: 2026-10-08. Analyst: the maintainer, headless Ghidra 11.3.2, read-only project `libnickel-4.38.23828`.
Scripts: `re/ghidra-scripts/` (run with `run-headless.sh <fw> -postScript <Script>.java <args>`).
Decompiled output stays in `~/.cache/nickeldrawer/decomp/` (Kobo code, never in the repo). Everything below
is our own description of behaviour, plus symbol names.

Symbol presence was checked against `~/.cache/nickeldrawer/symbols/<fw>/exports-mangled.txt` for all five builds
(4.38.23552, 23648, 23684, 23697, 23828). **Every symbol named in this note is exported by all five builds**,
except the Qt6 group in question 5 (23828 only); see the table in `symbols-required-draft.yaml`.

Only 23828 was decompiled. The other four builds were checked by exported names only, so behaviour could differ
(see open questions). Their symbol sets are almost identical (`re/symbols/DIFF.md`), so differences are unlikely
outside the Qt6 area.

Ghidra loads the library at image base 0x10000; addresses below are Ghidra addresses for 23828 and are given only
to make re-finding easy. Calls inside libnickel to exported functions go through the PLT (seen as thunks at
0x67xxxx–0x6cxxxx), so NickelHook's PLT/GOT hooks catch Nickel's own calls.

---

## Q1 — Adding an "Apps" row to the More tab

**Verdict: feasible.** Same pattern as NickelMenu's tab-bar button, one level deeper.

How the More tab is built:

- `MoreController` (QObject + `AbstractController` at offset 8). `MoreController::loadView()` allocates a
  `MoreView` (0x1c bytes), calls `MoreView::MoreView(QWidget*)` through the PLT, connects the view's signals
  (`wishlist()`, `notebooks()`, `dropbox()`, `googleDrive()`, `articles()`, `activity()`, `betaFeatures()`,
  `settings()`, `help()`) to the controller slots of the same names, stores the view, and installs itself as an
  event filter (for `SettingEvent`s).
- `MoreView::MoreView(QWidget*)` creates a `Ui_MoreView` (0x38 bytes), calls `Ui_MoreView::setupUi(QWidget*)`
  through the PLT, sets the properties `showIndicators=true`, `showStatusBottomLine=false`, `showTitle=true`,
  `mainNavView="More"`, then connects each button's tap signal to its own signal.
- `Ui_MoreView::setupUi` is uic output (no list model, no N3 list): objectName `MoreView`,
  `styleSheetFile=":qss/MoreView.qss"`, outer `QVBoxLayout "moreViewLayout"`, a `GenericContainerWidget`
  holding `QVBoxLayout "moreContainerLayout"`, a `FontSizeAdjustingLabel` title, then nine `IconLeftButton`s
  (each `new IconLeftButton(container)`, objectName, `setProperty("pixmap", QPixmap(":/images/..."))`,
  `addWidget`). Verified objectNames: `wishlistButton`, `dropboxButton`, `googleDriveButton`, `articlesButton`,
  `betaFeaturesButton`, `helpButton`; the others (notebooks, activity, settings) are pooled strings, likely
  `...Button` too — confirm by dumping `objectName()`s on device. Labels come from `retranslateUi`
  (context "MoreView": "My Wishlist", "My Notebooks", "My Dropbox", "My Google Drive", "My Articles",
  "Activity", "Beta Features", "Settings", "Help").
- `IconLeftButton` derives from `IconButtonBase`, which has the `tapped()` signal. API: ctor `(QWidget*)`,
  `setPixmap(QPixmap const&)`, `setText(QString const&)`, object size 0x58.
- `MoreView::updateVisibleButtons()` re-reads device/settings flags and shows/hides only its own known buttons
  (by Ui offsets), then `QLayout::activate()`. It runs on `SettingEvent`s. It will not touch an extra row.

Hook strategy:

1. Hook `MoreView::MoreView(QWidget*)` (`_ZN8MoreViewC1EP7QWidget`; called from `MoreController::loadView`
   via PLT). Call the original, then `findChild<QVBoxLayout*>("moreContainerLayout")`; fail-soft (log, return)
   if missing.
2. Build the row with Nickel's own `IconLeftButton` (dlsym ctor on a 256-byte calloc block like NickelMenu does
   for `MainNavButton`, then `setPixmap`, `setText`, objectName `nickelDrawerButton`) so it inherits
   `MoreView.qss`. Insert it with `QBoxLayout::insertWidget` before the Settings row (or after
   `betaFeaturesButton`).
3. Connect the old-style `SIGNAL(tapped())` to our slot (NickelMenu uses a QPushButton shim; we can use our own
   QObject since we have Qt). The slot opens the drawer (Q2).
4. Re-translate our label ourselves (locale change rebuilds text via `retranslateUi` only for Nickel's rows).

Alternative hook point: `Ui_MoreView::setupUi(QWidget*)` (`_ZN11Ui_MoreView7setupUiEP7QWidget`), called via PLT
from the `MoreView` ctor. Prefer the ctor: it is the same level NickelMenu hooks for `MainNavView`.

Comparison with NickelMenu (4.23.15505+): NickelMenu hooks `MainNavView::MainNavView(QWidget*)`, calls the
original, finds the `QHBoxLayout` with `findChild`, creates a `MainNavButton` via dlsym'd ctor on a calloc'd
block, configures pixmaps/text, and connects `SIGNAL(tapped())` through a QPushButton adaptor. Our hook is the
same recipe on `MoreView` + `IconLeftButton`. Both hooks can coexist (different symbols).

Open questions: exact objectNames of the settings/notebooks/activity rows; whether the row needs the same
`setProperty` calls Nickel makes (only `pixmap` is set; QSS keys on class and objectName).

## Q2 — Pushing a full-screen view, keeping the tab bar, back navigation

**Verdict: feasible with risk.** Two workable routes; the controller route matches Nickel's own More sub-pages.

How the window is built:

- `MainWindowController::createWindowStack(QWidget*)` makes a container widget (this+0x48) with a
  `QVBoxLayout` whose first item is a `QStackedWidget` (this+0x44, stretch 1).
- `MainWindowController::createNavFooter()` creates a `MainNavController` (this+0x4c/0x50) and adds its view
  (the bottom tab bar, `MainNavView`) to the window layout, below the stack. **The tab bar belongs to the window,
  not to the pushed view.**
- `MainWindowController::push(AbstractController*, bool)`: refuses duplicates ("This controller already
  exists"); unprepares the current top controller (calls its `viewWillDisappear`, hides the container);
  appends a `QPointer` to the controller list (this+0x3c) and connects the controller's `destroyed` to
  `controllerDestroyed`; loads the view (`AbstractController` vtable `loadView`); calls `pushView(view)`;
  then `prepareView(controller)`.
- `prepareView(AbstractController*)`: `updateOrientationMask`, **`updateNavFooter(controller)`**,
  `viewWillAppear`, `showStatusBarAsNecessary(view)`, show.
- `updateNavFooter(AbstractController*)`: calls the controller's virtual `navSection()`, calls
  `MainNavController::setActiveSection(section)`, and **shows the tab bar if `navSection() != 0`, hides it if
  `0`**.
- `pushView(QWidget*)`: if the widget is not already in the stack, fills missing `showIndicators` /
  `showStatusBottomLine` / `showTitle` properties from the status bar controller, closes touch menus,
  `addWidget` + `setCurrentWidget`. It does **not** touch the tab bar. Inside libnickel it is called only
  from `push()`; mods (NickelHardcover) call it directly.
- `popView(QWidget*)`: hides the widget, `deleteLater()`, `removeWidget`. Inside libnickel only `pop()` calls it.
- `pop(AbstractController*)`: needs the controller to be a QObject (`dynamic_cast` from `AbstractController`
  to `QObject`, else warns "Controller is not a QObject" and does nothing); unprepares it, calls
  `viewWillBeDestroyed`, removes it and re-prepares the controller underneath (which re-runs `updateNavFooter`).
- `show(AbstractController*)` (what tab taps use): pop the top controller, push the new one.

How Nickel's pages keep or hide the tab bar:

- `AbstractController::navSection()` returns 0 (no tab bar). `N3SettingsController` does not override it, so
  Settings (`MoreController::settings()` = `new N3SettingsController(nullptr)` + `show()` → `push(this+8)`)
  is shown **without** the tab bar.
- More sub-pages override it: `MoreController`, `DropboxController`, `GDriveController`,
  `WishlistController::navSection()` all return **6** (= More section). They keep the tab bar with "More"
  active. Their `back()` is `MainWindowController::sharedInstance()->pop(this)` (Dropbox first walks up its
  folder tree). The Wishlist page header is a `TextHeader` (from `Ui_WishlistView::setupUi`).

`AbstractController` layout (from the `MoreController` secondary vtable and callers): vptr, `QPointer<QWidget>`
(d-ptr at +4, widget at +8); size 0xc. Virtual slots (byte offsets): +0x0 D1, +0x4 D0, +0x8 `size()`,
+0xc `viewWillAppear()`, +0x10 `viewWillDisappear()`, +0x14 `viewWillBeDestroyed()`,
+0x18 `allowedOrientations() const`, +0x1c `navSection() const`, +0x20 `loadView()` (pure).
Real controllers are `class X : public QObject, public AbstractController` (AbstractController at +8), and Nickel
passes `this+8` to `push`.

What the drawer needs (route A, preferred — "native More sub-page"):

- A C++ class in the mod `DrawerController : public QObject, public AbstractController`, with a local declaration
  of `AbstractController` reproducing the exact virtual order and data layout above. Override `navSection()`
  → 6 and `loadView()` → create our view and store it in the QPointer. `__dynamic_cast` compares type_info by
  name on ARM EABI, so our own `AbstractController` typeinfo should satisfy Nickel's casts; alternatively link
  the typeinfo/ctor weakly from libnickel (`_ZTI18AbstractController`, `_ZN18AbstractControllerC2Ev`).
- Open: `MainWindowController::push(controller+8, false)`. Back (header chevron or our own back row):
  `MainWindowController::pop(controller+8)`. Tab taps and Home key (`clear()`) pop us like any Nickel page.
- Lifecycle: Nickel calls `viewWillBeDestroyed` and the `QObject` gets `destroyed`-tracking; we delete the
  controller ourselves after pop (Nickel logs "You're responsible for the lifecycle of this controller!").

Route B (simpler, already used by NickelHardcover): `pushView(ourWidget)` from the More tab. The tab bar stays
as it was (visible, More active) because `pushView` does not call `updateNavFooter`. Back = `popView(ourWidget)`.
Drawback: Nickel's controller stack does not know about us; a tab tap or Home key pops the controller **under**
us and leaves our widget orphaned in the `QStackedWidget`. We must watch `QStackedWidget::currentChanged` (or
NickelHardcover's "current view changed" approach) and `popView` ourselves when we stop being current.

Risks: route A depends on reproducing the vtable layout exactly (ABI mistake = crash on open, caught at G1-style
first run); route B leaks/ghosts if the cleanup watcher misses a path. Recommendation: prototype A first (2b.2),
keep B as the fallback inside the native-view plan, and plan B of the gate (N3Dialog) as last resort.

Open questions: whether `showStatusBarAsNecessary` needs `showTitle`/`showIndicators` properties on our view
(copy MoreView's: `showIndicators=true`, `showStatusBottomLine=false`, `showTitle=true`); whether
`allowedOrientations` should return portrait only; header widget for the back chevron (Wishlist uses
`TextHeader`; Settings pages use the N3 dialog header — not yet traced).

## Q3 — Hard page-turn keys

**Verdict: feasible.** A pushed view gets page-forward/back as ordinary arrow-key presses.

- `Nickel3Application::notify(QObject*, QEvent*)` runs a chain of static key filters on key events:
  `StylusManager`, `PowerKeyFilter`, `HallSensorFilter`, `HomePageKeyFilter`, `FrontLightKeyFilter`,
  **`TactileButtonFilter`**, `AVRCPButtonFilter`.
- `TactileButtonFilter::filter(QObject*, QKeyEvent*)`: ignores events while USB gadget mode is on or the sleep
  view is visible. The two page buttons arrive as **`Qt::Key_F23` (0x01000046) and `Qt::Key_F24`
  (0x01000047)**. It reads `ReadingSettings::invertPageTurnButtons()` (the "swap buttons" setting) and the
  current screen orientation, and maps F23/F24 to one of `Key_Left` (0x01000012), `Key_Up` (0x01000013),
  `Key_Right` (0x01000014), `Key_Down` (0x01000015). It then builds a new `QKeyEvent` with the mapped key and
  sends it to `MainWindowController::currentView()` (when the original receiver is the main window) or to
  `N3Dialog::content()` (when the receiver is an N3Dialog), otherwise to the original receiver.
- Nickel's paged views all decode it the same way, e.g. `N3SettingsPagedView::keyPressEvent`,
  `AwardsView::keyPressEvent`, `WirelessNetworksView::keyPressEvent`: **Left or Up → previous page;
  Right or Down → next page**; then pass to the base `keyPressEvent`.

What our view must do: be the current widget of the window stack (Q2 either route) and override
`keyPressEvent(QKeyEvent*)` (we have real Qt, so a normal subclass works; an event filter also works). Handle
Left/Up as back and Right/Down as forward; do not re-read the swap setting (Nickel already applied it and the
orientation). Ignore keys in dialogs/menus (they go to `N3Dialog::content()`, not to us). F23/F24 should not
be handled directly.

Open questions: what the `QMetaObject::cast` target class in `TactileButtonFilter` is (very likely the main
window); behaviour when a child of our view (e.g. a `TouchLineEdit`) has focus — the remapped key then goes to
that child, so the drawer list should not hold focus.

## Q4 — Native widgets for the UI kit

**Verdict: feasible with risk** (all constructors exported in 5/5; look and keyboard-outside-dialog unproven).

- **List rows (More style):** `IconLeftButton(QWidget*)` + `setPixmap` + `setText`, signal
  `IconButtonBase::tapped()`. Gives exactly the More-tab row look.
- **Settings rows:** `SettingContainer(QWidget*)` (signal `tapped()`, `setShowBottomLine(bool)`),
  `SettingItemBase(QWidget*)` (`setLabel`, `setActionWidget(QWidget*)`, `setAcceptTaps`),
  `SettingItemWithCheckBox`, `SettingItemWithMenu`, `SettingItemWithDropDown`, `SettingItemWithLabel`.
  NickelHardcover's `checkboxrow`/`menurow`/`staticrow` use `SettingContainer` + `TouchLabel` + `TouchCheckBox`.
- **Toggles:** `ToggleSwitch(QWidget*)` (`setState(bool)`, `isOn()`, `toggle()`, signal `toggled(bool)`,
  `setText(QString,QString)`), and the composite `SettingItemWithToggleSwitch(QWidget*)` (`setOn(bool)`,
  signals `toggled(bool)`, `tapped()`).
- **Headers:** `TextHeader(QWidget*)` (`setText`, `label()`, base `TextHeaderBase` with `tapped()`,
  `setPixmap`, `showSeparator`), variants `SmallTextHeader`, `MediumTextHeader`; used by `Ui_WishlistView` and
  reading menus. `HeaderBarLabel(QWidget*)` (`setElidedText`) is only used by `DragonMenuBarLabel` — not a page
  header. `N3SettingsHeader`/`HeaderView` do not exist in these builds.
- **Labels/buttons:** `TouchLabel(QWidget*, Qt::WindowFlags)` (+`setHitStateEnabled`), `N3ButtonLabel(QWidget*)`
  (`setPrimaryButton(bool)` / property `primaryButton`), `TouchCheckBox(QWidget*)`.
- **Keyboard:** `KeyboardFrame(QWidget*)` is a plain widget constructed by `Ui_N3Dialog`,
  `Ui_ConfirmationDialog`, `Ui_WebStoreKeyboard` and `Ui_N3BrowserView` — so it is not tied to N3Dialog.
  `KeyboardFrame::createKeyboard(KeyboardScript, QLocale const&)` cleans up any previous keyboard, sets the
  frame locale, builds a `SearchKeyboardController` through `SearchKeyboardControllerFactory::localizedKeyboard`
  and shows it. Wire it with `KeyboardReceiver(QLineEdit*, bool)` + `SearchKeyboardController::setReceiver` /
  `setGoText`; the controller emits `commitRequested()`. A `TouchLineEdit` emits `tapped()`; NickelHardcover
  connects that to its own "show keyboard". `N3Dialog::showKeyboard()` only sets the custom property
  `showKeyboard=true` (QSS shows the frame) and re-activates the layout. In our own view we would show/hide the
  `KeyboardFrame` ourselves. Script 0 is what NickelHardcover passes; the layout is chosen from the locale.
- **Dialogs:** `N3DialogFactory::getDialog(QWidget* content, bool)`, `N3Dialog::enableFullViewMode()` (sets
  custom property `fullViewMode=true`), `setTitle`, `enableBackButton`, `keyboardFrame()`, signals
  `closeTapped()`/`backTapped()`. Confirmations: `ConfirmationDialogFactory::getConfirmationDialog`,
  `showOKDialog`, `showErrorDialog`, `showTextEditDialog`.
- **Toasts:** `MainWindowController::toast(QString const&, QString const&, int)` forwards to
  `N3ToasterController::queueMeSomeToast` (shows immediately if idle, else queues, capped by
  `ApplicationSettings::numMaxToasts`). The int is a duration (40+ Nickel callers; unit still unverified).
- **E-ink refresh:** no "N3FullScreenRefresh" class. Kobo's patched Qt carries refresh hints as **custom
  `Qt::WidgetAttribute` values 0x80–0x9a** set with `QWidget::setAttribute` before a `repaint()`:
  `AddToCollectionDialog::flashWidget` sets attribute 0x80, `repaint()`, clears 0x80 (a flashing full update of
  that widget). `QuickTurnEventFilter::doRepaint` clears 0x80–0x9a, then sets combinations (0x82, 0x83, 0x86,
  0x88, 0x8f, 0x95, 0x96, 0x97, 0x80) before `repaint()` — fast/partial waveforms for page flipping. The flags
  reach `libkobo.so`'s `KoboScreenNXP::setDirty(QRect, unsigned)` / `updateModeFromQt` / `ditherModeFromQt`.
  Our kit can therefore request "flash this widget" with attribute 0x80 + `repaint()`, using only Qt API.

Open questions: meaning of each attribute 0x81–0x9a (needs the patched QtGui/QtWidgets from the device rootfs,
not in KoboRoot.tgz); whether `KeyboardFrame` outside an N3Dialog lays out correctly (prototype 2b.4);
`KeyboardScript` enum values and how kobo-tieng-viet's Vietnamese layout plugs into
`SearchKeyboardControllerFactory`; toast duration unit.

## Q5 — Qt6 upgrade prompt (4.38.23828)

**Verdict: understood; the drawer can warn without hooking anything** (read settings only).

Settings involved (`Kobo eReader.conf`):

| Section | Key | Meaning |
|---|---|---|
| `[ApplicationPreferences]` | `AvailableQt6Upgrade` | URL of the Qt6 (firmware 5.x) package from the last upgrade check (`UpgradeCheckCommand::onCheckResult` writes it) |
| `[ApplicationPreferences]` | `SkippedQt6UpgradeUrl` | URL the user declined |
| `[ApplicationPreferences]` | `Qt6UpgradeConfirmedByUser` | true after "Proceed"; the next sync then accepts the upgrade without asking |
| `[ApplicationPreferences]` | `Qt6UpgradeDialogVersion` | dialog version; a change re-arms the note |
| `[DialogSettings]` | `qt6UpgradeNoteDialogShown` | whether the note was shown |

Decision flow (paraphrased):

1. `HomePageController::viewWillAppear` → `maybeShowDialogs()` → `maybeShowQt6UpgradeDialog()` first (before
   Kobo Plus promo, Instapaper). It starts `Qt6UpgradeWorkflowManager::startWorkflow()` when
   `AvailableQt6Upgrade` is non-empty **and** `qt6UpgradeNoteDialogShown` is false. The workflow needs three
   notes from the Activity DB (`TYPE_QT6UPGRADE`); if any is missing it logs and does nothing.
2. During sync, `UpgradeCommand::onQt5UpgradeResult`: if no Qt6 package is offered, or its URL equals
   `SkippedQt6UpgradeUrl`, it logs "Qt6 upgrade not available or skipped; proceeding with Qt5 update." and
   continues with the normal 4.x update. Otherwise it clears `SkippedQt6UpgradeUrl` and goes the Qt6 way:
   `maybeStartQt6Confirmation()` → if `Qt6UpgradeConfirmedByUser` is true it resets it and **accepts
   immediately**; else it posts a Qt6 activity card and calls `UpgradeManager::requestQt6UpgradeConfirmation`,
   which (non-silent sync) resets `qt6UpgradeNoteDialogShown` and starts the workflow, or (silent sync)
   rejects.
3. In the workflow: **Skip** and **Cancel** (`onSkipTapped`, `onCancelTapped` → `onSkipHandling`) copy
   `AvailableQt6Upgrade` into `SkippedQt6UpgradeUrl` and emit `upgradeConfirmationRejected`. **Proceed**
   (`onProceedTapped`) sets `Qt6UpgradeConfirmedByUser=true`, clears `SkippedQt6UpgradeUrl` and starts a sync.
4. `IOUtil::isQt6UpgradePending(Device const&)`: true when `.kobo/KoboRoot.tgz` exists and
   `update-extractor <path> --check-update-tar` prints 1 (a downloaded Qt6 package is staged).
   `IOUtil::removeQt6UpgradeZip` also exists.

How to decline safely (for the user guide): tap **Skip** (or Cancel) on the note. That stores the offered URL as
skipped; Kobo asks again only when it publishes a different Qt6 URL. Keep auto-update off. Never tap Proceed:
it arms an automatic accept on the next sync.

Drawer warning (no hook needed, via `Settings::getSetting` or by reading the conf file):

- **info:** `AvailableQt6Upgrade` non-empty and equal to `SkippedQt6UpgradeUrl` → "declined, OK".
- **warn:** `AvailableQt6Upgrade` non-empty and not skipped → "Kobo will ask you to install firmware 5.x;
  tap Skip".
- **danger:** `Qt6UpgradeConfirmedByUser=true` or `IOUtil::isQt6UpgradePending` → "a 5.x upgrade is armed or
  staged; do not sync/reboot, follow the recovery note".

Builds 23552–23697 do not have this mechanism. They have an older, flag-gated
`OptionalQT6UpgradeDialog` (`ChooseSetupTypeController::maybeShowOptionalQT6Upgrade`,
`OneStoreServiceSettings::getShowOptionalQT6AccessibilityUpgrade`, `DevSettings::showOptionalQT6AccessibilityUpgrade`),
not decompiled yet. **Correction for `docs/DECISIONS.md`:** the `maybeShowQt6UpgradeDialog` /
`requestQt6UpgradeConfirmation` / `skippedQt6UpgradeUrl` symbols are 23828-only; 23697 has the older
`OptionalQT6UpgradeDialog` family (also in 23552–23684). The Qt6 warning symbols must be optional per build.

Open questions: safe recovery steps if `Qt6UpgradeConfirmedByUser` is already true (resetting the key
before the next sync is the obvious candidate, but writing Nickel's conf while Nickel runs needs testing);
the older OptionalQT6 path in 23552–23697.

---

## Firmware coverage

All Q1–Q4 symbols listed in `symbols-required-draft.yaml` are exported by **5/5** builds with identical mangled
names. Q5's Qt6 symbols: **23828 only**. Behaviour was decompiled on 23828 only.

## Recommendation for gate G2: **GO (conditional)**

Static evidence supports the native drawer: the More tab can take an extra row via a constructor hook, the tab
bar is window-owned and kept for any controller whose `navSection()` is 6, page keys arrive as arrow keys with
Nickel's swap setting already applied, and every widget we need is exported in all five builds. G2 still needs
the on-device prototypes (A: push with tab bar; C: keyboard with Vietnamese layout) before it is passed.

Order of work: (1) More-row hook + route B push (lowest ABI risk) to see the look on device; (2) route A
`AbstractController` subclass for correct tab-switch/Home behaviour; (3) keyboard in our own view.

### Plan B assessment (full-screen N3Dialog)

Proven by NickelHardcover on these builds (`getDialog` + `enableFullViewMode` + `pushView`). Keyboard hosting
comes for free (`keyboardFrame()` / `showKeyboard()`); page keys are delivered to `N3Dialog::content()` by
`TactileButtonFilter`, so paging still works. Costs: no tab bar (full-view dialog covers the window), title bar
and close/back from the dialog rather than a native More sub-page, and the dialog must be cleaned up on
view changes. It is a solid fallback, and partly reusable: if route A fails only for the keyboard, the text-entry
screens can still open as N3Dialogs from the native drawer.

## Top risks

1. **ABI of a mod-side `AbstractController` subclass** (vtable order, multiple-inheritance offset 8, RTTI cast
   in `push`/`pop`). Mitigation: route B first; unit-test layout with static asserts; first run with the user
   present.
2. **Orphaned raw-pushed views** (route B) when the user taps a tab or Home. Mitigation: watch the stack's
   current widget and self-pop; or move to route A.
3. **Keyboard outside an N3Dialog** and the Vietnamese layout path (`SearchKeyboardControllerFactory`) are
   unproven; fallback is an N3Dialog for text entry only.

---

## Prototype build (2026-10-08, `mod/drawer-proto/`, not yet run on device)

`libnddrawer.so` (NickelHook, NickelTC image pinned as in `mod/hello`), outputs `build/mod/libnddrawer.so`
and `build/mod/KoboRoot-nddrawer.tgz`. Prototype A = More row (flag `.adds/nickeldrawer/proto-a`),
prototype C = empty drawer page (flag `proto-c`; `proto-c-pushview` selects route B). Test script:
`mod/drawer-proto/README.md`. Static checks: `mod/drawer-proto/check-symbols.sh` (passes).

### Route chosen for C: route A (controller push), route B kept behind a flag

Further static work on 23828 removed the main uncertainties of route A:

- Base implementations are trivial and reproduced in the mod: `size()` returns `QSize(-1,-1)` (sret),
  `viewWillAppear/Disappear/BeDestroyed` empty, `allowedOrientations()` returns 5, `navSection()` 0, ctor
  zeroes the `QPointer` at +4/+8, dtor releases the QPointer d-ptr. Size 0xc confirmed.
- `MoreController`'s secondary vtable (offset-to-top −8) has exactly 9 slots: D1, D0, size, viewWillAppear,
  viewWillDisappear, viewWillBeDestroyed, allowedOrientations, navSection, loadView. The built mod's
  `DrawerController` secondary vtable was dumped from the `.so` and matches this order slot for slot,
  with offset-to-top −8 and a `__vmi_class_type_info` listing `QObject` @0 and `AbstractController` @8.
- Nickel's RTTI use: `push`, `pop`, `prepareView`, `ensureViewLoaded` and `view<QWidget*>` call
  `__dynamic_cast(controller, &typeinfo(AbstractController), &typeinfo(QObject), -2)`. The mod compiles
  only `drawer_controller.cc` with `-frtti`; type_info equality on ARM EABI is by name string, and our
  `_ZTS18AbstractController` is `"18AbstractController"` without the `*` local-type prefix, so Nickel's
  casts match our class. `typeinfo for QObject` is imported from QtCore (libnickel imports it too).
- **Lifecycle (correction to Q2):** after `loadView`, `ensureViewLoaded`/`view<QWidget*>` call
  `controllerAsQObject->setParent(view)` (disassembly at 0xad6d6e–0xad6d78 in the 23828 file: r0 = cast
  result, r1 = the view from the QPointer). The controller is a child of its view, so `popView`'s
  `deleteLater()` of the view also deletes the controller. The mod must **not** delete the controller
  after `pop`. "You're responsible for the lifecycle of this controller!" is logged only when the
  controller is not a QObject.
- Runtime guards: `static_assert(sizeof(AbstractController)==0xc)`, QObject/QPointer sizes, and a runtime
  check that the `AbstractController` subobject sits at +8 before calling `push`. After `push` the mod
  verifies the view landed in the window stack and that the controller was re-parented; otherwise it
  cleans up and toasts.
- `DrawerController` deliberately has no `Q_OBJECT`: moc would make `metaObject()` the key function and
  emit the vtable in a `-fno-rtti` object (null typeinfo → Nickel's `__dynamic_cast` would crash).

Route B (`pushView`/`popView`, as NickelHardcover) is compiled in and selected by `proto-c-pushview`; it
watches `QStackedWidget::currentChanged` and pops the drawer when another view becomes current.

### Symbols used (all optional nh_dlsym/nh_hook; 10/10 present in all 5 builds)

| Symbol | Use |
|---|---|
| `_ZN8MoreViewC1EP7QWidget` | hook (A) |
| `_ZN14IconLeftButtonC1EP7QWidget`, `_ZN14IconLeftButton9setPixmapERK7QPixmap`, `_ZN14IconLeftButton7setTextERK7QString` | row (A) |
| `_ZN20MainWindowController14sharedInstanceEv`, `_ZN20MainWindowController5toastERK7QStringS2_i` | window, toast |
| `_ZN20MainWindowController4pushEP18AbstractControllerb`, `_ZN20MainWindowController3popEP18AbstractController` | C route A |
| `_ZN20MainWindowController8pushViewEP7QWidget`, `_ZN20MainWindowController7popViewEP7QWidget` | C route B |

Signal by name (no dlsym): `IconButtonBase::tapped()`. Qt only: `findChild("moreContainerLayout")`,
`QBoxLayout::insertWidget` after `betaFeaturesButton`, `QSettings` read of
`[ApplicationPreferences] CurrentLocale` for the label, widget attribute 0x80 + `repaint()` on page flip.
Pixmap: Nickel resource `:/images/reading/beta_features.png` (placeholder).

### Residual risks for the first boot

1. **Route A ABI at open time** (not at boot): any mismatch in how Nickel uses the controller beyond
   what was decompiled (e.g. other `dynamic_cast` targets in `prepareView`, which returns null for us and
   falls back to the QPointer path) would crash on the first tap. Boot itself is unaffected; recovery =
   delete `proto-c`, try route B.
2. **`MoreView` hook runs on every More open.** A wrong assumption in `IconLeftButton` construction (256-byte
   calloc, 0x58 used) would crash when opening More, after the failsafe window. Recovery = delete `proto-a`
   or the uninstall flag over USB (Nickel restarts after a crash, USB still works).
3. **Plain Qt widgets on Nickel**: the drawer page uses `QLabel`/`QPushButton`, not Nickel widgets; fonts,
   sizes and touch-to-click synthesis for the back button are unproven (fallback: More tab/Home closes it).
4. **Tab-bar state with route B** is not re-evaluated (`pushView` skips `updateNavFooter`).
5. Behaviour was decompiled on 23828 only; the other four builds share the exported names, not
   verified code.
6. Shared uninstall flag with `ndhello`: with both installed, one uninstall run removes only one mod.

**Device result 2026-10-09:** the row shows, but tapping it crashed Nickel. Cause: our row children were built
with `-fno-rtti` (null typeinfo), and `IconButtonBase::tapGesture` `__dynamic_cast`s every child of the
button on press. Both mods are now built with `-frtti`, and a vtable check was added; see
`re/notes/protoA-crash.md`.

---

## Drawer host build (2026-10-09, `mod/drawer/`, not yet run on device)

`libnickeldrawer.so` (NickelHook name `nickeldrawer`), outputs `build/mod/libnickeldrawer.so` and
`build/mod/KoboRoot-nickeldrawer.tgz`. Evolved from `mod/drawer-proto/` (left untouched). Architecture,
flags, degradation table and the first-boot script: `mod/drawer/README.md`.

Structure: `nickel_bridge.cc` is the only file naming libnickel symbols (hook + dlsym tables, firmware
check, fail-soft wrappers); `check-symbols.sh` fails if any other source does. The route A controller is the
prototype's `AbstractController` mirror unchanged; the rebuilt `.so` again shows `DrawerController`'s
secondary vtable with offset-to-top −8 and 9 slots, typeinfo names `18AbstractController` /
`16DrawerController`. Kill switch `.adds/nickeldrawer/disable` keeps the bridge inert; route B via
`.adds/nickeldrawer/pushview`; uninstall flag `.adds/nickeldrawer/uninstall` (same path as the prototype:
uninstall `nddrawer` first).

### Symbols added since the prototype (all optional, 5/5 builds, mangled and demangled dumps)

| Symbol | Use | Source of the calling convention |
|---|---|---|
| `_ZN25ConfirmationDialogFactory12showOKDialogERK7QStringS2_` | result / Info dialog | NickelMenu (body is rich text, so the mod HTML-escapes and turns `\n` into `<br>`) |
| `_ZN25ConfirmationDialogFactory21getConfirmationDialogEP7QWidget` | confirm (Remove, Wi-Fi) | NickelHardcover (returns a `QDialog`; `accepted()`/`rejected()`, `open()`) |
| `_ZN18ConfirmationDialog8setTitleERK7QString`, `…7setTextERK7QString`, `…19setAcceptButtonTextERK7QString`, `…19setRejectButtonTextERK7QString`, `…15showCloseButtonEb` | confirm texts | NickelHardcover |
| `_ZN23WirelessWorkflowManager14sharedInstanceEv`, `…20isInternetAccessibleEv`, `…15connectWirelessEbb` | `needs_wifi` → connect, poll, retry once | NickelMenu `autoconnect` passes `(true, false)`; NickelHardcover polls `isInternetAccessible` 1 s × 30 because the signal is unreliable |
| `_ZN19PlugWorkflowManager14sharedInstanceEv`, `…4syncEv` | quick rescan | NickelMenu `rescan_books` |
| `_ZN19PlugWorkflowManager18onCancelAndConnectEv`, `…9unpluggedEv` | full rescan | NickelMenu `rescan_books_full` (it sleeps 1 s between them; the mod uses a 1 s timer) |

Total 25 names checked (24 hook/dlsym + the `IconButtonBase::tapped()` signal). Not found in any build:
a `WirelessWorkflowManager::connectingFailed()` export (NickelHardcover connects it by name; it is probably
an unexported signal), so the mod relies on polling only.

### Open questions for the device run

1. Does `TactileButtonFilter` deliver page keys to our view while Nickel's `ConfirmationDialog` is open?
   It is not an `N3Dialog`, so Q3's redirect to `N3Dialog::content()` may not apply. The mod blocks its own
   input while its plain dialogs and pending actions are up, but cannot see when Nickel's OK dialog closes.
2. `IconLeftButton::setText` with an embedded `\n` for the More row's status line (look not verified).
3. Touch → plain Qt widgets: `PressArea` accepts both `QTouchEvent` and synthesized mouse events.
4. Whether `QPixmap` loads `icon.svg` (needs the svg image-format plugin in Nickel's Qt).

---

## Phase 4 build: native app screens (2026-10-09, `mod/drawer/`, not run on device)

Behind the flag `.adds/nickeldrawer/screens` (default off). Tapping an app whose model item has `screen: true`
opens a `ScreenView` as a **child of our own drawer view** (it covers it), so Nickel's window stack is not
touched for screens; page keys still arrive at the drawer view (TactileButtonFilter → `currentView()`, Q3)
and an event filter on it hands them to the screen. Taps on the plain widget are handled from a 0 ms timer
(protoA lesson); every new class has typeinfo (`check-vtable-typeinfo.sh`: 24 vtables).

### Keyboard for `field` (Q4 path, unproven on device)

Proven recipe of NickelHardcover on these builds, not our own `KeyboardFrame`: `N3DialogFactory::getDialog(content,
true)` → `N3Dialog::setTitle` → `MainWindowController::pushView(dialog)` → `N3Dialog::keyboardFrame()` →
`KeyboardFrame::createKeyboard(frame, 0, QLocale())` (returns the `SearchKeyboardController`) → `KeyboardReceiver(QLineEdit*,
false)` in a zeroed 256-byte block (NickelHardcover uses 128; the receiver is never freed) →
`SearchKeyboardController::setReceiver(receiver, false)`, `setGoText` → `N3Dialog::showKeyboard()`. Signals by name:
`KeyboardController::commitRequested()` (Go) and `N3Dialog::closeTapped()` (cancel); our receiver is a parentless
QObject. The line edit is a plain `QLineEdit`. Close = `popView(dialog)`. The dialog carries the property
`nickelDrawerOverlay`, so route B's stack watcher does not close the drawer when it becomes current. Flag
`no-keyboard` forces the "edit on your phone" notice.

New symbols (all optional, `nh_dlsym`; 10/10 present in the 5 builds' mangled and demangled dumps, checked by
`check-symbols.sh`, now 35 names): `_ZN15N3DialogFactory9getDialogEP7QWidgetb`, `_ZN8N3Dialog8setTitleERK7QString`,
`_ZN8N3Dialog13keyboardFrameEv`, `_ZN8N3Dialog12showKeyboardEv`,
`_ZN13KeyboardFrame14createKeyboardE14KeyboardScriptRK7QLocale`, `_ZN16KeyboardReceiverC1EP9QLineEditb`,
`_ZN24SearchKeyboardController11setReceiverEP16KeyboardReceiverb`, `_ZN24SearchKeyboardController9setGoTextERK7QString`;
signals `_ZN18KeyboardController15commitRequestedEv`, `_ZN8N3Dialog11closeTappedEv`.

### Fonts

No libnickel API for UI fonts was identified. The screen uses the first installed family of Nickel's own fonts
as registered in-process with `QFontDatabase` (sans: Avenir Next, Avenir, Gill Sans, Kobo Nickel; serif: Georgia,
Amasis, Caecilia, Malabar; mono: Courier New, Courier), else Qt's default with a style hint. The chosen names
are written to the debug log; confirm them on the device.

### Open questions for the device run

1. Does Nickel's `QLocale()` select the Vietnamese keyboard (kobo-tieng-viet) or must the UI locale be passed?
2. Does `popView` of the keyboard dialog make the drawer view current again on route A (QStackedLayout picks
   the previous index), and does Home with the dialog open leave it orphaned? (The editor closes it when the
   screen view is destroyed.)
3. Partial refresh of a changed region (`update(QRegion)`) without attribute 0x80: waveform quality on the toggle.

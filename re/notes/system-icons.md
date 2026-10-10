# Nickel's system icons used by the mod (firmware 4.38.23828)

Date: 2026-10-10. Static only: `strings` over `libnickel.so.1.0.0` (405 `:/images/...` paths referenced by code),
the extracted QSS (`~/.cache/nickeldrawer/qss/`), the headless-Ghidra decompilation in
`~/.cache/nickeldrawer/decomp/` and a parse of the Qt resource trees compiled into the `nickel` binary (image sizes).
No Kobo image is copied into the repo; the mod loads them at run time with `QPixmap(":/images/...")`.

Rule (user decision): we never draw icons. Order for every icon: Nickel's resource -> Tabler Icons PNG (MIT)
installed at `/mnt/onboard/.adds/koboapps/icons/ui/<name>-<64|96>.png` -> no icon (the control shows its text label,
or is hidden, and the log says so). Code: `mod/common/system_icon.{h,cc}`.

## Where the images live, and their size on the Libra 2

The images are not in libnickel: the `nickel` executable carries five rcc resource trees (one per screen class; the
names are UTF-16 in each tree's name table). The Libra 2 build loads the 300 dpi set plus a small override set:

- a device debug log of ours read `:/images/reading/back_arrow.png 56x56` and `beta_features.png 50x50`;
- `back_arrow.png` is 56x56 only in the tree at file offset 0xb29d1a (sizes per tree: 72 / 56 / 48 / 48 px);
- `beta_features.png` is 50x50 only in the small override tree at 0x1461341 (325 files, also `settings_gear.png`
  80x80, `help_icon.png` 50x50, the More-tab icons).

So on the Libra 2 class: header icons 56-70 px, More-row icons 50 px (the gear may be 80 px from the override set).
The mod draws Nickel's images 1:1 (as Nickel does) centred in its 120 px tap targets, and logs the loaded size
once: `icon <name>: Nickel|Tabler|none WxH` (DebugLog, also nh_log/syslog).

## Chosen icons

| Our use | Nickel resource (pressed) | Size on Libra 2 | Where Nickel uses it (evidence) | Tabler fallback |
|---|---|---|---|---|
| Header back, full-view back | `:/images/reading/back_arrow.png` (`back_arrow_hit.png`) | 56 | `Ui_WishlistView::setupUi` sets it as `TextHeader` `deselectedPixmap` (decomp `header.c`); `N3Dialog::N3Dialog` sets `back_arrow_hit.png` as the back `TouchLabel`'s selected pixmap (`n3dialog.c`) | `back-64` |
| Close ✕ (tweaks overlay) | `:/images/search/global_close.png` (`global_close_hit.png`) | 56 | `N3Dialog::N3Dialog` close `TouchLabel` (`n3dialog.c`, `settings.c`); `N3Dialog.qss` | `close-64` |
| Header `settings`; More row "Tinh chỉnh" | `:/images/reading/settings_gear.png` (`settings_gear_hit.png`) | 70 (80 override) | `Ui_MoreView::setupUi` (0x676e14): the Settings `IconLeftButton`'s `pixmap` property (`more.c`); the same image the More tab's Settings row shows | `settings-64` (header), `adjustments-horizontal-64` (More row) |
| Header `import` | `:/images/library/library_download_lg_off.png` (`library_download_lg_hit.png`) | 70 | library download button (string in libnickel; a download arrow in a circle) | `import-64` |
| Header `refresh` | `:/images/navigation/browser_reload.png` (`browser_reload_hit.png`) | 70 | web browser toolbar reload (string in libnickel) | `refresh-64` |
| Header `add` | `:/images/reading/plus.png` (`plus_hit.png`) | 56 | reading settings "+" steppers (string in libnickel, with `minus.png`) | `add-64` |
| Gallery current-picture badge | `:/images/library/checkmark_hit.png` (white check on a black disc) | 56 | `AddToShelfWidget` QSS: `QCheckBox::indicator:checked` (unchecked: `menu_plus_off.png`) | `check-64`, drawn white on the layout's black disc |
| More row "Ứng dụng" | none fits (no apps/grid image among the 405) | — | — | `icons/apps.png` (Tabler `apps`, 50 px, installed by koboapps) |
| Drawer list / grid toggles | none (Nickel's library switches views from a text menu) | — | — | `list-64`, `grid-64` |
| Gallery picture placeholder, empty gallery | none (`reading_image_view.png` is a pan control, not a picture) | — | — | `picture-96` |
| App without its own icon | none | — | — | `apps-96` |
| App-screen toggle (on / off) | `:/images/widgets/switch_on.png` / `switch_off.png` (no `_hit`) | 94x56 | `ToggleSwitch::updateImage` loads `":/images/widgets/switch_%1.png"` with `"on"`/`"off"`; `SettingItemWithToggleSwitch` (Wi-Fi, Bluetooth, ... settings rows) owns one (see below) | `toggle-on-64`, `toggle-off-64` (Tabler `toggle-right` / `toggle-left`) |

All `_hit` images are Nickel's pressed look: the glyph in white on a black disc of the same size. With a Tabler icon
the press inverts the icon in a circle (`PressArea::HIGHLIGHT_CIRCLE`), which gives the same look.

## Nickel's settings switch (ToggleSwitch)

Found 2026-10-10 the same way (libnickel strings, headless Ghidra `DecompileByName` of `ToggleSwitch::*`,
`Ui_ToggleSwitch::setupUi`, `SettingItemWithToggleSwitch::SettingItemWithToggleSwitch`; output in
`~/.cache/nickeldrawer/decomp/toggleswitch*.c`, not in the repo; the rcc trees of `nickel` for sizes and QSS).

- Images: the only switch images among the `:/images/` strings are `:/images/widgets/switch_%1.png`; the `%1` is
  `"on"` or `"off"` (`ToggleSwitch::updateImage`, 0x112b660). No `_hit` / pressed variant exists, and the
  constructor calls `TouchLabel::setHitStateEnabled(false)` on both labels, so Nickel shows no pressed switch image.
  Other on/off images (`menu_checkbox_*`, `settings_checkbox_*` 34 px, `radio_btn_*` 56 px) are checkboxes and radio
  rings of other widgets, not the settings switch.
- Sizes per tree: 122x72 (0x8b4dcf), **94x56 (0xb29d1a, the Libra 2 / Storm set, same tree as the 56 px back
  arrow)**, 76x46 (0xca60c9), 63x36 (0xd1b134); not in the 0x1461341 override set, so the Libra 2 loads 94x56.
- Layout (`Ui_ToggleSwitch::setupUi`, object names read from the string table): `ToggleSwitch` > `container`
  (QHBoxLayout) holding `textContainer` (a spacer, then the `text` TouchLabel, alignment 0x82 = AlignRight |
  AlignVCenter) and then the `toggle` TouchLabel with the image. So the state word sits **left of** the image,
  right-aligned. `SettingItemWithToggleSwitch` sets the texts to `tr("On")` / `tr("Off")` (`ToggleSwitch::setText`,
  shown by `updateText`) and makes the switch the row's action widget (`SettingItemBase::setActionWidget`: added at
  the right end of the row's layout).
- `:qss/ToggleSwitch.qss` (rcc tree 0x19c9088, zlib): `#container` spacing 25 px on Storm; `#text` Avenir,
  `text-transform: uppercase`, 29 px on Storm (normal weight). So Nickel shows "ON" / "OFF" in
  uppercase next to the image; the mod matches it (`QString::toUpper` of the host's on/off text, 29 px, regular).
- Used by: `#bluetoothSwitch { qproperty-showBottomLine: true }` (a `SettingContainer` row: `border-bottom: 1px solid
  black`), `wifiSwitch`, `timeFormatSwitch`, `parentalControlSwitch`, `brightnessAutoSwitch` (libnickel strings).
- Present in all five target builds: `switch_on.png`, `switch_off.png` and `ToggleSwitch.qss` in the `nickel` name
  tables, `:/images/widgets/switch_%1.png` in libnickel (23552, 23648, 23684, 23697, 23828).

The mod's app-screen toggle (`mod/drawer/screen_layout.cc` `drawToggle`, `screen_canvas.cc` `paintSwitch`,
geometry in `screen_logic.cc` `toggleRow`) draws these images 1:1 in a 94x56 box at the content edge, the state
word 25 px to its left. The row press (whole row inverts) is ours and unchanged.

## Risks

- `mod/drawer/check-symbols.sh` checks symbols, not resources: a renamed or removed image on another firmware is
  only seen at run time. Each icon logs its source and size once (`icon back: Nickel 56x56, pressed image`); a
  device test should grep the debug log for `icon ` lines showing `Tabler` or `none`.
- All 13 chosen file names (normal and `_hit`; the two switch images are checked separately above) are in the resource name tables of the `nickel` binary of all five
  target builds (23552, 23648, 23684, 23697, 23828; file-name check, not full paths). The same check
  (`strings libnickel.so.1.0.0 | grep ':/images/'`, plus the UTF-16 file names in `nickel`) should be repeated by
  `firmware-watch` for a new build.

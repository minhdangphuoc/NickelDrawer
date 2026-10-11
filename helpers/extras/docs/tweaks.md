# Tweaks contract v1 (mod ⇄ extras-tweaks)

> Part of the **Extras helper** ([../README.md](../README.md)), not of NickelDrawer. The helper has no link to the drawer or its `appsctl`.

The "Tweaks" window (Vietnamese "Tinh chỉnh"; More tab, "Thêm") replaces the NickelMenu tweak items. The mod only draws the list and applies the optional native action; the Go program `bin/extras-tweaks` owns every setting. The mod runs it as a child process, one JSON document on stdout, `"v":1`, a hard timeout of 3 s per call, all text localized by the program from `locales/<lang>.json` (device `CurrentLocale`; exact locale, language, then English; English is also compiled in).

## `extras-tweaks list`

```json
{"v":1,"items":[
  {"id":"screensaver","label":"Sleep screen","kind":"toggle","value":true},
  {"id":"screenshot-button","label":"Screenshot button","caption":"In the status bar","kind":"toggle","value":true},
  {"id":"screenshots","label":"Screenshots","caption":"With the power button","kind":"toggle","value":true},
  {"id":"auto-usb","label":"Auto USB connect","kind":"toggle","value":false},
  {"id":"legibility","label":"Text legibility","kind":"toggle","value":false},
  {"id":"ip","label":"IP address","kind":"info","value":"192.168.1.5"},
  {"id":"invert-reboot","label":"Invert and reboot","kind":"action","confirm":"Invert the screen colours and reboot now?"},
  {"id":"sleep","label":"Sleep","kind":"action"},
  {"id":"reboot","label":"Reboot","kind":"action","confirm":"Reboot now?"},
  {"id":"library-rescan","label":"Rescan library","kind":"action"}
]}
```

- Items come in the order above, except that `screenshots` is listed only while it is on (see its row below), so the count varies and the mod lays out whatever it receives; `label`, `caption` and `confirm` are localized by the program.
- `caption` (optional, any kind except `info`): a short muted line under the label (one line, elided). A mod that does not know the field ignores it.
- `kind`:
  - `toggle`: `value` is a JSON boolean (the mod shows On/Off and flips it with `apply`).
  - `info`: `value` is a string shown read-only. `ip` is the first IPv4 address of `wlan0`, or the localized "No Wi-Fi" text.
  - `action`: no `value`. With `confirm` the mod asks first (title = label, text = `confirm`) and calls `apply` only on OK.
- A setting that cannot be read is reported as `false` (the screenshot button counts as on); the window never fails because of one tweak.

## `extras-tweaks apply <id>`

Applies one tweak (toggles flip their current value) and replies:

```json
{"v":1,"ok":true,"toast":"Sleep screen on","value":true}
{"v":1,"ok":true,"toast":"Rebooting","nickel":"reboot"}
```

- `toast`: localized text to show. `value`: the new boolean, only for toggles. `nickel`: optional native action the mod performs after the toast: `reboot`, `sleep`, `rescan-library`.
- After a toggle the mod updates the cell from `value`; it does not re-read the list. After any applied toggle it also re-reads the screenshot-off flag, so the status-bar button appears or disappears at once.
- Failure (exit code 2; unknown id, usage mistakes and write errors use the same shape):

```json
{"v":1,"ok":false,"error":"Could not save the setting: permission denied"}
```

- `info` ids and `sleep`/`reboot`/`library-rescan` have no side effect inside the program: the action is the `nickel` field. `invert-reboot` first flips the invert setting, then returns `"nickel":"reboot"`.
- Options: `--locale`, `--conf`, `--locales-dir`, `--flags-dir` (tests and tools); the defaults are the device paths.

## Behaviour of each tweak

| id | Reads / writes | Notes |
|---|---|---|
| `screensaver` | `<root>/.kobo/screensaver` (on) ⇄ `<root>/.kobo/screensaver.disabled` (off), by rename | Off keeps the pictures; Nickel then shows its own sleep screen. Toggling with neither folder present is an error. |
| `screenshot-button` | flag file `.adds/helpers/extras/screenshot-off`: absent = on (default), present = off | Shows or hides the status-bar button live. Replaces the old power-button tweak. |
| `screenshots` | `Kobo eReader.conf` `[FeatureSettings]` `Screenshots=true/false` | Kobo's own power-button screenshots. **Listed only while `true`** (missing or `false`: hidden), so a user who switched it on earlier can switch it off: while on, the power button takes screenshots instead of sleeping. Toggling off writes `false`; the cell stays until the window is reopened, then the list omits it. It cannot be switched on from here. |
| `auto-usb` | `Kobo eReader.conf` `[DeveloperSettings]` `AutoUsbGadget=true/false` | Same setting NickelMenu's `auto_usb_gadget` toggled. |
| `legibility` | `Kobo eReader.conf` `[Reading]` `webkitTextRendering=optimizeLegibility` (on) / line removed (off) | kepub books only; the toast says to reopen the book. Keys verified against libnickel 4.38.23828. |
| `invert-reboot` | `Kobo eReader.conf` `[FeatureSettings]` `InvertScreen=true/false` | Flips, then `nickel: reboot`. |
| `ip` | `wlan0` address | Read-only. |

`Kobo eReader.conf` is `<root>/.kobo/Kobo/Kobo eReader.conf`. Edits are atomic (temp file in the same folder, then rename), keep every other line, section and comment, and create a missing section or key. The program never backs up device files itself; the installer and the user do that. Section and key names are collected in `internal/tweaks/settings.go`; verify them on a device before shipping a new firmware.

## Mod side (libextras)

The row is added on each `MoreView`'s first Show, caught by an application event filter (see "Reaching the More tab" below), after NickelDrawer's row when present, else after Nickel's Beta features row, with Nickel's own Settings gear (`:/images/reading/settings_gear.png`; `icons/tweaks.png` when missing). The window is Nickel's own `N3Dialog` pushed as a controller without the tab bar; details and symbols in [tweaks-modal.md](tweaks-modal.md). There is **no fallback overlay**: if the N3Dialog, controller-push or `closeTapped()` pieces are missing, the row is not added, the log says why and the screenshot button keeps working. `rescan-library` is Nickel's quick rescan (`PlugWorkflowManager::sync`). An unknown `nickel` value or a missing Nickel function shows a toast instead.

## Reaching the More tab

The first build hooked `Ui_MoreView::setupUi(QWidget*)` (NickelDrawer owns the `MoreView::MoreView` hook, and
`nh_dlhook` keeps only one hook per symbol). On the device (4.38.23828) the hook never took: the log said
`libnickel symbol Ui_MoreView::setupUi(QWidget*) missing, tweaks disabled`. Cause: the symbol is defined and exported
by libnickel and has an `R_ARM_JUMP_SLOT`, but its binding is **WEAK** (uic's `setupUi` is inline header code, emitted
as a COMDAT weak definition) on all five builds, and `nh_dlhook` refuses any symbol that is not `STB_GLOBAL`
(`third_party/NickelHook/nh.c`), returning NULL. The old check only looked for the name in `exports.txt` and for a
jump slot, so it passed.

Now the row needs no hook. When the feature is on, `FirstShowFilter` installs one event filter on the application
(at once if `QCoreApplication` exists, otherwise from its constructor through `qAddPreRoutine`). It returns after one
integer compare for every event that is not `QEvent::Show`; for a Show it compares `metaObject()->className()` with
`MoreView` and, the first time per instance (a dynamic property marks it), calls `TweaksRow::addTo`. At that point
the constructor has returned (so NickelDrawer's row is already in the layout) and nothing has been painted, so the row
is laid out before the tab first draws. The filter never consumes the event.

Other options considered: hooking another `MoreView` symbol (`updateVisibleButtons`, `eventFilter`) would tie the
row to call paths we cannot see and still risk a clash with another mod; patching NickelHook to accept WEAK symbols
would change shared vendored code for one helper. `mod/common/check-nickel-bindings.sh` (run by both mods'
`check-symbols.sh`) now checks every hook for a defined FUNC with GLOBAL binding and a jump slot, and every dlsym'd
symbol for a defined, non-LOCAL entry, in the real libnickel of each cached build.

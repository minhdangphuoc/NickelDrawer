# Tweaks contract v1 (mod ⇄ appsctl)

The native "Tinh chỉnh" modal (More tab, "Thêm") replaces the NickelMenu tweak items. The mod only draws the list and applies the optional native action; `appsctl` owns every setting. Transport rules (child process, one JSON document on stdout, `"v":1`, hard timeout of 3 s per call, localized text) are those of [drawer-protocol.md](drawer-protocol.md).

## `appsctl drawer tweaks`

```json
{"v":1,"items":[
  {"id":"screensaver","label":"Sleep screen","kind":"toggle","value":true},
  {"id":"screenshots","label":"Screenshots","kind":"toggle","value":false},
  {"id":"auto-usb","label":"Auto USB connect","kind":"toggle","value":false},
  {"id":"legibility","label":"Text legibility","kind":"toggle","value":false},
  {"id":"ip","label":"IP address","kind":"info","value":"192.168.1.5"},
  {"id":"invert-reboot","label":"Invert and reboot","kind":"action","confirm":"Invert the screen colours and reboot now?"},
  {"id":"sleep","label":"Sleep","kind":"action"},
  {"id":"reboot","label":"Reboot","kind":"action","confirm":"Reboot now?"},
  {"id":"library-rescan","label":"Rescan library","kind":"action"}
]}
```

- Items come in the order above; all `label` and `confirm` texts are localized by `appsctl` (device `CurrentLocale`).
- `kind`:
  - `toggle`: `value` is a JSON boolean (the mod shows On/Off and flips it with `tweak`).
  - `info`: `value` is a string shown read-only (tapping does nothing). `ip` is the first non-loopback IPv4 address, or the localized "No Wi-Fi" text.
  - `action`: no `value`. When `confirm` is present the mod asks first (title = label, text = `confirm`) and calls `tweak` only on OK.
- A setting that cannot be read (missing or unreadable file) is reported as `false`; the modal never fails because of one tweak.

## `appsctl drawer tweak <id>`

Applies one tweak (toggles flip their current value) and replies:

```json
{"v":1,"ok":true,"toast":"Sleep screen on","value":true}
{"v":1,"ok":true,"toast":"Rebooting","nickel":"reboot"}
```

- `toast`: localized text to show. `value`: the new boolean, only for toggles. `nickel`: optional native action the mod performs after the toast: `reboot`, `sleep`, `rescan-library`. Without `nickel` the mod does nothing else.
- After a toggle the mod updates the row from `value`; it does not re-read the list.
- Error (exit code 2, same shape as the rest of the protocol, plus `ok:false` for convenience):

```json
{"v":1,"ok":false,"error":"Cannot write Kobo eReader.conf: permission denied"}
```

  Unknown id uses the same shape (`error` is a localized sentence). Usage mistakes follow the standard `{"error":{"code","message"}}` document of the protocol.
- `info` ids and `sleep`/`reboot`/`library-rescan` have no side effect inside `appsctl`: the action is the `nickel` field. `invert-reboot` first flips the invert setting, then returns `"nickel":"reboot"`.

## Behaviour of each tweak

| id | Reads / writes | Notes |
|---|---|---|
| `screensaver` | `<root>/.kobo/screensaver` (on) ⇄ `<root>/.kobo/screensaver.disabled` (off), by rename | Off keeps the pictures, Nickel then shows its own sleep screen. Toggling with neither folder present is an error. |
| `screenshots` | `Kobo eReader.conf` `[FeatureSettings]` `Screenshots=true/false` | |
| `auto-usb` | `Kobo eReader.conf` `[DeveloperSettings]` `AutoUsbGadget=true/false` | Same setting NickelMenu's `auto_usb_gadget` toggled. |
| `legibility` | `Kobo eReader.conf` `[ApplicationPreferences]` `webkitTextRendering=geometricPrecision` (on) / line removed (off) | Toast says a reboot is needed. |
| `invert-reboot` | `Kobo eReader.conf` `[FeatureSettings]` `InvertScreen=true/false` | Flips, then `nickel: reboot`. |
| `ip` | `wlan0` address | Read-only. |

`Kobo eReader.conf` is `<root>/.kobo/Kobo/Kobo eReader.conf`. Edits are atomic (temp file in the same folder, then rename), keep every other line, section and comment as they are, and create a missing section or key. `appsctl` never backs up device files itself; the installer and the user do that.

Section and key names are collected in `internal/tweaks/settings.go`. They are the Nickel QSettings names behind the NickelMenu `nickel_setting` items of the same name; verify them on a device before shipping a new firmware.

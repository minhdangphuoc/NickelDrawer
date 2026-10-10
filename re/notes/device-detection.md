# Device detection: which Kobo model the drawer runs on

Date: 2026-10-10. Decision: the drawer loads only on models with the tested Libra 2 screen, 1264×1680
(Libra 2, Libra H2O, Libra Colour). Any other model gets the same treatment as an unknown firmware: the
bridge stays inert, logs the reason, and shows one toast ("NickelDrawer does not support this model yet",
vi "NickelDrawer chưa hỗ trợ máy này"). Both the model check and the firmware build check must pass.
The model check runs first. Code: `mod/drawer/device_gate.{h,cc}` (pure rules, host-tested) and
`mod/drawer/nickel_bridge.cc` (`passesModelCheck`, `checkScreenSizeOnce`).

## Why the screen size, not the firmware, decides

`mod/drawer/screen_logic.h` fixes every layout metric to a 1264×1680 panel (300 dpi). The firmware
string 4.38.23828 is shared by the Touch through Elipsa 2E (docs/COMPATIBILITY.md), so the firmware check
alone would let the mod load on a 1072×1448 Clara or a 1404×1872 Elipsa, where the layout is wrong.

## Options looked at

| Source | What it gives | Cost and risk at `init()` | Used |
|---|---|---|---|
| Last field of `/mnt/onboard/.kobo/version` | Kobo device id, e.g. `00000000-0000-0000-0000-000000000388`; the last three digits name the model | The bridge already opens, reads one line and closes this file for the firmware build. Zero extra I/O, no new symbol | **Yes, decides** |
| First field of `.kobo/version` | The serial (`N418…` on a Libra 2) | Its prefix hints at the model but is not a documented mapping, and the serial is personal data we do not want in logs | No |
| `PRODUCT` environment variable | Codename (`io`, `storm`, `monza`) exported by Kobo's `rcS` from `/bin/kobo_config.sh` before it starts Nickel | `getenv` is free, but whether Nickel's environment always carries it (restarts, other launchers) is not verified | Logged only, as a cross-check |
| `/bin/kobo_config.sh` | Same codename, read from the NTX hardware config block | Starting a shell (which reads a raw block device) inside `init()` breaks the "tiny init" rule | No |
| Nickel's `Device::getCurrentDevice()` and model getters | Nickel's own model object | New libnickel symbols to dlsym and verify per firmware; we keep libnickel use to what the drawer needs | No |
| Live Qt screen size (`QApplication::desktop()->screenGeometry()`) | What Nickel's Qt actually draws on | Only valid after Nickel creates `QApplication`, which happens after our `init()`. Checked once on the first `MoreView` | **Yes, second check** |

## Evidence for the device id

- `.kobo/version` is one line: `serial,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000388`.
  The firmware check has read field 3 on the maintainer's Libra 2 since the G1 drill (the mod went active
  on 4.38.23828). The last field is the device id that Kobo's
  upgrade API takes (`docs/plan-detailed.md` §3: `UpgradeCheck/Device/00000000-0000-0000-0000-000000000388`,
  `re/firmware.lock`, `tools/fw-watch.sh`).
- FBInk (`fbink.h` `KOBO_DEVICE_ID_E`) and pgaskin's koboutils name models by the last three digits:
  384 Libra H2O (`storm`), 388 Libra 2 (`io`), 390 Libra Colour (`monza`). `internal/device/device.go`
  already maps them for `appsctl device-name`, and `internal/wallpaper` sizes pictures from the same id.
- The 1264×1680 size of these three panels comes from public spec sheets (docs/COMPATIBILITY.md); only the
  Libra 2 is measured. That is why the live screen check exists.

## The screen size check

`checkScreenSizeOnce()` runs in the `MoreView` constructor hook, after Nickel's original, before our rows
are added. It accepts 1264×1680 or 1680×1264 (rotated). Anything else, or no `QApplication`, disables the
bridge with the model toast. The drawer's own full-screen overlays already use
`QApplication::desktop()->screenGeometry()` (`plain_dialog.cc`, `tweaks_overlay.cc`), so it is the same
number our layout depends on. The panel's framebuffer may have a wider virtual width (1280 px on the
Libra 2 per FBInk notes); Qt reports the visible size, which is what the check compares. Confirm on the
device with the log line below.

## What the Libra 2 logs (syslog, `nickeldrawer`)

```
bridge: device 00000000-0000-0000-0000-000000000388 is model 388 (io, PRODUCT io), supported
active on 4.38.23828 (route A controller, app screens on)
bridge: screen 1264x1680, supported
```

`PRODUCT` may read `unset`; that changes nothing. The screen line appears when the More tab is first built.

## Single source of truth

`host/config/project.json` `supported_models` (id, codename, name) next to `supported_firmware`.
`mod/drawer/device_gate.cc` `SUPPORTED_MODELS` mirrors the ids and codenames;
`internal/project/project_test.go` fails when they differ, and checks the names against
`internal/device`. To add a model: it must have the 1264×1680 panel (or the layout must first learn its
size), then add it to both lists.

## Test hook

`.adds/nickeldrawer/test-unknown-model` (read at boot like `test-unknown-fw`) makes the model check fail
on any device: `bridge: test flag … present, device … treated as unsupported, drawer disabled (no More rows)`.
Checklist case FI-13 in `docs/testing/final-checklist.md`.

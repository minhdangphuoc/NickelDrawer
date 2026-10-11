# Compatibility

## Kobo Libra 2

Target: the five newest Libra 2 builds (firmware family `kobo9`, device id ending in `388`). Details and sources are in [plan-detailed.md](plan-detailed.md) section 3.

| Firmware | Released | NickelMenu (retired, not a fallback) | Native drawer | Status |
|---|---|---|---|---|
| 4.38.23828 | Sep 2026 | works (gate M1, 2026-10-09) | install/uninstall of a test mod works (gate G1); earlier drawer builds ran on this firmware; the current build is pending the final device session | menu apps tested on a device |
| 4.38.23697 | May 2026 | planned | planned | planned / untested (package verified as a Libra 2 build) |
| 4.38.23684 | Apr 2026 | planned | planned | planned / untested |
| 4.38.23648 | Mar 2026 | planned | planned | planned / untested |
| 4.38.23552 | Nov 2025 | planned | planned | planned / untested |

"Works" means it was run on a real device (see `docs/testing/final-checklist.md`). "Untested" means no one has run NickelDrawer on that build yet. This table will change as we test.

How we test: symbol checks against all five builds, and runtime tests on one real device with one build. We cannot run every build on hardware.

If the native drawer finds an unknown firmware or a model it does not support (see [The model gate](#the-model-gate)), it turns itself off: the More tab stays stock, the mod shows one toast and logs the reason. A missing symbol only switches off the feature that needs it (see `mod/drawer/README.md`). NickelMenu is not a fallback any more (DECISIONS.md), so there are no menu entries to fall back to. To rehearse this on a supported build, create the flag `.adds/nickeldrawer/test-unknown-fw` (plan case FI-04) or `.adds/nickeldrawer/test-unknown-model` (FI-13) and reboot; delete it to go back to normal.

Other flags in `.adds/nickeldrawer/`: `disable` (kill switch), `no-screens` (screens are on by default; this opts out and taps run the primary action again), `debug` (step log). The old `screens` flag is ignored.

## Firmware 5.x: not supported

NickelDrawer needs NickelHook (the native drawer), and NickelHook's author does not support 5.x yet; NickelMenu, which the project started from, does not either. So NickelDrawer is not supported on 5.x. Do not install it on a 5.x device.

The Libra 2 stays on the 4.38 line. Builds 4.39 to 4.45 are for other devices. Do not install them on a Libra 2: a 4.39 package once boot-looped Libra 2 devices.

## Supported devices

Every Kobo e-reader ever sold, with what NickelDrawer can honestly say about it. Only one row is tested.

- ✅ **Tested**: run on a real device. Only the Kobo Libra 2 on firmware 4.38.23828.
- 🟡 **Expected to work, untested**: the model has the same 1264×1680 screen as the Libra 2 and runs firmware 4.x (the same Nickel and Qt 5 base that NickelHook and NickelMenu support), but nobody has run NickelDrawer on it. Only the Libra H2O and the Libra Colour.
- ⏸ **Drawer stays off**: the model runs firmware 4.x but has another screen size. The mod loads, sees the model, logs it, shows one "NickelDrawer does not support this model yet" toast and stays off; the Kobo works as normal.
- ⛔ **Not supported**: firmware 5.x (Qt6), models that never got firmware 4.x, and Android tablets.

| Model | Year | Platform (codename, hardware) | Device id | Screen (px) | Newest firmware line | NickelDrawer |
|---|---|---|---|---|---|---|
| Kobo eReader (N416) | 2010 | pre-Nickel | n/a | 600×800 | 1.x/2.x, no Nickel | ⛔ |
| Kobo Wireless eReader (N647) | 2011 | pre-Nickel | n/a | 600×800 | 2.x, no Nickel | ⛔ |
| Kobo Touch (A/B and C) | 2011 | trilogy, kobo3/kobo4 | 310, 320 | 600×800 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Mini | 2012 | trilogy/pixie, kobo4 | 340 | 600×800 | 3.19.5761 (last build, verified) | ⛔ |
| Kobo Glo | 2012 | phoenix/kraken, kobo4 | 330 | 758×1024 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Aura HD | 2013 | dragon, kobo4 | 350 | 1080×1440 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Aura | 2013 | phoenix, kobo5 | 360 | 758×1024 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Aura H2O | 2014 | dragon/dahlia, kobo5 | 370 | 1080×1430 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Glo HD | 2015 | dragon/alyssum, kobo6 | 371 | 1072×1448 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Touch 2.0 | 2015 | trilogy/pika, kobo6 | 372 | 600×800 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Aura ONE (and Limited Edition) | 2016 | daylight, kobo6 | 373, 381 | 1404×1872 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Aura Edition 2 (v1, v2) | 2016 | phoenix/star, kobo6/kobo7 | 375, 379 | 758×1024 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Aura H2O Edition 2 (v1, v2) | 2017 | dragon/snow, kobo6/kobo7 | 374, 378 | 1080×1430 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Clara HD | 2018 | dragon/nova, kobo7 | 376 | 1072×1448 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Forma (8 GB, 32 GB) | 2018 | daylight/frost, kobo7 | 377, 380 | 1440×1920 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Libra H2O | 2019 | dragon/storm, kobo7 | 384 | 1264×1680 | 4.x | 🟡 |
| Kobo Nia | 2020 | phoenix/luna, kobo7 | 382 | 758×1024 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Sage | 2021 | daylight/cadmus, kobo8 | 383 | 1440×1920 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Elipsa | 2021 | dragon/europa, kobo8 | 387 | 1404×1872 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| **Kobo Libra 2** | 2021 | dragon/io, kobo9 | 388 | 1264×1680 | 4.38 | ✅ 4.38.23828 only |
| Kobo Clara 2E | 2022 | dragon/goldfinch, kobo10 | 386 | 1072×1448 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Elipsa 2E | 2023 | dragon/condor, kobo11 | 389 | 1404×1872 | 4.x | ⏸ drawer stays off (screen size not supported yet) |
| Kobo Libra Colour | 2024 | dragon/monza, kobo11 | 390 | 1264×1680 | 4.45, or 5.x if converted | 🟡 on 4.x, ⛔ on 5.x |
| Kobo Clara BW | 2024 | dragon/spaBW, kobo12 | 391 | 1072×1448 | 4.45, or 5.x if converted | ⏸ drawer stays off (screen size not supported yet) on 4.x, ⛔ on 5.x |
| Kobo Clara Colour | 2024 | dragon/spaColour, kobo12 | 393 | 1072×1448 | 4.45, or 5.x if converted | ⏸ drawer stays off (screen size not supported yet) on 4.x, ⛔ on 5.x |
| Kobo Vox | 2011 | Android tablet | n/a | 600×1024 | Android | ⛔ |
| Kobo Arc, Arc 7 HD, Arc 10 HD | 2012–2013 | Android tablets | n/a | various | Android | ⛔ |

Totals: 1 tested, 2 expected to work (untested), 19 where the drawer stays off (screen size not supported yet), 5 not supported (Kobo eReader, Wireless, Mini, Vox, Arc family), plus any Libra Colour or Clara unit converted to 5.x.

Reading the table:

- **Device id** is the last digits of Kobo's device id (`00000000-0000-0000-0000-000000000388` for the Libra 2), as in pgaskin's koboutils. **Platform** is koboutils' class/codename plus the firmware family folder (`kobo9`).
- **Newest firmware line** is what Kobo's upgrade API served for that device id on 2026-10-10 (4.38.23828 for the Touch through Elipsa 2E, 4.45.23697 for the Libra Colour and Clara models, 3.19.5761 for the Mini). For the Elipsa and Libra 2 the API answered with an older package for our placeholder query, so their line is taken from our own lock file (Libra 2) or is unverified (Elipsa). "4.x" means a 4.x build exists; it does not mean it is one of ours.
- **Year** and **screen** are from public spec sheets and are approximate. koboutils lists no resolutions, so these were not machine-checked.
- Tolino readers (shine, vision, epos) share Kobo hardware and device ids 6xx. They are not Kobo models and are not covered.
- Kobo's Android tablets and the Vox have no Nickel; the drawer cannot run on them.

### The model gate

The app screens are drawn for the Libra 2 panel: `SCREEN_WIDTH` 1264 and `SCREEN_HEIGHT` 1680 in `mod/drawer/screen_logic.h`. On another resolution they would be clipped, off-centre or too small, so the drawer refuses to run there. Two checks, and whichever fails first turns the drawer off:

1. **Model.** At boot the mod reads the device id, the last field of `.kobo/version` (the same line it reads the firmware build from). Its last three digits name the model; only 384 (Libra H2O), 388 (Libra 2) and 390 (Libra Colour) pass. The list is `supported_models` in `host/config/project.json`, mirrored in `mod/common/device_gate.cc` and kept equal by a test.
2. **Screen.** When the More tab is first built, the mod asks Nickel's Qt for the screen size and accepts only 1264×1680 (or 1680×1264 rotated). This catches a model whose real panel differs from the spec sheet.

A refused model gets the same treatment as an unknown firmware: no Apps row, one toast "NickelDrawer does not support this model yet" (vi: "NickelDrawer chưa hỗ trợ máy này"), a log line with the device id. The firmware build check still runs too; both must pass. Why the device id and not other sources: [re/notes/device-detection.md](../re/notes/device-detection.md). To rehearse it on a supported Kobo, create the flag `.adds/nickeldrawer/test-unknown-model` and reboot (checklist case FI-13); delete it to go back to normal.

### Why 🟡 is not a promise

- **Exact build list.** The mod reads the build from `.kobo/version` and loads only on the five builds in `host/config/project.json` and `mod/common/device_gate.cc`. On any other build it shows the "unsupported firmware" toast and turns itself off; the Kobo works as normal. The Libra H2O can run 4.38.23828, the same string as the Libra 2, but the mod's symbols were verified only against Libra 2 libnickel. Every libnickel symbol is looked up at boot and a missing one switches off just the feature that needs it. Do not rely on this: it is untested.
- **Newer 4.x lines.** The Libra Colour is on 4.45, which is not on our list, so it gets the firmware toast. Do not install packages meant for another model (a 4.39 package once boot-looped Libra 2 devices).
- **Same pixels, other panel.** The Libra H2O and Libra Colour have the 1264×1680 size of the Libra 2 and should lay out the same; the Libra Colour's colour screen has not been looked at.
- **Colour model.** The Libra Colour has extra Nickel parts (stylus notebooks, colour screen). Whether the same symbols exist is unknown.

### Firmware 5.x

Libra Colour and Clara BW/Colour (and their Tolino twins) can be converted to the 5.x (Qt6) line, and community notes say new units may ship with it. NickelHook and NickelMenu do not support 5.x. NickelDrawer is ⛔ there, with no plan. If your Kobo offers 5.x, decline.

### Help add a model or a build

1. Find your build: **Settings** → **Device information**, or the third field of `.kobo/version`.
2. Check the symbols off the device: get the matching update package for your model from Kobo (it is never committed here), then run `re/scripts/fetch-fw.sh` and `tools/symcheck.py` as `tools/fw-watch.sh --fetch` does for the Libra 2. `tools/fw-watch.sh` itself is hard-wired to the Libra 2 device id; for another model you pass your own id and package.
3. A model with another screen size also needs the layout in `mod/drawer/screen_logic.h` to learn that size before it can join `supported_models` in `host/config/project.json` (and `mod/common/device_gate.cc`).
4. Open an issue with the "Bug report" template: model, build, symbol-check result. Add only text, never Kobo files.
5. A device run needs the maintainer's uninstall drill first (gate G1). Do not test it on your only reader.

### Sources

Fetched 2026-10-10. Web pages are untrusted; facts were cross-checked where possible.

1. pgaskin, koboutils, `kobo/device.go`: <https://github.com/pgaskin/koboutils> (device names, ids, codenames, hardware families).
2. Kobo upgrade API, `api.kobobooks.com/1.0/UpgradeCheck/Device/<id>/kobo/0.0.0/N` (newest package per device id; same endpoint as `tools/fw-watch.sh`).
3. MobileRead wiki, Kobo: <https://wiki.mobileread.com/wiki/Kobo> (list of models, Vox and Arc as tablets).
4. pgaskin, NickelMenu README: <https://github.com/pgaskin/NickelMenu> ("Firmware 5.x is not supported yet").
5. notmarek, KoboTolinoFindings: <https://github.com/notmarek/KoboTolinoFindings> (5.x builds exist only for Libra Colour and Clara BW/Colour, via a conversion package).
6. nicoverbruggen, NickelMenu Qt6 release: <https://github.com/nicoverbruggen/NickelMenu/releases/tag/qt6-1.0> (unofficial 5.x/6.x fork) and NickelHome/NickelTypeFix READMEs (4.x only).
7. Kobo help center, manual firmware update: <https://help.kobo.com/hc/en-us/articles/35059171032727-Manually-Updating-your-Kobo-eReader-device-Firmware>.

## Languages

English and Vietnamese ship first. Other left-to-right languages can be added with a `locales/<lang>.json` file.

Right-to-left languages (Arabic, Hebrew, Persian) are not supported yet. The native UI kit does not lay out RTL. We plan to look at this in the UI kit phase.

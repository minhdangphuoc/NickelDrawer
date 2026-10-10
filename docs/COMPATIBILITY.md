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

If the native drawer finds an unknown firmware, it turns itself off: the More tab stays stock, the mod shows one toast and logs the reason. A missing symbol only switches off the feature that needs it (see `mod/drawer/README.md`). NickelMenu is not a fallback any more (DECISIONS.md), so there are no menu entries to fall back to. To rehearse this on a supported build, create the flag `.adds/nickeldrawer/test-unknown-fw` and reboot (plan case FI-04); delete it to go back to normal.

Other flags in `.adds/nickeldrawer/`: `disable` (kill switch), `no-screens` (screens are on by default; this opts out and taps run the primary action again), `debug` (step log). The old `screens` flag is ignored.

## Firmware 5.x: not supported

NickelDrawer needs NickelHook (the native drawer), and NickelHook's author does not support 5.x yet; NickelMenu, which the project started from, does not either. So NickelDrawer is not supported on 5.x. Do not install it on a 5.x device.

The Libra 2 stays on the 4.38 line. Builds 4.39 to 4.45 are for other devices. Do not install them on a Libra 2: a 4.39 package once boot-looped Libra 2 devices.

## Other devices

Other Kobo models: community-tested, not yet. We have not checked any other model. Tell us what you tried with the "Bug report" template, or send a compatibility note.

## Languages

English and Vietnamese ship first. Other left-to-right languages can be added with a `locales/<lang>.json` file.

Right-to-left languages (Arabic, Hebrew, Persian) are not supported yet. The native UI kit does not lay out RTL. We plan to look at this in the UI kit phase.

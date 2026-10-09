# Compatibility

## Kobo Libra 2

Target: the five newest Libra 2 builds (firmware family `kobo9`, device id ending in `388`). Details and sources are in [plan-detailed.md](plan-detailed.md) section 3.

| Firmware | Released | Menu apps (NickelMenu) | Native drawer | Status |
|---|---|---|---|---|
| 4.38.23828 | Sep 2026 | works (gate M1, 2026-10-09) | install/uninstall of a test mod works (gate G1); drawer pending (G2) | menu apps tested on a device |
| 4.38.23697 | May 2026 | planned | planned | planned / untested (package verified as a Libra 2 build) |
| 4.38.23684 | Apr 2026 | planned | planned | planned / untested |
| 4.38.23648 | Mar 2026 | planned | planned | planned / untested |
| 4.38.23552 | Nov 2025 | planned | planned | planned / untested |

"Works" means it was run on a real device (see `docs/sessions/`). "Untested" means no one has run NickelDrawer on that build yet. This table will change as we test.

How we test: symbol checks against all five builds, and runtime tests on one real device with one build. We cannot run every build on hardware.

If the native drawer finds an unknown firmware or a missing symbol, it turns itself off and logs the reason. The NickelMenu entries keep working.

## Firmware 5.x: not supported

NickelDrawer needs NickelMenu (the menu fallback) and NickelHook (the native drawer). NickelMenu's author says 5.x is not supported yet, so NickelDrawer is not supported on 5.x either. Do not install it on a 5.x device.

The Libra 2 stays on the 4.38 line. Builds 4.39 to 4.45 are for other devices. Do not install them on a Libra 2: a 4.39 package once boot-looped Libra 2 devices.

## Other devices

Other Kobo models: community-tested, not yet. The app menu part may work where NickelMenu works, but we have not checked. Tell us what you tried with the "Bug report" template, or send a compatibility note.

## Languages

English and Vietnamese ship first. Other left-to-right languages can be added with a `locales/<lang>.json` file.

Right-to-left languages (Arabic, Hebrew, Persian) are not supported yet. Menu text goes through NickelMenu, and the native UI kit does not lay out RTL. We plan to look at this in the UI kit phase.

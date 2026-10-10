# NickelDrawer: native drawer host (Phase 3)

`libnickeldrawer.so` is a NickelHook mod. It adds an **Apps** row to Nickel's More tab and opens a native
drawer page (list by default, 4-column launcher grid optional). It only renders and handles input. All app logic
lives in `appsctl`, which the mod runs as a child process (`docs/spec/drawer-protocol.md`). Screens follow
`docs/spec/ui-kit.md` §5. The libnickel facts behind it are in `re/notes/G2-findings.md`.

**Status: built and statically checked, never run on a device.** The first on-device run needs the user
present and gate G1 passed (see "First-boot test script" below).

## Build and checks (Mac, nothing touches the Kobo)

```sh
make -C mod/drawer docker-build   # build/mod/libnickeldrawer.so, build/mod/KoboRoot-nickeldrawer.tgz
make -C mod/drawer docker-test    # host unit tests of drawer_logic, screen_logic, qr_encoder (native g++, -nostdinc++)
mod/drawer/check-symbols.sh       # bridge-only symbols, 5/5 firmware exports, ARM hard-float, NEEDED, no STL, KoboRoot contents
```

Toolchain: the pinned NickelTC image (`mod/TOOLCHAIN.md`). The KoboRoot installs only
`/usr/local/Kobo/imageformats/libnickeldrawer.so`.

## Architecture

```
nickeldrawer.cc ── init(): flags + firmware check ──► nickel_bridge (the ONLY libnickel user)
                                                          │  MoreView ctor hook ─► entry_row
entry_row ── tap ─► drawer_navigator ── route A ─► drawer_controller (AbstractController, navSection 6)
                          │            └ route B ─► pushView/popView + stack watcher
                          └─ creates drawer_view + drawer_session
drawer_session ── model_client (appsctl via QProcess, async) ─► model_parser (QJsonDocument)
              └── action_runner ─► dialogs / plain_dialog, wifi_connector, bridge (toast, rescan)
drawer_logic: Qt-free rules (paging, refresh, protocol, retry, locale), unit-tested on the host

Phase 4 (on by default, opt out with `no-screens`): drawer_session ── tap on an app with "screen" ─► screen_session
screen_session ── model_client (`drawer screen`) ─► screen_view (child of drawer_view, covers it)
              └── taps ─► action_runner (`drawer event`), field_editor (Nickel keyboard via the bridge)
screen_layout (port of cmd/ndpreview) over screen_logic + qr_encoder: Qt-free, unit-tested on the host
```

| File | One job |
|---|---|
| `nickeldrawer.cc` | NickelHook declaration and the tiny `init()` |
| `nickel_bridge.{h,cc}` | hook/dlsym tables, firmware check, fail-soft wrappers for every libnickel call, e-ink full-refresh helper |
| `mod_switches.{h,cc}` | flag-file paths and the boot-time switch values |
| `drawer_logic.{h,cc}` | pure rules: paging math, key mapping, refresh policy, protocol version, exit codes, timeouts, Wi-Fi retry, locale |
| `fallback_strings.{h,cc}` | the drawer's own en/vi strings; reads `CurrentLocale` from `Kobo eReader.conf` (read-only) |
| `drawer_model.h` | plain structs of the protocol replies |
| `model_parser.{h,cc}` | JSON → structs, version check, problem rows |
| `appsctl_call.{h,cc}` | one `appsctl` QProcess run with a hard timeout, working dir `/` |
| `model_client.{h,cc}` | `drawer model / run / info / remove --confirm`, maps every failure to a localized reply |
| `view_settings.{h,cc}` | list/grid choice in `.adds/koboapps/data/drawer.ini` (open, write, close; only on change) |
| `painted_marks.{h,cc}` | update dot, list/grid toggles, header icon buttons, app mark; icons from `../common/system_icon` (Nickel's images, else Tabler PNGs, never drawn) |
| `item_widgets.{h,cc}` | list row (152 px) and grid cell (icon + 2-line name, no status line); status/dot change in place |
| `drawer_view.{h,cc}` | header, page of rows or grid cells (host items as header buttons: refresh, list, grid, settings gear with the update dot) with section labels, hint, `1 / N` with arrows, page keys |
| `plain_dialog.{h,cc}` | plain Qt dialog (menu, fallback for Nickel dialogs), 120 s timeout |
| `dialogs.{h,cc}` | result dialog and confirm question: Nickel's own first, plain fallback |
| `wifi_connector.{h,cc}` | Nickel's Wi-Fi connect, poll 1 s × 30 |
| `host_requests.{h,cc}`, `host_request_runner.{h,cc}` | reply `requests` + manifest `rescan` → one plan (Qt-free, tested): Wi-Fi first (ask, connect; skipped when online), then one rescan (full covers quick; full falls back to quick when its symbols are missing); never remembered. Run/event replies and `drawer screen` replies both go through it |
| `action_runner.{h,cc}` | tap → first action; `needs_wifi` → ask, connect, retry once; result; requests; long-press menu (Info, actions, Remove; at most 8 entries, extra actions and `menu:false` ones left out and logged) |
| `drawer_session.{h,cc}` | one open drawer: model → view, view taps → runner |
| `drawer_controller.{h,cc}` | ABI mirror of Nickel's `AbstractController` (built with `-frtti`, no `Q_OBJECT`) |
| `drawer_navigator.{h,cc}` | open/close by route A (push/pop) or route B (pushView/popView + watcher) |
| `entry_row.{h,cc}` | More-tab row, status summary line and update dot (items with a badge, or the model's `update_available`) |
| `more_row.{h,cc}` | builds and inserts one IconLeftButton row in the More tab (Apps and Tweaks rows) |
| `tweaks_logic.{h,cc}` | Qt-free: `drawer tweaks` / `drawer tweak` JSON replies, 2-column grid math (unit-tested) |
| `tweaks_model.h`, `tweaks_parser.{h,cc}` | Qt structs of the tweaks replies, built from tweaks_logic |
| `tweaks_row.{h,cc}` | the "Tweaks" More row after Apps; tap opens the modal |
| `tweaks_panel.{h,cc}` | modal body: rule, 2-column cells (italic serif label, On/Off or value), page keys |
| `tweaks_session.{h,cc}` | one open modal: Nickel's N3Dialog as a controller view (else the overlay), confirm, `tweak`, toast, Nickel sleep/reboot/rescan |
| `tweaks_overlay.{h,cc}` | own modal chrome when Nickel's N3Dialog cannot be used |
| `wake_rotation.{h,cc}` | random wallpaper: on `N3PowerWorkflowManager::resumedFromSleep()`, if the Wallpaper app's `data/rotate-on-wake` exists, starts `wallpaper next` detached (docs/spec/wallpaper.md "Random rotation") |
| `screen_model.h` | plain structs of a `drawer screen` reply |
| `screen_logic.{h,cc}` | pure ndpreview metrics and rules: wrap, elide, pages, component heights (Qt-free) |
| `qr_encoder.{h,cc}` | QR byte mode, level M, versions 1–20; checked against segno matrices (Qt-free) |
| `screen_layout.{h,cc}` | components → drawing primitives and tap targets, one page at a time (port of `cmd/ndpreview`) |
| `screen_view.{h,cc}` | paints a page, page keys (event filter on the drawer view), taps, full/partial refresh |
| `picture_loader.{h,cc}` | gallery/viewer pictures: read whole (cap 8 MB) and close, decode from memory scaled to the box |
| `tile_menu.{h,cc}` | gallery long-press: Nickel popup titled with the tile label, destructive confirm first |
| `row_menu_logic.{h,cc}` | Qt-free rules of list row menus: which declared entries are kept (cap 6), when a long press opens the menu (unit-tested) |
| `toast_limiter.{h,cc}`, `toast_gate.{h,cc}` | one toast per app per 2 s (Qt-free limiter, unit-tested) and the gate every drawer toast goes through |
| `qt6_notice.{h,cc}` | reads the 5.x offer from `Kobo eReader.conf` (rule: `DrawerLogic::isQt6UpgradePending`) |
| `event_queue.{h,cc}` | Qt-free FIFO of radio/toggle events: coalesce by control, drop in-flight duplicates, one at a time |
| `field_editor.{h,cc}` | `field` editing in Nickel's N3Dialog keyboard; else the caller shows "edit on your phone" |
| `screen_session.{h,cc}` | one open app screen: load, live re-read, taps → events, pushed screens, back |
| `../common/press_area.{h,cc}` | tap vs long-press (600 ms) from Nickel's tap gesture, touch or mouse; debounced, every event logged |
| `../common/nickel_fonts.{h,cc}` | title/body/mono families read from Nickel's live widgets (never generic), Vietnamese self-check |

### Protocol handling (v1)

- Timeouts: model 3 s, `run` of a `background` action 3 s, `run` of a `show` action 35 s (appsctl caps
  the script at 30 s and replies `ok:false` itself), info 3 s, remove 10 s. On timeout the process is killed.
- Exit 0 → parse the reply; `ok:false` is a normal result and its `text` is shown. Exit 2 → show
  `error.message`. Anything else, a missing binary, a crash, bad JSON or `v` missing → built-in fallback
  text in a problem row (model) or dialog (actions). `v > 1` → "Update NickelDrawer" row.
- The primary (tap) action is the **first** entry of `actions`. The long-press menu lists Info, the other
  actions except `status`, and Remove… (not for host items `_refresh`, `_settings`; broken apps from
  `errors[]` get Remove only). `errors[].message` is English and shown as-is.
- `needs_wifi` → No-Wi-Fi confirm (Cancel / Turn on Wi-Fi) → `WirelessWorkflowManager::connectWireless` →
  poll `isInternetAccessible` → retry once; a second `needs_wifi` shows "Could not connect to Wi-Fi."
- `rescan` `quick` → `PlugWorkflowManager::sync()`; `full` → `onCancelAndConnect()` then `unplugged()` 1 s
  later on a timer (NickelMenu's sequence, without sleeping on the GUI thread).
- `kind: background` with `ok:true` → toast; everything else with text → Nickel's OK dialog.

### Refresh

Open and page change: one flashing full update (attribute 0x80 + `repaint()` via the bridge, after the
layout settles). Same items with new status/dot: labels change in place (partial, Qt repaints only them);
the 9th partial update in a row is upgraded to a full one. A changed item list rebuilds the page (full).

## Flags (on the Kobo, read once at boot; reboot to apply)

Version `1.0.0-beta.1` (`NH_VERSION`, logged by NickelHook).

| Path | Effect |
|---|---|
| `.adds/nickeldrawer/uninstall` | NickelHook removes the mod and the flag at the next boot |
| `.adds/nickeldrawer/disable` | kill switch: the mod loads, the bridge stays inert, the More tab is stock |
| `.adds/nickeldrawer/pushview` | open the drawer with route B (`pushView`) instead of the controller push |
| `.adds/nickeldrawer/no-screens` | opt-out: tapping an app runs its primary action instead of opening its `ui.json` screen. Screens are on by default |
| `.adds/nickeldrawer/screens` | obsolete (screens used to be opt-in): still read, logged nowhere, changes nothing |
| `.adds/nickeldrawer/test-unknown-fw` | test hook (FI-04): the firmware check fails as if the build were unknown, so the drawer disables itself and shows the one "not supported" toast. Remove it and reboot to return to normal |
| `.adds/nickeldrawer/no-keyboard` | `field` taps show the "edit on your phone" notice instead of Nickel's keyboard |
| `.adds/nickeldrawer/debug` | step log of row construction and the tap path in `.adds/nickeldrawer/debug.log` (append, fsync, close per line; restarts at 64 KB). Survives a Nickel crash, unlike syslog. Shared with the prototype (`mod/common/debug_log.cc`) |

The prototype `libnddrawer.so` uses the same uninstall path: **uninstall the prototype first** (one flag
run removes only one of them).

## How it degrades

| Situation | Result |
|---|---|
| `disable` flag | inert; hooks pass straight through |
| firmware not one of the five 4.38 builds | inert, logged; no More rows. One Nickel toast a few seconds after the first More tab is built: "NickelDrawer does not support this firmware yet" (vi: "NickelDrawer chưa hỗ trợ firmware này"), only when `MainWindowController::toast` resolved, else log only. NickelMenu is not a fallback |
| a critical symbol missing (`MoreView` hook, `IconLeftButton`, `MainWindowController::sharedInstance`, neither push route) | same as an unknown firmware: inert, logged, the one toast. Test build: `make -C mod/drawer docker-build EXTRA_CPPFLAGS=-DND_TEST_MISSING_SYM` pretends one is missing (FI-03); never install that build for real use |
| Kobo offers 5.x (`AvailableQt6Upgrade` in `Kobo eReader.conf` is set and differs from `SkippedQt6UpgradeUrl`) | a bold notice row under the drawer header: "Kobo is offering update 5.x — NickelDrawer will stop working" (vi: "Kobo đang mời cập nhật 5.x — NickelDrawer sẽ ngừng hoạt động"), taking the hint line's room. Config keys, not `IOUtil::isQt6UpgradePending` (23828 only, needs a `Device`) |
| a non-critical libnickel symbol missing | only the feature using it is off: no row (IconLeftButton/hook), toast "not available" (push API), plain Qt dialog (Nickel dialogs), message instead of toast, no Wi-Fi retry, no rescan |
| `moreContainerLayout` not found | no row, logged |
| N3Dialog or controller push missing | Tweaks modal in the own overlay (re/notes/tweaks-modal.md) |
| N3PowerWorkflowManager / rescan missing | the tweak still runs; toast "Not available on this firmware." instead of sleep/reboot/rescan |
| N3PowerWorkflowManager or its `resumedFromSleep()` signal missing | logged; random wallpaper keeps the current picture (no change on wake) |
| `AbstractController` offset not 8, or push did not stack the view | page discarded, toast |
| appsctl missing / slow / crashing / bad JSON / too new | problem row + "Try again" row; the More row just reads "Apps" |
| `drawer.ini` not writable | logged, list/grid choice not kept |
| icon not loadable (e.g. SVG plugin missing) | the generic Tabler `apps` icon (`icons/ui/apps-96.png`), else no icon |
| `drawer screen` fails (no/invalid `ui.json`, appsctl error) | blocking error screen: reason, **Back**, **Try again** |
| keyboard symbols missing or `no-keyboard` | a `field` tap shows a toast "Change this from your phone or computer." |
| a QR value too long (> 666 bytes) | an empty square frame, the address stays as text |

Failsafe: a crash in the first 15 s after boot removes the mod (NickelHook).

## What needs device validation

1. Route A push/pop with our controller (proven only statically, prototype C not yet run).
2. Taps and long-press on plain Qt widgets (touch vs synthesized mouse events).
3. The row's second line (`IconLeftButton::setText` with `\n`) and the dot position.
4. Page keys reaching the view; keys while Nickel's OK/confirm dialog is open (it is a `ConfirmationDialog`,
   not an `N3Dialog`, so keys may still reach the drawer behind it).
5. Fonts, sizes and the 7-row page with two section labels fitting above the tab bar (KIT-12, largest font, vi).
6. Attribute 0x80 flash on our widget; partial updates of labels.
7. `connectWireless(true,false)` UI and `isInternetAccessible` polling; `PlugWorkflowManager` rescans.
8. SVG icons via `QPixmap` (needs Nickel's svg image plugin).
9. QSettings writing `drawer.ini` on the FAT partition while Nickel runs.
10. App screens (on by default): the screen view over the drawer page (fonts, sizes vs the ndpreview PNG),
    page keys on long screens, partial refresh of a toggled switch, the N3Dialog keyboard for a `field`
    (`KeyboardReceiver` size and `QLocale()` keyboard layout are unverified), QR scannable from a phone.

### App screens (Phase 4)

`appsctl drawer screen` returns the screen with texts and bindings resolved; the mod never evaluates
bindings. Layout is a line-by-line port of `cmd/ndpreview` (same metrics, the same average-advance text
wrap and elide, the same paging), so `ndpreview render` shows what the device draws; only the glyphs
differ (a real font wider than the estimate is elided again instead of clipped). The view height is the
real one (Nickel's status and tab bars take part of 1680 px), so long screens page earlier than the
preview. Taps: buttons, rows, `empty` and the header action run `drawer event`; toggles and radios show the
new value at once and run the action named like their variable with `--value`; list rows with `tap` pass
the row's primary text; `field` opens Nickel's keyboard. A held list row with a `menu` opens Nickel's popup menu (title: the row's primary text; `confirm`
entries ask first) and runs `appsctl drawer event <app> <menu-id> --list <bind> --index <n> --value <primary> [--screen <id>]`;
a row with only a `menu` still shows the pressed state on a short tap but sends nothing. Menu entries are `{id, label, confirm}`
(`confirm`: `true` for the generic destructive question, or a `{title,text,ok,cancel}` object), at most 6, taken from the list
component's `menu` array. Back pops
pushed screens, then returns to the drawer (one flash). A screen whose app has a `state` action is re-read
every 5 s while idle; unchanged primitives are not repainted.

## First-boot test script (later, with the user present)

Preconditions: G1 passed (ndhello toast, uninstall drill, failsafe drill); `ndhello` and `nddrawer`
uninstalled; backup of `.kobo/KoboReader.sqlite`, `.kobo/Kobo/Kobo eReader.conf` and `.adds/` in
`backups/<date>/`; battery > 50 %; NickelMenu working; `appsctl` installed in `.adds/koboapps/bin/`.
Shell rule: absolute paths, never `cd` into `/Volumes/KOBOeReader`.

0. Log items for NickelMenu (copy to `.adds/nm/nickeldrawer-log`, not installed by the build):
   ```
   menu_item:main:NickelDrawer log:cmd_output:5000:logread | grep nickeldrawer | tail -20
   menu_item:main:NickelDrawer save log:cmd_spawn:quiet:logread > /mnt/onboard/.adds/nickeldrawer/log.txt
   ```
1. Install: `mkdir -p /Volumes/KOBOeReader/.adds/nickeldrawer`, make sure `uninstall`, `disable`,
   `pushview` do not exist, `cp build/mod/KoboRoot-nickeldrawer.tgz /Volumes/KOBOeReader/.kobo/KoboRoot.tgz`,
   eject, unplug. Wait 20 s on Home (failsafe window). Log: `active on 4.38.<build> (route A controller)`.
2. More tab: "Apps" row below Beta Features; after a moment a second line if an app is running or an
   update exists, and a dot for updates. Leave and re-open More twice: one row each time.
3. Tap "Apps": drawer with back chevron, title, list/grid toggles (list underlined), sections
   INSTALLED / SYSTEM, hint, `1 / N`; **tab bar visible with More active**; one flash.
4. Page buttons: forward/back flip pages (one flash each); at the edges nothing changes. On-screen
   arrows do the same. Toggle grid: 4 columns of icon + name cells, refresh icon in the header; leave and re-open: grid is remembered.
5. Tap an app (e.g. LocalSend): result dialog; status line updates in place after closing it.
   Tap "Refresh list": dialog, list reloads.
6. Long-press an app: menu with Info, other actions, Remove…; Info shows version/storage/folder;
   Remove… → confirm (Cancel left, Remove right); Cancel leaves it.
7. With Wi-Fi off, tap an app that needs Wi-Fi: "Wi-Fi is off" dialog → Turn on Wi-Fi → Nickel connects
   → the action runs once.
8. Back chevron → More; open again and tap Home → Home; tap More → the normal More list with the Apps row
   (no leftover drawer page). Log: `navigator: drawer closed` once per close.
9. Sleep/wake with the drawer open; USB plug/eject; tap "Apps" twice quickly (one drawer).
10. Kill switch: `touch .adds/nickeldrawer/disable`, reboot → More is stock, log says "staying inert".
    Remove the flag, reboot → row back.
11. If route A crashes on tap: Nickel restarts by itself; `touch .adds/nickeldrawer/pushview`, reboot,
    repeat 3–9 (route B).

Uninstall (stop at the first that works): `touch .adds/nickeldrawer/uninstall` + reboot; power off within
15 s of boot (failsafe); `disable` flag; re-apply stock firmware (user's guide). NickelMenu is unaffected.

Send back: the log output after each step (or `log.txt`, kept in `backups/<date>/`, never committed),
photos of the More row, list page, grid page and a dialog, and the exact step before any freeze.

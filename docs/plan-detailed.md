# Ngăn Ứng Dụng — detailed project plan and timeline

Native "app drawer" mod for the Kobo Libra 2 (Mark 9 / "Io", 1264×1680, 300 dpi), stock Nickel UI, firmware 4.38.23828.

| | |
|---|---|
| Plan version | v3-detailed, 8 Oct 2026 (expands `docs/plan-overview.html` plan v2) |
| Developer | the maintainer, ~5 focused hours/day |
| Tester / owner | the user, batched on-device test sessions (the device stays in daily use) |
| Start | Mon 12 Oct 2026 |
| Planned end (v1.0 release) | Wed 23 Dec 2026 (11 weeks incl. 1 buffer week and the firmware-matrix work) |
| Source of truth for design | `docs/plan-overview.html` (architecture, wireframes, kit component list, protocol sketch) |

Anything marked **verify** was not confirmed first-hand during planning and must be checked before the task that depends on it.

---

## Table of contents

1. Goals, non-goals, success criteria
2. Architecture recap and fixed decisions
3. Firmware support matrix
4. Phase-by-phase breakdown (Phase 0 → 7)
5. Calendar timeline, milestones and gates
6. Reverse-engineering workplan
7. UI kit spec v1 and JSON-lines protocol spec
8. Safety and testing strategy
9. Validation (definition of done, test layers, test case tables, sign-off)
10. Blocker reports
11. Repo structure, build, CI, versioning and packaging
12. Risk register
13. Open questions and decisions for the user
14. Summary Gantt table
15. Appendix: reference facts collected during planning

---

## 1. Goals, non-goals, success criteria

### 1.1 Goals

> **Superseded by [DECISIONS.md](DECISIONS.md) (2026-10-10):** NickelMenu is retired and is not a fallback. On an unknown firmware the mod turns the drawer off, shows one toast and logs the reason (goal 5 below no longer promises NickelMenu items).

1. An **"Ứng dụng" (Apps) tab** in Nickel's bottom navigation bar that opens a native drawer view listing installed apps, drawn with Nickel's own widgets and stylesheet so it is indistinguishable from a Kobo screen.
2. A **UI kit renderer** inside the same mod that turns each app's `ui.json` into native Qt widgets (header with back chevron, list rows, toggles, text fields using Kobo's keyboard, buttons, progress, toasts, confirm dialogs).
3. **App backends as separate processes** (shell, Go, anything) speaking one JSON object per line over stdin/stdout. A crashing backend never takes Nickel down.
4. **One folder = one app**: `.adds/apps/<id>/` with `app.json`, `ui.json`, `bin/`, `data/`. Copy to install, delete to uninstall.
5. **Fail soft**: a per-firmware symbol table and a startup symbol check; on an unknown firmware or a missing symbol the drawer disables itself and every app stays reachable through NickelMenu items generated from `app.json`.
6. **Supported firmware**: the five newest Libra 2 firmware builds (see section 3), all on the 4.x line.
7. Three first apps: **LocalSend receiver**, **GitHub private-repo book sync** (add-only), **Library tool** (dedupe, `dc:language`/`xml:lang` → `vi`, NFC normalisation).
8. A **developer kit** (spec, template app, validator, Mac preview) so a new app needs no C++.

### 1.2 Non-goals (v1)

- No app catalog / store, no code signing, no sandboxing (backends run as root; only install apps you wrote or read).
- No support for firmware 3.x or 5.x (NickelMenu, NickelHook ecosystem and kobo-tieng-viet are 4.x only).
- No reader-view (in-book) integration; the drawer lives in the main navigation only.
- No replacement of Nickel's home screen, no theming beyond what Nickel's own stylesheet gives (dark mode follows the system).
- No Qt QML, no C++ standard library in the mod (NickelHook guidance), no browser-based UI.
- No background daemons that run all the time; backends run only while their screen is open or an explicit long-running job (sync, receive) is active.

### 1.3 Success criteria (measurable)

| ID | Criterion | Target | How measured |
|---|---|---|---|
| S1 | Drawer open latency (tab tap → drawer fully painted) | ≤ 600 ms with 3 apps, ≤ 1000 ms with 20 apps | `nh_log` timestamps around view push; user stopwatch for sanity |
| S2 | App screen open latency (tile tap → screen painted, backend started) | ≤ 800 ms for a shell backend, ≤ 1200 ms for a Go backend | log timestamps |
| S3 | Boot safety | 0 boot loops across all test installs; failsafe proven once on purpose | test log |
| S4 | Idle cost of the mod when no app screen is open | 0 extra processes, 0 timers firing, ≤ 1 MB RSS growth of Nickel after 72 h | `/proc/<pid>/status` VmRSS before/after soak; `top` |
| S5 | Battery impact idle | ≤ 1 % extra drain per 24 h in sleep compared with the user's baseline week | battery % readings at fixed times (section 9.6) |
| S6 | CPU while an app screen is open but idle | ≤ 2 % average over 60 s | `top -d 5` sampled via telnet/ssh or a log script |
| S7 | Memory of the host per open screen | ≤ 4 MB extra RSS per screen, released on back | VmRSS deltas |
| S8 | USB mass-storage | No file handles on `/mnt/onboard` held longer than 200 ms; USB connect during an open app screen works and the screen closes cleanly | `lsof`-style check via `/proc/*/fd`; manual test |
| S9 | Firmware coverage | Static symbol check passes on all 5 supported `libnickel.so` builds in CI; runtime detect disables cleanly on an unknown build | CI job + on-device test |
| S10 | Uninstall | Creating `.adds/koboapps/uninstall` removes the mod on next boot, leaving Nickel stock; NickelMenu items still work | manual test |
| S11 | UI parity | User rates each kit screen "looks like a Kobo screen" ≥ 4/5 in sign-off | sign-off checklist |
| S12 | Protocol robustness | 100 % of failure-injection cases (section 9.7) handled without a Nickel crash | test log |
| S13 | Developer kit | A new "hello" app can be written from the template and shown on device in ≤ 30 minutes by following `APPS.md` | timed exercise |

---

## 2. Architecture recap and fixed decisions

> **Superseded by [DECISIONS.md](DECISIONS.md) (2026-10-10):** the "NickelMenu fallback" bullet below is obsolete (`gen-menu` is gone; the installer removes the old generated file). The mod is `libnickeldrawer.so`, staged by `tools/install-to-kobo.sh` or an OTA update as `.kobo/KoboRoot.tgz`, and removed with the flag `.adds/nickeldrawer/uninstall` (the installer's `--uninstall` writes it).

Taken from `docs/plan-overview.html` and the NickelHook/NickelMenu/NickelDBus sources read during planning.

- **Mod name**: `libkoboapps.so` (NickelHook requires ≥ 3-letter unique names; under 3 letters are reserved for pgaskin/shermp). Installed by `KoboRoot.tgz` to `/usr/local/Kobo/imageformats/libkoboapps.so` (the Qt plugin path NickelHook mods use — **verify** exact path from `NickelHook.mk` when the toolchain is set up; NickelMenu installs `libnm.so` there).
- **Framework**: NickelHook (`nh.c`, `NickelHook.h`, `NickelHook.mk`) as a git submodule, built with NickelTC (`ghcr.io/pgaskin/nickeltc:1`, GCC 4.9.4, Qt 5.2.1 sysroot, `-march=armv7-a -mtune=cortex-a8 -mfpu=neon -mfloat-abi=hard -mthumb`). NickelTC is x86_64 only: on Apple Silicon run Docker with `--platform linux/amd64` (Rosetta/QEMU emulation; slower but works — **verify** on first build).
- **NickelHook features used**: `struct nh_info{ name, desc, uninstall_flag, uninstall_xflag, failsafe_delay }`, `struct nh_hook{ sym, sym_new, lib, out, desc, optional }`, `struct nh_dlsym{ name, out, desc, optional }`, `nh_log`, `nh_dump_log`, `nh_delete_file`. All libnickel symbols for the drawer are `optional = true` so a missing symbol never fails init; the mod's own `init()` then decides "drawer on" or "drawer off, menu fallback only".
- **Tab bar**: same technique as NickelMenu's `menu_main_15505` path: hook `MainNavView::MainNavView(QWidget*)` (`_ZN11MainNavViewC1EP7QWidget`), find its `QHBoxLayout`, construct a `MainNavButton` (`_ZN13MainNavButtonC1EP7QWidget`, `setPixmap`, `setActivePixmap`, `setText`, signal `tapped()`), bridge `tapped()` to our slot. Pixmap style: Nickel uses PNG icon paths; the drawer ships light/dark PNG icons matching Nickel's stroke weight.
- **View push**: `MainWindowController::sharedInstance()` + `MainWindowController::pushView(QWidget*)` (`_ZN20MainWindowController8pushViewEP7QWidget`, used by NickelHardcover) and `currentView()`. Whether `pushView` keeps the bottom tab bar visible, and how Nickel's own Home/Books views are constructed with the bar, is the main RE spike question (gate G2).
- **Dialogs/keyboard**: `ConfirmationDialogFactory::getConfirmationDialog`, `ConfirmationDialog::setTitle/setText/setAcceptButtonText/setRejectButtonText/showCloseButton/addWidget`, `N3DialogFactory::getDialog`, `N3Dialog::showKeyboard/hideKeyboard/keyboardFrame/enableFullViewMode/setTitle`, `KeyboardFrame::createKeyboard(KeyboardScript, QLocale const&)`, `KeyboardReceiver(QLineEdit*, bool)`, `TouchLineEdit`, `TouchTextEdit`, `TouchCheckBox`, `TouchLabel`, `N3ButtonLabel`, `SettingContainer`, `N3ConfirmationTextEditField` (all seen in NickelHardcover / NickelDBus sources).
- **Toast**: `MainWindowController::toast(QString const&, QString const&, int)`.
- **Wi-Fi / rescan / settings**: `WirelessWorkflowManager::sharedInstance/connectWireless(bool,bool)/connectWirelessSilently/isInternetAccessible`, `PlugWorkflowManager::sharedInstance/sync()` (rescan books), `N3FSSyncManager::sharedInstance/sync(QStringList*)`, `Settings` + `ReadingSettings::getDarkMode/setDarkMode`.
- **Backends**: `QProcess` per open app, JSON lines on stdin/stdout, stderr to the app's log, `SIGTERM` then `SIGKILL` on close; never a shell with the user's data path interpolated.
- **NickelMenu fallback**: `gen-menu` tool (part of the mod and also a standalone script) writes `.adds/nm/apps` entries from `app.json` files (NickelMenu reads every file in `.adds/nm/`). Items use `cmd_spawn`, `cmd_output`, `nickel_wifi:autoconnect`, `nickel_misc:rescan_books`.
- **Storage rule**: everything under `/mnt/onboard` is opened, read/written and closed within one Qt signal handler or the init function; nothing stays open. Backends own their `data/` folder and must obey the same rule (they are killed on USB connect, see section 8.7).

---

## 3. Firmware support matrix

### 3.1 Target: the five newest Libra 2 builds

Facts established during planning (8 Oct 2026):

- Kobo's upgrade API for device id `00000000-0000-0000-0000-000000000388` (Libra 2, "kobo9") returns `https://ereaderfiles.kobo.com/firmwares/kobo9/Sep2026/kobo-update-4.38.23828.zip` as the latest and nothing newer.
- The Libra 2 stays on a 4.38.2xxxx maintenance line. 4.39.22801 (Apr 2024) was **not** for the Libra 2: the kobo9-labelled package contained a MediaTek (`mt8113t-ntx`) image and boot-looped Libra 2 devices (MobileRead thread t=360326). 4.39.x–4.45.x are for the newer MediaTek devices (Clara/Libra Colour, Clara BW).
- NickelMenu, NickelHook mods and kobo-tieng-viet support 4.x only ("firmware 5.x is not supported yet").
- Each zip was confirmed by an HTTP HEAD on Kobo's CDN (status 200, size, Last-Modified).

| # | Version | CDN path (`https://ereaderfiles.kobo.com/firmwares/kobo9/…`) | Size | CDN Last-Modified | Source / notes |
|---|---|---|---|---|---|
| 1 | **4.38.23828** (device's current) | `Sep2026/kobo-update-4.38.23828.zip` | 121,126,185 B | 22 Sep 2026 | Kobo API "latest"; MobileRead t=375464 (released for "a bunch of Kobos"; patches from 23697 mostly work) |
| 2 | **4.38.23697** | `May2026/kobo-update-4.38.23697.zip` | 120,808,533 B | 26 May 2026 | HEAD returned 200 on the kobo9 path; mentioned in t=375464 as the prior build. **verify** that the package really contains a Freescale/Libra 2 image (check `upgrade/` kernel names and `KoboRoot.tgz` `usr/local/Kobo/version`), given the 4.39.22801 mislabel incident |
| 3 | **4.38.23684** | `Apr2026/kobo-update-4.38.23684.zip` | 120,807,586 B | 13 Apr 2026 | matches `EarliestChangeLog=4.38.23684` on the user's device |
| 4 | **4.38.23648** | `Mar2026/kobo-update-4.38.23648.zip` | 120,798,875 B | 18 Mar 2026 | MobileRead t=372740: Libra 2 rollout, Kobo Remote stability + credit-card fix |
| 5 | **4.38.23552** | `Nov2025/kobo-update-4.38.23552.zip` | 120,793,039 B | 17 Nov 2025 | pgaskin kfw DB entry for kobo9; MobileRead t=370942 (older BT devices incl. Libra 2) |
| – | 4.38.23429 (Aug 2025) | `Aug2025/kobo-update-4.38.23429.zip` | 121,752,299 B | 22 Aug 2025 | sixth newest, kept only as "known-older: drawer off" test input |

Order between 23697 and 23684 is by build number and CDN month (May 2026 vs Apr 2026). If 23697 turns out not to be a real Libra 2 build, 4.38.23429 moves into slot 5.

### 3.2 Design for multi-firmware support

1. **Symbol table per firmware** — `re/symbols/symbols-<fw>.md` (human) plus `re/symbols/symbols-<fw>.json` (machine), one per build, generated from the same source `re/symbols/required.yaml` which lists each capability with its mangled symbol and a status per firmware (`same`, `renamed:<new>`, `missing`, `untested`).
2. **CI symbol check against all 5 libnickel builds** — `tools/symcheck.py` runs `arm-nickel-linux-gnueabihf-nm -D --defined-only libnickel.so.1.0.0` (or `readelf --dyn-syms`) for every supported build and asserts every `required` symbol exists and every `optional` symbol is at least listed with a status. The libnickel files are **not** in the repo; CI downloads each official zip from Kobo's CDN into a cache keyed by version (URLs in `re/firmware.lock` with sha256), extracts `KoboRoot.tgz` → `usr/local/Kobo/libnickel.so.1.0.0`, runs the check and deletes the extraction. Nothing from Kobo is stored as an artifact.
3. **Runtime version detection** — at init read `/mnt/onboard/.kobo/version` (format `serial,…,4.38.23828,…`; field order **verify**, NickelDBus reads the same file) and compare against the table compiled into the mod (`fw_table.h` generated from `required.yaml`). Then do an actual `dlsym` of every needed symbol anyway. Decision rules:
   - known firmware, all symbols found → drawer on;
   - unknown 4.38.2xxxx build newer than the table, all symbols found → drawer on with a log line `fw not in table, symbols OK` and a one-time toast "Ứng dụng: firmware mới, chế độ thử"; 
   - any required symbol missing, or 3.x/5.x → drawer off, log reason, no toast at boot (toasts at boot can race the UI), the NickelMenu fallback is already there; the first time NickelMenu's "Ứng dụng" item is tapped it shows `cmd_output` text "Ngăn ứng dụng tắt: <reason>".
4. **Testing on one physical device** — the user has one Libra 2 on 4.38.23828 and uses it daily. Policy: **static checks for all five builds, dynamic tests on the installed build only.** Flashing a different build is optional and risky:
   - Upgrading/downgrading by dropping the official zip's contents (`upgrade/` folder, `KoboRoot.tgz`, `manifest.md5sum`) into `.kobo/` and ejecting is the documented manual-update route; downgrading within 4.38.2xxxx is reported to work on MobileRead but **verify**; it resets mods (every mod reinstalls because `KoboRoot.tgz` overwrites `/usr/local/Kobo`) and may re-trigger onboarding and Vietnamese UI fixes (kobo-tieng-viet auto-repairs after an extra reboot).
   - If a downgrade test is ever wanted, do it at most once, on 4.38.23684 (the previous build already seen on the device), with a full `.kobo` + `.adds` backup, a charged battery, and the NickelMenu-only configuration installed first. Not scheduled by default.
5. **Extra time** — about 2.5 developer days in total: 0.5 d firmware fetch/verification script, 1 d Ghidra batch import + diff of the four extra builds, 0.5 d `symcheck` CI, 0.5 d runtime detection and tests. Spread across Phases 2 and 3 and already included in the timeline (section 5).

---

## 4. Phase-by-phase breakdown

Estimates are agent working hours. "User" lines are what the user must do; they are grouped into batched test sessions (section 5.3).

### Phase 0 — Clean base (done, 8 Oct 2026)

- KOReader removed, backup at `~/kobo-backup-20261008`.
- Vietnamese UI mod (kobo-tieng-viet) and En–Vi dictionary (tudien) staged, install on eject.
- LocalSend receiver (`bethropolis/localgo`, linux/arm GOARM=7) at `.adds/localsend` with `start.sh`/`stop.sh` and NickelMenu items.
- NickelMenu with a custom "Tweak" bottom tab via `experimental:menu_main_15505_*`.
- `ExcludeSyncFolders` set so dot-folders are not imported.

**Remaining acceptance (user, session T0):** the device boots after eject, Vietnamese UI is active, dictionary works in a book, LocalSend start/stop menu items work and a phone can send one EPUB.

### Phase 1 — App folders + NickelMenu fallback (≈ 1 day)

**Objective:** define the on-disk app format now, so every later phase and the fallback use the same folders. Ship LocalSend as app #1 and GitHub sync as app #2 through NickelMenu only.

| # | Task | Est (h) |
|---|---|---|
| 1.1 | Write `spec/app.schema.json` (`id`, `name`, `name_vi`, `version`, `icon`, `backend` {cmd, cwd, env}, `actions[]` {id, label, label_vi, run, needs: [wifi, rescan_after], confirm}, `menu` {show: bool}, `min_host`, `fallback_only` flag) | 2 |
| 1.2 | Write `tools/gen-menu.sh` (POSIX sh, runs on the Kobo and on the Mac): scans `.adds/apps/*/app.json`, emits `.adds/nm/apps` with one `menu_item :main :Ứng dụng › <name> › <action> :cmd_spawn|cmd_output …` per action, chaining `nickel_wifi:autoconnect` / `nickel_misc:rescan_books` when `needs` says so. Uses a tiny `jq`-free parser (busybox `sed`/`awk`), so **verify** that the app.json subset stays flat enough for that; else ship a static `jsonq` Go binary | 3 |
| 1.3 | Move LocalSend to `.adds/apps/localsend/` (`app.json`, `bin/localgo`, `bin/start.sh`, `bin/stop.sh`, `data/`), keep the old `.adds/localsend` NickelMenu entries until the new ones are verified, then remove | 1 |
| 1.4 | GitHub sync backend v0 (Go, `bin/ghsync`): reads `data/config.json` {repo, path, branch, token_file}, lists the folder via the Contents API with the fine-grained read-only PAT from `data/token` (mode 600), downloads files not present in `/mnt/onboard/<dest>/` (add-only, never delete, atomic temp-file + rename, skip if same size+sha), writes `data/state.json`, prints one status line. Cross-compile `GOOS=linux GOARCH=arm GOARM=7`, static | 4 |
| 1.5 | Menu items for GitHub sync ("Đồng bộ GitHub ngay" = wifi autoconnect → ghsync → rescan_books) and Library tool placeholder | 0.5 |
| 1.6 | README for the user: where to paste the token, how to add an app folder | 0.5 |

**Deliverables:** `spec/app.schema.json`, `tools/gen-menu.sh`, `.adds/apps/localsend/`, `.adds/apps/ghsync/`, `.adds/nm/apps`.

**Acceptance (device, session T1):** NickelMenu shows "Ứng dụng › LocalSend › Bắt đầu nhận" etc.; a file from the phone arrives and appears in the library after "rescan"; "Đồng bộ GitHub ngay" pulls one new EPUB from the private repo and nothing is deleted; menu regenerates after adding a dummy app folder and rebooting.

**Risks:** NickelMenu's `cmd_spawn` runs from `/` as root: scripts must `cd` themselves. **Rollback:** delete `.adds/nm/apps`, restore `.adds/localsend` items. **Dependencies:** user supplies repo `owner/repo`, folder, token (section 13). **User:** create the PAT, paste into `data/token`, run session T1.

### Phase 2 — Toolchain + RE setup + hello-world (≈ 3 days, +1.5 d for the firmware matrix)

**Objective:** a reproducible build on the Mac, all five libnickel builds analysed in Ghidra with Ghidra reachable from the maintainer, and a hello-world mod whose failsafe and uninstall paths were exercised on the device.

| # | Task | Est (h) |
|---|---|---|
| 2.1 | Create repo skeleton (section 11), add NickelHook submodule, `make -f NickelHook/NickelHook.mk NAME=koboapps` generator, Makefile targets `all`, `koboroot`, `clangd` | 1.5 |
| 2.2 | Docker wrapper `tools/tc` that runs `ghcr.io/pgaskin/nickeltc:1` with `--platform linux/amd64`; confirm `arm-nickel-linux-gnueabihf-g++ --version` = 4.9.4 and `qmake -query` shows Qt 5.2.1 (**verify** image tag `:1` vs `:1.0`; nickel-screensaver uses `:1.0`) | 1.5 |
| 2.3 | `tools/fetch-fw.sh <ver>`: downloads the official zip for kobo9 into `~/.cache/koboapps/fw/<ver>/` (outside the repo), checks sha256 against `re/firmware.lock` (first run records it), extracts only `KoboRoot.tgz` → `usr/local/Kobo/libnickel.so.1.0.0`, `libnickel.so.1.0.0` symbol dump via `nm -D`, and the `version`/`upgrade` identifiers to verify it is a Freescale (Libra 2) image. Run for all five builds + 4.38.23429 | 2.5 |
| 2.4 | Ghidra: install (needs JDK 21 — **verify** current Ghidra 11.x requirement), create project `libnickel`, import the five `libnickel.so.1.0.0` as separate programs named by version, auto-analyse (ARM v7 LE, Thumb aware; enable "Demangler GNU" and "Shared Return Calls"). Expect ~30–60 min per binary on an M-series Mac; run as a batch overnight with `analyzeHeadless` | 3 |
| 2.5 | Ghidra: install the plugin zip in Ghidra (`File → Install Extensions`), enable `GhidraPlugin`, run `bridge_mcp_ghidra.py` (Python 3 + `mcp` SDK), register as an MCP server in the maintainer (`
| 2.6 | Hello-world mod: `nh_info{name="koboapps", uninstall_flag=".adds/koboapps/uninstall", failsafe_delay=15}`, one optional dlsym (`MainWindowController::sharedInstance` + `toast`), `init()` reads `/mnt/onboard/.kobo/version`, logs it, and schedules a toast "Xin chào từ Ứng dụng" 10 s after start through a `QTimer::singleShot` on the GUI thread. `make koboroot` | 3 |
| 2.7 | Failsafe drill design: build variant B that deliberately `abort()`s in `init()` to confirm NickelHook removes the mod and Nickel boots; variant C normal. Write the test script for the user (section 9.5, cases SAFE-01..04) | 1 |
| 2.8 | `tools/symcheck.py` v0: reads `re/symbols/required.yaml`, runs over every cached `libnickel` symbol dump, prints a matrix; wire into `make check` | 2.5 |
| 2.9 | `re/symbols/symbols-4.38.23828.md` skeleton with the symbols the hello-world and NickelMenu/NickelDBus/NickelHardcover already use (section 6.5 format) | 1.5 |
| 2.10 | Dev loop tooling: `tools/push.sh` (copies KoboRoot.tgz to `.kobo/` over USB or over LocalSend/FTP when the device is not mounted; **verify** which the user prefers), `tools/logs.sh` (collects `.kobo/koboapps_*.log`, `logread` via telnet if enabled) | 1.5 |

**Deliverables:** repo building in Docker, `KoboRoot.tgz` of hello-world (variants B and C), Ghidra project with 5 programs, Ghidra reachable, `symcheck` matrix for 5 builds, `symbols-4.38.23828.md` v0.

**Acceptance (device, session T2):** SAFE-01 variant B: Nickel boots within 2 minutes, mod file gone, log shows failsafe. SAFE-02 variant C: toast appears after boot, log shows firmware string. SAFE-03 uninstall file: mod removed on next boot. SAFE-04: USB connect/disconnect with the mod loaded, no errors. Gate **G1**.

**Risks:** NickelTC under amd64 emulation may be slow or crash (mitigation: build in a cloud x86_64 runner via CI if needed); Ghidra analysis of a ~40 MB C++ binary is slow (headless batch). **Rollback:** failsafe, uninstall flag, or delete `/usr/local/Kobo/imageformats/libkoboapps.so` over telnet. **User:** approve installing Ghidra/Ghidra and Docker use (section 13), run session T2 (~30 min).

### Phase 2b — RE spike (≈ 3 days)

**Objective:** answer the two questions that decide the drawer design before writing it: (a) can we push a full-screen view that keeps the bottom tab bar and the status bar, exactly like Home/Books; (b) can we instantiate Nickel's keyboard, list rows and toggles outside Nickel's own dialogs. See section 6 for the capability list.

| # | Task | Est (h) |
|---|---|---|
| 2b.1 | Trace `MainNavView` ↔ `MainNavController` ↔ `MainWindowController::pushView` ↔ the Home/Library view classes; find how the tab bar is attached (decompile constructors, `QObject::objectName` strings, stylesheet class names) | 4 |
| 2b.2 | Prototype A: `pushView` of a plain `QWidget` with a `QLabel`; observe whether the tab bar stays, which `objectName`/`setProperty` values are needed for the Nickel stylesheet to apply | 3 |
| 2b.3 | Prototype B: header bar. Find the header widget class used by Settings (likely `N3SettingsHeader`/`HeaderView` — **verify** name) with back chevron and title | 3 |
| 2b.4 | Prototype C: keyboard. Create `TouchLineEdit` + `KeyboardReceiver` + `KeyboardFrame::createKeyboard(KeyboardScript, QLocale)` inside our view (not inside an `N3Dialog`); confirm Vietnamese layout from kobo-tieng-viet appears | 3 |
| 2b.5 | Prototype D: list row + toggle. Instantiate `SettingContainer`-style rows and `TouchCheckBox`; compare pixel-by-pixel with a Settings screenshot | 2 |
| 2b.6 | Record findings in `re/notes/view-push.md`, `re/notes/keyboard.md`, update `symbols-4.38.23828.md`; cross-check each found symbol on the other four builds with `symcheck` and Ghidra "find function by name" | 2 |

**Acceptance (device, session T3):** prototypes A–D shown on device and screenshotted; user says the header/row/toggle look native. Gate **G2** (go/no-go for the native drawer). No-go fallback: "drawer as `N3Dialog` full-view mode" (NickelHardcover style) which is already proven to work, at the cost of no persistent tab bar.

### Phase 3 — Drawer host (≈ 1.5 weeks, +0.5 d firmware runtime detection)

**Objective:** the "Ứng dụng" tab, the drawer view with tiles and status, app discovery, backend process management, symbol/firmware check and the fallback switch.

| # | Task | Est (h) |
|---|---|---|
| 3.1 | `src/fw/` — version file parser, `fw_table.h` generator from `required.yaml`, decision function (section 3.2 rules), unit-tested on host | 3 |
| 3.2 | `src/nickel/` — thin wrappers around every dlsym'd symbol (`Nickel::toast()`, `Nickel::pushView()`, …) each checking for `nullptr`, so the rest of the code never touches raw symbols | 3 |
| 3.3 | Tab button: hook `MainNavView` ctor, add `MainNavButton` "Ứng dụng" with icons (`res/icons/apps.png`, `apps_active.png`, dark variants), bridge `tapped()`; respect NickelMenu's own extra button (both mods hook the same ctor — order test) | 4 |
| 3.4 | Drawer view: `DrawerView : QWidget` with header "Ứng dụng · N ứng dụng", a 2-column grid of `AppTile` (icon, name, one status line), "Thêm ứng dụng" tile, empty state text. Styled by Nickel's stylesheet through objectNames found in 2b | 6 |
| 3.5 | App registry: scan `/mnt/onboard/.adds/apps/*/app.json` on tab tap (not at boot), validate against schema (hand-written minimal validator, no std::), cache in memory, re-scan if directory mtime changed | 3 |
| 3.6 | Backend manager: `QProcess` per app, `start(app)`, `stop(app, graceful=true)`, stdin/stdout line framing (max line 64 KB), JSON parse with `QJsonDocument`, event dispatch, stderr → `data/log/backend.log` (rotate at 512 KB), exit code handling, restart policy (max 2 restarts per screen session) | 6 |
| 3.7 | Status line protocol (`{"status":"…"}`) for tiles; the host runs each app's `status` command (short-lived, 2 s timeout) when the drawer opens, in sequence, never in parallel | 2 |
| 3.8 | Host services the backend can `request`: `wifi_on`, `rescan_library`, `toast`, `confirm`, `open_url` (browser, optional), `sleep_inhibit` (for long syncs; **verify** the PowerTimer API seen in NickelHardcover `PowerTimer::fireAt`) | 4 |
| 3.9 | USB handling: hook/observe `PlugWorkflowManager` signals (NickelDBus uses `PlugManager::gadgetMode`); on USB connect: stop all backends, pop app screens, close drawer | 3 |
| 3.10 | Logging: `nh_log` with categories, ring buffer of the last 200 lines dumped to `.adds/koboapps/log/host.log` on error (write-and-close in a signal handler) | 2 |
| 3.11 | Settings screen for the mod itself (rendered with the kit later; for now a plain list): drawer on/off, verbose log, show hidden apps | 2 |
| 3.12 | `gen-menu` integrated: the host regenerates `.adds/nm/apps` when the app set changes (write-and-close), so the fallback is always current | 1.5 |
| 3.13 | Soak build: idle-cost instrumentation (`/proc/self/status` logged on tab open), no timers when closed | 1.5 |

**Deliverables:** `libkoboapps.so` v0.3 with drawer, `src/fw`, `src/nickel`, `src/host`, `symbols-4.38.23828.md` v1, `required.yaml` with status for all 5 builds.

**Acceptance (device, session T4):** DRW-01..12 (section 9.9). Gate **G3**.

**Risks:** NickelMenu and our mod both hooking `MainNavView` ctor; Qt object ownership when Nickel destroys views; backends outliving Nickel sleep. **Rollback:** uninstall flag; `.adds/koboapps/config` `drawer_enabled:0`. **User:** session T4 (~45 min) plus 72 h soak with the drawer installed but unused (section 9.6).

### Phase 4 — UI kit v1 (≈ 2 weeks)

**Objective:** the ten components plus confirm dialog, bindings, partial refresh, dark mode, Vietnamese strings, error states, Mac preview stub.

| # | Task | Est (h) |
|---|---|---|
| 4.1 | `ui.json` schema v1 (`spec/ui.schema.json`) and loader; screen stack (`push`/`pop`), back chevron pops | 4 |
| 4.2 | Binding engine: `$var` resolution, `set`/`append`/`remove` ops, dirty tracking per widget, batched repaint on the next event-loop tick | 5 |
| 4.3 | Components, each as `kit/<name>.cc` wrapping a Nickel widget where one exists: Header, Text, KeyValue, Toggle, TextField (keyboard), BigCode, Progress, List, Button, Toast; Confirm dialog via `ConfirmationDialogFactory` | 20 (2 h each) |
| 4.4 | Layout: vertical stack with Nickel spacing, scroll with Nickel's `QScrollArea` behaviour (page-wise), min tap target 120 px (≈ 10 mm at 300 dpi), footer button pinned | 4 |
| 4.5 | E-ink refresh strategy: use Nickel's own update path (no direct `mxcfb` ioctls in v1); full refresh on screen push/pop, partial on bound-value changes; count flashes in tests | 3 |
| 4.6 | Dark mode: verify every component inverts correctly via the Nickel stylesheet; fix icons (PNG pairs) | 2 |
| 4.7 | i18n: `name_vi`/`label_vi` fields; host strings in `res/strings_vi.json` and `strings_en.json`; pick by Nickel's locale setting (`Settings` key **verify**: `ApplicationSettings.Locale`?) | 2 |
| 4.8 | Error states: backend missing/crashed banner inside the screen with "Thử lại"/"Quay lại", schema error screen listing the offending JSON path, timeout spinner text | 3 |
| 4.9 | Host-side unit tests (Qt 5 on the Mac via Homebrew `qt@5`, **verify** availability on Apple Silicon): binding engine, JSON parser edge cases, layout math | 4 |
| 4.10 | `preview/` Mac app v0: Qt widgets at 1264×1680 scaled to 50 %, greyscale, renders `ui.json` with the same kit code behind a `#ifdef KA_HOST_PREVIEW` shim for Nickel widgets | 6 |
| 4.11 | Component sheet app `apps/_kitdemo` exercising every component with sample bindings; golden screenshots on device (NickelMenu `nickel_screenshot:capture` or `nickel_setting:toggle:screenshots`) | 2 |

**Acceptance (device, session T5):** KIT-01..14. Gate **G4**.

**Risks:** keyboard outside N3Dialog may need `N3Dialog` after all (then TextField opens a native text-edit dialog, which is acceptable and still native); partial refresh ghosting. **Rollback:** kit disabled by config, drawer still lists apps and runs actions via the fallback-style action list.

### Phase 5 — Port the tools (≈ 1 week)

| # | Task | Est (h) |
|---|---|---|
| 5.1 | LocalSend: rewrite `start.sh`/`stop.sh` into a long-running backend wrapper (`bin/ls-host.sh` or Go shim) that starts `localgo`, parses its output into `set`/`append` messages (pin, ip, alias, received list), handles `action:start/stop`, `toggle:auto_rescan`, emits `request:rescan_library` after each file. **verify** localgo prints PIN/received events in a parseable form; otherwise poll the receive folder | 5 |
| 5.2 | LocalSend `ui.json` per the wireframe (badge, text, bigcode, kv, toggle, list, primary button) | 1.5 |
| 5.3 | GitHub sync: extend `ghsync` to protocol mode (`--host`): progress `set` {done,total,current}, errors as `toast`, `confirm` before first sync of a new repo, settings screen with TextFields for repo/path and a toggle "Tự đồng bộ khi bật Wi-Fi" (implemented as: on `action:sync` the host first `request:wifi_on`) | 5 |
| 5.4 | GitHub `ui.json` (main + settings screens) | 1.5 |
| 5.5 | Library tool backend (Go, `bin/libtool`): (a) dedupe by normalised title+author (NFC, case-fold, strip punctuation) and by file sha256, listing candidates only; delete only after `confirm`; (b) fix `dc:language`/`xml:lang` → `vi` in `content.opf`/XHTML of chosen EPUBs (rewrite zip entries, keep order, `mimetype` first uncompressed, write temp + rename); (c) NFC-normalise file names and OPF title/creator; (d) always `request:rescan_library` after changes. Works on `/mnt/onboard` only in short bursts and never while USB is connected | 8 |
| 5.6 | Library `ui.json` (scan → results list → confirm → progress → done) | 1.5 |
| 5.7 | Remove the legacy `.adds/localsend` entries; regenerate menu fallback | 0.5 |
| 5.8 | App-level tests (section 9.9 LS-*, GH-*, LIB-*) and 72 h soak with LocalSend used daily | 2 |

**Acceptance (device, sessions T6, T7):** LS-01..08, GH-01..08, LIB-01..06. Gate **G5**.

**Risks:** EPUB rewriting bugs corrupt books — mitigations: operate on a copy, verify zip integrity (`unzip -t` equivalent in Go) before rename, keep `.bak` for 7 days in `data/trash/`. **User:** sessions T6/T7, confirm dedupe candidates by eye.

### Phase 6 — Developer kit (≈ 1 week)

| # | Task | Est (h) |
|---|---|---|
| 6.1 | `APPS.md`: folder layout, `app.json`, `ui.json` reference (every component, prop, event), protocol reference, lifecycle, limits, fallback behaviour, Vietnamese/English | 5 |
| 6.2 | `apps/_template/` hello app (shell backend) + `make new-app ID=…` | 2 |
| 6.3 | `tools/validate` (Go, single binary for Mac and Kobo): schema validation of `app.json`/`ui.json`, unknown `$var` detection, tap-target lint, string length lint for 1264 px width | 5 |
| 6.4 | Mac preview v1: hot-reload on file change, fake backend mode (feeds a `.jsonl` script), screenshot export for golden tests | 5 |
| 6.5 | `preview` golden tests in CI for `_kitdemo`, LocalSend, GitHub, Library screens | 2 |
| 6.6 | Release pipeline: `make release` → `koboapps-<ver>-KoboRoot.tgz`, `apps-<id>-<ver>.zip`, checksums, GitHub Release draft | 2 |
| 6.7 | Timed exercise: write a brand-new "Pin & giờ" (battery/time) app from the template using only `APPS.md`; measure against S13 | 2 |

**Acceptance:** CI green (build, symcheck ×5, unit, golden), S13 met, docs reviewed by the user. Gate **G6 = v1.0**.

### Phase 7 (optional, Jan 2027) — Catalog

Out of v1 scope. Minimal version: a `catalog.json` in a GitHub repo listing app zips with sha256; the host's "Thêm ứng dụng" tile downloads and unpacks into `.adds/apps/` after a confirm. Requires a trust decision (signing) — deferred.

---

## 5. Calendar timeline, milestones and gates

Assumptions: Mon–Fri, ~5 agent hours/day (25 h/week), user test sessions mostly in evenings or weekends, batched (the device is in daily use). Public holidays: none affecting Oct–Dec 2026 in the schedule except the week of 21 Dec which is a half week.

### 5.1 Week-by-week

| Week | Dates (2026) | Phase | Agent work (≈ h) | User sessions | Milestone / gate |
|---|---|---|---|---|---|
| W1 | Mon 12 – Fri 16 Oct | P1 (app folders, menu fallback), P2 start (repo, Docker TC, fetch-fw ×6) | 25 | **T0** (Phase 0 check, 15 min, Mon/Tue) · **T1** (Phase 1, 30 min, Thu/Fri) | M1: Phase 1 shipped via NickelMenu (Fri 16) |
| W2 | 19 – 23 Oct | P2 (Ghidra batch, Ghidra, hello-world, symcheck, push/log tools) | 25 | **T2** (failsafe drill + hello toast + USB, 30 min, Fri 23 or Sat 24) | **G1** go/no-go: failsafe and uninstall proven (Fri 23) |
| W3 | 26 – 30 Oct | P2b RE spike (view push, header, keyboard, rows), cross-check on 5 builds | 25 | **T3** (prototypes A–D, 30 min, Fri 30) | **G2** go/no-go: native view with tab bar + keyboard confirmed (Fri 30) |
| W4 | 2 – 6 Nov | P3 drawer host part 1 (fw detect, wrappers, tab, drawer view, registry) | 25 | – | – |
| W5 | 9 – 13 Nov | P3 part 2 (backend manager, services, USB, logging, settings) | 25 | **T4** (drawer tests DRW-*, 45 min, Fri 13); start 72 h idle soak Fri→Mon | **G3**: drawer host accepted (Mon 16 after soak numbers) |
| W6 | 16 – 20 Nov | P4 UI kit part 1 (schema, bindings, 5 components, layout) | 25 | – | – |
| W7 | 23 – 27 Nov | P4 part 2 (5 components, refresh, dark mode, i18n, errors, unit tests, preview v0, kit demo) | 25 | **T5** (KIT-*, 45 min, Fri 27 / Sat 28) | **G4**: kit v1 accepted (Mon 30) |
| W8 | 30 Nov – 4 Dec | P5 (LocalSend, GitHub sync) | 25 | **T6** (LS-*, GH-*, 45 min, Fri 4) | – |
| W9 | 7 – 11 Dec | P5 (Library tool, legacy cleanup, app tests), 72 h usage soak | 25 | **T7** (LIB-*, failure injection FI-*, 60 min, Fri 11) | **G5**: apps accepted (Mon 14) |
| W10 | 14 – 18 Dec | P6 dev kit (APPS.md, template, validator, preview v1, golden, release) | 25 | – | – |
| W11 | 21 – 23 Dec (3 days) | Buffer, timed exercise, release v1.0, firmware rehearsal (FW-* static) | 15 | **T8** (final sign-off + uninstall/reinstall check, 30 min, Wed 23) | **G6 = v1.0 released** (Wed 23 Dec) |
| (Jan 2027) | optional | P7 catalog | – | – | – |

Total: 11 weeks, ≈ 265 agent hours, 9 user sessions totalling ≈ 5.5 hours plus two 72 h soaks that need no attention beyond noting battery % twice a day.

### 5.2 Buffers and slip rules

- Built-in slack: W11 (3 days) plus ~2 h/week unallocated inside each phase estimate.
- If G2 is **no-go** (no clean view push with tab bar): switch to the `N3Dialog` full-view design; costs ~3 days; the kit and apps are unaffected because they target the kit API, not the container. Release moves to ~Thu 7 Jan 2027.
- If NickelTC under amd64 emulation fails: build in GitHub Actions (x86_64) and download artifacts; costs ~1 day in W2.
- If any phase slips > 3 days, the agent files a blocker report (section 10) and the timeline is replanned at the next gate rather than silently compressing later phases.

### 5.3 Batched user test sessions

Each session has a checklist (section 9.9), takes 15–60 minutes, starts with a backup of `.adds` and `.kobo/KoboReader.sqlite` to the Mac, and ends with log collection (`tools/logs.sh`). Between sessions the device runs the last accepted build so daily reading is never interrupted by a half-finished build. A "quick fix" re-test outside the batch is allowed only for S3-class safety issues.

### 5.4 Go/no-go gates

| Gate | When | Pass criteria | No-go action |
|---|---|---|---|
| G1 | end W2 | SAFE-01..04 pass; build reproducible; 5 libnickel builds fetched and verified as Libra 2 images; symcheck runs | Stop native work; keep Phase 1 fallback; investigate toolchain/failsafe blocker |
| G2 | end W3 | Prototype A pushes a view with tab bar; C shows Kobo keyboard with Vietnamese layout; all symbols used exist on 5/5 builds (or have per-build aliases) | Switch to N3Dialog full-view container (plan B) |
| G3 | after W5 soak | DRW-01..12 pass; S1, S4, S8 met; symbol/firmware check behaves on a simulated unknown firmware | Fix before kit work starts; kit work may begin on the Mac preview meanwhile |
| G4 | after W7 | KIT-01..14 pass; S11 ≥ 4/5; unit tests + golden tests green | Reduce v1 component set (drop BigCode/Progress) and continue |
| G5 | after W9 | LS/GH/LIB/FI cases pass; S5, S6, S7, S12 met | Ship LocalSend + GitHub only; Library tool to v1.1 |
| G6 | 23 Dec | CI green incl. symcheck ×5; S13 met; uninstall verified; user signs off | Release as 1.0-rc; fix list to January |

Note (2026-10-10, see `docs/DECISIONS.md`): the 72 h idle and usage soaks (SOAK-1, SOAK-2) are skipped. G3 and G5 pass on one batched device session, `docs/sessions/T-final-checklist.md`, which also covers G4; S4 and S5 are not measured, `debug.log` stays on in the beta. GitHub sync cases (GH-*) are postponed and do not block G5.

---

## 6. Reverse-engineering workplan

### 6.1 Legal stance

Analysis of `libnickel.so` is done for interoperability with software on a device the user owns, the same way NickelMenu, NickelDBus, NickelHardcover and kobopatch were made. Kobo binaries stay in a local cache on the Mac (and in a CI cache that is never published). The repository contains only our own code, symbol **names**, offsets-free notes and the download URLs of Kobo's public firmware packages. No decompiled Kobo code is committed; notes paraphrase behaviour.

### 6.2 Obtaining libnickel for each build

```
tools/fetch-fw.sh 4.38.23828
# 1. curl https://ereaderfiles.kobo.com/firmwares/kobo9/Sep2026/kobo-update-4.38.23828.zip
#    → ~/.cache/koboapps/fw/4.38.23828/kobo-update.zip  (121,126,185 bytes; sha256 recorded in re/firmware.lock on first fetch)
# 2. unzip -l shows: upgrade/ (kernel, u-boot images), KoboRoot.tgz, manifest.md5sum
#    verify Freescale/Libra 2 image: upgrade/ names contain imx6sll / "io" and NOT mt8113 (verify exact names)
# 3. tar -xzf KoboRoot.tgz ./usr/local/Kobo/libnickel.so.1.0.0 ./usr/local/Kobo/version (verify path of version file inside KoboRoot; otherwise read from etc/version)
# 4. arm-nickel-linux-gnueabihf-nm -D --defined-only libnickel.so.1.0.0 | c++filt > symbols.txt
# 5. sha256 of libnickel recorded next to it
```

The device's running copy can be cross-checked: `/usr/local/Kobo/libnickel.so.1.0.0` on the Kobo (via telnet/FTP from NickelMenu's example items) must have the same sha256 as the extracted 4.38.23828 one.

### 6.3 Ghidra + Ghidra setup (one-time, Mac)

1. Install JDK 21 (`brew install --cask temurin@21`) and Ghidra (`brew install --cask ghidra`, **verify** cask availability/version; else download from ghidra-sre.org).
2. Download the Ghidra release zip (plugin + `bridge_mcp_ghidra.py`). In Ghidra: `File → Install Extensions → +`, pick the zip, restart, enable `GhidraPlugin` under `File → Configure → Developer`. Default HTTP port 8080 (`Edit → Tool Options → Ghidra HTTP Server`).
3. `python3 -m venv ~/.venvs/Ghidra && pip install mcp requests` then register in the maintainer: `
4. Headless batch import/analysis for the five binaries:
   `analyzeHeadless ~/ghidra-projects libnickel -import libnickel-4.38.23828.so.1.0.0 -processor ARM:LE:32:v7 -analysisTimeoutPerFile 7200` (repeat per build, naming programs `libnickel-<ver>`).
5. Open the CodeBrowser on `libnickel-4.38.23828`; Ghidra serves whichever program is open, so switching builds means opening another program (or running a second Ghidra instance on another port).
6. Smoke test from the maintainer: `list_classes`, `search_functions_by_name MainNavView`, `decompile_function`.

### 6.4 Capabilities to locate (with starting points)

| # | Capability | Starting symbols/classes (known from other mods) | What to establish | Needed by |
|---|---|---|---|---|
| R1 | Push a full-screen view keeping the bottom tab bar and status bar | `MainWindowController::pushView(QWidget*)`, `currentView()`, `viewWithObjectName(QString)`, `MainNavController::home()`, `MainNavView` ctor | How Home/Library views are created and shown; whether the tab bar is part of the pushed view or of the window; objectNames/properties the stylesheet keys on | Drawer (G2) |
| R2 | Tab bar button | `MainNavButton` ctor, `setPixmap`, `setActivePixmap`, `setText`, `tapped()` | Active-state handling when our view is current; icon paths and sizes | Drawer |
| R3 | Header with back chevron and title/action | class used by Settings pages (**verify** name; candidates `N3SettingsHeader`, `HeaderView`, `TitleBar`) | ctor signature, how "back" pops the view | Kit Header |
| R4 | Settings-style list rows, toggles, static rows | `SettingContainer(QWidget*)`, `TouchCheckBox(QWidget*)`, `TouchLabel`, `N3ButtonLabel`, NickelHardcover's `settings/checkboxrow.cc`, `menurow.cc`, `staticrow.cc` | Which widget + objectName gives the exact Settings look | Kit KeyValue/Toggle/List |
| R5 | N3 dialogs | `N3DialogFactory::getDialog(QWidget*, bool)`, `N3Dialog::setTitle/enableFullViewMode/enableBackButton/disableCloseButton/showKeyboard/hideKeyboard/keyboardFrame` | Full-view mode as plan B container; keyboard hosting | Plan B, TextField |
| R6 | Confirmation dialogs | `ConfirmationDialogFactory::getConfirmationDialog(QWidget*)`, `showErrorDialog`, `showTextEditDialog(QString)`, `ConfirmationDialog::setTitle/setText/setAcceptButtonText/setRejectButtonText/showCloseButton/setRejectOnOutsideTap/addWidget` | Signals for accept/reject (QDialog accepted/rejected) | Kit Confirm |
| R7 | Toast | `MainWindowController::toast(QString, QString, int)` | Duration units (ms), threading rule (GUI thread only) | Kit Toast |
| R8 | Keyboard | `KeyboardFrame::createKeyboard(KeyboardScript, QLocale const&)`, `KeyboardReceiver(QLineEdit*, bool)`, `SearchKeyboardController::setReceiver/setGoText`, `TouchLineEdit`, `TouchTextEdit::setCustomPlaceholderText` | Whether a keyboard can be parented to our view; `KeyboardScript` enum value for Latin/Vietnamese; Go/Enter signal | Kit TextField |
| R9 | Wi-Fi connect | `WirelessWorkflowManager::sharedInstance/connectWireless(bool,bool)/connectWirelessSilently/isInternetAccessible/isAirplaneMode`, `WirelessManager` signals (NickelDBus `rvConnectSignals`) | Signal to await "connected"; timeout behaviour | Host service `wifi_on` |
| R10 | Library rescan | `PlugWorkflowManager::sharedInstance/sync()`, `N3FSSyncManager::sharedInstance/sync(QStringList*)`, `pfmDoneProcessing` signal (NickelDBus) | Partial rescan of one folder vs full | Host service `rescan_library` |
| R11 | Dark mode / settings | `Settings` ctor, `getSetting/saveSetting`, `ReadingSettings::getDarkMode`, vtables `_ZTV15ReadingSettings` | Read current dark mode so icons swap | Kit theming |
| R12 | Refresh / waveform control | Nickel's own screen update path (QPA plugin `libkobo.so`? **verify**), `FeatureSettings` flags, kobopatch notes on "full refresh" | Whether a widget-level `update()` yields a partial refresh; when Nickel forces a full flash | Kit refresh strategy |
| R13 | USB / power state | `PlugManager::sharedInstance/gadgetMode`, `PlugWorkflowManager` signals, `N3PowerWorkflowManager::sharedInstance/requestSleep`, `PowerTimer` | Signals for "USB connected", "going to sleep" to stop backends | Host |
| R14 | Locale | `SupportedLocales::supportedLocales(bool)`, `ApplicationSettings` locale key | Pick vi/en strings | i18n |
| R15 | Screenshot | `FeatureSettings.Screenshots` setting; NickelMenu `nickel_screenshot:capture` | Produce test screenshots | Validation |

### 6.5 Recording symbols: `re/symbols/symbols-<fw>.md` format

```
# libnickel symbols — firmware 4.38.23828 (Libra 2 / kobo9)
libnickel sha256: <hex>   source: kobo-update-4.38.23828.zip (Sep2026)   analysed: 2026-10-20

| cap | mangled | demangled | sig (our typedef) | kind | req | status | notes |
|-----|---------|-----------|-------------------|------|-----|--------|-------|
| R2 | _ZN13MainNavButtonC1EP7QWidget | MainNavButton::MainNavButton(QWidget*) | void (*)(MainNavButton*, QWidget*) | dlsym | required-for-drawer | ok | size ≥ 0x?? bytes, we calloc 256 |
| R7 | _ZN20MainWindowController5toastERK7QStringS2_i | MainWindowController::toast(QString const&, QString const&, int) | … | dlsym | optional | ok | duration ms |
| R1 | _ZN11MainNavViewC1EP7QWidget | MainNavView::MainNavView(QWidget*) | … | hook | required-for-drawer | ok | shared with NickelMenu |
```

Machine form `symbols-<fw>.json` is generated from `required.yaml` + the `nm` dump; `kind` ∈ {hook, dlsym, vtable}; `req` ∈ {required-for-drawer, optional, fallback-only}; `status` ∈ {ok, renamed, missing, untested}. `fw_table.h` is generated from the five JSON files.

### 6.6 Firmware-update diff procedure (each new Libra 2 build)

1. `tools/fetch-fw.sh <new>` → symbol dump; `tools/symcheck.py --against <new>` prints missing/renamed symbols in under a minute.
2. If all `required-for-drawer` symbols exist unchanged: add the build to `required.yaml` as `same`, regenerate `fw_table.h`, build, run the static CI, publish a patch release. Typical cost: 1 h.
3. If some are missing: import into Ghidra, use Ghidra `search_functions_by_name` and string references to find the renamed/moved function, record `renamed:<new>`, add a per-firmware alias in `required.yaml`, re-test on device if the user updates. Typical cost: 2–6 h.
4. Until then, the runtime check keeps the drawer off on the new build and the NickelMenu fallback carries the apps. Recommend the user keeps Kobo auto-update off (section 13).

---

## 7. UI kit spec v1 and protocol spec

> **Updated 2026-10-08:** the component set, sizes, drawer behaviour (list default + optional grid, More-tab entry row, page-turn keys) and screen patterns are now defined in [`docs/spec/ui-kit.md`](spec/ui-kit.md), which supersedes this section where they differ. The JSON-lines protocol in §7.7 was **not built**: DECISIONS.md (2026-10-10) replaced it with short action scripts, a `state` action the mod re-reads, and JSON signals; see [`spec/drawer-protocol.md`](spec/drawer-protocol.md).

### 7.1 Screen file (`ui.json`)

```json
{
  "kit": 1,
  "screens": {
    "main": {
      "title": "LocalSend",
      "badge": "$status",
      "action": { "label": "Cài đặt", "push": "settings" },
      "body": [ ...components... ],
      "footer": [ { "button": "Dừng nhận", "action": "stop", "style": "primary", "enabled": "$running" } ]
    },
    "settings": { "title": "Cài đặt", "body": [ ... ] }
  },
  "start": "main",
  "vars": { "status": "Đang tắt", "running": false, "pin": "", "received": [] }
}
```

Rules: `kit` is the schema version; unknown component keys are rendered as an error row, not a crash; `vars` gives defaults so a screen renders before the backend answers; strings may be plain or `{ "vi": "...", "en": "..." }`.

### 7.2 Components

> **Note:** the components, the app-to-host events and the toast/dialog/request signals are specified in [`spec/ui-json.md`](spec/ui-json.md) and [`spec/drawer-protocol.md`](spec/drawer-protocol.md), which win over this table.

| Component | JSON key | Props | Events to backend | Nickel widget (target) | Notes |
|---|---|---|---|---|---|
| Header | screen `title`, `badge`, `action` | title, badge ($var ok), action {label, push \| action} | `action` | Settings header class (R3) | Back chevron pops; at root pops to drawer |
| Text | `"text"` | value, style: body \| caption \| heading, align | – | `TouchLabel`/`QLabel` with Nickel font classes | Word-wrap; caption uses muted colour |
| KeyValue | `"kv"` | rows: [[key, value], …] (values may be `$var`) | – | static row (R4) | Value right-aligned, elided |
| Toggle | `"toggle"` | label, bind (var name), enabled | `{"event":"toggle","bind":…,"value":bool}` | `TouchCheckBox` in a `SettingContainer` row | Reflects `$bind`; host updates var optimistically, backend may `set` it back |
| TextField | `"field"` | label, bind, placeholder, secret: bool, keyboard: text \| number \| url | `{"event":"input","bind":…,"value":str}` on Go/Enter or blur | `TouchLineEdit` + Kobo keyboard (R8) | If R8 fails, opens native text-edit dialog |
| BigCode | `"bigcode"` | value ($var), digits | – | `QLabel` with large Nickel font class | For PINs; monospace digits |
| Progress | `"progress"` | value ($var 0..1 or {done,total}), label | – | Nickel progress bar class (**verify**; NickelDBus has `NDBProgressBar`) | Partial refresh only when changed ≥ 5 % |
| List | `"list"` | items ($var array of [primary, secondary] or strings), title, empty_text, max (default 50), tap: action name | `{"event":"tap","list":…,"index":n,"item":…}` | rows (R4) | Appends redraw only new rows |
| Button | `"button"` | label, action, style: primary \| secondary, enabled ($var), confirm: {text, ok, cancel} | `{"event":"action","name":…}` | `N3ButtonLabel`/Nickel button class | Debounced 500 ms; disabled while an action is pending unless `async: true` |
| Toast | backend message | text, duration (ms, default 3000) | – | `MainWindowController::toast` | Max 1 per 2 s per app |
| Confirm | backend message / button prop | title, text, ok, cancel | `{"event":"confirm","id":…,"ok":bool}` | `ConfirmationDialog` (R6) | Modal; timeout 120 s → cancel |

### 7.3 Binding syntax

- `$name` anywhere a string prop is allowed → replaced by `vars.name`, re-rendered on change. `"$name.sub"` indexes objects; `"$list[0]"` not supported in v1.
- Template strings: `"Đã nhận $count sách"` → substitution of each `$ident`; literal `$` as `$$`.
- Booleans: `enabled: "$running"` uses truthiness; `"!$running"` negates.
- Backend ops on vars: `set` (merge object), `append` (push to array, cap at `max` of the consuming List, oldest dropped), `remove` (`{"remove":{"received":0}}` by index), `clear`.

### 7.4 Layout rules for e-ink at 300 dpi (1264 × 1680)

- Content width 1264 − 2×72 px gutters = 1120 px; vertical rhythm 24 px; row height 120 px minimum (≈ 10 mm); button height 132 px; minimum tap target 120 × 120 px; touch spacing ≥ 24 px.
- Fonts: Nickel's defaults (follow the system font and size setting); heading ≈ 1.25× body; BigCode 3× body.
- Lines: 2 px separators in Nickel's row colour; no shadows, no gradients, no animation.
- Refresh: push/pop → full refresh (one flash); bound-value change → widget `update()` only; ≥ 8 partial updates since the last full → force a full refresh on the next change to clear ghosting; never more than 2 updates per second per screen (coalesce).
- Scrolling: page-wise (tap bottom/top zones or Nickel's scroll bar), no kinetic scrolling.
- Dark mode: all colours come from Nickel's stylesheet; icons supply `*_dark.png`.

### 7.5 i18n

Strings in `ui.json` may be `{ "vi": …, "en": … }`; the host picks the Nickel locale, falls back to `vi` then `en`. Host-owned strings (`Thử lại`, `Quay lại`, `Ứng dụng`, errors) live in `res/strings_<lang>.json`. Dates use `dd/MM HH:mm`.

### 7.6 Error states

| Situation | Screen behaviour |
|---|---|
| `ui.json` invalid | Native error screen: title "Không mở được ứng dụng", JSON path and message, buttons Quay lại / Mở menu dự phòng |
| Backend binary missing / not executable | Banner at top: "Không chạy được <cmd>" + Thử lại |
| Backend crashed (exit ≠ 0 or signal) | Banner "Ứng dụng dừng đột ngột (mã n)"; Thử lại restarts (max 2 per screen session), then "Xem log" shows the last 30 stderr lines in a Text block |
| Backend silent > `timeout` after an action | Button re-enabled, toast "Không phản hồi" |
| Protocol error (bad JSON line) | Line ignored, counted; after 20 bad lines the backend is stopped with reason `protocol` |
| Wi-Fi request failed | Toast "Không kết nối được Wi-Fi", `{"event":"wifi","ok":false}` to backend |

### 7.7 JSON-lines protocol v1

> **Not implemented, dropped 2026-10-10** in favour of scripts + state polling + signals ([`spec/drawer-protocol.md`](spec/drawer-protocol.md), DECISIONS.md). Kept below as the original design record only.

Transport: backend stdin (host→backend), stdout (backend→host), UTF-8, one JSON object per `\n`-terminated line, max 64 KB per line, no pretty-printing. stderr is logging only.

**Handshake (first lines):**

```
host → backend : {"hello":{"proto":1,"host":"koboapps/0.4.0","fw":"4.38.23828","lang":"vi","screen":"main","data_dir":"/mnt/onboard/.adds/apps/localsend/data","vars":{...current vars...}}}
backend → host : {"ready":{"proto":1,"name":"localsend","version":"1.2.0","caps":["wifi","rescan"]}}
```

If `ready` does not arrive within 5 s (configurable `app.json: backend.ready_timeout_ms`), the host shows the banner and keeps the screen with defaults. A backend that announces `proto` > 1 is rejected with an error screen.

**Host → backend events**

| Message | Meaning |
|---|---|
| `{"event":"action","name":"start","id":17}` | Button or header action tapped; `id` for reply correlation |
| `{"event":"toggle","bind":"auto_rescan","value":true}` | Toggle changed |
| `{"event":"input","bind":"repo","value":"owner/repo"}` | Text field committed |
| `{"event":"tap","list":"received","index":1,"item":["Đắc Nhân Tâm.epub","860 KB"]}` | List row tapped (only if `tap` set) |
| `{"event":"confirm","id":3,"ok":true}` | Answer to a `confirm` request |
| `{"event":"wifi","ok":true}` / `{"event":"rescan","ok":true}` | Result of a host service request |
| `{"event":"screen","name":"settings"}` | User navigated (push/pop) |
| `{"event":"pause"}` / `{"event":"resume"}` | Device going to sleep / waking while the screen is open |
| `{"event":"stop","reason":"back"|"usb"|"shutdown"|"timeout"}` | Final message; backend must exit within 3 s |

**Backend → host messages**

| Message | Meaning / limits |
|---|---|
| `{"set":{"status":"Đang nhận","pin":"4821"}}` | Merge vars; max 16 KB |
| `{"append":{"received":["Nhà Giả Kim.epub","1,2 MB"]}}` | Append one item |
| `{"remove":{"received":0}}`, `{"clear":["received"]}` | Remove/clear |
| `{"toast":"Đã nhận 1 sách","duration":3000}` | Toast (rate-limited) |
| `{"confirm":{"id":3,"title":"Xoá 2 sách trùng?","text":"…","ok":"Xoá","cancel":"Huỷ"}}` | Ask; answer comes as `event:confirm` |
| `{"request":"wifi_on"}` / `{"request":"rescan_library","paths":["/mnt/onboard/Books"]}` / `{"request":"sleep_inhibit","ms":600000}` / `{"request":"open","screen":"done"}` / `{"request":"back"}` | Host services; each answered by an `event` |
| `{"status":"Lần cuối: 08/10 20:15"}` | One-line status for the drawer tile (also the only line read by the `status` command mode) |
| `{"log":"…","level":"info"}` | Goes to the app log, not the screen |
| `{"done":{"code":0}}` | Backend finished its job voluntarily; host may keep the screen |

**Versioning:** `proto` integer; host supports proto 1; additive fields are allowed without a bump; a renamed/removed field bumps the number. `app.json: "min_host": "0.4.0"` lets an app refuse older hosts (fallback: the drawer shows "Cần phiên bản mới hơn").

**Timeouts:** ready 5 s; action reply (any message) 30 s then "Không phản hồi"; confirm 120 s; graceful stop 3 s then `SIGKILL`; status command 2 s.

**Lifecycle:** spawned on screen open (`QProcess`, cwd = app folder, env `KA_DATA`, `KA_FW`, `KA_LANG`, `PATH` with `bin/` first), stopped on back-to-drawer, USB connect, shutdown, or 10 min after sleep unless `sleep_inhibit` was granted. `app.json: backend.keep_alive: true` allows a backend (LocalSend receiver) to keep running after the screen closes, until USB connect, shutdown, or the user stops it; the drawer tile then shows its live status and the host polls nothing — the backend writes `data/status` (write-and-close) at most every 10 s and the tile reads it on drawer open.

**Crash handling:** exit code and signal logged; `QProcess::errorOccurred` → banner; stderr tail kept (4 KB ring); restart policy 2 per screen session; a backend that crashes 3 times in 10 minutes is marked "lỗi" on its tile until the device reboots or the user taps Thử lại.

**Logging:** host `nh_log` (`.kobo/koboapps_*.log` on error via `nh_dump_log`, plus `.adds/koboapps/log/host.log` ring dump); per-app `data/log/backend.log` (stderr, rotated 512 KB × 2). All writes are open-write-close.

**Resource limits:** per backend `ulimit -v 256 MB` (the Libra 2 has 512 MB RAM — **verify**), nice 5; host refuses to start a 4th concurrent backend; `append` arrays capped by `max`; total vars ≤ 256 KB.

---

## 8. Safety and testing strategy

### 8.1 Failsafe testing

NickelHook arms a failsafe before `init()` and disarms it `failsafe_delay` seconds later; if Nickel crashes before that, the mod is uninstalled on the next boot. We set `failsafe_delay = 15` and do, once per major build line (hello-world, first drawer build, first kit build), the deliberate-crash drill (SAFE-01). The user can also trigger it by powering off within ~20 s of boot (NickelMenu doc).

### 8.2 Uninstall file

`/mnt/onboard/.adds/koboapps/uninstall` (NickelHook `uninstall_flag`): on next boot the mod deletes itself, deletes `.adds/nm/apps` (our generated menu, via the `uninstall()` callback) and leaves `.adds/apps/` untouched. Documented in `README` and shown on the mod's settings screen.

### 8.3 Symbol and firmware check

Section 3.2. Additionally `init()` is tiny: read version file, dlsym table, decide, install the `MainNavView` hook only if "drawer on". Nothing else happens until the tab is tapped.

### 8.4 Watchdog

- Every host service call and backend wait runs on the GUI thread with Qt timers, never a blocking wait > 50 ms.
- A `QTimer` watchdog (only while a screen is open) checks every 5 s that the backend answered within its timeouts; otherwise banner + stop.
- Backends get `prctl(PR_SET_PDEATHSIG, SIGTERM)` equivalent: the host also writes `data/pid` and `gen-menu` adds a "Dừng mọi ứng dụng" fallback item (`pkill -f .adds/apps/`).

### 8.5 Memory / CPU budgets

Host ≤ 1 MB idle, ≤ 4 MB per open screen; backend ≤ 64 MB RSS typical, 256 MB hard; CPU ≤ 2 % idle-screen; no polling timers when the drawer is closed. Measured in soak tests (section 9.6).

### 8.6 USB mass-storage behaviour

All `/mnt/onboard` access is open-read/write-close inside signal handlers or `init()`. On `PlugWorkflowManager` "connect" signal (or NickelMenu-style detection of `gadgetMode`): stop every backend (SIGTERM, 3 s, SIGKILL), pop app screens, pop the drawer. Backends must not keep files open either: LocalSend writes each received file to `data/tmp/` then renames into the target folder; GitHub sync likewise. Tested by FI-05.

### 8.7 Test matrix

| Dimension | Values |
|---|---|
| Firmware (static symbol check) | 4.38.23828, 4.38.23697, 4.38.23684, 4.38.23648, 4.38.23552 (+ 4.38.23429 as "older: drawer off" case, + a synthetic 5.x version string) |
| Firmware (dynamic, on device) | 4.38.23828 only (others optional, section 3.2 item 4) |
| Theme | light, dark |
| Font size | smallest, default, largest Nickel setting |
| Locale | vi, en |
| Mods present | NickelMenu + kobo-tieng-viet + tudien (baseline); NickelMenu removed (fallback absent); NickelDBus present (conflict check, optional) |
| Power | battery > 50 %, battery < 15 %, charging over USB-power-only adapter |
| Connectivity | Wi-Fi on, off, airplane mode, captive/no-internet |
| Storage | `.adds/apps` empty, 3 apps, 20 dummy apps |

### 8.8 Manual test scripts for the user

Each session T0–T8 is a checklist file `docs/test-sessions/T<n>.md` with: preparation (backup command, build to install, charge level), numbered steps with the expected screen (reference screenshot where available), a "what to note" column (times, battery %), and the log-collection command at the end. Screenshots: enable `nickel_setting:toggle:screenshots` via NickelMenu or use `nickel_screenshot:capture`; files land on the onboard partition root as `screen_NNN.png` (**verify** path/name on 4.38) and are copied to `docs/screens/<session>/` on the Mac.

### 8.9 Crash log collection

`tools/logs.sh` collects: `.kobo/koboapps_*.log` (NickelHook dump on init failure), `.adds/koboapps/log/host.log`, `.adds/apps/*/data/log/backend.log`, and if telnet is enabled `logread` (syslog) and `dmesg | tail -200`. Nickel crash signatures appear in syslog as `nickel` segfault lines. Logs are attached to the blocker report or the session notes; nothing is uploaded anywhere.

---

## 9. Validation

### 9.1 Definition of done per phase

| Phase | Done when |
|---|---|
| P1 | Schema committed; `gen-menu` idempotent; LocalSend and GitHub sync reachable and working from NickelMenu on device; session T1 signed off |
| P2 | Reproducible Docker build; 5 firmware zips fetched, verified as Libra 2 images, sha256 locked; Ghidra project with 5 programs; Ghidra smoke test; hello-world variants built; SAFE-01..04 passed; `symcheck` matrix in CI; G1 signed |
| P2b | Prototypes A–D on device, screenshots in `docs/screens/T3/`; `re/notes/*.md` written; every symbol used exists on 5/5 builds or has aliases; G2 decision recorded |
| P3 | DRW-01..12 and FW-01..05 pass; S1, S4, S8 measured and met; 72 h idle soak numbers recorded; G3 signed |
| P4 | KIT-01..14 pass; unit tests + golden tests green; S11 score recorded; G4 signed |
| P5 | LS/GH/LIB cases pass; FI-01..10 pass; 72 h usage soak; S5–S7, S12 met; legacy folders removed; G5 signed |
| P6 | CI green on a clean clone; `APPS.md` reviewed; timed exercise ≤ 30 min; release artefacts built; uninstall/reinstall verified (SAFE-05/06); G6 signed |

### 9.2 Validation layers

1. **Static checks** (every commit, CI): compiler warnings as errors for our code (`-Wall -Wextra -Werror`, GCC 4.9 limits), `symcheck.py` against the 5 cached libnickel dumps, JSON-schema validation of all `app.json`/`ui.json` in `apps/` (`tools/validate`), `shellcheck` on every `.sh`, `go vet` + `staticcheck` for Go backends, a grep gate that no `std::` symbol appears in `nm libkoboapps.so` output (except what NickelHook itself uses).
2. **Unit tests** (host, Mac, Qt 5 via Homebrew — **verify**; else run inside the NickelTC container with `qemu-user`): protocol framing and parser (long lines, partial lines, invalid UTF-8, nested JSON), binding engine (templates, truthiness, cap/append/remove), layout math (row heights, tap targets at the three font sizes), firmware decision function (table of version strings → on/off/trial).
3. **Mac preview golden tests**: `preview --screenshot` renders each screen of `_kitdemo`, LocalSend, GitHub, Library at 1264×1680 greyscale in light and dark; compared to goldens with a 0.5 % pixel tolerance; goldens updated only by explicit `make goldens` with a reviewed diff.
4. **On-device tests**: scripted checklists (section 9.9) in sessions T0–T8, with expected screenshots and log collection.
5. **Soak tests** (section 9.6).
6. **Failure-injection tests** (section 9.7).
7. **Firmware-update rehearsal** (section 9.8).
8. **Uninstall / rollback verification** (SAFE-05/06).
9. **Acceptance sign-off**: at each gate the user fills `docs/signoff/G<n>.md` (checklist results, S-criteria values, score for S11, "go"/"no-go", date).

### 9.3 Build and static tooling commands

```
make all                  # cross-build in NickelTC (docker)
make koboroot             # KoboRoot.tgz
make check                # symcheck ×5 + validate + shellcheck + nostd grep
make test                 # host unit tests (Qt 5)
make goldens / make golden-test
make release VERSION=1.0.0
```

### 9.4 Device logging during tests

Verbose mode: `.adds/koboapps/config` → `log:2`. Every test case lists the log lines expected (e.g. `fw=4.38.23828 table=known drawer=on`).

### 9.5 Safety test cases

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| SAFE-01 | Install hello-world **variant B** (abort in init); eject; wait | Kobo shows the Kobo logo, maybe a longer boot, then Home; `.kobo/koboapps_*.log` exists; `/usr/local/Kobo/imageformats/libkoboapps.so` gone (check via telnet or by reinstalling and reading log) | Nickel boots ≤ 3 min; mod removed; no second reboot loop |
| SAFE-02 | Install **variant C**; eject | Boot normal; toast "Xin chào từ Ứng dụng" ~10 s after Home; log has fw line | Toast seen; log line `fw=4.38.23828` |
| SAFE-03 | Create `.adds/koboapps/uninstall`; reboot | No toast; file consumed; mod gone | Both true |
| SAFE-04 | With mod installed: connect USB, copy a book, eject, open book | Normal behaviour, no "repairing library" prompt | No errors in log; book opens |
| SAFE-05 (P6) | Full uninstall via flag with 3 apps installed | Drawer tab gone; NickelMenu "Ứng dụng" items gone (our file deleted); `.adds/apps` intact | True |
| SAFE-06 (P6) | Reinstall release KoboRoot.tgz over an existing install | Settings preserved; apps rediscovered | True |
| SAFE-07 | Power off within 10 s of boot (manual failsafe) twice in a row | Mod uninstalled by NickelHook; Nickel boots | Documented recovery works |
| SAFE-08 | Remove NickelMenu (`.adds/nm/uninstall`) with our mod installed | Drawer still works; no crash; fallback simply absent | True; reinstall NickelMenu afterwards |

### 9.6 Soak tests

| ID | Setup | Duration | Measurements (user notes at ~08:00 and ~20:00; agent reads logs) | Thresholds |
|---|---|---|---|---|
| SOAK-1 (W5) | Drawer installed, never opened; device in normal use; Wi-Fi as usual | 72 h | Battery % at each note; Nickel VmRSS logged at boot and on each drawer open (`tools/logs.sh` after); number of backend processes = 0 (`ps` via telnet or `data/pid` absent) | Extra drain ≤ 1 %/24 h vs the user's baseline week; VmRSS growth ≤ 1 MB; no log errors |
| SOAK-2 (W9) | LocalSend used once per day, GitHub sync once per day, Library tool once | 72 h | Same + per-backend RSS peaks from `backend.log` self-reports (`/proc/self/status` printed at exit) | S5–S7; zero crashes; no stuck backends after sleep |
| SOAK-3 (W11) | Release candidate, keep-alive LocalSend receiver left on for 24 h | 24 h | Battery, RSS, Wi-Fi state | Drain ≤ 8 %/24 h with Wi-Fi held on (expected cost of Wi-Fi; verify baseline), receiver still answers |

Baseline: the user records battery % twice a day for 3 days in W1 with no mod installed (only NickelMenu etc.).

### 9.7 Failure-injection tests

| ID | Injection | Expected | Pass criteria |
|---|---|---|---|
| FI-01 | `kill -9` the backend while its screen is open (via a hidden "Debug: crash" action in `_kitdemo` or telnet) | Banner "Ứng dụng dừng đột ngột"; Thử lại restarts; after 2 restarts the button becomes "Xem log" | No Nickel crash; log shows exit signal |
| FI-02 | Corrupt `ui.json` (truncate; invalid type; unknown component; `$undefined` var) | Error screen with path; unknown component → error row; undefined var renders empty and logs once | No crash; messages match |
| FI-03 | Delete one required symbol from the compiled table (build flag `-DKA_TEST_MISSING_SYM`) | Drawer off; log reason; NickelMenu fallback works; no boot toast | True |
| FI-04 | Fake firmware version (config override `fw_override:5.1.0` test hook) | Drawer off, reason "unsupported line" | True |
| FI-05 | Connect USB while GitHub sync is downloading | Backend gets `stop:usb`, exits ≤ 3 s; temp file left only in `data/tmp/`; after eject the next sync resumes cleanly; no half-written EPUB in the library | True; `.kobo` not corrupted |
| FI-06 | Turn Wi-Fi off / airplane mode mid-sync | Backend reports error toast; screen shows "Thử lại"; no hang | Within 30 s |
| FI-07 | Battery < 15 % and start a sync | Confirm dialog "Pin yếu, vẫn tiếp tục?" (host rule when battery < 15 %) | Dialog shown; cancel aborts |
| FI-08 | Backend floods 10,000 `append` lines | List capped at `max`; UI stays responsive; CPU returns to idle after flood | ≤ 2 s stall; no OOM |
| FI-09 | Backend prints 1 MB single line | Line dropped, counted; after 20 → backend stopped "protocol" | No crash |
| FI-10 | Sleep the device (power button) with a screen open; wake after 15 min | `pause`/`resume` delivered; backend still alive (< 10 min rule) or restarted after; screen consistent | True |
| FI-11 | 20 dummy app folders, one with a 5 MB icon and one with `app.json` of 1 MB | Drawer opens ≤ 1 s; oversize icon replaced by placeholder; oversize json rejected with error tile | True |
| FI-12 | Both NickelMenu "Tweak" tab and our tab configured | Both buttons visible, both work, order stable across reboots | True |

### 9.8 Firmware-update rehearsal

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| FW-01 | `make check` with the 5 cached builds | Matrix all `ok` for required symbols | CI green |
| FW-02 | Add 4.38.23429 dump to the matrix as "unsupported older" | Report lists it as not in table; drawer would be off | Report correct |
| FW-03 | Simulate a future build: copy 4.38.23828 dump, rename one required symbol | `symcheck` fails naming the symbol; runtime (FI-03) disables drawer | True |
| FW-04 | Procedure dry-run: pretend 4.38.24xxx appeared; follow section 6.6 steps 1–2 on the 23697 dump timing the work | Documented time ≤ 1 h for the "same" case | Time recorded |
| FW-05 (optional, user decides) | Real downgrade to 4.38.23684 then upgrade back to 23828 (section 3.2 item 4) | Drawer works on both; mods re-installed; Vietnamese UI self-repairs after extra reboot | Only if the user accepts the risk; otherwise N/A |

### 9.9 On-device test case tables

**Drawer (session T4)**

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| DRW-01 | Boot with 3 apps | Bottom bar shows Trang chủ · Sách · … · Ứng dụng with icon | Icon crisp, label in Vietnamese, same size as neighbours |
| DRW-02 | Tap Ứng dụng | Drawer with header "Ứng dụng · 3 ứng dụng", 2-column tiles, tab bar remains | Painted ≤ 600 ms (log); one full refresh only |
| DRW-03 | Tap Sách then Ứng dụng again | Returns to drawer; status lines refreshed | No duplicate views (log `views=1`) |
| DRW-04 | Tap a tile | App screen pushed with header back chevron; backend `ready` logged | ≤ 800 ms |
| DRW-05 | Tap back chevron | Back to drawer; backend `stop:back` and exit ≤ 3 s | Log confirms |
| DRW-06 | Empty `.adds/apps` | Drawer shows empty text + "Thêm ứng dụng" tile | Text correct |
| DRW-07 | Add a folder via USB, eject, open drawer | New app appears without reboot | True |
| DRW-08 | Dark mode on | Drawer and tiles inverted correctly; icons swap | No white boxes |
| DRW-09 | Largest font size | Tiles reflow, nothing clipped | True |
| DRW-10 | Open drawer, press power (sleep), wake | Drawer still shown, no artefacts | True |
| DRW-11 | Mod settings screen: drawer off → reboot | Tab absent; NickelMenu items present | True |
| DRW-12 | Backend `request:wifi_on` from `_kitdemo` | Nickel's Wi-Fi connect flow appears; event `wifi ok:true` logged | ≤ 20 s |

**UI kit (session T5, using `_kitdemo`)**

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| KIT-01 | Open component sheet | Header, Text(3 styles), KV, Toggle, Field, BigCode, Progress, List, Buttons, all visible | Matches golden ± tolerance |
| KIT-02 | Compare side by side with Settings › Reading | Row height, separators, font weight identical | User rating ≥ 4/5 |
| KIT-03 | Tap Toggle | Flips instantly; event logged; backend `set` back within 1 s keeps state | One partial refresh |
| KIT-04 | Tap TextField | Kobo keyboard appears (Vietnamese layout available); typing updates field; Go commits | Event logged |
| KIT-05 | Secret field | Dots shown; value not in logs | True |
| KIT-06 | Backend `set pin` every 2 s | Only BigCode repaints | No full flash; ≤ 1 partial/2 s |
| KIT-07 | `append` 60 items with `max` 50 | List shows last 50; no scroll jump | True |
| KIT-08 | Progress 0→100 % in 5 % steps | Smooth; ≤ 20 partial updates | True |
| KIT-09 | Button with `confirm` | Native Yes/No dialog; cancel sends nothing; ok sends action | True |
| KIT-10 | Toast burst (5 in 1 s) | Max 1 per 2 s shown, rest dropped and logged | True |
| KIT-11 | Dark mode | All components correct | No artefacts |
| KIT-12 | Font size small/large | Layout holds; tap targets ≥ 120 px | True |
| KIT-13 | Locale en | Host strings switch; `{vi,en}` strings switch | True |
| KIT-14 | Error screen (bad ui.json) | Native-looking error with path; "Mở menu dự phòng" opens NickelMenu list | True |

**LocalSend app (session T6)**

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| LS-01 | Open app, tap "Bắt đầu nhận" | PIN, IP, alias shown; badge "Đang nhận"; Wi-Fi came on if off | ≤ 5 s after Wi-Fi |
| LS-02 | Send one EPUB from the phone | Row appended with name and size; toast; book appears in Sách after auto-rescan | ≤ 10 s after transfer |
| LS-03 | Send 3 files at once | 3 rows; one rescan (debounced 5 s) | True |
| LS-04 | Toggle "Tự cập nhật thư viện" off, send a file | No rescan; NickelMenu "Import books" imports it | True |
| LS-05 | Back to drawer with keep_alive | Tile shows "Đang nhận"; receiver still works | True |
| LS-06 | "Dừng nhận" | Backend exits; badge "Đang tắt" | ≤ 3 s |
| LS-07 | Connect USB while receiving | Receiver stopped; after eject, "Bắt đầu nhận" works again | True |
| LS-08 | Phone sends a 200 MB file | Progress visible; completes; RSS of localgo < 64 MB | True |

**GitHub sync app (session T6)**

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| GH-01 | First open without token | Screen explains where to put the token; sync button disabled | True |
| GH-02 | With token, set repo and folder via fields | Saved to `data/config.json`; fields persist after reopen | True |
| GH-03 | "Đồng bộ ngay" with 3 new books in the repo | Wi-Fi on → progress 0/3…3/3 → toast "Đã đồng bộ 3 sách" → rescan → books in library | ≤ 2 min on normal Wi-Fi |
| GH-04 | Run again | "0 mới", nothing downloaded | True |
| GH-05 | Delete a book on the Kobo, run again | Book re-downloaded (add-only, no tombstones in v1) — or, if user prefers, skipped via `state.json` (**decision**, section 13) | Behaviour matches decision |
| GH-06 | Delete a book in the repo, run again | Nothing deleted on the Kobo | True |
| GH-07 | Wrong token | Toast "GitHub từ chối (401)"; no crash | True |
| GH-08 | No internet | Toast and "Thử lại" | ≤ 30 s |

**Library tool (session T7)**

| ID | Steps | Expected | Pass criteria |
|---|---|---|---|
| LIB-01 | Scan with 2 known duplicates planted | List shows the 2 groups with sizes | True |
| LIB-02 | Confirm delete | Confirm dialog; deleted files go to `data/trash/`; rescan; library shows one copy | True |
| LIB-03 | Fix language on 2 EPUBs with `en` | OPF and XHTML `lang` = `vi`; EPUB still opens; Vietnamese hyphenation/dictionary works | True |
| LIB-04 | NFC normalise a file with decomposed Vietnamese name | Renamed; library entry intact after rescan | True |
| LIB-05 | Cancel mid-run | Stops at a safe point; no partial files | True |
| LIB-06 | Run with USB connected (should be impossible) | Host refuses: "Đang kết nối USB" | True |

### 9.10 Sign-off record

`docs/signoff/G<n>.md` template: gate, date, build hash, session IDs run, table of case IDs with pass/fail/notes, S-criteria measured values, S11 score, open issues with links to blocker reports, decision (go/no-go), user initials.

---

## 10. Blocker reports

### 10.1 Definitions

| Term | Meaning | Example |
|---|---|---|
| **Blocker** | Something that stops planned work from continuing and that the agent cannot resolve alone within the time-box | A needed libnickel capability does not exist; a permission the agent lacks; hardware needed for a test |
| **Risk** | Something that *might* become a blocker; tracked in the risk register with a mitigation, work continues | NickelTC emulation may be slow |
| **Question** | A decision only the user can make; work can usually continue on an assumption that is written down | Which GitHub repo to sync |

### 10.2 Severity

| Level | Effect | Response time wanted from the user |
|---|---|---|
| **S1** | Stops all work (or threatens device safety) | Same day |
| **S2** | Stops the current phase; other phases can proceed | Within 2 working days |
| **S3** | Workaround exists; raised so the decision is explicit and the cost is visible | At the next gate |

### 10.3 When to raise

- Time-box rule: stop and write a report after **4 hours** on one obstacle, or after **2 failed, materially different approaches**, whichever comes first.
- Immediately (no time-box) for anything that could brick or boot-loop the device, risks user data, or needs a permission/credential.
- A question that blocks a task for more than 1 day becomes an S3 blocker.

### 10.4 Template (`docs/blockers/B-NNN.md`)

```
# B-NNN — <one-line summary>

- Date: YYYY-MM-DD   Severity: S1 | S2 | S3   Status: open | waiting-user | resolved | withdrawn
- Phase / task: P<n> / <task id>   Owner: agent   Needs: user | external | time
- Gate affected: G<n>   Timeline impact: <days> on <milestone>

## What was tried (with evidence)
1. <approach> — <command / log excerpt / screenshot path> — result
2. ...

## Impact
<what cannot proceed, what can; which gate moves and by how much>

## Options
| # | Option | Pros | Cons | Estimate |
|---|--------|------|------|----------|
| A | ... | ... | ... | ... |

## Recommendation
<option and why>

## Needed from the user, and by when
<decision / action / credential>, by <date> to hold the <gate> date

## Resolution
<filled when closed: what was done, date, follow-ups>
```

### 10.5 Where reports live and how they are communicated

- Files: `docs/blockers/B-001.md`, `B-002.md`, … plus `docs/blockers/INDEX.md` (table: ID, date, severity, status, one-line summary, gate affected). Reports are never deleted; superseded ones are marked `withdrawn`.
- Chat: the agent posts one short message when a report is opened: `BLOCKER B-003 (S2, P2b) — libnickel 4.38 has no exported symbol for pushing a view with the tab bar. Needs: your decision between options A/B by Fri 30 Oct. Details: docs/blockers/B-003.md`. Status changes get a one-line follow-up. No blocker is buried inside a long progress message.
- Replanning: at the next gate (or immediately for S1) the agent updates section 5 of this plan (dates, buffer consumption) with a short "replan note" referencing the blocker ID; the Gantt table at the end is regenerated; the user acknowledges the new dates in the sign-off file.

### 10.6 Example 1

```
# B-003 — libnickel 4.38 has no exported symbol for pushing a view with the tab bar

- Date: 2026-10-28   Severity: S2   Status: waiting-user
- Phase / task: P2b / 2b.1–2b.2   Owner: agent   Needs: user
- Gate affected: G2   Timeline impact: +3 days on G3 if option B; +0 if option A works within 1 day

## What was tried (with evidence)
1. MainWindowController::pushView(QWidget*) with a plain QWidget — view shows full screen,
   bottom MainNavView is hidden. Log: host.log 2026-10-28 14:02 "pushView ok; navbar visible=0".
   Screenshot docs/screens/T3/pushview-plain.png.
2. Searched Ghidra (Ghidra search_functions_by_name) for "MainNav", "TabBar", "Footer":
   MainNavView is created by MainNavController and parented to the Home/Library views'
   container, not to the window — i.e. each top-level view embeds its own nav bar
   (re/notes/view-push.md §3). No exported function builds that container for a foreign widget.
3. Tried calling the Library view's constructor family (LibraryNavMixin::showAllItemsWithoutSync)
   and replacing its content widget after construction — content replaced but the view also
   starts a library query (CPU 100 % for 3 s) and reverts on tab taps. Rejected.

## Impact
The "real tab next to Books" design cannot be done by one call. Phase 3 tasks 3.3–3.4 blocked;
3.1, 3.2, 3.5–3.10 (no UI dependency) can continue.

## Options
| # | Option | Pros | Cons | Estimate |
|---|--------|------|------|----------|
| A | Construct our own MainNavView instance (ctor is exported, NickelMenu already hooks it) and embed it at the bottom of our view; hook the active-state so our button is highlighted | Keeps the design; all symbols already in table | Two MainNavView instances alive; risk of double "tapped" routing; needs 1 more RE day | 1–2 d |
| B | Plan B container: N3Dialog enableFullViewMode with back button (NickelHardcover-proven) | Low risk, known symbols | No persistent tab bar on app screens; drawer opens like a dialog | 0.5 d |
| C | Hook the Home view's constructor and inject the drawer as a hidden page of its container | Perfect integration | Fragile across builds; NickelHome already hooks there (conflict) | 4–5 d |

## Recommendation
A, time-boxed to 1 day; fall back to B if the second MainNavView misbehaves.

## Needed from the user, and by when
Accept the 1-day spike (A) and the fallback (B), by Fri 30 Oct, to keep G2 on 30 Oct.
Optionally: say whether a drawer without a persistent tab bar (B) is acceptable for v1.

## Resolution
<pending>
```

### 10.7 Example 2

```
# B-001 — permission denied building external code on the Mac (Docker socket / NickelTC)

- Date: 2026-10-14   Severity: S1   Status: resolved 2026-10-15
- Phase / task: P2 / 2.2   Owner: agent   Needs: user
- Gate affected: G1   Timeline impact: 0 (resolved within 1 day, buffer used: 0)

## What was tried (with evidence)
1. `tools/tc make all` → "permission denied while trying to connect to the Docker daemon socket
   at unix:///var/run/docker.sock" (output saved docs/blockers/B-001-docker.txt). Docker Desktop
   was not running in the user's session; the agent's sandbox cannot start GUI apps.
2. After the user started Docker Desktop: `docker run --platform linux/amd64 ghcr.io/pgaskin/nickeltc:1`
   → "exec format error" because Rosetta for Linux containers was disabled in Docker settings.
3. Tried extracting the toolchain tarball to run natively (NickelTC README) — binaries are x86_64 ELF;
   cannot run on arm64 macOS without emulation. Rejected.

## Impact
No cross-build possible → hello-world and everything after it blocked. Phase 1 (shell/Go only) unaffected.

## Options
| # | Option | Pros | Cons | Estimate |
|---|--------|------|------|----------|
| A | User enables "Use Rosetta for x86_64/amd64 emulation" in Docker Desktop and keeps Docker running during dev hours | One click; fastest emulation | Needs the user's action each session | 0.1 d |
| B | Build in GitHub Actions (x86_64) on every push; download KoboRoot.tgz artifacts | No local emulation; CI needed anyway | Slower loop (~3 min); needs repo on GitHub now | 0.5 d |
| C | Build NickelTC image for arm64 from source | Native speed | Multi-hour build; "only tested on x86_64" per README; risk of subtle differences | 1–2 d |

## Recommendation
A now, B as the standing fallback (set up in Phase 6 anyway, pulled forward).

## Needed from the user, and by when
Enable Rosetta emulation in Docker Desktop → Settings → General, and start Docker Desktop
before test days; by Thu 15 Oct to keep G1 on 23 Oct.

## Resolution
2026-10-15: user enabled Rosetta; `make all` succeeds in 2 m 40 s; CI build job added early (tools/ci-build.yml).
```

---

## 11. Repo structure, build, CI, versioning and packaging

### 11.1 Layout (`github.com/<user>/koboapps`, private until v1.0)

```
koboapps/
├── README.md, LICENSE (MIT for our code), CHANGELOG.md
├── NickelHook/                 # git submodule (pgaskin/NickelHook)
├── Makefile                    # includes NickelHook.mk; targets all/koboroot/check/test/release
├── src/
│   ├── koboapps.cc             # NickelHook struct, init(), uninstall()
│   ├── fw/        version.cc fw_table.h(generated) decide.cc
│   ├── nickel/    symbols.h wrappers.cc (toast, pushView, wifi, rescan, dialogs, keyboard)
│   ├── host/      registry.cc appmodel.cc backend.cc protocol.cc services.cc usb.cc log.cc settings.cc
│   ├── drawer/    drawerview.cc apptile.cc navbutton.cc
│   ├── kit/       screen.cc binding.cc layout.cc refresh.cc  + one file per component
│   └── json/      minimal QJson helpers (no std::)
├── res/           icons/*.png (light/dark/active), strings_vi.json, strings_en.json, uninstall-doc
├── spec/          app.schema.json ui.schema.json protocol.md APPS.md
├── apps/          _template/ _kitdemo/ localsend/ ghsync/ libtool/   (each: app.json ui.json bin/ src/)
├── backends/      Go module: cmd/ghsync cmd/libtool cmd/ls-host cmd/validate cmd/jsonq
├── preview/       Mac Qt preview (CMake), shim for Nickel widgets, goldens/
├── tools/         tc  push.sh  logs.sh  fetch-fw.sh  gen-menu.sh  symcheck.py  new-app.sh
├── re/            firmware.lock  symbols/required.yaml  symbols/symbols-<fw>.md|json  notes/*.md
├── tests/         unit (Qt Test)  fixtures/*.jsonl  device/T0.md … T8.md
├── docs/          blockers/  signoff/  screens/  test-sessions/
└── .github/workflows/ci.yml release.yml
```

### 11.2 Build commands

```
git clone --recursive …
tools/tc make all koboroot                 # Docker NickelTC (amd64 emulation on Apple Silicon)
make check                                 # symcheck ×5, validate, shellcheck, nostd grep
cd backends && GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" ./cmd/...
cmake -S preview -B build/preview && cmake --build build/preview && build/preview/kapreview apps/localsend/ui.json
tools/push.sh KoboRoot.tgz                 # copies to /Volumes/KOBOeReader/.kobo/ when mounted
```

### 11.3 CI (GitHub Actions)

- `ci.yml` on push/PR: (1) `shellcheck`, `go vet`, `validate`; (2) cross-build in `ghcr.io/pgaskin/nickeltc:1` container, upload `KoboRoot.tgz` artifact; (3) `symcheck` against 5 libnickel symbol dumps — the dumps (`nm` text output, no binaries) are produced by a scheduled job that downloads the official zips into a cache keyed by sha256 and discards the binaries; only the text symbol lists are cached (**decision**: even the text dumps are kept in the CI cache, not committed); (4) host unit tests (ubuntu, `qtbase5-dev`); (5) preview golden tests (ubuntu, offscreen Qt platform).
- `release.yml` on tag `v*`: build, package, draft release with checksums.

### 11.4 Versioning and packaging

- Mod: semver `MAJOR.MINOR.PATCH`; `MINOR` bump when the protocol or `ui.json` schema gains fields; `MAJOR` when `proto` changes. Version string embedded (`KA_VERSION` from git describe) and written to `.adds/koboapps/version` at init (write-and-close).
- Artefacts: `koboapps-<ver>-KoboRoot.tgz` (contains `usr/local/Kobo/imageformats/libkoboapps.so`, `mnt/onboard/.adds/koboapps/{doc,default-config}`); `koboapps-apps-<ver>.zip` with `.adds/apps/{localsend,ghsync,libtool}`; individual `app-<id>-<ver>.zip`. SHA256SUMS alongside.
- Install: copy KoboRoot.tgz into `.kobo/`, unzip apps into `.adds/apps/`, eject; Kobo installs and reboots.
- `required.yaml` lists `supported_firmware: [4.38.23828, 4.38.23697, 4.38.23684, 4.38.23648, 4.38.23552]`; the release notes print the matrix.

---

## 12. Risk register

| ID | Risk | Likelihood | Impact | Mitigation | Owner |
|---|---|---|---|---|---|
| R-01 | Boot loop from a bad mod build | Low | High | NickelHook failsafe, `failsafe_delay=15`, tiny init, SAFE-01 drill per build line, uninstall flag, telnet enabled before risky sessions | agent |
| R-02 | No clean way to push a view with the tab bar (G2 no-go) | Medium | Medium | Plan B N3Dialog full-view container; kit unaffected; +3 days | agent |
| R-03 | Keyboard cannot be hosted outside N3Dialog | Medium | Low | TextField opens native text-edit dialog instead (still native) | agent |
| R-04 | NickelTC on Apple Silicon slow/broken | Medium | Medium | Rosetta emulation; CI build fallback | user/agent |
| R-05 | Firmware update changes symbols | Medium over months | Medium | Symbol check + 5-build matrix + runtime disable + menu fallback; auto-update off | agent |
| R-06 | 4.38.23697 kobo9 package is not a Libra 2 build | Medium | Low | fetch-fw verifies image type; replace with 4.38.23429 in the matrix | agent |
| R-07 | Conflict with NickelMenu/NickelHome hooks on `MainNavView` | Medium | Medium | Order-independent injection (append after existing buttons), FI-12, test with "Tweak" tab present | agent |
| R-08 | Memory growth / leaks inside Nickel over weeks | Low | High | No timers when idle, QObject parenting audit, SOAK tests, VmRSS logging | agent |
| R-09 | USB corruption from held file handles | Low | High | open-write-close rule, backend stop on USB, FI-05 | agent |
| R-10 | Library tool corrupts an EPUB | Low | High | copy-verify-rename, trash folder, `unzip -t` equivalent, user confirms candidates | agent |
| R-11 | GitHub PAT leak | Low | Medium | token file mode 600 in `data/`, never logged, fine-grained read-only, user pastes it | user |
| R-12 | Old GCC 4.9 / Qt 5.2 lacks features used | Medium | Low | No std::, C++11 subset only, clangd database for early errors, CI build | agent |
| R-13 | Emulated Qt build mismatches device (ABI) | Low | High | Only NickelTC output goes to the device; Mac preview is for layout only | agent |
| R-14 | User availability for batched sessions slips | Medium | Medium | Sessions are short, scheduled at gates, agent continues on Mac-only work (preview, docs, backends) | user |
| R-15 | Ghidra analysis time on 5 binaries | Medium | Low | headless batch overnight; analyse 23828 first, others in background | agent |
| R-16 | Vietnamese keyboard layout from kobo-tieng-viet not available in our field | Low | Medium | Use `KeyboardScript` the mod registers (**verify**); else fall back to native text-edit dialog which the mod already patches | agent |
| R-17 | Battery drain from keep-alive LocalSend receiver (Wi-Fi held on) | High (by design) | Low | Tile shows "Đang nhận"; auto-stop after N hours (configurable, default 2 h); SOAK-3 numbers | agent |
| R-18 | Backends run as root; a bug can delete books | Low | High | Add-only policies, confirm dialogs, trash folder, apps limited to those written in this repo | user/agent |

---

## 13. Open questions and decisions for the user

| # | Question | Needed by | Default assumption if no answer |
|---|---|---|---|
| Q1 | OK to install JDK 21, Ghidra, Ghidra and to register an MCP server in the maintainer on the Mac? | W2 start (19 Oct) | Yes |
| Q2 | OK to run Docker Desktop with Rosetta emulation during dev days? | W1 | Yes; CI fallback otherwise |
| Q3 | OK to install test mods (hello-world incl. the deliberate-crash variant) on the Kobo in session T2? | 23 Oct | Yes |
| Q4 | Turn off Kobo auto-update (and keep Wi-Fi mostly off) while the drawer is young? | W1 | Yes (recommended) |
| Q5 | GitHub repo `owner/repo`, branch, folder, destination folder on the Kobo; who creates the fine-grained PAT (read-only Contents on that repo only) | T1 (15 Oct) | User creates and pastes into `.adds/apps/ghsync/data/token` |
| Q6 | GH-05: re-download books deleted on the Kobo, or remember them in `state.json` and skip? | W8 | Skip (remember deletions) |
| Q7 | Enable telnet/FTP via NickelMenu items for log collection during test sessions (off otherwise)? | T2 | Yes, toggled per session |
| Q8 | Accept plan B (drawer as full-view dialog without persistent tab bar) if G2 fails? | G2 | Yes |
| Q9 | Willing to do the optional downgrade/upgrade rehearsal FW-05? | W11 | No (static checks only) |
| Q10 | Preferred time windows for the 9 batched sessions (evenings / weekends)? | W1 | Fridays 20:00–21:00 |
| Q11 | Language of app names in the drawer: Vietnamese only, or follow Nickel locale? | W6 | Follow locale, Vietnamese default |
| Q12 | LocalSend keep-alive auto-stop after how long? | W8 | 2 hours |
| Q13 | Repo visibility (private until 1.0, then public under MIT)? | W10 | Private → public at 1.0 |

---

## 14. Summary Gantt table

```
2026            Oct             |        Nov            |        Dec
Week            W1  W2  W3  W4  | W5  W6  W7  W8  W9    | W10 W11
Dates (Mon)     12  19  26  2   | 9   16  23  30  7     | 14  21
P1 menu fallback ██
P2 toolchain+RE  ░█  ██
P2b RE spike             ██
P3 drawer host               ██  ██
P4 UI kit v1                         ██  ██
P5 port tools                                ██  ██
P6 dev kit                                           ██
Buffer/release                                           ░█
Gates            M1  G1  G2      G3      G4      G5      G6
User sessions   T0T1 T2  T3      T4      T5      T6  T7  T8
Soaks                            ▒▒▒(72h)        ▒▒▒(72h) ▒(24h)
```

| Milestone | Date | Deliverable |
|---|---|---|
| M1 | Fri 16 Oct | Apps via NickelMenu (LocalSend, GitHub sync) |
| G1 | Fri 23 Oct | Toolchain, 5 firmware builds analysed, hello-world failsafe proven |
| G2 | Fri 30 Oct | Native view/keyboard feasibility decided |
| G3 | Mon 16 Nov | Drawer host accepted, idle soak passed |
| G4 | Mon 30 Nov | UI kit v1 accepted |
| G5 | Mon 14 Dec | Three apps accepted, failure injection passed |
| G6 | Wed 23 Dec | v1.0 release, dev kit, CI green on 5 firmware builds |

---

## 15. Appendix: reference facts collected during planning (8 Oct 2026)

- NickelHook API (`NickelHook.h`): `nh_info{name, desc, uninstall_flag, uninstall_xflag, failsafe_delay}`, `nh_hook{sym, sym_new, lib, out, desc, optional}`, `nh_dlsym{name, out, desc, optional}`, `nh_log`, `nh_dump_log` (writes `/mnt/onboard/.kobo/<name>_YYYY-MM-DD_HH-MM-SS.log`), `nh_delete_file/dir`. README: avoid the C++ standard library; never hold `/mnt/onboard` file handles longer than a few hundred ms; the init blocks Nickel startup so keep it fast; build with NickelTC; names < 3 letters reserved.
- NickelTC: Docker `ghcr.io/pgaskin/nickeltc:1` (README) / `:1.0` (nickel-screensaver README); GCC 4.9.4; Qt 5.2.1 at `/usr/local/Qt-5.2.1-arm`; flags `-march=armv7-a -mtune=cortex-a8 -mfpu=neon -mfloat-abi=hard -mthumb`; "only tested on x86_64"; last updated Apr 2021.
- NickelMenu: `menu_main_15505_*` experimental options control the extra bottom-bar button (label/icon); hooks `MainNavView::MainNavView`, creates `MainNavButton`, bridges `tapped()` through a `QPushButton` shim, shows `NickelTouchMenu` with `MenuTextItem`s; all these symbols are `optional` and the feature degrades with a log message. Actions available for the fallback: `cmd_spawn`, `cmd_output` (≤ 10 s), `nickel_wifi:autoconnect|autoconnect_silent|enable|disable|toggle`, `nickel_misc:rescan_books|rescan_books_full|home|force_usb_connection`, `nickel_setting:toggle:dark_mode|screenshots|invert|lockscreen`, `nickel_screenshot:capture`, `dbg_toast`, `power:sleep|reboot|shutdown`. Uninstall by `.adds/nm/uninstall` or power-off within 20 s of boot. "Firmware 5.x is not supported yet."
- NickelDBus symbols: `PlugManager::sharedInstance/gadgetMode`, `PlugWorkflowManager::sharedInstance`, `WirelessManager::sharedInstance`, `MainWindowController::sharedInstance/toast/currentView`, `N3Dialog::content`, `Device::getCurrentDevice`, `N3FSSyncManager::sharedInstance/sync(QStringList*)`, `WirelessWatchdog::sharedInstance`, `ConfirmationDialogFactory::getConfirmationDialog/showTextEditDialog`, `ConfirmationDialog::setTitle/setText/setAcceptButtonText/setRejectButtonText/showCloseButton/setRejectOnOutsideTap/addWidget`, `N3ConfirmationTextEditField` ctor(s) with `KeyboardScript`, `TouchLineEdit`; signal `pfmDoneProcessing`.
- NickelHardcover (hook/src): `MainWindowController::pushView(QWidget*)`, `viewWithObjectName`, `N3DialogFactory::getDialog(QWidget*, bool)`, `N3Dialog::showKeyboard/hideKeyboard/keyboardFrame/enableBackButton/disableCloseButton/enableFullViewMode/setTitle`, `KeyboardFrame::createKeyboard(KeyboardScript, QLocale const&)`, `KeyboardReceiver(QLineEdit*|QTextEdit*, bool)`, `SearchKeyboardController::setReceiver/setGoText`, `TouchLineEdit`, `TouchTextEdit::setCustomPlaceholderText`, `TouchCheckBox`, `TouchLabel::setHitStateEnabled`, `N3ButtonLabel`, `SettingContainer`, `PowerTimer::fireAt/timeRemaining`, `WirelessWorkflowManager::connectWireless/connectWirelessSilently/isInternetAccessible`, `SupportedLocales::supportedLocales`, `ConfirmationDialogFactory::showErrorDialog`; widgets dir has `dialog, label, elidedlabel, pagedstack, buttongroup, rating`; settings dir has `checkboxrow, menurow, staticrow, settingsdialog`. "Firmware 5.x not supported." Primary repo on Codeberg.
- Ghidra: Ghidra extension zip installed via `File → Install Extensions`; plugin `GhidraPlugin` under `File → Configure → Developer`; HTTP server default `127.0.0.1:8080`; Python bridge `bridge_mcp_ghidra.py --ghidra-server http://127.0.0.1:8080/` (needs the `mcp` Python SDK); tools include list classes/methods/imports/exports, decompile, rename.
- Kobo firmware: upgrade API `https://api.kobobooks.com/1.0/UpgradeCheck/Device/<deviceid>/<affiliate>/<ver>/<serial>`; Libra 2 device id `…0388`, hardware folder `kobo9`; zip contains `upgrade/`, `KoboRoot.tgz`, `manifest.md5sum`; manual install = copy contents into `.kobo/` and eject. Five newest Libra 2 builds and CDN facts in section 3.1. kobopatch-patches `src/versions` ends at 4.38.21908 and 4.39.22801 (no patch sets for the 4.38.23xxx Libra 2 builds there; MobileRead patch threads exist for 4.38.23648 and 4.38.23828).
- kobo-tieng-viet: 4.x only ("không hỗ trợ 3.x và 5.x"); since build 20260319 it self-repairs Vietnamese after a firmware update on the next reboot.
- Device: Kobo Libra 2, N418, Mark 9 "Io", NXP i.MX6SLL, 512 MB RAM (**verify**), 7" 1264×1680 300 ppi, firmware 4.38.23828.

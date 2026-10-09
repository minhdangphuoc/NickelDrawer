# Decisions

| Date | Decision | Notes |
|---|---|---|
| 2026-10-08 | Stock Nickel UI, KOReader removed | User prefers the native Kobo look. |
| 2026-10-08 | Native drawer via NickelHook mod + UI kit (`ui.json`) | Menu (NickelMenu) stays as permanent fallback. |
| 2026-10-08 | Support the 5 newest Libra 2 firmware builds (4.38.23552 → 4.38.23828) | 4.x only; 5.x unsupported by NickelMenu / kobo-tieng-viet. See plan §3. |
| 2026-10-08 | Model routing: developer codes, reviewer debugs and reverse-engineers | See CONTRIBUTING.md. |
| 2026-10-08 | Any UI language: English source + `locales/<lang>.json` | **English and Vietnamese ship first** and are maintained by us; other languages come from community translations later. RTL layout deferred to UI kit phase. |
| 2026-10-08 | Public open-source project, name **NickelDrawer** | No GitHub repo with that name existed on 8 Oct 2026. |
| 2026-10-08 | License **MIT** | Matches NickelMenu, NickelHook, localgo. |
| 2026-10-08 | Public beta after gate G3 (mid-Nov 2026); v1.0 at G6 | Publish once drawer + failsafe are proven. |
| 2026-10-08 | OTA updates from GitHub Releases, signed with ed25519 | Private key stays on the maintainer's Mac; never in repo/CI. |
| 2026-10-08 | Repo owner: the maintainer's GitHub account (confirm before creating) | |
| 2026-10-08 | **Decline Kobo's optional "Qt6 upgrade"** (= firmware 5.x) on the Libra 2 | The server-offered upgrade flow (`ApplicationSettings::availableQt6Upgrade`, `HomePageController::maybeShowQt6UpgradeDialog`, `skippedQt6UpgradeUrl`) exists only in 4.38.23828; 4.38.23552–23697 have an older optional dialog (17 related symbols). It asks the user first: Skip/Cancel records the offered URL as declined (it returns only for a new URL); Proceed makes the next sync install it. See re/notes/G2-findings.md Q5. Accepting it would break NickelMenu, NickelDrawer and kobo-tieng-viet. Future: the drawer warns when `IOUtil::isQt6UpgradePending` is true. |
| 2026-10-08 | Symbol dumps (`re/symbols/*/exports.txt`, ~25 MB) not committed | Regenerate with `re/scripts/fetch-fw.sh` + `re/scripts/symreport.py`. |
| 2026-10-08 | Drawer: native list (sections "Installed" / "System", icon + name + status + chevron) is the default view; optional grid view behind a toggle in the header | Approved on the design canvas. Lives as one "Apps" row in Nickel's More tab, not a separate bottom tab. |
| 2026-10-08 | Hard keys: page-turn buttons flip pages (drawer list/grid and long app screens), ignored in dialogs/menus, follow Nickel's direction setting | Key-event delivery to a pushed view is a G2 item for Ghidra. |
| 2026-10-08 | Gate G2 (static): conditional GO for the native drawer | More-tab row via MoreView hook (NickelMenu recipe); drawer as AbstractController subclass with navSection 6 (keeps tab bar, More active); page keys arrive as arrow keys via TactileButtonFilter; plan B = full-screen N3Dialog. Needs on-device prototypes before G2 is signed. See re/notes/G2-findings.md. |
| 2026-10-09 | GitHub sync on device postponed by the user | Code, pairing and tests stay; device acceptance (GH-*) moves to a later session. Repo `main` branch currently has no books. |
| 2026-10-10 | Keep current app screens; no Settings-style options window or Kobo Settings row for now | Improve back instead: working arrow on app screens, one-level back stack restoring the previous view (page + list/grid). |

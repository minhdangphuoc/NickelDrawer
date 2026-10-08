#!/usr/bin/env python3
"""Reports over libnickel export lists produced by re/scripts/fetch-fw.sh.

  symreport.py diff   -> re/symbols/DIFF.md
  symreport.py matrix -> re/symbols/known-mods-matrix.md

Reads mangled export lists from the cache ($NICKELDRAWER_CACHE/symbols/<fw>/),
never from Kobo binaries directly. Run with `python3 -I`.
"""

import os
import re
import subprocess
import sys
from collections import Counter
from pathlib import Path

RE_DIR = Path(__file__).resolve().parent.parent
CACHE = Path(os.environ.get("NICKELDRAWER_CACHE", Path.home() / ".cache/nickeldrawer"))
FIRMWARES = ["4.38.23552", "4.38.23648", "4.38.23684", "4.38.23697", "4.38.23828"]

MOD_SOURCES = {
    "NickelMenu": ["src/nickelmenu.cc", "src/action_cc.cc"],
    "NickelDBus": ["src/ndb/NDBDbus.cc", "src/ndb/NDBCfmDlg.cc"],
    "NickelHardcover": ["hook/src/nickelhardcover.cc"],
}

# Capability ids follow docs/plan-detailed.md section 6.4. First match wins.
CAPABILITIES = [
    ("R7", r"^MainWindowController::toast", "toast"),
    ("R1", r"^(MainWindowController|MainNavController|MainNavView|AbstractController|StatusBarController)::",
     "view stack / main navigation"),
    ("R2", r"^MainNavButton::", "bottom tab bar button"),
    ("R4", r"^(SettingContainer|TouchCheckBox|TouchLabel|N3ButtonLabel|ToggleSwitch|SettingItemWithToggleSwitch|"
           r"N3Settings\w*|SettingsMixin|PagingFooter)::", "settings-style widgets / settings pages"),
    ("R6", r"^(ConfirmationDialog|ConfirmationDialogFactory)::", "confirmation dialog"),
    ("R5", r"^(N3Dialog|N3DialogFactory)::", "N3 dialog (full-view container, keyboard host)"),
    ("R8", r"^(KeyboardFrame|KeyboardReceiver|SearchKeyboardController|TouchLineEdit|TouchTextEdit|"
           r"N3ConfirmationTextEditField)::", "keyboard / text input"),
    ("R9", r"^(WirelessWorkflowManager|WirelessManager|WirelessWatchdog)::", "Wi-Fi"),
    ("R10", r"^(PlugWorkflowManager|N3FSSyncManager)::", "library rescan / USB workflow"),
    ("R11", r"^(Settings|ReadingSettings|FeatureSettings|ApplicationSettings|PowerSettings|DevSettings)::|"
            r"vtable for (Settings|ReadingSettings|FeatureSettings|ApplicationSettings|PowerSettings|DevSettings)$",
     "settings store / dark mode"),
    ("R13", r"^(PlugManager|N3PowerWorkflowManager|PowerTimer)::", "USB / power state"),
    ("R14", r"^SupportedLocales::", "locale"),
    ("menu", r"^(NickelTouchMenu|MenuTextItem|LightMenuSeparator|BoldMenuSeparator|AbstractNickelMenuController|"
             r"AbstractMenuController|SelectionMenu\w*|ReadingMenuView|WebSearchMixinBase)::", "menus (NickelMenu-style)"),
    ("nav", r"^(\w+NavMixin|MoreController)::", "jump to a Nickel screen"),
]
DRAWER_CAPABILITIES = {"R1", "R2", "R4", "R5", "R6", "R7", "R8", "R9", "R10", "R11"}

# Symbols no known mod uses yet but the drawer probably needs (plan 6.4 + exports survey).
# Demangled forms; the purpose column is a guess until checked in Ghidra.
DRAWER_CANDIDATES = {
    "MainWindowController::popView(QWidget*)": "pop our pushed view (back)",
    "MainWindowController::push(AbstractController*, bool)": "controller-level push (guess: what Nickel uses for its own screens)",
    "MainWindowController::pop(AbstractController*)": "controller-level pop",
    "MainWindowController::updateNavFooter(AbstractController*)": "guess: tab bar visibility per controller",
    "MainWindowController::createNavFooter()": "guess: builds the bottom tab bar (window-owned)",
    "MainWindowController::showStatusBarAsNecessary(QWidget*)": "status bar visibility per view",
    "MainWindowController::currentViewName() const": "detect when our view is current",
    "MainWindowController::previousViewTitle() const": "guess: back-header title source",
    "MainWindowController::closeActiveN3Dialogs()": "cleanup before push",
    "MainNavController::home()": "go Home",
    "MainNavController::setActiveSection(MainNavSection)": "tab active state",
    "MainNavView::updateVisibleButtons()": "guess: re-layout tab bar after adding a button",
    "MainNavView::setHomeActive(bool)": "clear Home active state when drawer is current",
    "MainNavButton::setActive(bool)": "active state of our tab",
    "AbstractController::AbstractController()": "base for a controller-based drawer view (vtable needed)",
    "N3SettingsControllerBase::N3SettingsControllerBase(QObject*)": "settings page controller base",
    "N3SettingsPagedView::N3SettingsPagedView(QWidget*)": "paged settings list look",
    "PagingFooter::PagingFooter(QWidget*)": "page footer for long lists",
    "SettingsMixin::wirelessConnection()": "open Wi-Fi settings (23828 name)",
    "N3SettingsController::wirelessConnection()": "open Wi-Fi settings (pre-23828 name)",
    "WirelessWorkflowManager::showNetworksDialog(bool)": "network picker (pre-23828 signature)",
    "WirelessWorkflowManager::showNetworksDialog(bool, bool)": "network picker (23828 signature)",
    "N3PowerWorkflowManager::sharedInstance()": "power workflow",
    "N3PowerWorkflowManager::requestSleep()": "sleep",
    "PlugWorkflowManager::sync()": "full library rescan",
}


def read_lines(path):
    with open(path, encoding="utf-8") as handle:
        return [line.rstrip("\n") for line in handle]


def demangle(names):
    cxxfilt = subprocess.run(["xcrun", "--find", "llvm-cxxfilt"], check=True, capture_output=True, text=True)
    result = subprocess.run([cxxfilt.stdout.strip()], input="\n".join(names) + "\n",
                            check=True, capture_output=True, text=True)
    demangled = result.stdout.splitlines()
    if len(demangled) != len(names):
        sys.exit("llvm-cxxfilt returned a different number of lines; cannot pair names")
    return dict(zip(names, demangled))


def load_exports():
    exports = {}
    for firmware in FIRMWARES:
        path = CACHE / "symbols" / firmware / "exports-mangled.txt"
        if not path.exists():
            sys.exit(f"missing {path}: run re/scripts/fetch-fw.sh {firmware} <month> first")
        exports[firmware] = set(read_lines(path))
    return exports


def owner_of(demangled):
    """Class (or 'vtable for X' etc.) a symbol belongs to, for grouping."""
    head = demangled.split("(", 1)[0]
    return head.rsplit("::", 1)[0] if "::" in head else head


def capability_of(demangled):
    for cap_id, pattern, purpose in CAPABILITIES:
        if re.search(pattern, demangled):
            return cap_id, purpose
    return "-", "other"


def write_diff(exports):
    lines = ["# libnickel export diff between consecutive Libra 2 builds", "",
             "Generated by `python3 -I re/scripts/symreport.py diff` from `nm -D --defined-only` dumps.", "",
             "| from | to | exports (to) | added | removed |", "|---|---|---|---|---|"]
    details = []
    for older, newer in zip(FIRMWARES, FIRMWARES[1:]):
        added = sorted(exports[newer] - exports[older])
        removed = sorted(exports[older] - exports[newer])
        lines.append(f"| {older} | {newer} | {len(exports[newer])} | {len(added)} | {len(removed)} |")
        names = demangle(added + removed) if added or removed else {}
        details += [f"", f"## {older} -> {newer}", ""]
        for label, symbols in (("Added", added), ("Removed", removed)):
            if not symbols:
                details.append(f"{label}: none.")
                continue
            per_class = Counter(owner_of(names[s]) for s in symbols)
            summary = ", ".join(f"{cls} ({count})" for cls, count in per_class.most_common())
            details += [f"**{label} ({len(symbols)})** by class: {summary}", "", "<details><summary>full list</summary>", "",
                        "```"] + sorted(names[s] for s in symbols) + ["```", "", "</details>", ""]
    path = RE_DIR / "symbols" / "DIFF.md"
    path.write_text("\n".join(lines + details) + "\n", encoding="utf-8")
    print(f"wrote {path}")


def collect_mod_symbols(src_root):
    usages = {}
    for mod, files in MOD_SOURCES.items():
        for relative in files:
            path = src_root / mod / relative
            for number, line in enumerate(read_lines(path), start=1):
                for symbol in re.findall(r'"(_Z[A-Za-z0-9_]+)"', line):
                    usages.setdefault(symbol, set()).add(f"{mod} ({Path(relative).name}:{number})")
    return usages


def presence_cells(symbol, exports):
    return ["yes" if symbol in exports[fw] else "**NO**" for fw in FIRMWARES]


def write_matrix(exports, src_root):
    usages = collect_mod_symbols(src_root)
    names = demangle(sorted(usages))
    rows = []
    for symbol, places in usages.items():
        cap_id, purpose = capability_of(names[symbol])
        rows.append((cap_id not in DRAWER_CAPABILITIES, cap_id, names[symbol], symbol, places, purpose))
    rows.sort()

    newest_pairs = demangle(sorted(exports[FIRMWARES[-1]] | exports[FIRMWARES[0]]))
    mangled_by_demangled = {d: m for m, d in newest_pairs.items()}

    header = ("| drawer | cap | symbol (demangled) | mangled | used by | purpose (guess) | "
              + " | ".join(fw.split(".")[-1] for fw in FIRMWARES) + " |")
    divider = "|" + "---|" * (6 + len(FIRMWARES))
    lines = ["# Known-mod libnickel symbols vs the 5 target Libra 2 builds", "",
             "Generated by `python3 -I re/scripts/symreport.py matrix`. Sources: NickelMenu, NickelDBus, NickelHardcover",
             "(commits in re/README.md). `NO` = not exported by that build's libnickel.so.1.0.0 (some are Qt symbols,",
             "resolved from libQt5*.so at runtime, and are expected to be absent here). Purpose column = capability",
             "bucket from plan section 6.4; it is a guess from the name, not verified in Ghidra.", "",
             "## Symbols used by existing mods", "", header, divider]
    for not_drawer, cap_id, demangled, symbol, places, purpose in rows:
        mark = "" if not_drawer else "**D**"
        lines.append(f"| {mark} | {cap_id} | `{demangled}` | `{symbol}` | {'<br>'.join(sorted(places))} | {purpose} | "
                     + " | ".join(presence_cells(symbol, exports)) + " |")

    lines += ["", "## Drawer candidates not used by any of the three mods", "",
              "| symbol (demangled) | mangled | purpose (guess) | " + " | ".join(fw.split(".")[-1] for fw in FIRMWARES) + " |",
              "|" + "---|" * (3 + len(FIRMWARES))]
    for demangled, purpose in DRAWER_CANDIDATES.items():
        symbol = mangled_by_demangled.get(demangled)
        cells = presence_cells(symbol, exports) if symbol else ["**NO**"] * len(FIRMWARES)
        lines.append(f"| `{demangled}` | `{symbol or 'not found'}` | {purpose} | " + " | ".join(cells) + " |")

    path = RE_DIR / "symbols" / "known-mods-matrix.md"
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"wrote {path} ({len(rows)} mod symbols, {len(DRAWER_CANDIDATES)} candidates)")


def main():
    command = sys.argv[1] if len(sys.argv) > 1 else ""
    exports = load_exports()
    if command == "diff":
        write_diff(exports)
    elif command == "matrix":
        write_matrix(exports, CACHE / "src")
    else:
        sys.exit("usage: symreport.py diff|matrix")


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Checks that every symbol in a symbols list exists in each firmware's libnickel export dump.

Usage: tools/symcheck.py [--symbols FILE] [--fw-dir DIR] [--builds V,V,...] [--quiet]
Exit 0 when every required-for-drawer symbol is present in every build, 1 when one is missing,
2 on usage or input errors. Optional and fallback-only symbols are reported, never fatal.

Dumps are mangled names, one per line: <fw-dir>/<version>/exports-mangled.txt (or exports.txt).
Default fw-dir is $NICKELDRAWER_CACHE/symbols (~/.cache/nickeldrawer/symbols). Missing dumps are
regenerated with re/scripts/fetch-fw.sh using re/firmware.lock, which downloads into the cache only.
"""
import argparse
import os
import re
import subprocess
import sys

REPO_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_SYMBOLS = os.path.join(REPO_ROOT, "re", "notes", "symbols-required-draft.yaml")
FIRMWARE_LOCK = os.path.join(REPO_ROOT, "re", "firmware.lock")
FETCH_SCRIPT = os.path.join(REPO_ROOT, "re", "scripts", "fetch-fw.sh")
DUMP_NAMES = ("exports-mangled.txt", "exports.txt")
REQUIRED = "required-for-drawer"


class InputError(Exception):
    pass


def parse_symbols_file(path):
    """Returns (builds, symbols); symbols are dicts with mangled and req. Reads the simple YAML subset we write."""
    builds, symbols = [], []
    try:
        lines = open(path, encoding="utf-8").read().splitlines()
    except OSError as error:
        raise InputError(f"cannot read symbols file {path}: {error}")
    for line in lines:
        builds_match = re.match(r"^builds:\s*\[(.*)\]\s*$", line)
        mangled_match = re.match(r"^\s*-\s+mangled:\s*(\S+)\s*$", line)
        req_match = re.match(r"^\s+req:\s*(\S+)\s*$", line)
        if builds_match:
            builds = [item.strip() for item in builds_match.group(1).split(",") if item.strip()]
        elif mangled_match:
            symbols.append({"mangled": mangled_match.group(1), "req": None})
        elif req_match and symbols and symbols[-1]["req"] is None:
            symbols[-1]["req"] = req_match.group(1)
    unclassified = [s["mangled"] for s in symbols if s["req"] is None]
    if unclassified:
        raise InputError(f"{path}: symbols without a req field: {', '.join(unclassified)}")
    if not symbols:
        raise InputError(f"{path}: no 'mangled:' entries found")
    return builds, symbols


def find_dump(fw_dir, version):
    for name in DUMP_NAMES:
        candidate = os.path.join(fw_dir, version, name)
        if os.path.isfile(candidate):
            return candidate
    return None


def lock_month(version):
    with open(FIRMWARE_LOCK, encoding="utf-8") as lock:
        for line in lock:
            fields = line.rstrip("\n").split("\t")
            if len(fields) > 1 and fields[0] == version:
                return fields[1]
    return None


def regenerate_dump(version):
    month = lock_month(version)
    if month is None:
        raise InputError(f"{version} is not in re/firmware.lock; add it or pass --fw-dir with its dump")
    print(f"regenerating dump for {version} ({month}) via re/scripts/fetch-fw.sh", file=sys.stderr)
    result = subprocess.run(["sh", FETCH_SCRIPT, version, month], stdout=subprocess.DEVNULL)
    if result.returncode != 0:
        raise InputError(f"fetch-fw.sh failed for {version} (exit {result.returncode}); see its output above")


def load_exports(fw_dir, version, canRegenerate):
    dump = find_dump(fw_dir, version)
    if dump is None and canRegenerate:
        regenerate_dump(version)
        dump = find_dump(fw_dir, version)
    if dump is None:
        raise InputError(f"no export dump for {version} under {fw_dir}/{version}/ (expected {' or '.join(DUMP_NAMES)})")
    with open(dump, encoding="utf-8", errors="replace") as handle:
        return {line.strip() for line in handle}


def print_matrix(builds, symbols, missing):
    width = max(len(s["mangled"]) for s in symbols)
    print(f"{'symbol':<{width}}  {'req':<20} " + " ".join(b.split('.')[-1].rjust(6) for b in builds))
    for symbol in symbols:
        cells = " ".join(("ok" if symbol["mangled"] not in missing[b] else "MISSING").rjust(6) for b in builds)
        print(f"{symbol['mangled']:<{width}}  {symbol['req']:<20} {cells}")


def summarize(builds, symbols, missing):
    fatal = [(s["mangled"], b) for s in symbols if s["req"] == REQUIRED for b in builds if s["mangled"] in missing[b]]
    optionalMissing = sum(1 for s in symbols if s["req"] != REQUIRED for b in builds if s["mangled"] in missing[b])
    requiredCount = sum(1 for s in symbols if s["req"] == REQUIRED)
    print(f"\n{len(symbols)} symbols ({requiredCount} required) x {len(builds)} builds; "
          f"{len(fatal)} required missing, {optionalMissing} optional/fallback missing")
    for mangled, build in fatal:
        print(f"FAIL: required symbol {mangled} missing in {build}")
    return fatal


def default_fw_dir():
    cache = os.environ.get("NICKELDRAWER_CACHE", os.path.expanduser("~/.cache/nickeldrawer"))
    return os.path.join(cache, "symbols")


def parse_arguments(argv):
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--symbols", default=DEFAULT_SYMBOLS)
    parser.add_argument("--fw-dir", default=default_fw_dir(),
                        help="directory holding <version>/exports-mangled.txt dumps")
    parser.add_argument("--builds", help="comma-separated versions (default: builds: of the symbols file)")
    parser.add_argument("--no-regenerate", action="store_true", help="never run fetch-fw.sh for a missing dump")
    parser.add_argument("--quiet", action="store_true", help="skip the matrix, print only the summary")
    return parser.parse_args(argv)


def main(argv):
    args = parse_arguments(argv)
    try:
        listedBuilds, symbols = parse_symbols_file(args.symbols)
        builds = args.builds.split(",") if args.builds else listedBuilds
        if not builds:
            raise InputError("no builds: pass --builds or add a builds: [..] line to the symbols file")
        canRegenerate = args.fw_dir == default_fw_dir() and not args.no_regenerate
        missing = {b: {s["mangled"] for s in symbols} - load_exports(args.fw_dir, b, canRegenerate) for b in builds}
    except InputError as error:
        print(f"symcheck: {error}", file=sys.stderr)
        return 2
    if not args.quiet:
        print_matrix(builds, symbols, missing)
    return 1 if summarize(builds, symbols, missing) else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))

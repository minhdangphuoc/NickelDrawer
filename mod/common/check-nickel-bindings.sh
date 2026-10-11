#!/bin/sh
# Checks a mod's libnickel symbols against the real libnickel of every cached firmware build, the
# way NickelHook resolves them at runtime (third_party/NickelHook/nh.c):
#   hooked (.sym = "...")    nh_dlhook patches libnickel's GOT. It needs the symbol in libnickel's
#                            .dynsym, defined (not UND), type FUNC, binding GLOBAL, and an
#                            R_ARM_JUMP_SLOT relocation for it. It refuses WEAK symbols: inline code
#                            such as uic's Ui_<Form>::setupUi is WEAK, is listed in exports.txt and
#                            has a jump slot, yet nh_dlhook returns NULL for it on the device.
#   dlsym'd (.name = "...")  dlsym(RTLD_DEFAULT) needs a defined, non-LOCAL .dynsym entry.
# Usage: check-nickel-bindings.sh <bridge.cc>
# Needs the firmware cache (re/scripts/fetch-fw.sh) and docker; skips with a notice without them.
# Prints every failure; exits non-zero when any check failed.
set -eu

bridge=${1:?usage: check-nickel-bindings.sh <bridge.cc>}
firmware_cache="${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}/fw"
libnickel_path=root/usr/local/Kobo/libnickel.so.1.0.0
nickeltc_image="ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840"

failures=0
fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}

hooked_symbols() {
	grep -hoE '\.sym = "_Z[^"]+"' "$bridge" | sed -E 's/.*"(.*)"/\1/' | sort -u
}

dlsymed_symbols() {
	grep -hoE '(\.name = |NICKEL_SYMBOL\()"_Z[^"]+"' "$bridge" | sed -E 's/.*"(.*)"/\1/' | sort -u
}

if [ ! -d "$firmware_cache" ] || ! command -v docker >/dev/null 2>&1; then
	echo "bindings: skipped (no firmware cache at $firmware_cache or no docker; run re/scripts/fetch-fw.sh)"
	exit 0
fi

hooks=$(hooked_symbols)
dlsyms=$(dlsymed_symbols)
all_symbols=$(printf '%s\n%s\n' "$hooks" "$dlsyms" | grep -v '^$' | sort -u)
[ -n "$all_symbols" ] || {
	echo "FAIL: no libnickel symbols found in $bridge" >&2
	exit 1
}
pattern=$(echo "$all_symbols" | paste -sd'|' -)

# One container for all builds: per build, the matching .dynsym rows and jump-slot relocations,
# each line prefixed with "<firmware> sym|rel".
readelf_rows=$(docker run --rm --platform linux/amd64 -v "$firmware_cache:/fw:ro" -e PATTERN="$pattern" \
	--entrypoint sh "$nickeltc_image" -c '
	for dir in /fw/*/; do
		lib="${dir}'"$libnickel_path"'"
		[ -f "$lib" ] || continue
		firmware=$(basename "$dir")
		readelf -W --dyn-syms "$lib" | grep -E " ($PATTERN)\$" | sed "s|^|$firmware sym |"
		readelf -rW "$lib" | grep JUMP_SLOT | grep -E " ($PATTERN)\$" | sed "s|^|$firmware rel |"
		echo "$firmware seen"
	done')

firmwares=$(echo "$readelf_rows" | awk '$2 == "seen" { print $1 }')
[ -n "$firmwares" ] || {
	echo "FAIL: no libnickel found under $firmware_cache/*/$libnickel_path" >&2
	exit 1
}

# .dynsym columns after the prefix: Num: Value Size Type Bind Vis Ndx Name.
dynsym_row() {
	echo "$readelf_rows" | awk -v fw="$1" -v name="$2" '$1 == fw && $2 == "sym" && $10 == name { print $6, $7, $9; exit }'
}

check_hook() {
	firmware=$1 symbol=$2
	row=$(dynsym_row "$firmware" "$symbol")
	if [ -z "$row" ]; then
		fail "$firmware: hooked $symbol is not in libnickel's .dynsym"
		return
	fi
	set -- $row # type bind ndx
	[ "$3" != UND ] || fail "$firmware: hooked $symbol is undefined in libnickel (imported, defined elsewhere)"
	[ "$1" = FUNC ] || fail "$firmware: hooked $symbol is $1, nh_dlhook needs FUNC"
	[ "$2" = GLOBAL ] || fail "$firmware: hooked $symbol is $2, nh_dlhook only patches GLOBAL symbols"
	echo "$readelf_rows" | awk -v fw="$firmware" -v name="$symbol" '$1 == fw && $2 == "rel" && $NF == name { found = 1 } END { exit !found }' ||
		fail "$firmware: no R_ARM_JUMP_SLOT for hooked $symbol (nh_dlhook has no GOT entry to patch)"
}

check_dlsym() {
	firmware=$1 symbol=$2
	row=$(dynsym_row "$firmware" "$symbol")
	if [ -z "$row" ]; then
		fail "$firmware: $symbol is not in libnickel's .dynsym"
		return
	fi
	set -- $row
	[ "$3" != UND ] || fail "$firmware: $symbol is undefined in libnickel (imported, defined elsewhere)"
	[ "$2" != LOCAL ] || fail "$firmware: $symbol is LOCAL, dlsym cannot see it"
}

for firmware in $firmwares; do
	for symbol in $hooks; do
		check_hook "$firmware" "$symbol"
	done
	for symbol in $dlsyms; do
		check_dlsym "$firmware" "$symbol"
	done
done

build_count=$(echo "$firmwares" | wc -l | tr -d ' ')
if [ "$failures" -ne 0 ]; then
	echo "bindings: $failures failure(s) over $build_count build(s)" >&2
	exit 1
fi
echo "bindings: $(echo "$hooks" | grep -c . || true) hook(s) FUNC GLOBAL with jump slots, $(echo "$dlsyms" | grep -c . || true) dlsym(s) defined, on $build_count build(s)"

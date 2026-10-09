#!/bin/sh
# Static checks for build/mod/libnddrawer.so (gate-G2 prototypes). Run from anywhere:
#   mod/drawer-proto/check-symbols.sh
# 1. Every libnickel symbol the mod hooks or dlsyms is exported by all firmware
#    builds in re/symbols/<fw>/exports.txt (demangled), and by the mangled dumps
#    in ~/.cache/nickeldrawer/symbols/<fw>/ when those exist.
# 2. The library is ARM, hard-float, needs only the expected libraries, and
#    imports nothing from the C++ standard library beyond the ABI runtime.
# 3. Every vtable has typeinfo (mod/common/check-vtable-typeinfo.sh).
# Exits non-zero on the first category that fails; prints every failure.
set -eu

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)
library="$repo_root/build/mod/libnddrawer.so"
symbols_dir="$repo_root/re/symbols"
mangled_cache="${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}/symbols"
nickeltc_image="ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840"
allowed_needed="libdl.so.2 libQt5Widgets.so.5 libQt5Gui.so.5 libQt5Core.so.5 libstdc++.so.6 libgcc_s.so.1 libpthread.so.0 libc.so.6 ld-linux-armhf.so.3"

failures=0
fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}

# Mangled names from the nh_dlsym (.name) and nh_hook (.sym) tables.
list_nickel_symbols() {
	grep -hoE '\.(name|sym) = "_Z[^"]+"' "$script_dir"/*.cc | sed -E 's/.*"(_Z[^"]+)"/\1/' | sort -u
}

check_firmware_exports() {
	symbols=$(list_nickel_symbols)
	if [ -z "$symbols" ]; then
		fail "no libnickel symbols found in $script_dir/*.cc"
		return
	fi
	firmware_count=0
	for exports in "$symbols_dir"/*/exports.txt; do
		firmware=$(basename "$(dirname "$exports")")
		firmware_count=$((firmware_count + 1))
		mangled_dump="$mangled_cache/$firmware/exports-mangled.txt"
		for symbol in $symbols; do
			demangled=$(echo "$symbol" | c++filt)
			grep -Fxq -- "$demangled" "$exports" || fail "$firmware: $demangled ($symbol) not exported"
			if [ -f "$mangled_dump" ]; then
				grep -Fxq -- "$symbol" "$mangled_dump" || fail "$firmware: $symbol missing from $mangled_dump"
			fi
		done
	done
	[ "$firmware_count" -eq 5 ] || fail "expected 5 firmware dumps in $symbols_dir, found $firmware_count (run re/scripts/fetch-fw.sh)"
	echo "symbols: $(echo "$symbols" | wc -l | tr -d ' ') libnickel symbols checked against $firmware_count builds"
}

nickeltc() {
	docker run --rm --platform linux/amd64 -v "$repo_root/build/mod:/o:ro" --entrypoint sh "$nickeltc_image" -c "$1"
}

check_binary() {
	if [ ! -f "$library" ]; then
		fail "$library missing; run: make -C mod/drawer-proto docker-build"
		return
	fi
	if ! command -v docker >/dev/null 2>&1; then
		fail "docker not found; binary checks need the NickelTC image"
		return
	fi
	tools=arm-nickel-linux-gnueabihf-
	attributes=$(nickeltc "${tools}readelf -h -A /o/libnddrawer.so")
	echo "$attributes" | grep -q 'Machine: *ARM' || fail "not an ARM ELF"
	echo "$attributes" | grep -q 'Tag_ABI_VFP_args: VFP registers' || fail "not hard-float (Tag_ABI_VFP_args)"

	needed=$(nickeltc "${tools}readelf -d /o/libnddrawer.so" | sed -n 's/.*(NEEDED).*\[\(.*\)\]/\1/p')
	for library_name in $needed; do
		case " $allowed_needed " in
		*" $library_name "*) ;;
		*) fail "unexpected NEEDED library $library_name" ;;
		esac
	done

	# std:: (STL) symbols are _ZSt/_ZNSt/_ZNKSt; the ABI runtime (__cxa_*,
	# __cxxabiv1 typeinfo vtables, operator new/delete) is allowed.
	stl=$(nickeltc "${tools}nm -D /o/libnddrawer.so" | grep -E ' _Z(N|NK)?St' || true)
	[ -z "$stl" ] || fail "C++ standard library symbols: $stl"
	echo "binary: ARM hard-float; NEEDED: $(echo $needed)"
}

# Every vtable must carry typeinfo: Nickel __dynamic_casts the mod's objects.
check_vtable_typeinfo() {
	[ -f "$library" ] || return 0 # check_binary already reported it
	"$repo_root/mod/common/check-vtable-typeinfo.sh" libnddrawer.so || fail "vtable typeinfo check"
}

check_firmware_exports
check_binary
check_vtable_typeinfo

if [ "$failures" -ne 0 ]; then
	echo "$failures check(s) failed" >&2
	exit 1
fi
echo "OK: all static checks passed"

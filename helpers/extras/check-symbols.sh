#!/bin/sh
# Static checks for build/helpers/libextras.so. Run from anywhere:
#   helpers/extras/check-symbols.sh
# 1. nickel_bridge.cc is the only source naming libnickel symbols.
# 2. Every libnickel symbol the bridge hooks, dlsyms or connects as a signal is exported by all five
#    firmware builds (re/symbols/<fw>/exports.txt, demangled; plus
#    ~/.cache/nickeldrawer/symbols/<fw>/exports-mangled.txt when present).
# 3. Every symbol the bridge hooks has a PLT (R_ARM_JUMP_SLOT) relocation in libnickel on all five
#    builds (needs the firmware cache ~/.cache/nickeldrawer/fw; skipped with a notice without it),
#    and no other mod in the repo hooks the same symbol: NickelHook's nh_dlhook returns the original
#    through dlsym, so a second hook on one symbol silently drops the first.
# 4. The library is ARM, hard-float, needs only the expected libraries, imports nothing from the C++
#    standard library beyond the ABI runtime, exports both hook targets, carries its own NickelHook
#    plugin key (nh.extras, not another mod's) and never names the drawer's folders.
# 5. The KoboRoot installs exactly the library.
# 6. Every vtable has typeinfo (mod/common/check-vtable-typeinfo.sh).
# Prints every failure; exits non-zero when any check failed.
set -eu

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)
library_name=libextras.so
build_subfolder=helpers
library="$repo_root/build/$build_subfolder/$library_name"
koboroot="$repo_root/build/$build_subfolder/KoboRoot-extras.tgz"
bridge="$script_dir/nickel_bridge.cc"
symbols_dir="$repo_root/re/symbols"
mangled_cache="${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}/symbols"
nickeltc_image="ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840"
allowed_needed="libdl.so.2 libQt5Widgets.so.5 libQt5Gui.so.5 libQt5Core.so.5 libstdc++.so.6 libgcc_s.so.1 libpthread.so.0 libc.so.6 ld-linux-armhf.so.3"
expected_firmware_count=5
firmware_cache="${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}/fw"
libnickel_path=root/usr/local/Kobo/libnickel.so.1.0.0
# The bridges of the repo's other mods whose hook tables must stay disjoint from ours.
other_bridges="$repo_root/mod/drawer/nickel_bridge.cc"
plugin_key=nh.extras
# CI sets CXXFILT=llvm-cxxfilt-NN so names demangle exactly as in re/symbols/<fw>/exports.txt.
cxxfilt=${CXXFILT:-c++filt}

failures=0
fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}

# Quoted mangled names (_ZN...) in the bridge: the hook table, the dlsym table and the
# "signal-symbol:" comments for SIGNAL() connections.
list_nickel_symbols() {
	grep -hoE '"_ZN[^"]+"|signal-symbol: _ZN[A-Za-z0-9_]+' "$bridge" | sed -E 's/^"//; s/"$//; s/^signal-symbol: //' | sort -u
}

check_bridge_is_only_symbol_user() {
	offenders=$(grep -lE '"_ZN[0-9]+[A-Za-z]|nh_dlsym|nh_hook|dlsym\(' "$script_dir"/*.cc "$script_dir"/*.h | grep -v -e '/nickel_bridge\.' -e '/extras\.cc$' || true)
	[ -z "$offenders" ] || fail "libnickel symbols outside the bridge: $offenders"
}

check_firmware_exports() {
	symbols=$(list_nickel_symbols)
	if [ -z "$symbols" ]; then
		fail "no libnickel symbols found in $bridge"
		return
	fi
	firmware_count=0
	for exports in "$symbols_dir"/*/exports.txt; do
		firmware=$(basename "$(dirname "$exports")")
		firmware_count=$((firmware_count + 1))
		mangled_dump="$mangled_cache/$firmware/exports-mangled.txt"
		for symbol in $symbols; do
			demangled=$(echo "$symbol" | "$cxxfilt")
			grep -Fxq -- "$demangled" "$exports" || fail "$firmware: $demangled ($symbol) not exported"
			if [ -f "$mangled_dump" ]; then
				grep -Fxq -- "$symbol" "$mangled_dump" || fail "$firmware: $symbol missing from $mangled_dump"
			fi
		done
	done
	[ "$firmware_count" -eq "$expected_firmware_count" ] ||
		fail "expected $expected_firmware_count firmware dumps in $symbols_dir, found $firmware_count (run re/scripts/fetch-fw.sh)"
	echo "symbols: $(echo "$symbols" | wc -l | tr -d ' ') libnickel symbols checked against $firmware_count builds"
}

hook_symbols_of() {
	grep -hoE '\.sym = "_ZN[^"]+"' "$1" | sed -E 's/.*"(.*)"/\1/' | sort -u
}

# No two mods may hook one symbol (see the header of this file).
check_hooks_are_not_shared() {
	ours=$(hook_symbols_of "$bridge")
	[ -n "$ours" ] || {
		fail "no hook symbols found in $bridge"
		return
	}
	for other in $other_bridges; do
		[ -f "$other" ] || continue
		for symbol in $(hook_symbols_of "$other"); do
			echo "$ours" | grep -Fxq -- "$symbol" && fail "$symbol is hooked here and in $other"
		done
	done
	echo "hooks: $(echo "$ours" | wc -l | tr -d ' ') symbols, none shared with another mod"
}

nickeltc() {
	docker run --rm --platform linux/amd64 -v "$repo_root/build/$build_subfolder:/o:ro" --entrypoint sh "$nickeltc_image" -c "$1"
}

# Each hooked symbol needs a jump-slot relocation in libnickel: that is what nh_dlhook patches.
check_hooks_have_plt_slots() {
	if [ ! -d "$firmware_cache" ] || ! command -v docker >/dev/null 2>&1; then
		echo "plt: skipped (no firmware cache at $firmware_cache or no docker; run re/scripts/fetch-fw.sh)"
		return
	fi
	checked=0
	for firmware_dir in "$firmware_cache"/*/; do
		firmware=$(basename "$firmware_dir")
		[ -f "$firmware_dir$libnickel_path" ] || continue
		slots=$(docker run --rm --platform linux/amd64 -v "$firmware_cache:/fw:ro" --entrypoint sh "$nickeltc_image" \
			-c "readelf -rW /fw/$firmware/$libnickel_path | grep JUMP_SLOT" || true)
		for symbol in $(hook_symbols_of "$bridge"); do
			echo "$slots" | grep -q " $symbol\$" || fail "$firmware: no PLT slot for $symbol (nh_hook cannot patch it)"
		done
		checked=$((checked + 1))
	done
	echo "plt: hook symbols have jump slots on $checked build(s)"
}

check_plugin_key() {
	# The manifest is generated into the vendored NickelHook folder, which the drawer build shares.
	grep -aq "$plugin_key" "$library" || fail "plugin key $plugin_key not in $library_name (stale shared manifest? run make -C helpers/extras docker-build)"
	if grep -aq 'nh\.nickeldrawer' "$library"; then
		fail "the drawer's plugin key nh.nickeldrawer is in $library_name"
	fi
}

check_independent_of_the_drawer() {
	grep -aq 'nickeldrawer\|koboapps' "$library" && fail "$library_name names the drawer's folders (nickeldrawer/koboapps); the helper must stand alone"
	return 0
}

# The other direction of "no conflict": when the drawer's library is built, it must not name this helper,
# its folder or its tweaks, or it would not stand alone.
check_drawer_library_is_independent() {
	drawer_library="$repo_root/build/mod/libnickeldrawer.so"
	[ -f "$drawer_library" ] || return 0
	if grep -aiq 'extras\|tweak\|adds/helpers' "$drawer_library"; then
		fail "libnickeldrawer.so names this helper (extras, tweak, adds/helpers); the drawer must stand alone"
	fi
}

check_binary() {
	if [ ! -f "$library" ]; then
		fail "$library missing; run: make -C helpers/extras docker-build"
		return
	fi
	if ! command -v docker >/dev/null 2>&1; then
		fail "docker not found; binary checks need the NickelTC image"
		return
	fi
	tools=arm-nickel-linux-gnueabihf-
	attributes=$(nickeltc "${tools}readelf -h -A /o/$library_name")
	echo "$attributes" | grep -q 'Machine: *ARM' || fail "not an ARM ELF"
	echo "$attributes" | grep -q 'Tag_ABI_VFP_args: VFP registers' || fail "not hard-float (Tag_ABI_VFP_args)"

	needed=$(nickeltc "${tools}readelf -d /o/$library_name" | sed -n 's/.*(NEEDED).*\[\(.*\)\]/\1/p')
	for needed_library in $needed; do
		case " $allowed_needed " in
		*" $needed_library "*) ;;
		*) fail "unexpected NEEDED library $needed_library" ;;
		esac
	done

	# std:: (STL) symbols are _ZSt/_ZNSt/_ZNKSt; the ABI runtime (__cxa_*, __cxxabiv1 typeinfo
	# vtables, operator new/delete) is allowed.
	dynamic_symbols=$(nickeltc "${tools}nm -D /o/$library_name")
	stl=$(echo "$dynamic_symbols" | grep -E ' _Z(N|NK)?St' || true)
	[ -z "$stl" ] || fail "C++ standard library symbols: $stl"
	for hook_target in $(grep -hoE '\.sym_new = "[A-Za-z0-9_]+"' "$bridge" | sed -E 's/.*"(.*)"/\1/'); do
		echo "$dynamic_symbols" | grep -q " T $hook_target\$" || fail "hook target $hook_target not exported"
	done
	echo "$dynamic_symbols" | grep -q ' D NickelHook$' || fail "NickelHook struct not exported"
	echo "binary: ARM hard-float; NEEDED: $(echo $needed)"
}

check_koboroot() {
	if [ ! -f "$koboroot" ]; then
		fail "$koboroot missing; run: make -C helpers/extras docker-build"
		return
	fi
	contents=$(tar -tzf "$koboroot" | grep -v '/$')
	[ "$contents" = "./usr/local/Kobo/imageformats/$library_name" ] ||
		[ "$contents" = "usr/local/Kobo/imageformats/$library_name" ] ||
		fail "KoboRoot contents unexpected: $contents"
	echo "koboroot: $contents"
}

# Every vtable must carry typeinfo: Nickel __dynamic_casts objects a mod gives it.
check_vtable_typeinfo() {
	[ -f "$library" ] || return 0 # check_binary already reported it
	"$repo_root/mod/common/check-vtable-typeinfo.sh" $library_name $build_subfolder || fail "vtable typeinfo check"
}

check_bridge_is_only_symbol_user
check_firmware_exports
check_hooks_are_not_shared
check_hooks_have_plt_slots
if [ -f "$library" ]; then
	check_plugin_key
	check_independent_of_the_drawer
	check_drawer_library_is_independent
fi
check_binary
check_koboroot
check_vtable_typeinfo

if [ "$failures" -ne 0 ]; then
	echo "$failures check(s) failed" >&2
	exit 1
fi
echo "OK: all static checks passed"

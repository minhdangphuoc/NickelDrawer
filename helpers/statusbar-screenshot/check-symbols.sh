#!/bin/sh
# Static checks for build/helpers/libstatusbarshot.so. Run from anywhere:
#   helpers/statusbar-screenshot/check-symbols.sh
# 1. nickel_bridge.cc is the only source naming libnickel symbols.
# 2. Every libnickel symbol the bridge hooks, dlsyms or connects as a signal is exported by all five
#    firmware builds (re/symbols/<fw>/exports.txt, demangled; plus
#    ~/.cache/nickeldrawer/symbols/<fw>/exports-mangled.txt when present).
# 3. The library is ARM, hard-float, needs only the expected libraries, imports nothing from the C++
#    standard library beyond the ABI runtime, exports the hook target, carries its own NickelHook
#    plugin key (nh.statusbarshot, not another mod's) and never names the drawer's folders.
# 4. The KoboRoot installs exactly the library.
# 5. Every vtable has typeinfo (mod/common/check-vtable-typeinfo.sh).
# Prints every failure; exits non-zero when any check failed.
set -eu

script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)
library_name=libstatusbarshot.so
build_subfolder=helpers
library="$repo_root/build/$build_subfolder/$library_name"
koboroot="$repo_root/build/$build_subfolder/KoboRoot-statusbarshot.tgz"
bridge="$script_dir/nickel_bridge.cc"
symbols_dir="$repo_root/re/symbols"
mangled_cache="${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}/symbols"
nickeltc_image="ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840"
allowed_needed="libdl.so.2 libQt5Widgets.so.5 libQt5Gui.so.5 libQt5Core.so.5 libstdc++.so.6 libgcc_s.so.1 libpthread.so.0 libc.so.6 ld-linux-armhf.so.3"
expected_firmware_count=5
plugin_key=nh.statusbarshot
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
	offenders=$(grep -lE '"_ZN[0-9]+[A-Za-z]|nh_dlsym|nh_hook|dlsym\(' "$script_dir"/*.cc "$script_dir"/*.h | grep -v -e '/nickel_bridge\.' -e '/statusbarshot\.cc$' || true)
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

nickeltc() {
	docker run --rm --platform linux/amd64 -v "$repo_root/build/$build_subfolder:/o:ro" --entrypoint sh "$nickeltc_image" -c "$1"
}

check_plugin_key() {
	# The manifest is generated into the vendored NickelHook folder, which the drawer build shares.
	grep -aq "$plugin_key" "$library" || fail "plugin key $plugin_key not in $library_name (stale shared manifest? run make -C helpers/statusbar-screenshot docker-build)"
	if grep -aq 'nh\.nickeldrawer' "$library"; then
		fail "the drawer's plugin key nh.nickeldrawer is in $library_name"
	fi
}

check_independent_of_the_drawer() {
	grep -aq 'nickeldrawer\|koboapps' "$library" && fail "$library_name names the drawer's folders (nickeldrawer/koboapps); the helper must stand alone"
	return 0
}

check_binary() {
	if [ ! -f "$library" ]; then
		fail "$library missing; run: make -C helpers/statusbar-screenshot docker-build"
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
	echo "$dynamic_symbols" | grep -q ' T statusbarshot_StatusBarView_construct$' || fail "hook target statusbarshot_StatusBarView_construct not exported"
	echo "$dynamic_symbols" | grep -q ' D NickelHook$' || fail "NickelHook struct not exported"
	echo "binary: ARM hard-float; NEEDED: $(echo $needed)"
}

check_koboroot() {
	if [ ! -f "$koboroot" ]; then
		fail "$koboroot missing; run: make -C helpers/statusbar-screenshot docker-build"
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
if [ -f "$library" ]; then
	check_plugin_key
	check_independent_of_the_drawer
fi
check_binary
check_koboroot
check_vtable_typeinfo

if [ "$failures" -ne 0 ]; then
	echo "$failures check(s) failed" >&2
	exit 1
fi
echo "OK: all static checks passed"

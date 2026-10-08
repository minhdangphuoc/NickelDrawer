#!/bin/sh
# Download one Libra 2 (kobo9) firmware package into the local cache, verify it
# is a Freescale i.MX6SLL image, extract libnickel and dump its exported symbols.
#
# Usage: re/scripts/fetch-fw.sh <version> <cdn-month>
#   e.g. re/scripts/fetch-fw.sh 4.38.23828 Sep2026
#
# Kobo binaries stay in $NICKELDRAWER_CACHE (default ~/.cache/nickeldrawer).
# Only symbol names are written into the repo (re/symbols/<version>/exports.txt).
set -eu

version=${1:?usage: fetch-fw.sh <version> <cdn-month>}
month=${2:?usage: fetch-fw.sh <version> <cdn-month>}

repo_re_dir=$(cd "$(dirname "$0")/.." && pwd)
cache=${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}
fw_dir=$cache/fw/$version
sym_cache=$cache/symbols/$version
zip_name=kobo-update-$version.zip
url=https://ereaderfiles.kobo.com/firmwares/kobo9/$month/$zip_name

llvm_nm=$(xcrun --find llvm-nm) || { echo "llvm-nm not found: install Xcode command line tools" >&2; exit 1; }
cxxfilt=$(xcrun --find llvm-cxxfilt) || { echo "llvm-cxxfilt not found: install Xcode command line tools" >&2; exit 1; }

download_zip() {
	mkdir -p "$fw_dir"
	if [ -f "$fw_dir/$zip_name" ]; then
		echo "cached: $fw_dir/$zip_name"
		return
	fi
	curl -fSL -o "$fw_dir/$zip_name.part" "$url"
	mv "$fw_dir/$zip_name.part" "$fw_dir/$zip_name"
}

extract_package() {
	rm -rf "$fw_dir/unz" "$fw_dir/root"
	mkdir -p "$fw_dir/unz" "$fw_dir/root"
	unzip -q -o "$fw_dir/$zip_name" KoboRoot.tgz manifest.md5sum 'upgrade/*' -d "$fw_dir/unz"
	tar -tzvf "$fw_dir/unz/KoboRoot.tgz" >"$fw_dir/koboroot-list.txt"
	tar -xzf "$fw_dir/unz/KoboRoot.tgz" -C "$fw_dir/root" \
		./usr/local/Kobo/libnickel.so.1.0.0 ./usr/local/Kobo/nickel ./usr/local/Kobo/platforms/libkobo.so
}

# The 4.39.22801 incident shipped a MediaTek (mt8113t-ntx) image under kobo9.
# A real Libra 2 package has only upgrade/mx6sll-ntx/*E70K10* boot files.
verify_libra2_image() {
	if ! grep -q 'upgrade/mx6sll-ntx/zImage-E70K10' "$fw_dir/unz/manifest.md5sum"; then
		echo "NOT a Libra 2 image: no mx6sll-ntx/zImage-E70K10 in manifest" >&2
		exit 2
	fi
	if grep -q 'mt8113' "$fw_dir/unz/manifest.md5sum"; then
		echo "NOT a Libra 2 image: MediaTek (mt8113) boot files present" >&2
		exit 2
	fi
	file "$fw_dir/root/usr/local/Kobo/libnickel.so.1.0.0" | grep -q 'ELF 32-bit LSB shared object, ARM, EABI5' ||
		{ echo "libnickel is not an ARM EABI5 ELF" >&2; exit 2; }
	echo "verified: mx6sll-ntx / E70K10 (Libra 2), ARM EABI5 libnickel"
}

dump_exports() {
	mkdir -p "$sym_cache" "$repo_re_dir/symbols/$version"
	"$llvm_nm" -D --defined-only --format=just-symbols "$fw_dir/root/usr/local/Kobo/libnickel.so.1.0.0" |
		LC_ALL=C sort -u >"$sym_cache/exports-mangled.txt"
	"$llvm_nm" -D --defined-only --format=just-symbols "$fw_dir/root/usr/local/Kobo/nickel" |
		LC_ALL=C sort -u >"$sym_cache/nickel-exports-mangled.txt"
	"$cxxfilt" <"$sym_cache/exports-mangled.txt" | LC_ALL=C sort >"$repo_re_dir/symbols/$version/exports.txt"
	echo "exports: $(wc -l <"$sym_cache/exports-mangled.txt") -> re/symbols/$version/exports.txt"
}

print_lock_line() {
	zip_sha=$(shasum -a 256 "$fw_dir/$zip_name" | cut -d' ' -f1)
	zip_size=$(stat -f %z "$fw_dir/$zip_name")
	lib_sha=$(shasum -a 256 "$fw_dir/root/usr/local/Kobo/libnickel.so.1.0.0" | cut -d' ' -f1)
	printf 'lock: %s\t%s\t%s\t%s\t%s\n' "$version" "$url" "$zip_size" "$zip_sha" "$lib_sha"
}

download_zip
extract_package
verify_libra2_image
dump_exports
print_lock_line

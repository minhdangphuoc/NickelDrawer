#!/bin/sh
# Fails when any vtable in a built mod library has a null typeinfo slot.
#   mod/common/check-vtable-typeinfo.sh <library file name in build/mod>
# Nickel runs __dynamic_cast on objects the mod hands it (the More row's
# children in IconButtonBase::tapGesture, controllers in push/pop). A class
# compiled with -fno-rtti has 0 in its vtable's typeinfo slot, and
# __dynamic_cast dereferences it: Nickel crashes (re/notes/protoA-crash.md).
# A typeinfo slot is fine when its word is non-zero (R_ARM_RELATIVE addend) or
# a relocation fills it (imported typeinfo).
set -eu

if [ $# -ne 1 ]; then
	echo "usage: $0 <library file name in build/mod>" >&2
	exit 2
fi
library_name=$1
repo_root=$(cd "$(dirname "$0")/../.." && pwd)
nickeltc_image="ghcr.io/pgaskin/nickeltc@sha256:3296eb400d270cfd26f5773ab09625d4469f85680b3a191951dfb78d8ade7840"

# Runs inside the NickelTC image; prints "checked <n>" and one "null <symbol>" per bad vtable.
inspect_vtables='
tools=arm-nickel-linux-gnueabihf-
library=/o/$0
relocated=$(${tools}readelf -rW "$library" | awk "/^[0-9a-f]+ /{print \$1}")
count=0
for entry in $(${tools}nm "$library" | awk "\$3 ~ /^_ZTV/ {print \$1 \":\" \$3}"); do
	address=${entry%%:*}
	symbol=${entry#*:}
	slot=$(printf "%08x" $((0x$address + 4)))
	word=$(${tools}objdump -s --start-address=0x$slot --stop-address=$(printf "0x%x" $((0x$slot + 4))) "$library" |
		awk "/^ [0-9a-f]+ [0-9a-f]+/ {print \$2; exit}")
	count=$((count + 1))
	if [ "$word" = "00000000" ] && ! echo "$relocated" | grep -qx "$slot"; then
		echo "null $symbol"
	fi
done
echo "checked $count"
'

report=$(docker run --rm --platform linux/amd64 -v "$repo_root/build/mod:/o:ro" --entrypoint sh "$nickeltc_image" \
	-c "$inspect_vtables" "$library_name")
checked=$(echo "$report" | sed -n 's/^checked //p')
nulls=$(echo "$report" | sed -n 's/^null //p')

if [ -z "$checked" ] || [ "$checked" -eq 0 ]; then
	echo "FAIL: no vtables found in $library_name (stripped?); cannot check typeinfo" >&2
	exit 1
fi
if [ -n "$nulls" ]; then
	echo "FAIL: vtables without typeinfo (build with -frtti): $(echo "$nulls" | c++filt | tr '\n' ' ')" >&2
	exit 1
fi
echo "vtables: $checked checked, all have typeinfo"

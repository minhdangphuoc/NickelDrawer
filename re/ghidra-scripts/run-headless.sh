#!/bin/sh
# Runs NickelDrawer GhidraScripts headless, read-only, against an analysed libnickel project.
# Usage: run-headless.sh <fw-version> -postScript <Script>.java <args...> [-postScript ...]
# Output of DecompileByName is Kobo code: point it at ~/.cache/nickeldrawer/decomp/, never the repo.
set -eu

if [ $# -lt 3 ]; then
	echo "usage: $0 <fw-version> -postScript <Script>.java <args...>" >&2
	exit 2
fi
fw_version=$1
shift

ghidra_home=${GHIDRA_HOME:-$HOME/Applications/ghidra_11.3.2_PUBLIC}
jdk_bin=${JDK21_BIN:-/opt/homebrew/opt/openjdk@21/bin}
project_dir=${NICKELDRAWER_GHIDRA_PROJECTS:-$HOME/.local/share/nickeldrawer/ghidra-projects}
script_dir=$(cd "$(dirname "$0")" && pwd)

if [ ! -d "$project_dir/libnickel-$fw_version.rep" ]; then
	echo "no Ghidra project for $fw_version in $project_dir; import it first (docs/GHIDRA.md)" >&2
	exit 1
fi
mkdir -p "${NICKELDRAWER_CACHE:-$HOME/.cache/nickeldrawer}/decomp"

PATH="$jdk_bin:$PATH" exec "$ghidra_home/support/analyzeHeadless" "$project_dir" "libnickel-$fw_version" \
	-process libnickel.so.1.0.0 -noanalysis -readOnly -scriptPath "$script_dir" "$@"

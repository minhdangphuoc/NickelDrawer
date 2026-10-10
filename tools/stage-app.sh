#!/bin/bash
# Stages one app for packaging: copies apps/<id> into <tree>/.adds/apps/<id> without dev-only
# files (README, preview-state*, data/) and adds its built binary.
#   tools/stage-app.sh <tree> <id> [<built-binary> <name-in-bin>]
# Used by `make dist`; the tree is then packed into dist/apps/app-<id>-<version>.zip.
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)

fail() {
    echo "Error: $*" >&2
    exit 1
}

[ "$#" -eq 2 ] || [ "$#" -eq 4 ] || fail "usage: $0 <tree> <id> [<built-binary> <name-in-bin>]"
tree=$1
appId=$2
source=$REPO_ROOT/apps/$appId
target=$tree/.adds/apps/$appId
[ -f "$source/app.json" ] || fail "apps/$appId/app.json not found"

rm -rf "$target"
mkdir -p "$target"
(cd "$source" && find . -type f \
    ! -path './data/*' ! -name 'preview-state*' ! -name 'README*' \
    ! -name '.DS_Store' ! -name '._*' -print) | while IFS= read -r file; do
    mkdir -p "$target/$(dirname "$file")"
    cp "$source/$file" "$target/$file"
done

if [ "$#" -eq 4 ]; then
    [ -f "$3" ] || fail "$3 is missing; run: make build"
    mkdir -p "$target/bin"
    cp "$3" "$target/bin/$4"
fi

[ ! -d "$target/bin" ] || chmod 755 "$target"/bin/*
find "$target" -type f ! -path "$target/bin/*" -exec chmod 644 {} +

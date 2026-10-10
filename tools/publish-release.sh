#!/bin/bash
# Signs and publishes a NickelDrawer drawer release (maintainer only; needs the release key and gh).
# Usage: tools/publish-release.sh VERSION [--prerelease] [--repo owner/name] [--target branch]
# Push the release commit first: the tag is created on GitHub at --target (default main).
set -euo pipefail

usage() {
    echo "Usage: tools/publish-release.sh VERSION [--prerelease] [--repo owner/name] [--target branch]" >&2
    exit 2
}

[ $# -ge 1 ] || usage
version=$1
shift
prerelease=()
repo=$(python3 -c 'import json; print(json.load(open("host/config/project.json"))["repo"])')
target=main
while [ $# -gt 0 ]; do
    case $1 in
    --prerelease) prerelease=(--prerelease) ;;
    --repo) repo=$2; shift ;;
    --target) target=$2; shift ;;
    *) usage ;;
    esac
    shift
done

cd "$(dirname "$0")/.."
if ! grep -q "^## \[$version\]" CHANGELOG.md; then
    echo "CHANGELOG.md has no '## [$version]' section. Move the Unreleased items under it first." >&2
    exit 1
fi

rm -rf dist/release
make release VERSION="$version"
build/ndrelease-host verify

cp build/mod/KoboRoot-nickeldrawer.tgz dist/release/
cp "dist/koboapps-$version.zip" dist/release/
assets=(release.json release.json.sig "nickeldrawer-$version.zip" "koboapps-$version.zip" KoboRoot-nickeldrawer.tgz)
(cd dist/release && shasum -a 256 "${assets[@]}" >SHA256SUMS.txt && cat SHA256SUMS.txt)
awk "/^## \\[$version\\]/{keep=1;next} /^## \\[/{keep=0} keep" CHANGELOG.md >dist/release/notes.md

paths=()
for asset in "${assets[@]}" SHA256SUMS.txt; do
    paths+=("dist/release/$asset")
done
gh release create "v$version" "${paths[@]}" "${prerelease[@]}" -R "$repo" --target "$target" \
    --title "NickelDrawer $version" --notes-file dist/release/notes.md
echo "Published v$version on $repo"

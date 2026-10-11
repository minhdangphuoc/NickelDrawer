# Releasing

For maintainers. Nothing here is automated: signing and publishing are manual steps on the maintainer's Mac.

**Shortcut for a drawer release:** after the CHANGELOG section, the `Makefile` version and the release commit are pushed, `tools/publish-release.sh X.Y.Z [--prerelease]` runs `make release`, verifies, writes `SHA256SUMS.txt` and the notes, and creates the GitHub release (the tag is made on GitHub at `main`). The steps below are what it does.

## The signing key

The ed25519 private key lives only at `~/.config/nickeldrawer/release-ed25519.key` (mode 600) on the maintainer's Mac. It never goes in the repo, CI, chat, logs or a release. The public key is `spec/release-ed25519.pub`; `appsctl` embeds it. If the key is lost or leaked, follow the key-loss paragraph in [UPDATES.md](UPDATES.md).

## Versions

`MAJOR.MINOR.PATCH` with an optional `-prerelease`, for example `1.0.0-beta.1`. Pre-releases sort before their release (`1.0.0-beta.1` < `1.0.0-beta.2` < `1.0.0`). The git tag is `v` plus the version. GitHub pre-releases are for betas; `stable` clients ignore them. The exact commands for the first public beta are in [RELEASE-CHECKLIST-1.0.md](RELEASE-CHECKLIST-1.0.md).

## Steps

1. Start from a clean `main`. Update `CHANGELOG.md`: move "Unreleased" items under the new version and date.
2. Set the default `VERSION ?=` in the `Makefile`. (App versions are released on their own, see [App releases](#app-releases).)
3. Run the checks, and the mod checks if the mod changed (needs Docker):
   ```sh
   make check
   tools/test-install.sh
   make -C mod/drawer docker-build docker-test
   mod/drawer/check-symbols.sh      # needs re/symbols/<fw>/exports.txt, made by re/scripts/fetch-fw.sh
   ```
   CI runs the same checks on every push and, for a `v*` tag, builds the unsigned release inputs as a workflow artifact ([CI.md](CI.md)). CI never signs.
4. Build and sign (the key must exist; `VERSION` must be on the command line):
   ```sh
   make release VERSION=X.Y.Z
   ```
   This runs `check build dist`, then `ndrelease sign` and `ndrelease verify`. Output is in `dist/release/`: `release.json`, `release.json.sig`, `nickeldrawer-X.Y.Z.zip` (the drawer: host tool and mod, no apps) and all listed with sha256 and size in the signed `release.json`. App zips are not part of it: `make dist` still writes them to `dist/apps/` for [app releases](#app-releases) and manual installs.
5. Verify again by hand and record the hashes:
   ```sh
   build/ndrelease-host verify
   shasum -a 256 dist/release/*
   ```
6. Commit and tag (Conventional Commits; no tool or assistant names anywhere):
   ```sh
   git commit -am "chore: release X.Y.Z"
   git tag -s vX.Y.Z -m "X.Y.Z"
   git push origin main vX.Y.Z
   ```
7. Publish the assets: the three update files, `koboapps-X.Y.Z.zip` (the drawer tree for a manual install), `KoboRoot-nickeldrawer.tgz` and a `SHA256SUMS.txt` for users who install by hand. Use `--prerelease` for betas; `stable` clients ignore pre-releases. Drawer tags are always `v` plus the version: the drawer updater ignores every other tag.
   ```sh
   awk '/^## \[X.Y.Z\]/{keep=1;next} /^## \[/{keep=0} keep' CHANGELOG.md >/tmp/notes.md
   gh release create vX.Y.Z \
     dist/release/release.json dist/release/release.json.sig dist/release/nickeldrawer-X.Y.Z.zip \
     dist/koboapps-X.Y.Z.zip build/mod/KoboRoot-nickeldrawer.tgz SHA256SUMS.txt \
     --title "NickelDrawer X.Y.Z" --notes-file /tmp/notes.md
   ```
8. Test the update on a device you control: **Apps · Check for updates**, then **Apps · Update status**. Check `appsctl update rollback` still works.

## App releases

Each app is released on its own, without a drawer release. The device installs and updates apps from a signed catalog; the design is in [spec/app-updates.md](spec/app-updates.md) and the user side in [UPDATES.md](UPDATES.md).

- Every app release is a GitHub release in this repo with the tag `<id>-v<version>` (for example `todo-v1.0.2`) holding one file, `app-<id>-<version>.zip`. Create it with `--prerelease --latest=false`: while every drawer release is a beta, GitHub would otherwise mark the newest app release as the repo's "Latest" (the device ignores both flags).
- The catalog `apps.json` + `apps.json.sig` lives on one fixed release tagged `apps`. Its assets are replaced on every app release. The device downloads it from `https://github.com/<repo>/releases/download/apps/apps.json`.
- The same private key signs drawer releases and the catalog (the catalog over `nickeldrawer-apps-v1\n` + the bytes).

### Ordering rules (read before the first app release)

1. **Publish a drawer beta that contains the tag filter first**, then create the `apps` release. A drawer built before the filter (1.0.0-beta.5) takes the newest release of any kind on the beta channel, so it would try to update from `apps` or `todo-v1.0.2` and fail with "missing file".
2. Until every device runs the fix, **publish a drawer beta after any app release** (the app and catalog releases are newer than the last drawer release, which is what beta.5 devices look at). A newer drawer release restores their update path. Once the fixed drawer is the only one in use, this rule can go.
3. Create the app's own release (tag and zip) **before** you replace the catalog: a catalog that names a tag that does not exist yet gives devices a 404. Upload `apps.json.sig` and `apps.json` last; they are the switch.
4. A catalog lists every app at its current version. Before the first catalog, release **every** app once (`make release-app APP=all`), so no catalog entry points at a missing tag. Afterwards release only the app whose version you bumped.

### Steps

1. Bump the version in the changed app's `app.json` (semver, see [APPS.md](APPS.md); it names the zip and the row in [apps/README.md](../apps/README.md)), update its README and `CHANGELOG.md`, and run `make check`.
2. Pick the catalog sequence: one more than the `sequence` in the catalog now published (`curl -sL https://github.com/<repo>/releases/download/apps/apps.json | grep sequence`; use 1 for the first).
3. Build, pack, sign and verify (the key must exist):
   ```sh
   make release-app APP=todo SEQUENCE=8
   ```
   This runs `check build dist`, signs a catalog of every app zip in `dist/apps`, and writes `dist/apps-release/`: `apps.json`, `apps.json.sig` and `app-todo-<version>.zip` (with `APP=all` every zip). Then `ndrelease verify-apps` checks the signature and every zip against the catalog. Use `build/ndrelease-host verify-apps --dir dist/apps-release --apps-dir dist/apps` to run it again by hand.
4. Check that the tag of every catalog entry exists, except the one you are about to create:
   ```sh
   for tag in $(grep '"tag"' dist/apps-release/apps.json | sed 's/.*: "\(.*\)".*/\1/'); do
     gh release view "$tag" >/dev/null 2>&1 || echo "missing release: $tag"
   done
   ```
5. Create the app's release with its zip, then replace the catalog (zip and new release first, `apps.json.sig` and `apps.json` last):
   ```sh
   gh release create todo-v1.0.2 dist/apps-release/app-todo-1.0.2.zip \
     --title "To-do 1.0.2" --notes "To-do 1.0.2" --prerelease --latest=false
   gh release upload apps dist/apps-release/apps.json.sig dist/apps-release/apps.json --clobber
   ```
6. Check on a device you control: **Settings > Updates > Apps > Check** shows the new version; update it; its `data/` is still there.

### First time only: create the `apps` release

Do this after the drawer beta of rule 1 is published and, ideally, installed on your test device. `--prerelease --latest=false` keeps it from becoming the repo's "Latest" release.

```sh
make release-app APP=all SEQUENCE=1
for tag in $(sed -n 's/.*"tag": "\(.*\)".*/\1/p' dist/apps-release/apps.json); do
  zip="dist/apps-release/app-$(echo "$tag" | sed 's/-v\([0-9]\)/-\1/').zip"      # todo-v1.0.2 -> app-todo-1.0.2.zip
  gh release create "$tag" "$zip" --title "$tag" --notes "$tag" --prerelease --latest=false
done
gh release create apps dist/apps-release/apps.json.sig dist/apps-release/apps.json \
  --title "App catalog" --notes "Signed app catalog read by NickelDrawer. Do not edit." --prerelease --latest=false
```

If a release was created by mistake with the wrong asset, `gh release upload <tag> <file> --clobber` replaces the asset, and `gh release delete <tag> --cleanup-tag` removes the release.

## Before the first public release

- No Kobo binaries, firmware files or personal data in the tree, the zips or the KoboRoot (`unzip -l dist/release/*.zip`, `tar -tzf build/mod/KoboRoot-nickeldrawer.tgz`).
- The public key in `spec/release-ed25519.pub` is the real one, and `make release` succeeds (it refuses a key that does not match).
- `SECURITY.md` names a contact; `LICENSE` and `third_party/README.md` are current.
- `docs/COMPATIBILITY.md` matches what was tested.

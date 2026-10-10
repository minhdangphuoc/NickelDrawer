# Releasing

For maintainers. Nothing here is automated: signing and publishing are manual steps on the maintainer's Mac.

## The signing key

The ed25519 private key lives only at `~/.config/nickeldrawer/release-ed25519.key` (mode 600) on the maintainer's Mac. It never goes in the repo, CI, chat, logs or a release. The public key is `spec/release-ed25519.pub`; `appsctl` embeds it. If the key is lost or leaked, follow the key-loss paragraph in [UPDATES.md](UPDATES.md).

## Versions

`MAJOR.MINOR.PATCH` with an optional `-prerelease`, for example `1.0.0-beta.1`. Pre-releases sort before their release (`1.0.0-beta.1` < `1.0.0-beta.2` < `1.0.0`). The git tag is `v` plus the version. GitHub pre-releases are for betas; `stable` clients ignore them. The exact commands for the first public beta are in [RELEASE-CHECKLIST-1.0.md](RELEASE-CHECKLIST-1.0.md).

## Steps

1. Start from a clean `main`. Update `CHANGELOG.md`: move "Unreleased" items under the new version and date.
2. Bump the version in each changed app's `app.json` (semver, see [APPS.md](APPS.md); it names the app zip and the row in [apps/README.md](../apps/README.md)) and set the default `VERSION ?=` in the `Makefile`.
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
   This runs `check build dist`, then `ndrelease sign` and `ndrelease verify`. Output is in `dist/release/`: `release.json`, `release.json.sig`, `nickeldrawer-X.Y.Z.zip` (the drawer: host tool and mod, no apps) and one `app-<id>-<version>.zip` per app (also in `dist/apps/`), all listed with sha256 and size in the signed `release.json`.
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
7. Publish the assets: the three update files, `koboapps-X.Y.Z.zip` (the drawer tree for a manual install), `KoboRoot-nickeldrawer.tgz`, every app zip from `dist/apps/` and a `SHA256SUMS.txt` for users who install by hand. Apps are installed one by one from these zips ([apps/README.md](../apps/README.md)). Use `--prerelease` for betas; `stable` clients ignore pre-releases.
   ```sh
   awk '/^## \[X.Y.Z\]/{keep=1;next} /^## \[/{keep=0} keep' CHANGELOG.md >/tmp/notes.md
   gh release create vX.Y.Z \
     dist/release/release.json dist/release/release.json.sig dist/release/nickeldrawer-X.Y.Z.zip \
     dist/koboapps-X.Y.Z.zip build/mod/KoboRoot-nickeldrawer.tgz dist/apps/app-*.zip SHA256SUMS.txt \
     --title "NickelDrawer X.Y.Z" --notes-file /tmp/notes.md
   ```
8. Test the update on a device you control: **Apps · Check for updates**, then **Apps · Update status**. Check `appsctl update rollback` still works.

## Before the first public release

- No Kobo binaries, firmware files or personal data in the tree, the zips or the KoboRoot (`unzip -l dist/release/*.zip`, `tar -tzf build/mod/KoboRoot-nickeldrawer.tgz`).
- The public key in `spec/release-ed25519.pub` is the real one, and `make release` succeeds (it refuses a key that does not match).
- `SECURITY.md` names a contact; `LICENSE` and `third_party/README.md` are current.
- `docs/COMPATIBILITY.md` matches what was tested.

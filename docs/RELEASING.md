# Releasing

For maintainers. Nothing here is automated: signing and publishing are manual steps on the maintainer's Mac.

## The signing key

The ed25519 private key lives only at `~/.config/nickeldrawer/release-ed25519.key` (mode 600) on the maintainer's Mac. It never goes in the repo, CI, chat, logs or a release. The public key is `spec/release-ed25519.pub`; `appsctl` embeds it. If the key is lost or leaked, follow the key-loss paragraph in [UPDATES.md](UPDATES.md).

## Steps

1. Start from a clean `main`. Update `CHANGELOG.md`: move "Unreleased" items under the new version and date.
2. Bump the version in each changed app's `app.json` (semver, see [APPS.md](APPS.md)) and set the default `VERSION ?=` in the `Makefile`.
3. Run the checks:
   ```sh
   make check
   tools/test-install.sh
   ```
4. Build and sign (the key must exist; `VERSION` must be on the command line):
   ```sh
   make release VERSION=X.Y.Z
   ```
   This runs `check build dist`, then `ndrelease sign` and `ndrelease verify`. Output is in `dist/release/`: `release.json`, `release.json.sig`, `nickeldrawer-X.Y.Z.zip`.
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
7. Publish the three assets. Use `--prerelease` for betas; `stable` clients ignore pre-releases.
   ```sh
   gh release create vX.Y.Z \
     dist/release/release.json dist/release/release.json.sig dist/release/nickeldrawer-X.Y.Z.zip \
     --title "NickelDrawer X.Y.Z" --notes-file <(sed -n '/^## \[X.Y.Z\]/,/^## \[/p' CHANGELOG.md | sed '$d')
   ```
8. Test the update on a device you control: **Apps · Check for updates**, then **Apps · Update status**. Check `appsctl update rollback` still works.

## Before the first public release

- No Kobo binaries, firmware files or personal data in the tree or the zip (`unzip -l dist/release/*.zip`).
- `SECURITY.md` names a contact; `LICENSE` and `third_party/README.md` are current.
- `docs/COMPATIBILITY.md` matches what was tested.

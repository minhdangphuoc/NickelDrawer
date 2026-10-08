# Signed updates

NickelDrawer can update its own apps and host tool over Wi-Fi. An update is only applied if it is signed with the maintainer's release key. Nothing touches the network unless you run an update from the menu.

## For users

1. Create `/mnt/onboard/.adds/koboapps/data/update.json` once (copy it from the computer):

   ```json
   {"repo": "owner/name", "channel": "stable"}
   ```

   `channel` is `stable` (default) or `beta`. Only `beta` accepts GitHub pre-releases.
2. Open the menu and tap **Apps · Check for updates**. It connects Wi-Fi and starts the update in the background (a menu command may run for under 10 seconds only).
3. Tap **Apps · Update status** a little later to see the result.
4. The menu is rebuilt by the update. If a new item is missing, tap **Apps · Refresh list**.

From a shell: `appsctl update check | apply | status | rollback`. `apply` takes `--allow-downgrade`; every subcommand takes `--onboard-root`, `--api-base`, `--tmp-dir`, `--locale`.

If `update.json` is missing, or this build has no release key, the update stops with a message that says what to fix. Nothing is changed.

## Trust model

- The release key pair is ed25519. The **private key** lives only on the maintainer's Mac (`~/.config/nickeldrawer/release-ed25519.key`, mode 600). It is never in the repo, never on the Kobo, never printed.
- The **public key** is `spec/release-ed25519.pub` (base64 of the raw 32 bytes). `appsctl` embeds it at build time. A placeholder or empty key means updates are refused.
- A release has three assets on GitHub: `release.json`, `release.json.sig` and `nickeldrawer-<version>.zip`. The signature is detached, base64 of the raw 64 bytes, over the exact bytes of `release.json`.
- `release.json` lists every file with its path, sha256, size and mode. The device checks the signature first, then every file of the zip against that list. A valid signature on a wrong file is still rejected.
- Updates can only write host `bin/`, `lib/`, `locales/` and each app's `app.json`, `bin/`, `locales/`. Any other path, including every `data/` folder, makes the manifest invalid, even when it is signed. App data (tokens, configs, PINs) is never read or written.
- Downgrades are refused unless you pass `--allow-downgrade`. `min_appsctl` in the manifest stops an old `appsctl` from applying a release it does not understand.
- The GitHub account is not trusted for integrity: only the signature is. TLS uses CA roots embedded in `appsctl`, because the Kobo may lack a system bundle.

If the private key is lost or leaked, generate a new one, ship a build that embeds the new public key by hand (USB install), and treat releases signed by the old key as untrusted.

## What an update does

1. Reads `update.json`, finds the latest release (`stable`: GitHub's latest non-pre-release; `beta`: newest non-draft).
2. Downloads `release.json` and its signature to `/tmp`, verifies the signature, parses and validates the manifest.
3. Same version as installed: nothing happens ("up to date"). Older: refused without `--allow-downgrade`.
4. Downloads the zip to `/tmp`, verifies every file's sha256 and size.
5. Stages the files that differ into `.adds/koboapps/data/update/staging/` (same storage as the targets, so the final step is a rename).
6. Copies the files it will replace into `.adds/koboapps/data/rollback/<old-version>/` and writes a journal. New files are recorded so a rollback deletes them. The two newest backups are kept.
7. Renames each staged file over its target. Replacing the running `appsctl` works because the rename leaves the running copy alone.
8. Regenerates the menu, then self-checks: no app became invalid, every app in the manifest has the listed version, and the new `appsctl --version` runs and reports the release version.
9. If anything from step 7 on fails, the backup is restored automatically, the menu is rebuilt again, and the status says the old version was restored. The last result is kept in `.adds/koboapps/data/update/status.json`.

## Rollback

`appsctl update rollback` restores the newest backup (the version before the last update), removes files the update added, rebuilds the menu and drops that backup. Run it again to go back one more step, while backups remain. App `data/` folders are not part of any backup and are not touched.

If the self-check fails and the automatic restore also fails, the message names the backup folder. Copy its `files/` tree over `/mnt/onboard` by hand, or reinstall from the computer with `tools/install-to-kobo.sh`.

## For the maintainer

One-time setup:

```sh
go run ./cmd/ndrelease keygen      # private key to ~/.config/nickeldrawer, public key to spec/ and the embedded copy
git add spec/release-ed25519.pub internal/release/release-ed25519.pub
```

`keygen` refuses to overwrite an existing key. Back the private key up somewhere safe and offline. Commit the public key and cut the first release only from a build that embeds it.

Publish a release:

```sh
make release VERSION=0.2.0         # check + build + dist + sign + verify; stops if the key is missing
gh release create v0.2.0 dist/release/release.json dist/release/release.json.sig \
    dist/release/nickeldrawer-0.2.0.zip --title "v0.2.0" --notes-file CHANGELOG-0.2.0.md
# beta: add --prerelease and set "channel": "beta" on the test device
```

`make release` builds `appsctl` with the version baked in (`appsctl --version`), signs `dist/onboard` and writes `dist/release/`. `build/ndrelease-host verify` re-checks the signature, every sha256 and the zip. The first install still goes through `tools/install-to-kobo.sh`; updates only work on a device that already has `appsctl` with the embedded key and an `update.json`.

Two files are mirrored because `go:embed` cannot read outside a package folder: `spec/release-ed25519.pub` to `internal/release/` and `host/locales/en.json` to `cmd/appsctl/embedded/`. `make sync-embedded` (part of `check` and `build`) refreshes them and a test fails if they drift.

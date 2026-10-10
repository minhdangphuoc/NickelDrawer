# Signed updates

NickelDrawer can update itself over Wi-Fi: the drawer's host tool and the Nickel mod. The project's own apps update separately, one by one, from a signed catalog (see [App updates](#app-updates)); they are never part of a drawer update. An update is only applied if it is signed with the maintainer's release key. Nothing touches the network unless you start a check or an update from the settings screen of the Apps drawer (the gear in its header).

## For users

1. Updates come from the project's GitHub repo by default. The installer (`tools/install-to-kobo.sh`) creates `/mnt/onboard/.adds/koboapps/data/update.json` from `.adds/koboapps/config/update.json`, which `make dist` generates from `host/config/project.json` (the one file that names the repo, maintainer and supported firmware, so a fork edits only that). The installer never overwrites an existing `data/update.json`, with one exception: if its `repo` is empty (older installers wrote it that way) the installer fills in the default repo and leaves `channel` alone. A repo you set yourself is kept. To follow a fork, edit the file:

   ```json
   {"repo": "owner/name", "channel": "stable"}
   ```

   `channel` is `stable` (default) or `beta`. Only `beta` accepts GitHub pre-releases. The repo also holds one release per app (`todo-v1.0.2`) and the catalog release `apps`; the drawer ignores them: it considers only tags that are `v` plus a version (`v1.0.0-beta.5`) and takes the highest one the channel allows. Versions follow Semantic Versioning, so `1.0.0-beta.2` is newer than `1.0.0-beta.1` and older than `1.0.0`. An empty `repo` switches updates off: the settings screen shows the source as "Off" and both buttons stay disabled.
2. Open More, Apps, then the gear (**⚙**) in the header. The **Updates** section shows the installed version, the new version (with a black dot, only when a check found one) and the source.
3. Tap **Check**. It connects Wi-Fi if needed, fetches and verifies the signed release index and the signed app catalog (two independent checks: one failing does not block the other) and records the results; nothing is installed. While it runs the Updates section says "Checking for updates…" and **Check** is disabled; the screen refreshes itself. When it ends, one status line stays until the next check: "Checked at 17:02 · You are on the latest version (1.0.0-beta.1)", "Checked at 17:02 · New version 1.0.0-beta.2 available", or "Could not check at 17:02: no Wi-Fi" (other reasons: cannot reach GitHub, GitHub is limiting requests, bad signature, no release on this channel, or a short message for anything else). When a newer release exists the gear in the header also gets a black dot, the **New version** row appears and the primary button reads **Update to <version>**; a short "What's new" text shows when the release carries notes.
4. Tap **Update to <version>** and confirm. The update runs in the background; the status line reads "Downloading and installing…", then "Installed 1.0.0-beta.2 — restart the Kobo to finish" or the reason it failed. Keep Wi-Fi on until it ends.
5. When the result says to restart the Kobo, restart it: the Nickel mod of the release is installed by the reboot (see "The Nickel mod" below).

From a shell: `appsctl update check | apply | status | rollback` for the drawer and `appsctl update apps | app <id> | rollback-app <id>` for apps. `check` writes `.adds/koboapps/data/update/available.json` (`available`, `version`, `current`, `notes`, `has_mod`, `checked`), which the drawer reads for the dot on the gear and for the screen; it is a cache, and `apply` verifies everything again. It also writes `.adds/koboapps/data/update/last-check.json` after every check, successful or not: `{"time": "<RFC 3339 UTC>", "result": "up_to_date" | "available" | "failed", "version": "…", "error_key": "no_network" | "unreachable" | "rate_limited" | "bad_signature" | "no_release_on_channel" | "other", "report": {…}}`. `version` is the installed version for `up_to_date` and the new one for `available`; `error_key` and (only for `other`) `report`, the message key and parameters of the failure, appear for `failed`. The settings screen renders it in the device language (the time in the device's time zone); an `available` record for a version that is installed by now is not shown. `no_network` means the connection itself failed (no Wi-Fi, no DNS); `unreachable` means GitHub answered with an error or not at all; `rate_limited` is HTTP 429 or 403 with an exhausted rate limit. `apply` takes `--allow-downgrade`; every subcommand takes `--onboard-root`, `--api-base`, `--tmp-dir`, `--locale`.

If `update.json` is missing or has an empty `repo`, or this build has no release key, the update stops with a message that says what to fix. Nothing is changed.

## App updates

The project's own apps (`apps/*`) install and update from the same screen, **Settings > Updates > Apps**. Design and rules: [spec/app-updates.md](spec/app-updates.md); how a maintainer publishes one: [RELEASING.md](RELEASING.md).

For users:

1. Open More, Apps, the gear, then the **Apps** row of the Updates section. The row's caption says "2 updates · 1 not installed" or "Up to date" (or "Not checked yet"); the gear in the drawer header shows the update dot when a drawer or app update waits.
2. Tap **Check** (here or on the Settings screen: the Settings button checks the drawer and the apps, each on its own). It fetches the signed catalog `apps.json` from the project's `apps` release and verifies it before reading it. Nothing is installed.
3. **Updates available** lists apps with a newer version ("0.3.0 → 0.4.0"). Tap one, confirm, and only that app updates: its files are replaced, its `data/` (settings, tokens, lists) is kept and no restart is needed. **Update all (n)** updates every app in that list. **Installed** shows the versions you have. **Not installed** lists the project's other apps; tap one and confirm to install it. Hidden apps (the UI kit) appear there only after you switch them on in Settings > Hidden apps.
4. While it runs the screen shows "Updating 1 / 2" and what it does ("Wallpaper · downloading and verifying") and the buttons are dimmed; keep Wi-Fi on. An app that is running is stopped first through its own stop action, or left alone with "app is running": close it, then try again.
5. If something is wrong the line under the header says so in a few words: no Wi-Fi, cannot reach GitHub, bad signature, old app list, damaged file, refused package, not a valid app, needs a newer drawer, not enough space, or "update failed, old version restored".

From a shell: `appsctl update apps` (check and list), `appsctl update apps --all`, `appsctl update app <id>`, `appsctl update rollback-app <id>`. The files are in `.adds/koboapps/data/update/`: `apps-catalog.json` (the verified catalog, a cache), `apps-sequence.json` (the newest catalog accepted), `apps-check.json` and `apps-job.json` (last check and last or running job), `app-rollback/<id>/` (the version before the last update, without `data/`).

How it stays safe:

- The catalog is signed with the same release key as drawer releases, over the exact bytes `nickeldrawer-apps-v1\n` + `apps.json`; a drawer signature is not a valid catalog signature and the reverse. A catalog with a lower `sequence` than the last one accepted is refused, and an app is never replaced by an equal or older version.
- Each zip must have exactly the signed size and sha256 before it is opened, then pass the zip rules (regular files only, only under `.adds/apps/<id>/`, never a `data/` folder, no `..` or links, size limits).
- The app is swapped by renames: its `data/` moves into the new folder first and is never copied or read. After a power cut, the next start (any app job, or opening the drawer) puts things right; a folder that holds `data/` is never deleted. The previous version is kept once (without `data/`), and `rollback-app` puts it back.

## Trust model

This page covers drawer updates; app updates are in the section above. `release.json` no longer lists app zips: the device gets the app list only from the signed `apps.json` catalog. The old `app_packages` field is still parsed, so a device accepts an older release that has it.


- The release key pair is ed25519. The **private key** lives only on the maintainer's Mac (`~/.config/nickeldrawer/release-ed25519.key`, mode 600). It is never in the repo, never on the Kobo, never printed.
- The **public key** is `spec/release-ed25519.pub` (base64 of the raw 32 bytes). `appsctl` embeds it at build time. A placeholder or empty key means updates are refused.
- A drawer release has three update assets on GitHub: `release.json`, `release.json.sig` and `nickeldrawer-<version>.zip` (the drawer only). Apps are published on their own releases. `release.json` may carry an optional plain-text `notes` summary (up to 600 characters; `ndrelease sign --notes-file`) that the settings screen shows. The signature is detached, base64 of the raw 64 bytes, over the exact bytes of `release.json`.
- `release.json` lists every drawer file with its path, sha256, size and mode. The device checks the signature first, then every file of the zip against that list. A valid signature on a wrong file is still rejected.
- Updates can only write host `bin/`, `lib/`, `locales/`, `icons/`, `screens/`, `config/` and the one file `mod/KoboRoot-nickeldrawer.tgz`. Any other path makes the manifest invalid, even when it is signed: every path under `.adds/apps/` (apps are installed and removed by hand, never by an update) and every `data/` folder. Installed apps and their data (tokens, configs, PINs) are never read, changed or deleted by an update or a rollback; a test (`TestDrawerUpdateNeverTouchesInstalledApps`) pins this.
- Downgrades are refused unless you pass `--allow-downgrade`. `min_appsctl` in the manifest stops an old `appsctl` from applying a release it does not understand.
- The GitHub account is not trusted for integrity: only the signature is. TLS uses CA roots embedded in `appsctl`, because the Kobo may lack a system bundle.

If the private key is lost or leaked, generate a new one, ship a build that embeds the new public key by hand (USB install), and treat releases signed by the old key as untrusted.

## What an update does

1. Reads `update.json`, lists the repo's releases and takes the highest drawer version (`v` + version tag; `stable`: no pre-releases; `beta`: pre-releases too; drafts, app releases and the `apps` release never count).
2. Downloads `release.json` and its signature to `/tmp`, verifies the signature, parses and validates the manifest.
3. Same version as installed: nothing happens ("up to date"). Older: refused without `--allow-downgrade`.
4. Downloads the zip to `/tmp`, verifies every file's sha256 and size.
5. Stages the files that differ into `.adds/koboapps/data/update/staging/` (same storage as the targets, so the final step is a rename).
6. Copies the files it will replace into `.adds/koboapps/data/rollback/<old-version>/` and writes a journal. New files are recorded so a rollback deletes them. The two newest backups are kept.
7. Renames each staged file over its target. Replacing the running `appsctl` works because the rename leaves the running copy alone.
8. Self-checks: no installed app became invalid, and the new `appsctl --version` runs and reports the release version.
9. If the release carries a changed mod, stages it (next section). The cached check result is dropped.
10. If anything from step 7 on fails, the backup is restored automatically and the status says the old version was restored. The last result is kept in `.adds/koboapps/data/update/status.json`.

## The Nickel mod

A release may include the mod as `.adds/koboapps/mod/KoboRoot-nickeldrawer.tgz` (built by the mod build and copied into `dist/` by `make dist` when `build/mod/KoboRoot-nickeldrawer.tgz` exists). It is an ordinary manifest file, so the signature and the sha256 cover it. The updater treats it specially:

1. Before changing anything it checks that `.kobo/KoboRoot.tgz` is free (or already identical): Nickel installs one archive per reboot, so a different waiting archive makes the update stop with "restart the Kobo, then update again".
2. It checks the archive itself: gzip + tar holding nothing but `usr/local/Kobo/imageformats/*.so` (and its parent folders). Anything else rejects the whole update.
3. Only when the archive differs from the copy recorded in `.adds/koboapps/mod/` (so an unchanged mod is not installed again), after the other files are swapped and the self-check passed, it copies the archive to `.kobo/KoboRoot.tgz`. The result is then "Updated X to Y. Restart the Kobo to finish." (`update.applied_reboot`).
4. Nickel installs the mod at the next boot and deletes `KoboRoot.tgz`. The recorded copy in `.adds/koboapps/mod/` stays, so the next update can tell whether the mod changed. `tools/install-to-kobo.sh` writes the same record when it stages the mod.

The mod itself is not replaced while Nickel runs. If the mod fails to load after the restart, NickelHook's failsafe (power off once within 15 s of the mod loading) uninstalls it.

## Rollback

`appsctl update rollback` restores the newest backup (the version before the last update), removes files the update added and drops that backup. A mod archive that was staged but not yet installed by a reboot is withdrawn. Run it again to go back one more step, while backups remain. App `data/` folders are not part of any backup and are not touched.

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

To show a changelog summary on the device, pass `build/ndrelease-host sign ... --notes-file notes.txt` (plain text, up to 600 characters). Build the mod first (`build/mod/KoboRoot-nickeldrawer.tgz`) so `make dist` includes it. `make release` builds `appsctl` with the version baked in (`appsctl --version`), signs `dist/onboard` (host + mod only) and writes `dist/release/` (the three update files). `build/ndrelease-host verify` re-checks the signature, every sha256 and the zip against the index. App releases are separate (`make release-app`, see [RELEASING.md](RELEASING.md)). The first install still goes through `tools/install-to-kobo.sh`; updates only work on a device that already has `appsctl` with the embedded key and an `update.json`.

Two files are mirrored because `go:embed` cannot read outside a package folder: `spec/release-ed25519.pub` to `internal/release/` and `host/locales/en.json` to `cmd/appsctl/embedded/`. `make sync-embedded` (part of `check` and `build`) refreshes them and a test fails if they drift.

# App updates

> **Status.** Phase 1 is implemented: the project's own apps (`apps/*` in this repo) are installed and updated over the air from a signed catalog. Phase 2, the per-app `update.json` of third-party apps described from [Declaring the source](#declaring-the-source) onward, is still a design: nothing in those sections exists in the current build and details may still change. How to publish an app with it: [publishing-apps.md](../guide/publishing-apps.md).

Over-the-air updates for apps, one app at a time. Drawer updates are separate and keep the format in [UPDATES.md](../UPDATES.md). The app folder itself is defined in [app-format.md](app-format.md).

## Phase 1: project apps (implemented)

The project's own apps are released one by one and listed in a signed catalog. The drawer installs, updates and rolls back from that catalog only.

### Release model

- Each app release is its own GitHub release in this repo with the tag `<id>-v<version>` (for example `todo-v1.0.2`). It holds one asset, `app-<id>-<version>.zip`, which carries only `.adds/apps/<id>/` (no `data/`, no previews).
- One fixed release, tagged `apps`, holds the **catalog**: `apps.json` and `apps.json.sig`. It is created once with `--latest=false` and its two assets are replaced (`gh release upload apps ... --clobber`) on every app release.
- The device downloads `https://github.com/<repo>/releases/download/apps/apps.json` (+ `.sig`) and each zip from `https://github.com/<repo>/releases/download/<tag>/<zip>`. Plain downloads, no API, no rate limit. `<repo>` is the `repo` of `data/update.json`, the same source as drawer updates. Apps have no channels.
- The drawer's own release selection ignores these releases: it considers only tags that are `v` plus a semantic version, on both channels ([UPDATES.md](../UPDATES.md)).

### apps.json

```json
{
  "v": 1,
  "sequence": 7,
  "issued": "2026-10-11T09:00:00Z",
  "apps": [
    {"id": "todo", "version": "1.0.2", "tag": "todo-v1.0.2", "zip": "app-todo-1.0.2.zip",
     "sha256": "9f2c...64 lowercase hex...", "size": 48213,
     "name": "To-do", "names": {"vi": "Việc cần làm"}, "hidden": false,
     "requires": {"drawer": ">=1.0.0"}}
  ]
}
```

| Field | Rules |
|---|---|
| `v` | must be `1` |
| `sequence` | integer of 1 or more; one more than the catalog it replaces |
| `issued` | RFC 3339 time |
| `apps[].tag` | exactly `<id>-v<version>` |
| `apps[].zip` | exactly `app-<id>-<version>.zip` |
| `apps[].sha256`, `size` | of the zip; size 1 to 33554432 (32 MiB) |
| `apps[].name`, `names` | English name, and translations by language code; at most 60 characters, no control characters |
| `apps[].hidden` | copied from `app.json`; see [the Not installed list](#device-screen) |
| `apps[].requires.drawer` | `>=X.Y.Z`, from the app's `min_host`; absent means any drawer |

The file is at most 64 KiB and lists at most 200 apps.

### Signature and anti-rollback

- The catalog is signed with the project release key (ed25519, the same key as `release.json`) over the exact bytes `nickeldrawer-apps-v1\n` + the raw bytes of `apps.json`. `apps.json.sig` is base64 of the raw 64 bytes. The prefix separates the two uses of the key: a drawer `release.json` signature (which has no prefix, so that releases such as 1.0.0-beta.5 keep updating) never verifies a catalog, and a catalog signature never verifies a drawer release.
- The device verifies the signature **before** it parses anything, and reads at most 64 KiB.
- Anti-rollback: the device stores the last accepted `sequence` (`data/update/apps-sequence.json`). A lower sequence is refused ("old app list"); an equal one is the same catalog. An app version is never installed over an equal or higher installed version, whatever the catalog says.
- The device checks that each zip has exactly the signed `size` and `sha256` before it opens it, and that the app inside has the signed id and version.
- Phase 2 will add a drawer `release.json` v2 signed with its own prefix (`nickeldrawer-release-v2\n`), so that no signature of the project key can ever be replayed across formats.

### The zip

The rules in [The zip](#the-zip) below apply, with the folder layout used by `make dist`: entries are `.adds/apps/<id>/...` so a manual unzip merges onto the Kobo root. The device accepts only regular files and folders, nothing outside `.adds/apps/<id>/`, nothing under any `data/`, no `..`, backslash or absolute path, no `preview-state*` file, NFC names without case-fold duplicates, at most 4096 entries and 64 MiB unpacked. Modes become `0755` (executables) or `0644`.

### Applying

Everything runs in a detached `appsctl update ...` job that writes `data/update/apps-job.json` (state, which app, which phase, how many); the screen reads it. There is no reboot, and the job asks for a drawer rescan at the end.

1. Recover any interrupted swap (below).
2. Download the catalog and verify it. Pick the apps (`update app <id>`: one; `update apps --all`: every installed app with a newer version).
3. For each app: refuse an equal or older target; check `requires.drawer`; if the app is installed and running, stop it through its `stop` action (the `status` action says if it runs) or fail with "app busy".
4. Download the zip to the temp folder, check size then sha256, check the zip rules and that the free space is enough, unpack into the staging folder `.adds/apps/.<id>.new`.
5. Move `<id>/data` into the staging folder, rename `<id>` to `.<id>.old`, rename the staging folder to `<id>`. Each is a rename on one file system, so `data/` is moved, never copied or read.
6. Validate the installed folder with the manifest loader (and its version). On failure, undo: data goes back to the old folder, which is renamed back; the error says the old version was restored.
7. On success the old folder (it holds no `data/`) becomes the app's one rollback copy in `data/update/app-rollback/<id>/`, replacing the earlier one.

The staging and old folders start with a dot (`.<id>.new`, `.<id>.old`), so the app scanner never lists them, even after a crash. A failure on one app of `--all` does not stop the others; the result counts both.

`appsctl update rollback-app <id>` swaps the rollback copy in with the same steps (data stays in place) and consumes it.

### Recovery after power loss

On every app job and every time the drawer model is built (opening the drawer), each app with a leftover `.<id>.old` or `.<id>.new` is recovered: a missing `<id>` is restored from `.<id>.old`; any `data/` in a leftover folder is moved into `<id>`; a `<id>` that does not load is replaced by the old folder (or the rollback copy), data moving along; leftovers are removed only when they hold no `data/`. A folder that holds `data/` is never deleted: if two folders hold one, both stay and the problem is logged. Tests cut the power after each step of the swap and require the data byte for byte.

### Device screen

Settings > Updates has an **Apps** row (caption "2 updates · 1 not installed", or "Up to date") that opens the **Updates for apps** screen. The gear in the drawer header gets the update dot when an app update waits, too.

- Header status "2 updates". Sections **Updates available** (rows: name and `0.3.0 → 0.4.0`; a tap asks "Update <name>?" and says the data and settings are kept and no restart is needed), **Installed** (name and version, not tappable) and **Not installed** (tap asks "Install <name>?"). Hidden catalog apps (for example the UI kit) are offered for install only when the user switched them on in Settings > Hidden apps; installed ones are listed normally.
- Footer: primary **Update all (n)** and secondary **Check** when n > 0, else **Check** is the primary. While a job runs: a steps block ("Updating 1 / 2", detail "Wallpaper · downloading and verifying") and disabled buttons.
- The Settings **Check** button runs the drawer check and the catalog check independently: one failing does not block the other.
- Errors are short localized lines: no Wi-Fi, cannot reach GitHub, bad signature, old app list, damaged file, refused package, not a valid app, needs a newer drawer, app is running, not enough space, update failed and old version restored.

### Command line

`appsctl update apps` (check, then list every catalog app), `update apps --all`, `update app <id>` (install or update), `update rollback-app <id>`; flags as for the other update commands, plus `--download-base` for tests.

### Maintainer

`ndrelease sign-apps --apps-dir dist/apps --sequence N [--only <id>...]` writes `dist/apps-release/{apps.json,apps.json.sig}` for every zip in `dist/apps` (the tags come from the id and version) and copies the `--only` zips beside them for upload; `ndrelease verify-apps` re-checks it. The steps are in [RELEASING.md](../RELEASING.md).

### Not in phase 1

Third-party apps, per-app signing keys, mirrors and channels for apps, background checks and delta updates. The drawer `release.json` no longer lists app zips (`app_packages`); the field is still parsed for old releases.

## Phase 2: third-party apps (design, not implemented)

The rest of this page. The catalog above stays the only source for project apps; phase 2 adds each app's own `update.json` for apps that are not in it.

## Goals and limits

- An app author publishes three files. The device downloads them, checks them and swaps the app folder. No reboot.
- Every file is verified on the device. Hosts, mirrors and gateways are untrusted transport.
- Out of scope: catalogs added by users, background checks, delta updates, unsigned or dev-mode updates.

## Threat model

- Protected against: a compromised release host, a hostile mirror or gateway, and a network attacker. They cannot make the device install a file the author did not sign, and they cannot roll an app back.
- Not protected against: an app that is already installed. Apps run as root on the Kobo, so an installed app can edit any file, including the pinned keys. Only install apps you trust.
- Not fully preventable: a freeze attack, where a mirror withholds new releases. See [Anti-rollback and freshness](#anti-rollback-and-freshness).
- Residual risk: the original source itself (the release host account or web server) is trusted for freshness. If it is compromised and also holds an old signed `update.json` and `current-key.json`, it can serve stale but validly signed files; anti-rollback and the sequence numbers limit this to a freeze, never a downgrade or a revived old key.
- A leaked signing key lets the thief publish updates until users switch to a new key. Only a recovery key makes that possible over the air; see [Key rotation](#key-rotation).

## Declaring the source

In `app.json`, one of two forms:

```json
"update": {
  "github": "alice/kobo-weather",
  "key": "ed25519:kP3b...43 chars of base64...=",
  "recovery_key": "ed25519:Q9xT...43 chars of base64...="
}
```

```json
"update": { "url": "https://example.org/weather/", "key": "ed25519:kP3b...=" }
```

| Field | Rules |
|---|---|
| `github` | `owner/repo`, `^[A-Za-z0-9._-]{1,100}/[A-Za-z0-9._-]{1,100}$`. Exactly one of `github` and `url` |
| `url` | base URL ending in `/` (added if missing). `https://`; plain `http://` only when the host is a private-range IP address (10.x, 172.16-31.x, 192.168.x) or a `.local` name, for a NAS on the home network. Any other `http://` is invalid |
| `key` | required. The signing key |
| `recovery_key` | optional but strongly recommended. Kept offline; used only to rotate the signing key |

Key format for both: `ed25519:` plus standard base64 of the **raw 32-byte** public key (44 characters with padding). Not PKIX or DER. Any other length is invalid.

An app without `update` is never checked. The `channel` (`stable` or `beta`) is a device setting, default `stable`. It is not in `app.json`.

### Fingerprint

The fingerprint of a key is the first 16 hex characters of the sha256 of the raw 32 key bytes, in groups of four: `3fa9-02bc-77de-1a40`. The app Info dialog shows it. Developers print it in the README so a user can compare it after a manual install.

## Developer tools

| Command | What it does |
|---|---|
| `appsctl app keygen <id> [--recovery] [--new] [--key-dir DIR]` | writes a private key and prints the `app.json` line and the fingerprint |
| `appsctl app pack <app-dir> --out dist` | validates the app, scans for private keys, builds the zip and `update.json` |
| `appsctl app sign <dist-dir> [--key FILE] [--key-dir DIR]` | writes `update.json.sig` |
| `appsctl app rotate <id> [--recovery-key FILE] [--key-dir DIR] --sequence N` | adds `key_rotation` to `update.json` and writes `current-key.json` and its signature |
| `appsctl app verify <dist-dir>` | re-checks signature, hash, size, zip rules and the private-key scan |

Key files are looked up as `<id>.key` and `<id>.recovery.key`. The default folder is `~/.config/nickeldrawer/apps/` (mode 600 files). `--key-dir DIR` on `keygen` writes there instead, and on `sign` and `rotate` looks there first, then in the default folder. Use it for a USB stick, a password-manager export folder or an encrypted volume. The location is the developer's choice and responsibility; the only rule is that `keygen` refuses a folder inside a git work tree or inside an app folder.

## Private apps

The source repository of an app may stay private. Only the three release files must be reachable by the device, on a public or LAN host (use the `url` source). A private GitHub repository read directly by the device is **not supported**: the device would need an access token. That token would sit on a device where every app runs as root, it expires, and it grants access to the source code.

## Release files

Three assets per release:

| File | Content |
|---|---|
| `app-<id>-<version>.zip` | the app folder, rooted at `<id>/` |
| `update.json` | the manifest below |
| `update.json.sig` | detached signature: base64 of the raw 64 bytes (see [Signature](#signature)) |

### update.json

```json
{
  "v": 1,
  "id": "weather",
  "version": "1.2.0",
  "zip": "app-weather-1.2.0.zip",
  "sha256": "9f2c...64 lowercase hex chars...",
  "size": 48213,
  "requires": { "drawer": ">=1.0.0" },
  "notes": "Adds a 7-day forecast. Fixes the crash when Wi-Fi is off."
}
```

| Field | Rules |
|---|---|
| `v` | must be `1`. A higher value is an unknown format: `bad_manifest` |
| `id` | must equal the installed app's id |
| `version` | semver `MAJOR.MINOR.PATCH`; must equal `app.json` `version` inside the zip |
| `zip` | file name only: `^app-<id>-<version>\.zip$`. No `/` |
| `sha256` | 64 lowercase hex chars, of the zip file |
| `size` | zip size in bytes, 1 to 33554432 (32 MiB). The downloaded zip must have exactly this size |
| `requires.drawer` | one constraint `>=X.Y.Z`; default `>=1.0.0`. `appsctl app pack` sets it from `min_host` when the app has one |
| `notes` | optional plain text, at most 600 characters. Control characters (except line feed) are stripped; it is never rendered as markup |
| `key_rotation` | optional, see [Key rotation](#key-rotation) |

`update.json` is at most 16 KiB. Unknown fields are ignored.

### Signature

The signature is Ed25519 over the exact bytes:

```
nickeldrawer-app-update-v1\n  +  <raw bytes of update.json>
```

The prefix is ASCII, `\n` is one line feed. Today the drawer's own `release.json` is signed over its raw bytes with no prefix, and the project catalog (phase 1) is signed over `nickeldrawer-apps-v1\n` + its bytes. Phase 2 introduces drawer domain separation: `release.json` v2 is signed over `nickeldrawer-release-v2\n` + the bytes, and a device accepts an unprefixed signature only for `release.json` schema v1. This is required before the project key signs any third-party app update, so that a drawer signature can never be replayed as an app update or the other way round. The device verifies the signature **before** it parses the JSON, and refuses a file over 16 KiB before reading more.

### The zip

Normative rules; the device and `appsctl app verify` apply them:

- Regular files and directories only. Symlinks, hardlinks and devices are rejected.
- All entries are under `<id>/`. No absolute path, no `..`, no backslash, nothing under `<id>/data/`, no preview-state files.
- Names are Unicode NFC. Two names that are equal after case folding are a duplicate and rejected.
- At most 4096 entries, 64 MiB uncompressed in total, 32 MiB download.
- File modes are masked to `0755` (executables) or `0644`. Setuid, setgid and sticky bits are dropped.
- The sha256 of the downloaded file is checked **before** the zip is opened.
- `appsctl validate` must pass on the unpacked folder.

## URL resolution

`<base>` is the `url` value (with the trailing `/`). Mirrors are applied after this step. Sources use HTTPS (the `url` private-host exception aside).

| Source | `update.json` | `update.json.sig` | zip |
|---|---|---|---|
| `github`, stable | `https://github.com/<repo>/releases/latest/download/update.json` | `.../update.json.sig` | `.../<zip>` |
| `github`, beta | the newest prerelease from `https://api.github.com/repos/<repo>/releases?per_page=10`, then its assets | its assets | its assets |
| `url` | `<base>update.json` | `<base>update.json.sig` | `<base><zip>` |

- Stable uses the `releases/latest/download` redirect, not the GitHub API, so there is no rate limit. GitHub marks a release "latest" only when it is not a prerelease or draft.
- Beta uses the API. Unauthenticated calls are limited to 60 requests per hour per IP address. A limit answer is the `rate_limited` error. If the repo has no prerelease, beta falls back to stable.
- The zip name comes from the verified `update.json`, never from the unverified API answer.
- The `url` source works with any static host: GitHub Pages, Codeberg or Gitea (through a fixed path), your own server, a NAS. It has no channels: whatever `update.json` is there is the release.

## Mirrors

Optional list in `.adds/koboapps/config/update.json`:

```json
{ "mirrors": ["https://ghproxy.example/", "http://my-nas.local/kobo-mirror/"] }
```

For app updates, mirrors serve **only the zip**. The device fetches `update.json`, `update.json.sig`, `current-key.json` and `current-key.json.sig` from the original source, never from a mirror (see [Key rotation](#key-rotation)). The zip is tried on the mirrors in order, then on the original URL; its signed `sha256` and `size` decide if it is accepted. For drawer updates, mirrors apply to all files as before, since the release is signed as a whole.

Rewrite rule: **mirror URL = prefix + original URL without its scheme.** The prefix must end in `/` (added if missing).

| Original | Prefix | Mirror URL |
|---|---|---|
| `https://github.com/alice/kobo-weather/releases/latest/download/update.json` | `https://my-nas.local/kobo-mirror/` | `https://my-nas.local/kobo-mirror/github.com/alice/kobo-weather/releases/latest/download/update.json` |
| `https://example.org/weather/app-weather-1.2.0.zip` | `https://ghproxy.example/` | `https://ghproxy.example/example.org/weather/app-weather-1.2.0.zip` |

- Mirrors may use plain HTTP, because every file is signed and checked. A `github` source is always HTTPS; a `url` source may use HTTP only under the private-host rule above.
- Query strings are kept; a static mirror ignores them.
- The device follows at most 5 redirects per request.
- Privacy: a mirror sees which apps a device has and when it checks. It can also withhold updates. Use only mirrors you run or trust with that.
- A mirror cannot forge a release. For apps it can serve a bad zip (rejected by the hash) or nothing. Because it never serves `update.json`, it cannot serve an old version or a stale rotation either.

### Running a simple mirror

Any web server that serves a folder works. Layout: `<root>/<original host>/<original path>`. A cron job downloads the three files of each app's latest release:

```sh
#!/bin/sh
# Run from cron every hour. ROOT is the folder your web server publishes at /kobo-mirror/.
ROOT=/srv/www/kobo-mirror
REPO=alice/kobo-weather
DIR="$ROOT/github.com/$REPO/releases/latest/download"
mkdir -p "$DIR"
curl -fsSL -o "$DIR/update.json.new" "https://github.com/$REPO/releases/latest/download/update.json" || exit 1
ZIP=$(sed -n 's/.*"zip" *: *"\([^"]*\)".*/\1/p' "$DIR/update.json.new")
curl -fsSL -o "$DIR/$ZIP" "https://github.com/$REPO/releases/latest/download/$ZIP" || exit 1
curl -fsSL -o "$DIR/update.json.sig" "https://github.com/$REPO/releases/latest/download/update.json.sig" || exit 1
mv "$DIR/update.json.new" "$DIR/update.json"
```

Only the zip is used from a mirror, so the script may skip the other files; they are kept here so the folder is also a complete copy for backup. Old zips can be deleted after a day.

## Trust and pinning

- The signature is mandatory. There is no unsigned mode.
- Pins live **outside** the app folder, in `.adds/koboapps/data/app-keys.json`:

  ```json
  { "weather": { "signing_key": "ed25519:...", "recovery_key": "ed25519:...", "pinned_at": "2026-11-02T10:00:00Z", "source": "usb", "rotation_sequence": 0, "last_seen_version": "1.2.0" } }
  ```

  `recovery_key` is absent when the app has none. `source` is `usb` or `catalog`.
- A pin is recorded at the first install or the first check, from the installed `app.json` (trust on first use, over USB). Project catalog apps are pinned from the signed `apps.json` catalog instead: no trust on first use.
- An update's own `app.json` `update.key` and `update.recovery_key` must equal the pinned ones, or the update is rejected (`key_changed`). The recovery key can never change over the air.
- The key inside a downloaded zip never verifies that same zip.
- The project's own apps are signed with the project release key and listed in the signed `apps.json` catalog (phase 1). That catalog feeds the **Not installed** list, which is curated: only project apps. Third-party apps are installed by copying the folder, then update from their own source.
- An author must never reuse the drawer's release key for an app.

### Key rotation

Rotation by the current signing key would add no protection against a leaked key, since the thief could rotate to their own. So rotation is signed by the **recovery key**, which stays offline.

`update.json` carries the rotation statement for the update that switches keys:

```json
"key_rotation": {
  "new_key": "ed25519:<base64 raw public key>",
  "sequence": 1,
  "signature": "<base64 signature by the RECOVERY key>"
}
```

The signature is Ed25519, made with the recovery key, over these exact bytes (UTF-8, each `\n` a line feed, final line feed included):

```
nickeldrawer-key-rotation-v1\n<id>\n<new_signing_key>\n<sequence>\n
```

`<id>` is the app id, `<new_signing_key>` the full `ed25519:...` string, `<sequence>` a decimal integer without leading zeros.

Rules:

1. The app must have a pinned recovery key. Without one, rotation is impossible.
2. `key_rotation` is parsed from bytes that are not verified yet, so the parser applies strict limits first: `update.json` at most 16 KiB, exactly the three fields above, `new_key` a valid 44-character key, `signature` valid base64 of 64 bytes, `sequence` an integer below 2^31. It is **accepted only after the recovery signature verifies**; until then its content is not used for anything else.
3. `sequence` must be greater than the stored `rotation_sequence`. Older rotations are rejected.
4. `update.json.sig` is verified with the **new** signing key, and the rotation `signature` with the pinned recovery key.
5. Both must pass. Then the update is applied, the new key and sequence are pinned, and the zip's `app.json` must carry the same `update.key`.
6. The old signing key is rejected forever after.
7. On any failure the update is rejected with `key_changed` and nothing changes.
8. `update.json` and its rotation come from the original source only, never from a mirror.

### Current-key statement

A device that missed later rotations could otherwise be fed an old, validly signed rotation to a key that has since leaked. To stop that, an app that has ever rotated publishes a **current-key statement**: `current-key.json` and `current-key.json.sig` next to `update.json` at the original source (same URL rules).

```json
{ "v": 1, "id": "weather", "key": "ed25519:<current signing key>", "sequence": 2, "issued": "2026-11-02T10:00:00Z" }
```

The signature is by the **recovery key** over the bytes `nickeldrawer-current-key-v1\n<id>\n<key>\n<sequence>\n<issued>\n` (same conventions as above), checked before parsing, file at most 1 KiB.

On each check the device fetches it from the original source (a 404 means the app never rotated). If it verifies:

- `sequence` below the stored `rotation_sequence`: ignored as stale.
- A `key_rotation` in `update.json` is accepted only when its `sequence` equals the statement's `sequence` and its `new_key` equals the statement's `key`. An older rotation to a leaked key is therefore rejected, even though its signature is valid.
- If the statement's key differs from the pinned signing key, the device pins it only together with a verified update signed by that key (the rotation rules above still apply).

If an app has no recovery key, a lost or leaked signing key cannot be fixed over the air: users reinstall over USB. Every developer should create a recovery key (`appsctl app keygen --recovery`) and keep it offline.

## Anti-rollback and freshness

- An update is offered only if its `version` is greater than the installed version (semver compare). An old, validly signed `update.json` gives "already up to date", never a downgrade.
- The device records `last_seen_version` per app (in the pin file) after each verified check.
- Freeze attacks by a mirror do not apply to app manifests: `update.json` always comes from the original source. A hostile or stale original source can still withhold new releases; that cannot be prevented. The App updates screen shows which host answered.

## Key-leak safeguards

Only the public key is inside a zip, and an Ed25519 private key cannot be derived from a public key or from signatures. So holding a release does not reveal the signing key. The real risk is a developer slip: the key in the app folder, committed to git, printed in a CI log or pasted somewhere. The tools guard against the common slips:

- `appsctl app pack` refuses to build if any file in the app looks like a private key. By content: the private-key file format `keygen` writes, PEM `PRIVATE KEY` blocks, OpenSSH private keys. By name: `*.key`, `*.pem`, `id_ed25519*`. The error names the file.
- `appsctl app verify` runs the same scan on a built zip.
- `appsctl app keygen` refuses to write inside a git work tree or inside an app folder. It prints a `.gitignore` suggestion (`*.key`, `*.pem`, `id_ed25519*`) and offers a passphrase.

## Verification order

The device stops at the first failure.

1. From the original source, download `update.json` (stop at 16 KiB), `update.json.sig`, and `current-key.json` with its signature if present.
2. Choose the key: the pinned signing key, or the `new_key` of a `key_rotation` that passed the rules above (recovery signature and sequence, and the current-key statement when one exists). The rotation is read from the unverified bytes only under the strict limits and trusted only after the recovery signature verifies.
3. Verify the signature over `"nickeldrawer-app-update-v1\n"` + the exact bytes. Only then parse the JSON.
4. Check `v`, the field rules, and `id` equals the installed id.
5. Anti-rollback: `version` must be greater than the installed version. Check `requires.drawer`.
6. Download the zip (mirrors allowed), stopping at 32 MiB or `size`. Its size must equal `size`.
7. Check `sha256` of the file.
8. Open the zip and check every entry against the zip rules. `app.json` inside must have the same `id`, `version` and keys as the pins.
9. Unpack to a temporary folder next to the app (same file system) and run `appsctl validate` on it.

## Applying an update

1. If the app is running, ask the user to stop it. Declined: nothing changes (`app_running`).
2. Copy the old folder without `data/` to the host's rollback folder (`.adds/koboapps/data/update/rollback/<id>/`).
3. Move the old `<id>/data` into the **staged** new folder (a rename on one file system). If there is no `data/`, skip this step.
4. Rename `<id>` to `<id>.old`, then rename the staged folder to `<id>`.
5. Validate the installed app, then update the pin (`last_seen_version`, and key and sequence after a rotation). On any failure in steps 3 to 5, put `data/` back into the old folder, restore the old folder as `<id>` and report `rolled_back`.
6. Remove `<id>.old` (it holds no `data/` any more). Keep the rollback copy until the next successful update of that app.

`data/` is moved, never copied, read, changed or deleted by the update. Because it moves into the staged folder first, a crash at any point leaves it inside exactly one of `<id>`, `<id>.old` or the staged folder.

### Recovery after power loss

At the next drawer start, for each app with a leftover `<id>.old` or staged folder:

1. **Never delete a folder that contains `data/`.**
2. If `<id>` exists and `<id>/data` is missing, but `<id>.old/data` or the staged folder's `data/` exists, move that `data/` into `<id>` first.
3. If `<id>` is missing and `<id>.old` exists, rename `<id>.old` back to `<id>` (restore).
4. Only when no leftover folder contains `data/` and `<id>` validates, remove `<id>.old` and any staged folder. If `<id>` fails validation, restore from `<id>.old` or the rollback copy.

The crash cases to test: after step 2, after step 3, between the two renames of step 4, after step 4, and during step 5 and step 6.

## Device behaviour

- No background checks. One **Check** in Settings › Updates checks the drawer and every installed app that has an `update` source.
- Failures are isolated per app.
- Results are cached in `.adds/koboapps/data/update/apps.json` (a cache; apply verifies everything again).
- Screen: Settings › Updates has an **Apps** row. It opens the App updates screen with three sections, **Updates**, **Installed** and **Not installed**. **Update all (n)** is the primary button, **Check** the secondary one. It shows which host answered.
- Apps with no `update` source appear in **Installed** with no update.

## Errors

Each error has a message key in `host/locales/<lang>.json` and shows as a short per-app status line. Texts come from locale files, never from code.

| Error | Message key | When |
|---|---|---|
| no network | `appupdate.error.no_network` | no Wi-Fi, DNS failure |
| unreachable | `appupdate.error.unreachable` | host answers with an error, or all mirrors and the original fail |
| rate limited | `appupdate.error.rate_limited` | HTTP 429, or 403 with an exhausted limit (beta only) |
| no release | `appupdate.error.no_release` | no release, or no `update.json` asset |
| bad manifest | `appupdate.error.bad_manifest` | JSON invalid, field rules broken, unknown `v`, wrong `id` |
| bad signature | `appupdate.error.bad_signature` | signature missing or does not match |
| key changed | `appupdate.error.key_changed` | key differs from the pin without a valid rotation; rotation replayed or signed by the wrong key |
| hash mismatch | `appupdate.error.hash_mismatch` | zip `sha256` or `size` differs |
| unsafe zip | `appupdate.error.unsafe_zip` | any zip rule broken |
| invalid app | `appupdate.error.invalid_app` | `appsctl validate` fails on the new folder |
| needs newer drawer | `appupdate.error.needs_newer_drawer` | `requires.drawer` is not met |
| app running | `appupdate.error.app_running` | the user declined to stop the app |
| rolled back | `appupdate.error.rolled_back` | the swap or final validation failed; the old version is back |
| disk full | `appupdate.error.no_space` | not enough free space for the download and the rollback copy |

An old version is not an error: the status is `appupdate.status.up_to_date`. Other keys: `appupdate.status.available` (`{version}`), `appupdate.status.updated` (`{version}`), `appupdate.status.no_source`.

## Tests

Unit tests cover: manifest rules; signature with the right prefix and with the drawer's prefix (must fail); rotation (right key, wrong key, wrong id, replayed sequence, old signing key afterwards, an old valid rotation after a newer current-key statement, rotation fields outside the strict limits, a rotation or `update.json` served by a mirror is never used); the swap and power loss, cutting power after every numbered step of Applying an update (see below); anti-rollback; URL resolution; mirror rewriting; every zip rule; size limits; swap and rollback with a simulated failure at each step; after every simulated crash the next start must leave `<id>/data` intact. Device cases go in `docs/plan-detailed.md` §9 when the feature is built.

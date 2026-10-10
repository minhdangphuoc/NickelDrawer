# Publishing apps with updates

> **Status: planned for 1.1 — not implemented yet.** The commands and screens below describe the approved design. `appsctl app ...` does not exist in the current build. The format is in [app-updates.md](../spec/app-updates.md); when the two disagree, the spec wins.

How to take a working app to a signed release on GitHub, and how to ship updates that Kobo owners get from the drawer. Build the app first with the [developer guide](developer-guide.md).

You need: a finished app folder, a GitHub account (or any static web host), and a computer with `appsctl`.

## How it works in one minute

- Your `app.json` says where updates come from and which public key signs them.
- A release is three files: a zip of the app, `update.json` (version, hash, size) and `update.json.sig` (your signature).
- On the Kobo, **Settings › Updates › Check** fetches those files, verifies the signature and hash, and swaps the app folder. Your users' `data/` is never touched.
- The device remembers your key from the first install. An update cannot change it.

## 1. Create your keys

```sh
appsctl app keygen weather               # signing key
appsctl app keygen weather --recovery    # recovery key, kept offline
```

Each command writes a private key to `~/.config/nickeldrawer/apps/` (mode 600), or to the folder you give with `--key-dir DIR` (a USB stick, a password-manager export folder, an encrypted volume; `sign` and `rotate` accept the same option and look there first). Where you keep it is your choice and your responsibility. The only limit is that `keygen` refuses a folder inside a git work tree or an app folder. Each command and prints the line for `app.json`. Both offer an optional passphrase: use one. Back up both private keys. Move the recovery key to offline storage, such as a USB stick in a drawer, and do not keep it on the machine you publish from. If you lose a private key, no one can recover it for you.

Never reuse the drawer's own release key for an app.

`keygen` refuses to write inside a git work tree or an app folder, and prints a `.gitignore` suggestion (`*.key`, `*.pem`, `id_ed25519*`). `pack` refuses to build, and `verify` refuses a zip, if any file looks like a private key (by name or by content).

**Why a release cannot leak your key.** The zip holds only the public key, and an Ed25519 private key cannot be derived from a public key or from signatures. The real risks are your own slips: the key in the app folder, committed to git, printed in a CI log or pasted into a chat. If that happens, treat the signing key as leaked: remove it from every place, then rotate with the recovery key ([Rotating a key](#rotating-a-key)) and publish the new release at once. Without a recovery key, users must reinstall over USB.

## 2. Declare the source in app.json

```json
"update": {
  "github": "alice/kobo-weather",
  "key": "ed25519:kP3b...=",
  "recovery_key": "ed25519:Q9xT...="
}
```

For a different host use `"url": "https://example.org/weather/"` instead of `github` (see [Hosting elsewhere](#hosting-elsewhere)). Keys are `ed25519:` plus base64 of the raw 32-byte public key, exactly what `keygen` prints.

Print the key fingerprint in your README, for example `Signing key fingerprint: 3fa9-02bc-77de-1a40`. The app Info dialog shows the same fingerprint, so a user can compare after a manual install. `keygen` prints it.

Run `appsctl validate .` after the edit.

## 3. First release

1. Set `version` in `app.json` (for example `1.0.0`) and run the checks from [Testing](developer-guide.md#testing).
2. Pack:

   ```sh
   appsctl app pack ./weather --out dist
   ```

   It validates the app, then writes `dist/app-weather-1.0.0.zip` and `dist/update.json`. The zip has the app folder contents under `weather/`, without `data/`. Check `notes` in `update.json`: plain text, at most 600 characters.
3. Sign on your machine:

   ```sh
   appsctl app sign dist --key ~/.config/nickeldrawer/apps/weather.key
   appsctl app verify dist
   ```

   `sign` writes `dist/update.json.sig`. `verify` re-checks the signature, hash, size and every zip rule.
4. Create a GitHub release with tag `v1.0.0` and attach the three files. Do not mark it as a prerelease: stable users get the release GitHub calls "latest".
5. Tell your users to copy the `weather` folder into `.adds/apps/` over USB once. After that it updates itself.

## 4. Shipping an update

1. Change the code. **Bump `version`**: the device ignores a version that is not higher than the installed one.
2. `appsctl app pack`, `sign`, `verify` as above.
3. New GitHub release `v1.1.0` with the three files.
4. Users tap **Check** in Settings › Updates, then **Update all** or the app's row.

Rules that keep users safe:

- Never remove a version's files from a release you want users to still install.
- Keep the `data/` format compatible, or migrate it at start ([Versioning](developer-guide.md#versioning-and-min_host)).
- Set `min_host` when you need a newer drawer; `pack` copies it to `requires.drawer`.

## Using GitHub Actions

Copy `.github/workflows/release.yml` from the app template (`appsctl new` creates it). On a `v*` tag it runs `appsctl app pack` and uploads `dist/` as a workflow artifact. It does **not** sign. Download the artifact, then `appsctl app sign` and `verify` on your machine, then attach the files to the release.

Optional signing in CI is possible but discouraged. If you do it:

- Put only the **signing** key in a GitHub environment secret, never the recovery key.
- Use a protected environment with required reviewers, so every signing needs your approval.
- Know the trade-off: anyone who can push a tag and approve, or who breaks into your GitHub account, can sign an update. A key on your own machine does not have that exposure. The template has a commented job to start from.

## Beta channel

Mark a release as a **prerelease** on GitHub. Users who choose the beta channel in the update settings get the newest prerelease; everyone else keeps getting stable. Beta checks use the GitHub API, which allows 60 requests per hour per IP address without login, so a busy network can see "GitHub is limiting requests". Stable has no such limit. Promote a beta by publishing the same files as a normal release with a higher version.

## Hosting elsewhere

Use `"url"` instead of `"github"`:

```json
"update": { "url": "https://example.org/weather/", "key": "ed25519:kP3b...=" }
```

Put `update.json`, `update.json.sig` and the zip in that folder. HTTPS is required (a LAN host is the only exception, see [Private apps](#private-apps)). Any static host works: GitHub Pages, Codeberg or Gitea, your server, a NAS. Upload the zip first and `update.json` last. There are no channels for `url`.

## Private apps

Your source repository can stay private. Only the three release files must be reachable. Use the `url` source and publish the files to a public host:

1. Pick the host: a small separate public repo (use its releases or GitHub Pages), R2, S3, Netlify or your own server.
2. In the private repo, add a GitHub Actions job that runs after `pack` and `sign` and uploads `update.json`, `update.json.sig` and the zip to that host. Upload the zip first and `update.json` last. Store the host's deploy token as an environment secret and keep the signing key out of CI (see above).
3. Set `"url"` in `app.json` to the public folder.

For a personal app, run a small web server on a home NAS and use `"url": "http://my-nas.local/weather/"`. Plain `http://` is accepted for a `url` source only when the host is a private-range IP address or a `.local` name. Everything else must be HTTPS. (Mirrors may always use HTTP, because everything is signed.)

A private GitHub repository read directly by the device is not supported. The device would need a token. That token would live on a device where every app runs as root, it expires, and it grants access to your source code.

## Mirrors

Mirrors help where GitHub is slow or blocked. The owner of a Kobo adds them in `.adds/koboapps/config/update.json`:

```json
{ "mirrors": ["https://ghproxy.example/", "http://my-nas.local/kobo-mirror/"] }
```

The device takes only the zip from a mirror (tried in order, then the original address); `update.json` always comes from your own source. A mirror URL is the prefix plus the original URL without `https://`. Mirrors cannot forge anything, because the device checks the zip against the signed hash, but a mirror sees which apps a device has and can hold back the zip. Full rules and a cron script for a home mirror: [app-updates.md](../spec/app-updates.md#mirrors).

You do not set mirrors in `app.json`.

## Rotating a key

Rotation needs the **recovery key**. A leak of the signing key is the case it exists for.

1. Create a new signing key: `appsctl app keygen weather --new` (the old file is kept as `weather.key.old`).
2. Sign a rotation with the recovery key: `appsctl app rotate weather --recovery-key <file> --sequence 1`. It adds `key_rotation` to the next `update.json`. Use a higher `sequence` each time.
3. Put the new public key in `update.key` in `app.json`, then pack and sign the release with the **new** key.
4. Keep `key_rotation` in the next releases too. `rotate` also writes `current-key.json` and `current-key.json.sig`: upload them next to `update.json`, and again after every later rotation. The device fetches them from your source on each check, so an old rotation to a key that has since leaked is rejected.

After the rotation, the old signing key is rejected forever. If you never made a recovery key, you cannot rotate: tell users to reinstall the app over USB with a new key.

## License

You choose your app's license, commercial included. NickelDrawer's license does not apply to apps that only use its app format, protocol and tools, and using `appsctl` to build, sign and publish your app is allowed for any purpose. See [LICENSE-APPS.md](../../LICENSE-APPS.md).

## Security

- Keep private keys out of the app folder, off GitHub and off shared machines. Use a passphrase.
- Make a recovery key at the start and keep it offline.
- Publish the fingerprint of your key in the README.
- Never reuse the drawer's release key.
- Bump `version` for every release. The device refuses the same or a lower version.
- Remember the limits: pins protect against a bad host, mirror or network. They do not protect against a malicious app already on the device. Apps run as root.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| "No update found" after a release | version not higher than installed, or release is a draft or prerelease on the stable channel | bump `version`; publish as a normal release |
| Bad signature | signed an older `update.json`, or edited it after signing, or wrong key | run `appsctl app sign` again, then `verify` |
| Key changed | `update.key` in the new `app.json` differs from the pinned key | restore the old key, or do a proper rotation |
| Hash or size mismatch | zip rebuilt or re-uploaded after `pack` | run `pack` and `sign` again and upload all three files |
| Unsafe zip | `data/`, a symlink, `..` or a path outside `<id>/` | remove it; `appsctl app verify` names the entry |
| Needs a newer drawer | `requires.drawer` is above the user's version | lower `min_host`, or ask users to update the drawer |
| GitHub is limiting requests | beta channel, many checks from one IP address | wait an hour, or use stable |
| Unreachable | host down, or Wi-Fi without internet | try again; add a mirror |
| Rolled back | the new version failed validation on the device | run `appsctl validate` on the zip contents and fix |

## Checklist before a release

- [ ] `version` is higher than the last release.
- [ ] `appsctl validate` has no errors or warnings.
- [ ] No `data/` and no personal files in the app folder.
- [ ] `update.key` and `update.recovery_key` match your key files.
- [ ] `notes` is short plain text.
- [ ] `pack`, `sign`, `verify` all passed on the final files.
- [ ] The release has exactly three assets: `app-<id>-<version>.zip`, `update.json`, `update.json.sig`.
- [ ] The release is not a prerelease (stable) or is a prerelease (beta).
- [ ] The README shows the key fingerprint and the data the app touches.
- [ ] The recovery key is offline and backed up.

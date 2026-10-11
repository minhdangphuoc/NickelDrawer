# Release checklist: 1.0.0-beta.1

Exact steps for the first public beta. Nothing here has been run. Work top to bottom in one terminal at the repository root. General background is in [RELEASING.md](RELEASING.md); the update trust model is in [UPDATES.md](UPDATES.md).

Rules that apply to every step:

- Commit messages, tags and release notes carry no co-author or session trailers.
- The signing key (`~/.config/nickeldrawer/release-ed25519.key`) is only ever read by `make release` on your Mac. Never print it, copy it, upload it or put it in CI.
- Do not `cd` into `/Volumes/KOBOeReader`.

## 0. Decide first

Answer these before step 1 (the audit lists the facts):

1. **Repository name and owner.** Assumed below: your GitHub account, name `NickelDrawer`, public.
2. **History.** The 54 existing commits are clean (no secrets, binaries or serials) but carry your personal git email. Publish as is, or squash into one commit with an address you are fine to show (`git commit-tree`/orphan branch). Assumed: publish as is.
3. **`docs/testing/`** holds the final on-device checklist. Device session logs live in the gitignored `local/sessions/`.
4. **`screens` flag.** App screens (`ui.json`) are off unless `.adds/nickeldrawer/screens` exists. Decide whether the beta ships with them on by default (code change in `mod/drawer/mod_switches.cc` plus a device test) or documents the flag (the README does).

## 1. Final checks (local, no device)

```sh
cd path/to/your/clone
git status --short                                     # review every line; nothing unexpected
set -o pipefail && make check build dist && bash tools/test-install.sh
build/appsctl-host validate --apps-dir apps
make -C mod/drawer docker-build && make -C mod/drawer docker-test && mod/drawer/check-symbols.sh
```

Public-readiness scan of what will be published:

```sh
# binaries, firmware, databases: expect no output
git ls-files | grep -Ei '\.(so|tgz|sqlite|zip|ttf|otf|epub)$|^(appsctl|ghsync|wallpaper)$'
# personal data: expect only RFC 1918 example addresses (192.168.1.x) and test token fixtures
git grep -nIE '/Users/|@gmail|ghp_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|N[0-9]{12}' -- . ':!third_party'
# nothing from Kobo in the mod package or zips
tar -tzf build/mod/KoboRoot-nickeldrawer.tgz
unzip -l dist/koboapps-1.0.0-beta.1.zip | grep -Ei '\.so|\.tgz|sqlite' || echo clean
```

The `KoboRoot` must list exactly `usr/local/Kobo/imageformats/libnickeldrawer.so`.

On the device you own, before you publish anything you tell other people to install: run the failsafe and uninstall drill with this exact build (plan gate G1 and `docs/testing/final-checklist.md`), then one normal boot, the Apps row and one app.

## 2. Release commit

```sh
sed -i '' "s/^## \[1.0.0-beta.1\] - unreleased/## [1.0.0-beta.1] - $(date +%F)/" CHANGELOG.md
git add -A
git status --short                                     # check the staged list once more
git diff --cached --name-only | grep -Ei '^(backups|local|downloads|build|dist)/|\.(so|tgz|sqlite)$' && echo STOP || echo staged-list-ok
git commit -m "chore: release 1.0.0-beta.1" \
  -m "Version 1.0.0-beta.1: native drawer, UI kit, apps, signed updates, English and Vietnamese. Release inputs are built unsigned in CI and signed locally."
```

## 3. Create the repository and push (no secrets are needed)

```sh
gh auth status
GH_USER=$(gh api user -q .login)
gh repo create "$GH_USER/NickelDrawer" --public \
  --description "App drawer and tools for Kobo e-readers running the stock firmware" \
  --homepage "https://github.com/$GH_USER/NickelDrawer"
git remote add origin "git@github.com:$GH_USER/NickelDrawer.git"
git push -u origin main
gh api -X PUT "repos/$GH_USER/NickelDrawer/private-vulnerability-reporting"   # SECURITY.md points reporters here
gh repo edit "$GH_USER/NickelDrawer" --add-topic kobo --add-topic ereader --add-topic nickel --add-topic nickelmenu
gh run watch --exit-status                             # CI must be green before you tag
```

If `private-vulnerability-reporting` returns 404, enable it in Settings, Code security.

CI uses no repository secrets. Leave **Settings, Secrets and variables** empty. Keep the default `GITHUB_TOKEN` permissions at read (the workflow also sets `contents: read`).

## 4. Build the release assets and sign (your Mac only)

```sh
VERSION=1.0.0-beta.1
make -C mod/drawer docker-build
make release VERSION=$VERSION            # check, build, dist, sign with your key, verify
build/ndrelease-host verify              # again, by hand, against spec/release-ed25519.pub
```

`make release` stops with a message if the key is missing, readable by others, or does not match `spec/release-ed25519.pub`.

Collect and hash everything that goes on the release page:

```sh
STAGE=build/release-assets && rm -rf "$STAGE" && mkdir -p "$STAGE"
cp dist/release/release.json dist/release/release.json.sig "dist/release/nickeldrawer-$VERSION.zip" "$STAGE"/
cp "dist/koboapps-$VERSION.zip" build/mod/KoboRoot-nickeldrawer.tgz "$STAGE"/
(cd "$STAGE" && shasum -a 256 release.json release.json.sig "nickeldrawer-$VERSION.zip" "koboapps-$VERSION.zip" KoboRoot-nickeldrawer.tgz >SHA256SUMS.txt && cat SHA256SUMS.txt)
```

The CI `release-inputs` artifact (step 5) is a second build of the same sources; its zip and tgz may differ in bytes (timestamps). Publish the files you built and tested here.

## 5. Tag and publish

```sh
git tag -a "v$VERSION" -m "NickelDrawer $VERSION"       # add -s if you have a signing identity set up for git
git push origin "v$VERSION"                            # CI builds the unsigned release-inputs artifact
gh run watch --exit-status

awk '/^## \[1.0.0-beta.1\]/{keep=1;next} /^## \[/{keep=0} keep' CHANGELOG.md >build/release-notes.md
gh release create "v$VERSION" --verify-tag --prerelease --title "NickelDrawer $VERSION" \
  --notes-file build/release-notes.md \
  "$STAGE"/release.json "$STAGE"/release.json.sig "$STAGE/nickeldrawer-$VERSION.zip" \
  "$STAGE/koboapps-$VERSION.zip" "$STAGE"/KoboRoot-nickeldrawer.tgz "$STAGE"/SHA256SUMS.txt
```

`--prerelease` keeps `stable` clients away; only `"channel": "beta"` devices update to it.

## 6. Verify what is public

```sh
CHECK=$(mktemp -d)
gh release download "v$VERSION" --repo "$GH_USER/NickelDrawer" --dir "$CHECK"
(cd "$CHECK" && shasum -a 256 -c SHA256SUMS.txt)
build/ndrelease-host verify --dir "$CHECK"             # signature, every file's sha256 and size, with the public key from the tree
unzip -l "$CHECK/koboapps-$VERSION.zip" | grep -Ei '\.so|\.tgz|sqlite' || echo clean
rm -rf "$CHECK"
```

Then on the device you control (it must run a build with the embedded release key and an older version): copy `{"repo": "<owner>/NickelDrawer", "channel": "beta"}` to `/mnt/onboard/.adds/koboapps/data/update.json`, tap Apps, the gear (⚙), Check, then Update to the new version, then confirm `appsctl update rollback` still works. Do the fresh-install path from the release zip on a spare folder or after a backup.

## 7. Announce

Post only after step 6 passes. Keep it short, say it is a beta, say what is tested, link the README. Suggested text:

> **NickelDrawer 1.0.0-beta.1.** An app drawer for Kobo e-readers on the stock firmware: an Apps row in the More tab, plus LocalSend, GitHub book sync, a sleep-screen gallery and a library tool. Kobo Libra 2, firmware 4.38 (23828, 23697, 23684, 23648, 23552). Tested on one device on 4.38.23828; the others are symbol-checked only. It has a failsafe and a one-file uninstall; read the Safety section first. MIT, signed updates, English and Vietnamese. https://github.com/OWNER/NickelDrawer/releases/tag/v1.0.0-beta.1

Places: the repository's Discussions (enable with `gh repo edit "$GH_USER/NickelDrawer" --enable-discussions`), the MobileRead Kobo forum, the Vietnamese Kobo communities you use.

## If something is wrong after publishing

```sh
gh release delete "v$VERSION" --cleanup-tag --yes      # also removes the remote tag
git tag -d "v$VERSION"
```

A leaked or lost signing key: follow the key-loss paragraph in [UPDATES.md](UPDATES.md). A leaked token or personal file in a pushed commit: rotate the secret first, then rewrite history; do not just delete the file in a new commit.

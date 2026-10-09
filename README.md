# NickelDrawer

An app drawer and small tools for Kobo e-readers on stock firmware (Nickel). Apps are folders; the menu is built from them; a native drawer is planned. First target: Kobo Libra 2, firmware 4.38.x. English and Vietnamese UI.

- Plan overview with wireframes: [docs/plan-overview.html](docs/plan-overview.html)
- Detailed plan and timeline: [docs/plan-detailed.html](docs/plan-detailed.html) ([Markdown](docs/plan-detailed.md))

## Install

```sh
make check build dist          # tests, ARMv7 binaries, dist/ tree and dist/koboapps-<version>.zip
tools/install-to-kobo.sh --dry-run                    # show what would happen
tools/install-to-kobo.sh                              # install (Kobo mounted at /Volumes/KOBOeReader)
tools/install-to-kobo.sh --remove-legacy              # also drop old .adds/localsend and its two menu items
tools/test-install.sh                                 # run the installer against a fake Kobo in a temp dir
```

The installer backs up `.adds/nm`, `.adds/apps`, `.adds/koboapps` (and `.adds/localsend` with `--remove-legacy`) to `backups/<date-time>/` first. It replaces `app.json` and `bin/` of each app but never touches an existing `data/` (token, config, PIN, state). Set `KOBO_ROOT` if the volume has another name. Run it only with the Kobo mounted and you present; eject afterwards. `make clean` removes `build/` and `dist/`.

## Updates

Once installed, apps and the host tool update over Wi-Fi from a signed GitHub Release: put `{"repo": "owner/name"}` in `.adds/koboapps/data/update.json`, then tap **Apps · Check for updates** and later **Apps · Update status**. Updates are verified with an ed25519 key embedded in `appsctl`, never touch app `data/` folders, keep the two newest backups and restore automatically if a self-check fails (`appsctl update rollback` undoes the last one). Maintainers publish with `make release VERSION=x.y.z`. Details and trust model: [docs/UPDATES.md](docs/UPDATES.md).

## Tools

- `tools/vncheck.py FONT...` — count missing Vietnamese glyphs in fonts (needs `fonttools`).
- `tools/nfc.py DIR` — check unpacked EPUB HTML for non-NFC text / combining marks.
- `tools/md2html.py plan.md out.html tools/plan-tpl.html` — render the plan page (needs `markdown`).
- `go run ./cmd/ndpreview serve apps/localsend --locale vi` — preview a native screen (`ui.json`) at Libra 2 size in the browser on 127.0.0.1; `ndpreview render ... --out screen.png` writes a PNG. See [docs/APPS.md](docs/APPS.md), "Design a native screen".
- `go run ./cmd/appsctl validate apps/localsend` — also checks `ui.json` ([docs/spec/ui-json.md](docs/spec/ui-json.md)).
- `make schema` — rewrite `spec/ui.schema.json` from `internal/uikit`.

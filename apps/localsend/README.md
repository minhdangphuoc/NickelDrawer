# LocalSend app

Receives files from a phone via LocalSend and sorts them into the library.

## How it works

- **Start receiving** runs `localgo` with `--dir` set to the inbox (`/mnt/onboard/.adds/koboapps/inbox`) and `--exec` set to `appsctl inbox route --from-env --source localsend`. localgo runs that hook once for each received file.
- The router checks the file against an allowlist (extension and content must agree, size caps) and moves it: books (EPUB, KEPUB, MOBI) to `Books`, PDF/TXT/HTML/RTF to `Documents`, CBZ/CBR to `Comics`, PNG/JPEG/GIF to `Pictures`. Anything else goes to `inbox/rejected/` and is deleted after 7 days. Existing files are never overwritten (`name (2).epub`). Rules and format: `docs/spec/inbox.md`.
- Receiving **stops by itself after 15 minutes without a received file** (`bin/watch.sh`, started with the receiver). The screen then says so, and the library is rescanned (a `rescan` request from `state`, declared in `app.json`). No script can keep the Kobo awake: Nickel's own sleep timer still applies, so wake the Kobo while sending.
- **Stop receiving** stops localgo, waits up to 2 seconds for running hooks, sorts anything left in the inbox (and in the old `/mnt/onboard/LocalSend` folder that older versions used), shows a summary (for example `Books: 2`, `Rejected: 1`) and rescans the library.

## Security note

The `--exec` string (`ROUTE_HOOK` in `bin/env.sh`) must never contain a `%` placeholder such as `%f` or `%n`. localgo pastes those into the shell command, and the file name and sender alias come from the sender. The router reads the path from the `LOCALGO_FILE` environment variable instead. A test in `cmd/appsctl` enforces this. See `docs/ARCHITECTURE.md` section 4.4.

## Files

`bin/localgo` is not committed (third-party binary). `make dist` copies it from `build/localgo-linux-armv7` (built from `third_party/localgo`) into `bin/localgo`.

State lives in `data/` (PIN in `data/pin.txt`, config in `data/home`). While running, localgo logs to `/tmp/localsend.log` so nothing stays open on the USB-visible storage; "Stop receiving" appends it to `data/log.txt` (last 500 lines kept). The journal of routed files is `.adds/koboapps/data/inbox-journal.jsonl`.

## Install, update, remove

- **Install:** download `app-localsend-<version>.zip` from the [release](../../../../releases), unzip it and merge its `.adds` folder onto the root of the Kobo drive (Finder: choose **Merge**, never Replace). Or, from a checkout, `make dist` then `tools/install-to-kobo.sh --app localsend`. The drawer must already be installed. Steps and details: [apps/README.md](../README.md).
- **Update:** install the new zip the same way. The app's files are replaced and its `data/` is kept.
- **Remove:** long-press the app in the drawer and choose Remove, or delete `.adds/apps/localsend` on the drive.

# ghsync phone pairing

`ghsync pair` lets the user enter the GitHub repo and token on a phone instead of typing them on the Kobo. In the drawer: **GitHub**, then the row **Pair with your phone** (the `pair` screen), then **Start pairing** (`bin/pair.sh`) and, to cancel, **Stop pairing** (`bin/stop-pairing.sh`).

## Flow

1. The pairing screen's **Start pairing** runs `bin/pair.sh`. It checks Wi-Fi, starts `ghsync pair` detached (output to `/tmp/ghsync-pair.txt`, pid in `/tmp/ghsync-pair.pid`), waits about 1 s and answers with a toast. Nothing is printed for the drawer to show: the screen reads the session from `bin/state.sh`.
2. While the session is open the screen shows, from `ghsync state` (re-read every 5 s): the QR code of the address, the address `http://<ip>:8765`, the one-time 6-digit code in large digits, a hint, and how long it stays open ("Open 9 more min, until 21:30"). The session writes the address, code and expiry to `/tmp/ghsync-pair.json` (mode 600) while it is open; the file is removed when it ends.
3. On the phone (same Wi-Fi) the user scans the QR code or types the address and fills one form: code, repo (`owner/name`), branch (optional), folder (optional), token (password field). The page uses inline CSS only, no scripts and no external resources, and is shown in the device language (`apps/ghsync/locales`, English and Vietnamese; other languages by adding a locale file).
4. The server checks the code, validates the fields, then proves the token works against the GitHub API before saving anything (see "What the check tells apart" below).
5. On success it writes `data/token` then `data/config.json` (temp file, fsync, rename, mode 600) and stops. `dest` and `extensions` of an existing config are kept.
6. The session records how it ended (paired, locked, timed out, stopped) in `/tmp/ghsync-pair-result`; the screen shows it as a note ("Paired with owner/name. Tap Sync now."), and the main screen's header changes from "Not paired" to "Ready".

## What the check tells apart

GitHub answers 404 for a repository the token cannot see, a branch that does not exist, a folder that does not exist, and a repository without commits. The first version showed one vague message for all of them (device session of 2026-10-09: the repo `main` branch had no books). The check (`internal/ghsync/access.go`) now asks in order and says which one failed, on the phone form and, for sync, in the screen's error notice:

| Step | Request | Result |
|---|---|---|
| Repository | `GET /repos/{owner}/{name}` | 404: "Repository {repo} not found, or the token cannot see it" (a private repo outside the token's scope looks the same on purpose); 401: bad or expired token; 403: the token lacks Contents: Read-only |
| Commits | `GET .../branches?per_page=1` | empty list: the repository is empty. It **pairs successfully**; the first sync finds 0 books |
| Branch (only when typed) | `GET .../branches/{branch}` | 404: "Branch {branch} does not exist in {repo}" |
| Folder (only when typed) | `GET .../contents/{folder}?ref={branch}` | 404 or a single file: "Folder {folder} does not exist on branch {branch}" |

A sync of an empty repository (tree answers 409) is a clean run of 0 books; a missing branch (tree 404) is named; a configured folder without files only logs a warning and reports 0 books.

## Rules

| Rule | Value |
|---|---|
| Bind address | the Wi-Fi LAN IPv4 only (private range); wildcard addresses are refused; no IP means no server |
| Port | 8765 |
| Code | 6 digits from `crypto/rand`, compared in constant time |
| Wrong codes | 5 allowed in total; the 5th closes the server |
| Lifetime | 10 minutes, success, lockout, or `ghsync pair --stop` (SIGTERM via the pid file) |
| Token format | starts with `github_pat_` or `ghp_`, 20 to 255 chars of `[A-Za-z0-9_]` |
| Repo / folder | same rules as `config.json` (`owner/name`, no `..`) |
| Token handling | never logged, never in any page, never in the URL (POST), redacted from error text; the form that comes back after an error is refilled without the token and code |
| Responses | `Cache-Control: no-store`, strict CSP, `Referrer-Policy: no-referrer`, request bodies limited to 8 KiB |

A mistyped repo or token does not use up a code attempt; only a wrong code does.

## Where the token goes (audit, 2026-10-10)

| Place | Holds the token? | How it is kept that way (tests in `internal/ghsync/token_audit_test.go`, `cmd/ghsync`) |
|---|---|---|
| `data/token` | yes, the only copy; written by the pairing page or by hand | temp file, fsync, rename, mode 600 (`TestTokenAndConfigAreWrittenPrivate...`); `ghsync run` refuses a token file others can read where the filesystem enforces modes |
| `data/config.json`, `data/state.json`, `data/log.txt` | no | the logger replaces the token with `***`; errors saved to `state.json` are redacted; `scrubbed()` rebuilds any returned error that quotes the token |
| `ghsync state` output (the screen) | no | the output type is an allowlist without a token field; the failure reason shown on screen is cut of the token a second time |
| process arguments, environment, stdout, `/tmp` files | no | the token is read from the file by `ghsync` only; the pairing info file holds address and code, not the token |
| error pages on the phone | no | the form comes back without the token field; messages are redacted |
| HTTP | only in the `Authorization` header to `api.github.com` over HTTPS | Go drops that header on cross-host redirects |

Caveat: `/mnt/onboard` is FAT, so mode 600 cannot be enforced there; anyone who mounts the Kobo over USB can read `data/token`. That is why the token is fine-grained, limited to one repository and read-only; revoke it on GitHub if the Kobo is lost.

## Security note: HTTP versus self-signed HTTPS

The page is plain HTTP on the home LAN. The token and code travel unencrypted, so someone on the same Wi-Fi who can sniff traffic (or an attacker on a shared network) could read the token during the few seconds of the request.

| | HTTP + one-time code (v1) | Self-signed HTTPS |
|---|---|---|
| Eavesdropping | possible on the LAN | blocked |
| Phone experience | opens at once | full-page browser warning; users learn to click through, and iOS/Android may refuse to submit forms on an untrusted page |
| Active attacker | cannot guess the code in 5 tries, but a sniffer sees the code and token in the same request | can still serve a different certificate; the user cannot tell the difference without comparing fingerprints |
| Kobo cost | none | key generation and TLS in a tiny binary, certificate lifecycle |

Why v1 picks HTTP: a self-signed certificate protects against sniffing only if the user verifies a fingerprint, which nobody does, and the browser warning trains users to ignore warnings. Instead v1 shrinks the exposure: a 10-minute window, one use, 5 attempts, LAN-only binding, a fine-grained read-only token limited to one repo (the worst case is read access to one book repo), and a plain "use only on your own Wi-Fi" warning on the form and in the docs. Revoke the token on GitHub if you paired on a network you do not trust.

The QR code on the Kobo carries only the address; the 6-digit code is still typed. A later version may put a long random secret (256 bits) in the URL fragment, so the secret is never sent to the server in clear and nothing has to be typed.

## Files

- `internal/ghsync/pair.go` session, LAN IP detection, lifetime; `pairform.go` handler, validation, GitHub check, atomic save; `pairpage.go` HTML.
- `cmd/ghsync/pair.go` CLI (`pair`, `pair --stop`, flags `--port`, `--ip`, `--lifetime`, `--pid-file`, `--api-base`).
- `internal/ghsync/access.go` the access check; `internal/ghsync/screenstate.go` and `cmd/ghsync/state.go` what the screen shows.
- `apps/ghsync/bin/pair.sh` and `stop-pairing.sh`, actions `pair` and `stop-pairing` in `apps/ghsync/app.json`, the `pair` screen in `ui.json`, strings `msg.pair_*`, `msg.err_pair_*` and `msg.err_*_not_found` in `locales/`.

# ghsync phone pairing

`ghsync pair` lets the user enter the GitHub repo and token on a phone instead of typing them on the Kobo. Menu entry: **GitHub · Pair with phone** (`bin/pair.sh`).

## Flow

1. The menu runs `bin/pair.sh`. It waits for Wi-Fi, starts `ghsync pair` detached (output to `/tmp/ghsync-pair.txt`, pid in `/tmp/ghsync-pair.pid`), waits about 1 s and prints three lines: the address `http://<ip>:8765`, a one-time 6-digit code, and the validity (10 minutes). `cmd_output` stays under 10 s.
2. On the phone (same Wi-Fi) the user opens the address and fills one form: code, repo (`owner/name`), branch (optional), folder (optional), token (password field). The page uses inline CSS only, no scripts and no external resources, and is shown in the device language (`apps/ghsync/locales`, English and Vietnamese; other languages by adding a locale file).
3. The server checks the code, validates the fields, then proves the token works against the GitHub API (repo readable, folder listing via the contents API) before saving anything.
4. On success it writes `data/token` then `data/config.json` (temp file, fsync, rename, mode 600) and stops. `dest` and `extensions` of an existing config are kept.
5. **GitHub · Status** shows the pairing address and code while it is open and the outcome (paired, locked, timed out, stopped) afterwards.

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

## Security note: HTTP versus self-signed HTTPS

The page is plain HTTP on the home LAN. The token and code travel unencrypted, so someone on the same Wi-Fi who can sniff traffic (or an attacker on a shared network) could read the token during the few seconds of the request.

| | HTTP + one-time code (v1) | Self-signed HTTPS |
|---|---|---|
| Eavesdropping | possible on the LAN | blocked |
| Phone experience | opens at once | full-page browser warning; users learn to click through, and iOS/Android may refuse to submit forms on an untrusted page |
| Active attacker | cannot guess the code in 5 tries, but a sniffer sees the code and token in the same request | can still serve a different certificate; the user cannot tell the difference without comparing fingerprints |
| Kobo cost | none | key generation and TLS in a tiny binary, certificate lifecycle |

Why v1 picks HTTP: a self-signed certificate protects against sniffing only if the user verifies a fingerprint, which nobody does, and the browser warning trains users to ignore warnings. Instead v1 shrinks the exposure: a 10-minute window, one use, 5 attempts, LAN-only binding, a fine-grained read-only token limited to one repo (the worst case is read access to one book repo), and a plain "use only on your own Wi-Fi" warning on the form and in the docs. Revoke the token on GitHub if you paired on a network you do not trust.

Later: the native drawer phase will show a QR code on the Kobo that carries the address and a long random secret (256 bits) in the URL fragment, so the secret is never sent to the server in clear and the code is not guessable or typed.

## Files

- `internal/ghsync/pair.go` session, LAN IP detection, lifetime; `pairform.go` handler, validation, GitHub check, atomic save; `pairpage.go` HTML.
- `cmd/ghsync/pair.go` CLI (`pair`, `pair --stop`, flags `--port`, `--ip`, `--lifetime`, `--pid-file`, `--api-base`).
- `apps/ghsync/bin/pair.sh`, action `pair` in `apps/ghsync/app.json`, strings `msg.pair_*` and `msg.err_pair_*` in `locales/`.

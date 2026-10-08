# Inbox router

Sorts files received on the Kobo (LocalSend, USB) into the library folders. Design and threat model: [ARCHITECTURE.md](../ARCHITECTURE.md) §4. Code: `internal/inbox`, `cmd/appsctl/inbox.go`.

## Commands

```
appsctl inbox route --from-env [--source NAME]   path from $LOCALGO_FILE only (the LocalSend hook)
appsctl inbox route [--source NAME] PATH         one file (tests, USB)
appsctl inbox sort [--source-dir DIR]            every file directly in DIR
appsctl inbox status                             summary of the last session, at most 8 short lines
appsctl inbox purge                              delete rejected files older than 7 days
```

All take `--onboard-root DIR` (default `/mnt/onboard`) plus `--locale`, `--conf`, `--koboapps-dir` like the other commands. Text comes from `host/locales/<lang>.json` (`inbox.*` keys).

- `route` and `sort` accept files only from the **inbox** (`.adds/koboapps/inbox`) or the old **LocalSend folder** (`/mnt/onboard/LocalSend`); any other path is an error. `sort` skips hidden names and sub-folders.
- `--source` is a label for the journal (`^[a-z0-9][a-z0-9-]{0,31}$`), default `manual`.
- `--from-env` with a path argument, or with an empty `LOCALGO_FILE`, is an error.
- Exit code 0 for a routed **or rejected** file (both are normal outcomes) and when another process already took the file; 2 for usage errors, rules errors and I/O failures.

## Pipeline per file

1. The folder must be an allowed source (after `EvalSymlinks`); the entry must not be a folder.
2. A symlink or non-regular file is rejected (`symlink`, `not_regular`), never followed.
3. The name is cleaned and matched against the rules by the longest extension. No match: `type_not_allowed` (a `*.tmp` leftover of a cut transfer is deleted: `incomplete`).
4. Empty file: `empty`. Over the size cap: `too_large`.
5. The content must pass the rule's check, else `content_mismatch`.
6. The name is sanitized (see below), claimed in the destination with `O_EXCL` (`name.ext`, `name (2).ext`, ...), and the file is renamed onto the claim.
7. A rejected file goes to `rejected/` the same way, with its mtime set to the rejection time.
8. One JSON line is appended to the journal under `flock`.

Several `route` processes may run at once (localgo starts one hook per file). Two processes that sanitize to the same name get different final names; if two race for the same file, one wins and the other reports "already handled".

## Rules: `config/routing.json`

```json
{ "version": 1, "rules": [
  { "type": "pdf", "extensions": [".pdf"], "check": "pdf", "max_bytes": 314572800, "destination": "documents" }
] }
```

| Field | Meaning |
|---|---|
| `type` | unique name; used by overrides, the journal |
| `extensions` | lowercase, with the dot; unique across rules; the longest match wins (`.kepub.epub` over `.epub`) |
| `check` | `epub`, `mobi`, `pdf`, `utf8text`, `html`, `rtf`, `zip`, `rar`, `png`, `jpeg`, `gif` |
| `max_bytes` | size cap |
| `destination` | `books`, `documents`, `comics` or `pictures` |

The shipped file is installed to `.adds/koboapps/config/routing.json` and replaced on every install. Missing or invalid: routing stops (fail closed).

### User override: `data/routing.json`

```json
{ "remove": ["gif"], "destinations": { "txt": "books" } }
```

Only `remove` (type names) and `destinations` (type name to a library folder) exist. Unknown fields, unknown types, or a destination that is not `books`, `documents`, `comics` or `pictures` make the whole file invalid: a warning is printed and the shipped defaults apply. The installer never writes this file.

## Names

NFC normalization, `/` and `\` and control characters removed, `<>:"|?*` replaced by `_`, leading dots and spaces and trailing dots and spaces removed (an empty stem becomes `file`), at most 180 bytes with the extension kept, extension lower-cased.

## Journal: `data/inbox-journal.jsonl`

```json
{"time":"2026-10-08T09:00:00Z","source":"localsend","name":"Truyện Kiều.pdf","type":"pdf","destination":"documents","stored_as":"Truyện Kiều.pdf"}
{"time":"2026-10-08T09:00:02Z","source":"localsend","name":"x.exe","destination":"rejected","stored_as":"x.exe","reason":"type_not_allowed"}
```

`status` counts the newest run of lines whose gaps are at most 30 minutes. Reasons: `symlink`, `not_regular`, `type_not_allowed`, `too_large`, `empty`, `content_mismatch`, `incomplete`; each has a message `inbox.reason.<code>` in the host locales.

The journal is append-only and is never rotated or truncated: a line is about 150 bytes, so a thousand files cost roughly 150 KB. Delete the file by hand if it ever matters; `status` then starts from an empty history.

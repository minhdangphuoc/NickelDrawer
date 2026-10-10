# Library app (v1)

`apps/library`, binary `bin/library` (`cmd/library`, `internal/library`). It fixes two real problems of a Vietnamese reader:

- Vietnamese EPUBs tagged `dc:language` "en" (or nothing) get English hyphenation on Kobo.
- The same book arrives twice (LocalSend, GitHub sync, USB) and piles up.

Read-only by default: only `fix-language`, `normalize-nfc`, `remove-duplicate`, `undo` and `restore` change files, and each keeps the original in `data/`.

## Commands

All commands accept `--app-dir DIR` (default `/mnt/onboard/.adds/apps/library`), `--locale CODE`, `--conf FILE` and `--root DIR` (storage root, for tests and Mac runs). Flags go before positional arguments.

| Command | What it does |
|---|---|
| `scan` | Walks the library, writes `data/report.json` and `data/summary.json`, prints "Books / Wrong language / Not NFC / Duplicate groups". Purges old trash first. |
| `fix-language [--dry-run]` | Re-analyzes the library and fixes every Vietnamese EPUB whose language is not `vi`. `--dry-run` lists what would change and writes nothing. |
| `normalize-nfc [--dry-run]` | Renames files whose name is not Unicode NFC and rewrites a non-NFC `dc:title` / `dc:creator` in EPUBs. Content changes are backed up like a language fix; every rename is journaled so `undo` renames it back. A rename onto an existing file is refused (counted as failed, nothing overwritten). |
| `cancel` | Asks the running scan or fix to stop after the book it is working on (`data/cancel`; the run clears it). A stopped scan saves nothing; a stopped fix keeps what it finished (undoable). Prints "Nothing is running" otherwise. |
| `dedupe-list` | Prints the duplicate groups of the last scan. Never deletes. |
| `remove-duplicate --confirm <path>` or `--selected --confirm` | Moves one file of a duplicate group to `data/trash/`. |
| `undo [YYYY-MM-DD]` | Puts the originals of one fix or normalisation run back (newest run when no date), including old file names. |
| `restore [YYYY-MM-DD]` | Puts trashed duplicates of one day back (newest day when no date); never overwrites. |
| `pick <path>` | Remembers a duplicate tapped on the screen (`data/selected.txt`). |
| `page` | Next page of the duplicate list (`data/page.txt`). |
| `state` | Prints the `ui.json` variables as one JSON object. |

Changing commands take the lock `data/lock` (pid inside; a dead pid is taken over), so only one runs at a time; a second one prints "Still working".

## What is scanned

Only the logical folders `books`, `documents` and `comics` of `internal/storage` (`/mnt/onboard/Books`, `Documents`, `Comics`), recursively to depth 8 and at most 20,000 files. Nothing else is read: not `Pictures`, not the old `/mnt/onboard/LocalSend` folder (use LocalSend's "Sort old files" to move it into `Books` first), not any other top-level folder. Adding a folder means adding a logical name to `internal/storage`, a doc entry here and a test.

Safe-read rules:

- hidden names and names starting with `_` are skipped;
- **symlinks are never followed**, neither files nor folders (stricter than the "resolves inside" rule of `storage.ListImages`: a book library has no reason to contain links);
- only regular files with a book extension (`epub mobi pdf txt html htm rtf cbz cbr`) at most 200 MB; larger files are counted as skipped;
- only `.epub` (including `.kepub.epub`) is opened; the zip is read with streaming, entries are size-limited (container 256 KB, package 4 MB, 96 KB of markup per chapter), at most 20,000 entries.

## What the scan reads from an EPUB

`META-INF/container.xml` -> rootfile -> OPF: first non-empty `dc:title`, `dc:creator`, `dc:language`. Then about 20 KB of visible text from the spine documents in order (at most 6 KB per chapter; scripts, styles and `<head>` dropped, entities decoded).

**Vietnamese detection** on the NFC-normalized sample: share of letters that are `ă đ ơ ư`, the tone-marked block U+1EA0-U+1EF9, or `â ê ô` / accented vowels shared with French and Spanish. Vietnamese when there are at least 100 letters, the share is at least 0.20 and the Vietnamese-only letters alone are at least 0.08 (Vietnamese prose is about 0.30-0.40; French and Portuguese stay under 0.10).

**Wrong language**: Vietnamese text and the primary subtag of `dc:language` is not `vi` (`vi-VN` is fine; a missing tag is wrong).

**Not NFC**: the file name, the title or the author is not in Unicode NFC (decomposed tone marks display badly in some fonts and break search). Body text is not judged: `normalize-nfc` fixes exactly what is counted. Report rows carry `name_not_nfc` and `meta_not_nfc`.

**Duplicates**: files of equal size are hashed with sha256 (other files are never read in full). Groups join files with the same sha256 (kind `identical`) or the same normalized title and author (kind `probable`; case, tone marks and punctuation ignored, `đ` = `d`). A group is `identical` only when every member has the same hash.

## data/report.json

```json
{ "version": 1, "scanned_at": "2026-10-09T14:30:00+07:00",
  "counts": { "files": 340, "epubs": 300, "wrong_language": 12, "not_nfc": 3, "unreadable": 1, "duplicate_groups": 5, "skipped": 0 },
  "duplicates": [ { "id": 1, "kind": "identical", "paths": ["Books/a.epub", "Books/GitHub/a.epub"] } ],
  "books": [ { "path": "Books/a.epub", "size": 123456, "format": "epub", "title": "...", "author": "...", "language": "en",
               "vietnamese": true, "vietnamese_share": 0.31, "not_nfc": false, "sha256": "...", "error": "" } ] }
```

Paths are relative to the storage root (`/mnt/onboard`), slash-separated. `summary.json` is the same without `books`, so `state` stays fast. Every path that comes back from the screen or the command line is validated again: clean, inside `Books/`, `Documents/` or `Comics/`, no hidden part, parent resolves inside the folder, regular file.

## fix-language

For each book: refuse above 100 MB; require free space of `2 x size + 16 MB`; **copy the original to `data/backups/<date>/<same relative path>`** (an existing backup of the same day is the true original and is kept); write a new zip to a hidden temp file next to the book; **re-read it** (every entry decompresses, `mimetype` is the first entry and stored, the package now says `vi`); then atomically rename it over the book. On any failure the temp file is removed and the original stays.

What changes: the text of every `dc:language` in the OPF (a `dc:language` is added when there is none), and `lang="en*"` / `xml:lang="en*"` attributes in the OPF, `.xhtml`, `.html`, `.htm` and `.ncx` entries (`hreflang` and `data-lang` are untouched). All other entries are copied raw, byte for byte, still compressed. After the run the report rows of the fixed books are re-read, so the counts drop without a new scan.

**Kobo database**: the app never touches `KoboReader.sqlite`. Nickel reads `dc:language` when it imports a book, so after a change the library has to be rescanned. The app queues one quick rescan request (`data/requests.json`) after every fix, normalisation, undo, trash and restore that changed files; the `state` action hands it over once, after the job has ended, as `"requests":[{"request":"rescan","mode":"quick"}]` (`app.json` declares `"requests": ["rescan"]`). From a shell the command prints a hint to rescan. Editing the database from outside Nickel is the riskiest thing a mod can do and is not needed.

## Duplicates and trash

Nothing is deleted automatically and the app never deletes a book. `remove-duplicate` needs `--confirm`, a path that is a member of a group in the last report, and at least one other member that still exists (the last copy is never moved). The file goes to `data/trash/<date>/<relative path>` (rename; copy and remove if on another file system). The report is updated. `restore` puts a day's trash back without overwriting (the screen's **Restore from trash** button restores the newest day). Trash day-folders older than 30 days are deleted at the start of each `scan`, only below `data/trash/`.

On the screen a duplicate is first selected by tapping its row (`pick`), then removed with a confirmed button (`trash`); a tap alone never removes anything.

## data/ layout

```
data/report.json  summary.json  progress.json  selected.txt  page.txt  lock  cancel  requests.json  log.txt
data/backups/<date>.renames.json   names changed by normalize-nfc that undo reverses
data/backups/<date>/Books/...      originals before fix-language (undo)
data/trash/<date>/Books/...        removed duplicates (restore, purged after 30 days)
```

`log.txt` is truncated above 256 KB; every line is written by opening and closing the file.

## Actions and screen

`app.json` storage: `reads: [books, documents, comics]`, `writes: [books, documents, comics]` (a fixed or trashed book can live in any of them). Declares `requests: ["rescan"]`.

| Action | Script | Runs |
|---|---|---|
| `scan` | `bin/scan.sh` | detached; the screen follows it through `state` |
| `fix` | `bin/fix.sh` | detached, same pattern |
| `undo` | `bin/undo.sh` | detached, same pattern |
| `normalize` | `bin/normalize.sh` | detached (`normalize-nfc`) |
| `restore` | `bin/restore.sh` | foreground; restores the newest trash day |
| `cancel` | `bin/cancel.sh` | foreground; asks the running job to stop |
| `pick` | `bin/pick.sh` | foreground; `NICKELDRAWER_VALUE` is the tapped row's path |
| `page` | `bin/page.sh` | foreground |
| `trash` | `bin/trash.sh` | foreground; moves the selected duplicate |
| `state` | `bin/state.sh` | foreground, 1 s budget; prints the bindings |

`state` variables: `status` ("Working" while running), `books`, `wrong`, `not_nfc`, `groups`, `last_scan`, `running`, `phase`, `done`, `total`, `detail` (feeds the `steps` component with the progress file), `can_fix`, `can_undo`, `has_pages`, `has_selection`, `selected`, `duplicates` (rows `[path, "Group 2 · same file · 1.2 MB"]`, 4 groups per page; `page` cycles through the pages).

The screen (ui.json): counts as `kv` rows, `steps`, secondary buttons Fix language and Undo last fix (both with a confirm dialog), the duplicate `list` (tap selects), More duplicates, the selection, Move to trash (confirm), a notice about backups, and the primary **Scan** button in the footer.

## Validation cases

| ID | Case | Expected |
|---|---|---|
| LIB-01 | Vietnamese EPUB tagged `en`, scan, fix, reload | Report shows 1 wrong language; after fix the book hyphenates as Vietnamese; `data/backups/<date>/` holds the original |
| LIB-02 | `undo` after LIB-01 | Original bytes back, backup removed |
| LIB-03 | Same EPUB in Books and Books/GitHub | One `identical` group; removing one moves it to `data/trash/`; the other is refused as the last copy |
| LIB-04 | Symlink inside Books pointing at a folder outside | Not scanned, not read |
| LIB-05 | Scan with Wi-Fi off and sleep timer on a 300-book library | Finishes in the background; the screen shows progress |

Automated: `internal/library` and `cmd/library` tests build EPUB fixtures (Vietnamese, English, French, wrong or missing tags, decomposed text, duplicates, damaged zips, symlink escapes, oversized files).

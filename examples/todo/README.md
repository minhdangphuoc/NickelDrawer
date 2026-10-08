# To-do (v1 menu app)

A small to-do list in plain POSIX `sh`. No binary, no Go. It teaches the shape of a v1 app:
one folder, a manifest, thin scripts, English plus Vietnamese messages, data in `data/`.

## Actions

| Menu item | What it does |
|---|---|
| To-do · Show list | Counts, then up to 10 open items, numbered |
| To-do · Mark next done | Turns the first `[ ] ` item into `[x] ` |
| To-do · Clear done items | Removes every `[x] ` line |
| To-do · Add from file | Appends each line of the inbox as a new open item, then empties the inbox |

## Data

`data/todo.txt`, one item per line: `[ ] buy milk` (open) or `[x] call Mai` (done). You can edit
it by hand over USB. `data/` is never shipped in a release zip.

## Adding items (v1)

The Kobo has no good keyboard for a menu app, so v1 uses a file:

1. Write one item per line in a text file named `todo-inbox.txt`.
2. Put it in the Kobo root over USB (`/mnt/onboard/todo-inbox.txt`). If you send it with
   LocalSend it lands in `/mnt/onboard/LocalSend/todo-inbox.txt`, which the app also checks.
3. Tap **To-do · Add from file**. The lines are appended and the inbox is emptied (never deleted).

Typing on the device or entering items from your phone arrives with the native drawer and
`ui.json`. See `ui.json` here for a preview of that screen.

## Design notes

- Every action finishes in well under a second, so it runs in the foreground with
  `"output": "show"` and a 3 s timeout.
- Output is at most 12 lines: counts + 10 items + "... and N more".
- The list is rewritten through a temp file in `data/` plus `mv`, so a power cut never leaves a
  half-written list. No file stays open on `/mnt/onboard` after the script exits.
- Scripts print only through `appsctl msg`; wording lives in `locales/`.
- Shared code is in `bin/lib.sh` (not an action, so it is not in `app.json`).

## Running on the Mac

The scripts take their paths from environment variables (documented at the top of `bin/lib.sh`):

```sh
tmp=$(mktemp -d)
export TODO_DATA_DIR=$tmp/data TODO_INBOXES=$tmp/inbox.txt
export TODO_APPSCTL=$PWD/build/appsctl-host TODO_APPS_DIR=$PWD/examples
printf 'buy milk\ncall Mai\n' > "$tmp/inbox.txt"
examples/todo/bin/add-from-file.sh
examples/todo/bin/show.sh
```

## Preview: ui.json

`ui.json` is a **preview**. It is not supported until the native drawer ships, and nothing reads
it today. It shows the same app as a native screen: a header, a list whose rows toggle done on
tap, a "Hide done items" toggle, a text field for a new item and one primary button (rules in
`docs/spec/ui-kit.md`). The `_preview` key is there to say so; it is ignored.

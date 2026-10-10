# To-do (drawer app with a native screen)

A small to-do list in plain POSIX `sh`, no binary and no Go. It is the reference for an app that
talks to the drawer **only through the NickelDrawer API**: a `ui.json` screen, a `state` action
and scripts that answer with declared toasts.

## What the screen has

| Component | Backed by |
|---|---|
| "Hide done items" switch (`toggle`) | action `hidedone` (the variable `hidedone`) |
| "New item" text box (`field`) | action `newitem`; the typed text arrives as `NICKELDRAWER_VALUE` |
| Item list (`list`), tap flips open/done | action `toggle`, row text in `NICKELDRAWER_VALUE` |
| Long-press a row: Remove... (list `menu`) | action `remove`, with `NICKELDRAWER_LIST`, `NICKELDRAWER_INDEX`, `NICKELDRAWER_VALUE` |
| Clear done items (`button`, with a confirm) | action `clear-done` |
| Add from file (primary `button`) | action `add-from-file` |

An input control runs the action whose id equals its variable name, which is why the variables
are `hidedone` and `newitem` (action ids allow no underscore).

## Data

`data/todo.txt`, one item per line: `[ ] buy milk` (open) or `[x] call Mai` (done). You can edit
it by hand over USB. `data/hide-done` exists while the switch is on. `data/` is never shipped.

## Adding items from a file

Write one item per line in `todo-inbox.txt` in the Kobo root (or send it with LocalSend, which
puts it in `/mnt/onboard/LocalSend/`), then tap **Add from file**. The lines are appended and the
inbox is emptied, never deleted.

## Design notes

- Scripts return at once and print only `{"toast":"id"}`; `report` in `lib/common.sh` does this
  and prints the localized text instead when you run a script from a shell.
- `state.sh` prints the variables as one JSON object. Item text is escaped by `awk`.
- The list is rewritten through a temp file in `data/` plus `mv`, so a power cut never leaves a
  half-written list. No file stays open on `/mnt/onboard` after a script exits.
- Wording lives in `locales/`; the screen texts are `ui.*` keys, the shell messages `msg.*`.

## Running on the Mac

The scripts take their paths from environment variables (see the top of `bin/lib.sh`):

```sh
tmp=$(mktemp -d)
export TODO_DATA_DIR=$tmp/data TODO_INBOXES=$tmp/inbox.txt TODO_COMMON=$PWD/host/lib/common.sh
export TODO_APPSCTL=$PWD/build/appsctl-host TODO_APPS_DIR=$PWD/examples
printf 'buy milk\ncall Mai\n' > "$tmp/inbox.txt"
examples/todo/bin/add-from-file.sh
examples/todo/bin/state.sh
ndpreview serve examples/todo
```

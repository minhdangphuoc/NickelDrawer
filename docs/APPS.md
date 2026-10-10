# Build your first NickelDrawer app

An app is a folder with a manifest, a native screen (`ui.json`) and a few short scripts. Copy it to the Kobo and it shows up in the Apps drawer (More tab, Apps). This guide builds a small "Hello" app from start to finish.

The drawer is built and NickelMenu is retired: apps no longer generate NickelMenu items, and nothing an app prints is shown as a text dialog. See [DECISIONS.md](DECISIONS.md).

The full format rules are in [spec/app-format.md](spec/app-format.md). This guide is the short path.

Skip to [Guidelines](#guidelines) for design rules, or to the worked examples: [To-do](../apps/todo/README.md) (a shipped app: field with the Nickel keyboard, list with a row menu, a tiny Go helper), [UI Kit](../apps/uikit/README.md) (every component with sample content) and [Image viewer](../examples/imageview/README.md) (background job, native reader, Go helper). See [examples/README.md](../examples/README.md).

## Guidelines

Read these before you design an action. An e-ink Kobo is slow, shared with the reader and
battery powered; a good app is small and quiet.

### Design for e-ink

- **One job per action.** "Sync now" and "Status" are two actions, not one with options. Max 12 actions, and fewer is better.
- **Results in 12 short lines or fewer.** A dialog is not a log. Show the outcome and the one next step; put details in `data/log.txt`.
- **Return at once.** A script the drawer starts must return quickly (a hard cap of 30 s applies, then the drawer shows a timeout message). For longer work use the background pattern: start the job detached (`start_detached` in `common.sh`, which redirects every fd), write a pid file and a result file in `/tmp` or `data/`, answer with a "started" toast, and let the `state` action report progress and the result. See `apps/library/bin/lib.sh` and `examples/imageview`.
- **Progress comes from `state`, not from the script.** The screen re-reads your `state` action every 5 seconds while it is open. Report step counts (a `steps` component), not a stream of text, and keep e-ink repaints few.
- **Page-turn keys.** The page-turn buttons flip list pages; the host does this for you with a `list`. Keep one screen short enough to need little paging.
- **Long or secret input comes from the phone.** The Kobo keyboard is poor for tokens and paths. Use the pairing pattern: a tiny HTTP page on the LAN with a one-time code, as in [spec/ghsync-pairing.md](spec/ghsync-pairing.md). Short text can use a `field` (Nickel's keyboard); see the To-do example.
- **Keep your data in `data/`.** Config, state, tokens and logs. Never in the app root, never in `/tmp` if it must survive a reboot.
- **Hold no files open on `/mnt/onboard`** after the script exits (it blocks USB ejecting). Write to a temp file in the same folder as the target and `mv` it into place, so a power cut never leaves half a file.
- **Sleep and Wi-Fi are not guaranteed.** The device may sleep and drop Wi-Fi at any time, and no script can keep Nickel awake (the idle timer is Nickel's). `wifi: true` makes the drawer connect Wi-Fi before the action runs (and report `needs_wifi` if it cannot); check again for an IP inside long jobs and give a clear message if it fails. Background jobs must tolerate the connection going away, and should stop themselves after a timeout instead of running forever.
- **Battery.** No polling loops, no busy waits. Do the work, then exit. Prefer `sleep` over spinning, short bursts over long ones, and no work at all when nothing changed (skip files that are already up to date).

### Localization (i18n)

- English is the source: `app.json` strings and `locales/en.json`. Always ship `vi.json` too; other languages are welcome.
- Scripts never print literal text. Use `appsctl msg --app <id> msg.<key> name=value`. The key in the call includes the `msg.` prefix.
- Never build a sentence from pieces. Use one string with `{placeholders}`; a translation must use the same placeholders (validated).
- Keep Vietnamese short. It runs 20 to 40 percent longer than English and has tall diacritics: prefer short verbs ("Xong việc kế tiếp"), no filler words, and keep labels under 32 characters including diacritics.
- Labels and names may not contain `:`. Use `\n` for line breaks in dialog text; mind the 12-line budget in every language.
- Check each language with the override: `appsctl msg --locale vi ...`.

### Safety

Apps run as **root** on a device that holds the user's library. So:

- Touch only your own files: your `data/` folder and the one folder your manifest documents (for example `/mnt/onboard/LocalSend`).
- When you create files in a shared folder (the library root, `Images/`, `Albums/`), give them a name that shows they are yours (a prefix such as `todo-` or a dedicated subfolder) so you can find them later and never collide with the user's.
- **Never delete or overwrite user files.** Delete only what your app created (or content the user explicitly handed you, like the to-do inbox, which is emptied, not removed). Move to `data/trash/` instead of deleting when unsure.
- Validate every path from config or the network: no `..`, no absolute paths from outside.
- Secrets live in `data/` with mode 600 and never appear in output or logs.
- Do not ship Kobo binaries or firmware files.

### Running inside the drawer

Every script the native drawer starts (actions, events, the `state` read) has `NICKELDRAWER=1` in its environment. In that case return at once (no waiting loops) and print only JSON signals (`{"toast":"id"}`, `{"dialog":"id"}`, `{"request":"rescan"}`) or nothing; never print prose meant for a dialog (`appsctl` logs and drops it). Declare the toast or dialog in `ui.json` instead. In `common.sh`, `report <toast-id> <app> <msg-key>` prints the signal in the drawer and the localized text on the command line (tests, debugging); `report_failure`, `fail_with` and `run_quick_command` do the same for failures and quick commands; `emit_request` and `start_detached` are described below. Declare a `failed` toast on every screen.

### Naming and versioning

- `id`: lowercase, digits, `-`; it equals the folder name and never changes after release (the user's data lives under it).
- `name` is short (24 chars max) because it is the row title in the drawer.
- Labels are verbs or short noun phrases: "Make albums", "Clear done items". Keep the same label for the same job across apps.
- Semver: PATCH for fixes, MINOR for new actions or messages, MAJOR when `data/` formats change incompatibly (write a migration or refuse with a clear message). Bump `version` on every release.

### Testing checklist

1. **Validate:** `build/appsctl-host validate <app>` or `go run ./cmd/appsctl validate <app>` has no errors or warnings.
2. **Lint:** `sh -n bin/*.sh` and `shellcheck bin/*.sh`.
3. **Mac run with temp dirs:** give every script an environment variable for each path it uses (data dir, input folder, `appsctl`), defaulting to the Kobo path. Run the scripts against a temp copy and check output length, exit codes and that nothing is left open or half written. The apps and examples show this (`TODO_DATA_DIR`, `UIKIT_DATA_DIR`, `IMAGEVIEW_IMAGES_DIR`).
4. **Drive it through the protocol:** `appsctl drawer screen <app>` and `appsctl drawer event <app> <action> --value ...` (with `--apps-dir` and a temp copy) print exactly what the mod would get. `cmd/appsctl/examples_test.go` does this for To-do, UI Kit and the examples; copy the pattern.
5. **Fake-Kobo install test:** `tools/test-install.sh` installs into a fake Kobo in a temp dir and covers the mod staging and `--uninstall`.
6. **Device checklist** (user present, backups first): refresh list shows the app; each tap answers within a second or two with a toast; texts fit in English and Vietnamese (switch language, refresh); failure paths (no Wi-Fi, empty input, missing file) end in the `failed` toast, never a crash; eject works right after (nothing held open); a rescan shows new files; battery and sleep: let the Kobo sleep during a background job and confirm it ends cleanly.

### How an app talks to the drawer

The drawer renders; the app decides. The app never draws and never prints text for a dialog. The model is deliberately small:

1. **Short scripts.** Each tap runs one action script (`appsctl drawer event`). It does a quick job or starts a detached background job, then answers with at most a few JSON lines.
2. **State is re-read.** The screen's values come from the app's `state` action (one JSON object of the `vars` in `ui.json`). The mod re-reads it every 5 seconds while the screen is open, so a background job shows progress by changing files that `state` reads. There is no long-running backend process and no JSON-lines protocol: that design was dropped (DECISIONS.md).
3. **Signals.** A script answers with `{"toast":"id"}` or `{"dialog":"id"}` naming something declared on the screen, and may add requests for the host (below).

| Signal (one JSON object per line) | Effect |
|---|---|
| `{"toast":"id"}` | the host shows the toast declared in `ui.json` `toasts` |
| `{"dialog":"id"}` | the host shows the declared dialog; OK runs its `action` |
| `{"request":"rescan","mode":"quick"}` or `"full"` | Nickel rescans the library (`quick` is the default mode) |
| `{"request":"wifi"}` | Nickel's Wi-Fi connect flow, then the host retries |

**Requests.** A script asks for a host service by printing the request line, or a finished background job puts `"requests":[{"request":"rescan","mode":"quick"}]` in its `state` object (the state script must hand each request over once, for example by deleting a marker file when it prints it; see `apps/library` and `examples/imageview`). The app must list the services it uses in `app.json`:

```json
"requests": ["rescan", "wifi"]
```

`appsctl` drops a request the app did not declare (and logs it), then passes the rest to the mod in the reply (`requests`). `common.sh` has `emit_request rescan quick` / `emit_request wifi`. The manifest field `rescan` on an action (run the rescan after a successful action) still works; requests are for the cases where the script or a finished job knows better.

**Newer-host check.** An app that needs a newer NickelDrawer sets `"min_host": "1.2.0"` in `app.json`; on an older host its drawer row reads "Needs a newer version" (a status line only; the app still runs).

Rule of thumb: if you are about to `echo` a sentence, declare it as a toast in `ui.json` instead.

### Design a native screen

A native screen is a `ui.json` next to `app.json` (optionally more screens in `screens/<id>.json`). You can write and check one without a Kobo:

1. **Write it.** Start from `apps/todo/ui.json` (the smallest) or `apps/localsend/ui.json` (texts as `"@ui.key"` references into `locales/*.json`). Declare every variable a screen uses in `vars`, give buttons the id of an action from `app.json`, keep one primary button per screen, and keep labels free of `:`.
2. **Validate it.** `go run ./cmd/appsctl validate apps/myapp` (or `build/appsctl-host`) checks `app.json`, the locales and the `ui.json`. Each problem is printed as `path: [code] message`; the codes are listed in [spec/ui-json.md](spec/ui-json.md) section 7.
3. **Preview it.** `go run ./cmd/ndpreview serve apps/myapp --locale vi` (or `build/ndpreview-host`) prints `http://127.0.0.1:8790/`. The page draws the screen at the Libra 2 size (1264 x 1680, shown at 50 %, a link switches to 100 %) in black, white and greys with the same row, button and header sizes as the kit, and no animation. It re-reads your files on every reload, so edit, save, refresh.
   - The sidebar picks the screen and language, switches to 16 grey levels, and edits the variables (the `vars` of `ui.json`, or a file you give with `--state state.json`). Use it to try long lists, long Vietnamese strings and empty states.
   - Long screens and lists page. The left and right arrow keys, or the on-screen arrows, stand in for the page-turn buttons.
   - Tap a button, row or toggle: the sidebar shows the JSON line the host would send to the backend and the app.json action that would run. A button with `confirm` opens the confirm dialog first. Screens can declare `dialogs` and `toasts`; the sidebar has links to show them.
   - `--exec` also runs the action script, with `sh`, inside a temporary folder that stands in for the Kobo (`KOBO_ROOT`, `HOME`, `ND_DATA_DIR`, `ND_EVENT`). Scripts that read `/mnt/onboard` paths fail on a computer and say so in the output; give your scripts path overrides as in "Test on the Mac".
   - The server listens on `127.0.0.1` only and refuses other `Host` headers. The QR code is a placeholder that shows the layout; it does not encode the address.
4. **Render a picture.** `go run ./cmd/ndpreview render apps/myapp --out screen.png --screen main --locale vi [--state state.json] [--grey 16] [--font /path/font.ttf]` writes a 1264 x 1680 PNG for reviews and bug reports. Text needs a TrueType or OpenType font: without `--font` the tool tries common system fonts (Arial Unicode, DejaVu, Georgia) and otherwise draws the shapes only and prints a warning. Pick a font that has Vietnamese letters.

The preview lays out in the page area the device gives a screen: 1264 x 1426 px below Nickel's 128 px status bar, with a 126 px tab bar under it (measured on a Libra 2, firmware 4.38), so page counts and the footer position equal the device. Both bars are drawn as plain labelled strips. Icons come from the repository's Tabler PNGs (`host/icons/ui`, found by walking up from the working directory; `--icons DIR` overrides; a missing folder prints a warning and the icons are left out). With `--locale en`, `--state preview-state.json` uses `preview-state.en.json` when that file sits next to it, so English previews do not show Vietnamese sample data. `ndpreview drawer-list`, `drawer-grid` and `tweaks` render the drawer's list view, grid view and the Tweaks modal with sample content (`--locale en|vi --out FILE`), and `tools/preview-all.sh` renders every app screen, the settings screens and these three views for both languages into `build/preview/<locale>/`.

The preview is a layout check, not Nickel: real fonts and the real refresh behaviour differ. Test on the device after the native drawer ships (plan gates G1 and G4).

### Worked examples

- [To-do](../apps/todo/README.md): a shipped app and the model for a small one: a `field` for the Nickel keyboard, a `list` with tap and row menu, a button that is disabled through a `$var`, declared toasts, tasks in `data/todo.json` written atomically by a tiny Go helper, and a `ui.json` you can preview with `ndpreview`.
- [UI Kit](../apps/uikit/README.md): every `ui.json` component on its own page (a grouped home, a states matrix, a live sample and a readout of the last event, 18 pages and a picture viewer); hidden by default. To see it in the drawer, tap the gear (⚙) in the drawer header and turn on **UI Kit** under **Hidden apps**; no file needs editing. Copy its `ui.json` when you need an example of a component.
- [Image viewer](../examples/imageview/README.md): a background job that builds `.cbz` albums so Kobo's own reader shows the photos, reports through `state`, asks the host for a rescan, and uses a Go helper for what BusyBox lacks.

### Nested views and back

NickelDrawer has one back stack, like Nickel's own pages. Design your screens to fit it.

```
More (Nickel) → Apps drawer → your app's main screen → sub-screen → (detail)
                                   ←                 ←             ←
```

**Rules**

- **"←" goes back exactly one level** and restores the previous view as the user left it: same page, same list/grid mode, fresh data. You never draw your own "Back" button in the content.
- **The main screen's "←" returns to the drawer.** A tap on a bottom tab (Home, Books, …) leaves your app entirely; that is Nickel's normal behaviour, don't fight it.
- **Keep it shallow: at most two levels inside your app** (main → sub-screen → detail). If you need a third level, the design is too deep: merge screens or use a popup menu.
- **No cycles.** A sub-screen never pushes its own parent again (A → B → A). Use "←" instead.
- **Each screen has its own short title** (≤ 24 characters, no app-name prefix) and its status, if any, on the right of the header.
- **Leaving a screen never cancels work.** Background jobs keep running; when the user comes back, the screen shows the current status.
- **A bottom-tab tap clears the stack.** The drawer drops every stacked screen and its cached state; reopening Apps starts fresh at the drawer's first page. Only preferences (list/grid) survive. Don't rely on screen state surviving a tab tap: keep anything important in your app's `data/` and return it from your `state` action.
- **Define the flow up front.** Your screens and their `push` links *are* the flow. `appsctl validate` checks it when you build and install: a `main` screen must exist, every screen must be reachable from `main`, at most two levels below it, no cycles and no push to a missing screen. Sketch the flow first (main → which sub-screens → which details), then write the screens.

**Which container to use**

| Need | Use | Example |
|---|---|---|
| A place the user stays: state, lists, the main buttons | **Screen** (`screens/<id>.json`) | LocalSend receiving screen |
| Details or a group of options for one thing | **Sub-screen**, opened with `push` | Library → "Duplicates" |
| Choose one of several actions on an item | **Popup menu** (long-press) | Duplicate → Move to trash / Keep |
| Confirm something destructive, or show one result | **Dialog** (OK / Cancel) | "Remove this app?" |
| Tell the user something finished | **Toast** (one line, disappears) | "3 books added" |
| Browse many pictures as a grid, then view one | **Gallery** on `main` and **viewer** on a pushed screen (both bound to one array from `state`; header icon `actions`) | Wallpaper ([spec/wallpaper-gallery.md](spec/wallpaper-gallery.md)) |

- **Never stack modals:** no dialog on top of a popup, no navigation from inside a dialog, at most one dialog at a time.
- **Don't hide screens behind long-press only.** Long-press is a shortcut; every screen must also be reachable by a visible row or header action.
- **After an action, stay on the screen** and let it refresh. Show a toast or a short dialog for the result; don't push a "result screen".

**How to declare it**

Open a sub-screen with a header action that has `push` (the target screen id; `appsctl validate` checks that it exists):

```json
{
  "title": "@screen.main.title",
  "action": { "label": "@action.duplicates", "push": "duplicates" },
  "body": [ … ]
}
```

Sub-screens read the same `state` action as the main screen, so bindings stay consistent across the stack. Preview the whole flow with `ndpreview serve`: the arrow keys flip pages and "←" pops, exactly like on the device.

## Quick start: `appsctl new`

```sh
go run ./cmd/appsctl new my-app --name "My App" [--with-ui] [--dir apps]
```

This creates `my-app/` from `templates/app`: a valid `app.json`, `bin/run.sh` (answers with a toast through `report` from `common.sh`), `locales/en.json` and `vi.json`, and a README that explains how to declare `network` and `storage`. With `--with-ui` it also writes a `ui.json` (with the declared toast) and a `state` action. It refuses an existing folder, validates the result with `appsctl validate` and prints the next steps (validate, `ndpreview serve`, install). The rest of this guide explains each file.

## 1. Folder layout

```
hello/
├─ app.json            manifest (required)
├─ bin/hello.sh        script the action runs
├─ locales/
│  ├─ en.json          English messages printed by scripts
│  └─ vi.json          Vietnamese translation
└─ data/               created at run time: config, tokens, logs
```

On the Kobo the folder lives at `/mnt/onboard/.adds/apps/<id>/`. The folder name must equal the app `id`.

## 2. The manifest (app.json)

```json
{
  "schema": 1,
  "id": "hello",
  "name": "Hello",
  "version": "1.0.0",
  "description": "Says hello and shows the Wi-Fi address",
  "actions": [
    {
      "id": "greet",
      "label": "Say hello",
      "run": "bin/hello.sh",
      "output": "show",
      "timeout_ms": 5000,
      "wifi": false,
      "rescan": "none"
    }
  ]
}
```

- `schema` is always `1`.
- `id` is lowercase letters, digits and `-`, 2 to 31 characters.
- `name` is 1 to 24 characters. `label` is 1 to 32. Neither may contain `:`.
- `version` is semver, for example `1.0.0`.
- An app has 1 to 12 actions. Each action runs one script.
- `output: "show"` runs the script and waits for it (at most 30 s); only its JSON signals reach the screen, any other output is dropped. `output: "none"` starts the script detached and returns at once.
- `timeout_ms` (only with `show`) is 500 to 9999. Default is 5000.
- `wifi: true` makes the drawer connect Wi-Fi first: without an IP, `appsctl` answers `needs_wifi` and the mod connects and retries once.
- `rescan` is `none`, `quick` or `full`. The library rescans only if the script exited with 0.
- `requests` (optional) lists the host services scripts may ask for: `rescan`, `wifi`.
- `min_host` (optional) is the oldest NickelDrawer version the app works with.
- Unknown fields are ignored with a warning.

The `name`, `description` and `label` values are the English source text.

## 3. Locales

English lives in the manifest (for `name`, `description`, `label`) and in `locales/en.json` (for script messages). Translations are `locales/<lang>.json`, for example `vi.json` or `pt_BR.json`.

Keys:

| Key | Used for |
|---|---|
| `name` | app name |
| `description` | app description |
| `actions.<id>.label` | action label |
| `msg.<key>` | a message printed by your scripts |

Rules:

- A placeholder looks like `{name}`. A translation must use the same placeholders as English. `appsctl validate` checks this.
- `\n` is allowed for multi-line dialog text.
- The `:` and length rules apply to translated labels too.
- Missing keys fall back to English, key by key.
- Locale comes from `CurrentLocale` in `/mnt/onboard/.kobo/Kobo/Kobo eReader.conf`. Lookup order: `pt_BR.json`, then `pt.json`, then English.

Scripts never print literal text. They call `appsctl msg`:

```sh
appsctl msg --app hello msg.greeting name="$name" ip="$ip"
```

The drawer answers in the current language (`CurrentLocale`); change it and reopen the drawer.

## 4. Script contract

- Scripts run as root, from `/`, with a minimal environment. Start with `cd "$(dirname "$0")/.."` or use absolute paths. Source `/mnt/onboard/.adds/koboapps/lib/common.sh`.
- Exit `0` for success, non-zero for failure. A failed action shows a short "did not work" message; say why with a declared `failed` toast and the reason in `data/log.txt`.
- **Return at once.** The drawer waits at most 30 s and `timeout_ms` is at most 9999. Keep scripts well under a second where you can.
- **Long work goes to the background.** Start the job with all file descriptors redirected (`start_detached`), return at once with a "started" toast, and let the `state` action report progress and the result. `apps/library/bin/lib.sh` is the model: it writes a pid file and output to `/tmp`, and `state.sh` reads them.
- **Hold no files open on `/mnt/onboard`** after the script exits. Write temporary files to `/tmp` and copy them to `data/` at the end. An open file blocks USB ejecting.
- **Write only to your own `data/` folder** and to the folder your app declares (for example `/mnt/onboard/LocalSend`).
- **Secrets** (tokens, passwords) live only in `data/`, mode 600. Never print them. Never put them in `log.txt`.
- Log with `app_log "$APP_DIR" "message"`. It appends to `data/log.txt`.

Helpers from `common.sh`:

| Function | What it does |
|---|---|
| `current_ip` | prints the first non-loopback IPv4 address |
| `wait_for_wifi N` | waits up to N seconds for an IP; prints it, returns 0 or 1 |
| `app_log DIR MSG...` | appends a timestamped line to `DIR/data/log.txt` |
| `report TOAST APP KEY [k=v...]` | drawer: prints `{"toast":"TOAST"}`; command line: the localized message |
| `report_failure`, `fail_with`, `run_quick_command` | the same for failures and quick commands |
| `emit_request rescan [quick\|full]`, `emit_request wifi` | prints a host request (declare it in `requests`) |
| `start_detached CMD...` | runs CMD in the background with every fd closed |

## 5. A complete example

`hello/app.json`: as in section 2.

`hello/locales/en.json`:

```json
{
  "msg.greeting": "Hello, {name}!\nIP: {ip}",
  "msg.no_wifi": "No Wi-Fi. Turn it on and try again."
}
```

`hello/locales/vi.json`:

```json
{
  "name": "Xin chào",
  "description": "Chào bạn và hiện địa chỉ Wi-Fi",
  "actions.greet.label": "Chào",
  "msg.greeting": "Xin chào, {name}!\nIP: {ip}",
  "msg.no_wifi": "Chưa có Wi-Fi. Bật Wi-Fi rồi thử lại."
}
```

`hello/ui.json` (the screen; the toast text may use state variables):

```json
{
  "kit": 1,
  "start": "main",
  "vars": { "ip": "" },
  "screens": {
    "main": {
      "title": "Hello",
      "body": [ { "kv": [["IP", "$ip"]] } ],
      "footer": [ { "button": "Say hello", "action": "greet", "style": "primary" } ],
      "toasts": { "greeted": "Hello, reader! IP: $ip", "no_wifi": "No Wi-Fi", "failed": "That did not work" }
    }
  }
}
```

`hello/bin/hello.sh` (make it executable) answers with a toast; `hello/bin/state.sh` prints `{"ip":"…"}` with `printf '{"ip":"%s"}\n' "$(current_ip)"` and is declared as the `state` action in `app.json`:

```sh
#!/bin/sh
# Greets the user. Fast, so it runs in the foreground.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APP_ID=hello

ip=$(current_ip)
if [ -z "$ip" ]; then
    report_failure no_wifi "$APP_ID" msg.no_wifi
fi

app_log "$APP_DIR" "greeted from $ip"
report greeted "$APP_ID" msg.greeting name=reader ip="$ip"
```

## 6. Validate

```sh
make build
build/appsctl-host validate path/to/hello
```

`validate` checks the manifest, that `run` files exist and are executable, and that translations match English. Exit code 1 means an app is invalid. Fix every message before you go on.

## 7. Test on the Mac

The scripts need `/mnt/onboard`, so test the logic with a fake root:

- Run `shellcheck` and `sh -n bin/*.sh`.
- Keep real logic in a Go or Python program that you can unit test, and keep the shell script thin.
- Run `appsctl drawer model --apps-dir <dir>`, `appsctl drawer screen <app>` and `appsctl drawer event <app> <action>` and read the JSON the mod would get.
- `tools/test-install.sh` runs the installer against a fake Kobo in a temp dir. Run it before touching a real device.

## 8. Test on the device

Read the safety rules in [CONTRIBUTING.md](../CONTRIBUTING.md) first. Short version:

1. Back up `.adds/apps` before you write to the Kobo.
2. Copy the `hello` folder to `/mnt/onboard/.adds/apps/hello/`. Do not leave your shell's working directory on the Kobo volume.
3. Eject the Kobo.
4. Open More, Apps, and tap **Refresh list** (System section). Your app appears as a row.
5. Tap it, then tap **Say hello**. If it fails, read `data/log.txt` in the app folder.

## 9. Package

Zip the app folder so the folder itself is at the top level:

```sh
zip -r hello-1.0.0.zip hello -x 'hello/data/*'
```

Do not ship `data/`. It holds the user's tokens and logs. If you need a default config, ship `config.example.json` next to `app.json` (outside `data/`) as ghsync does.

Do not include Kobo binaries or firmware files in a package.

# Build your first NickelDrawer app

An app is a folder with a manifest and a few scripts. Copy it to the Kobo and it shows up in the menu. This guide builds a small "Hello" app from start to finish.

The full format rules are in [spec/app-format.md](spec/app-format.md). This guide is the short path.

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
- An app has 1 to 8 actions. Each action runs one script.
- `output: "show"` prints the script's stdout in a dialog. Keep it under about 12 short lines. `output: "none"` runs the script in the background and shows nothing.
- `timeout_ms` (only with `show`) is 500 to 9999. Default is 5000.
- `wifi: true` asks Nickel to connect to Wi-Fi first. It does not wait. Your script must wait for an IP itself (see `wait_for_wifi` below).
- `rescan` is `none`, `quick` or `full`. The library rescans only if the script exited with 0.
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
appsctl msg --app hello greeting name="$name" ip="$ip"
```

The menu is built in the current language. After you change the Kobo language, tap the refresh item once.

## 4. Script contract

- Scripts run as root, from `/`, with a minimal environment. Start with `cd "$(dirname "$0")/.."` or use absolute paths. Source `/mnt/onboard/.adds/koboapps/lib/common.sh`.
- Exit `0` for success, non-zero for failure. A failure is shown to the user.
- **10 second limit.** Nickel's `cmd_output` must return in under 10 s. Keep `timeout_ms` at or below 9999.
- **Long work goes to the background.** Start the job with all file descriptors redirected, return at once, and let a second action ("Status") report the result. `apps/ghsync/bin/sync.sh` is the model: it writes a pid file and output to `/tmp`, starts the job with `</dev/null >/dev/null 2>&1 &`, and the status action reads the result.
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

`hello/bin/hello.sh` (make it executable):

```sh
#!/bin/sh
# Shows a greeting and the current IP. Fast, so it runs in the foreground.

APP_DIR=$(cd "$(dirname "$0")/.." && pwd)
. /mnt/onboard/.adds/koboapps/lib/common.sh

APPSCTL=/mnt/onboard/.adds/koboapps/bin/appsctl

ip=$(current_ip)
if [ -z "$ip" ]; then
    "$APPSCTL" msg --app hello no_wifi
    exit 1
fi

app_log "$APP_DIR" "greeted from $ip"
"$APPSCTL" msg --app hello greeting name=reader ip="$ip"
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
- Run `appsctl gen-menu --apps-dir <dir> --out /tmp/menu` and read the generated menu lines.
- `tools/test-install.sh` runs the installer against a fake Kobo in a temp dir. Run it before touching a real device.

## 8. Test on the device

Read the safety rules in [CONTRIBUTING.md](../CONTRIBUTING.md) first. Short version:

1. Back up `.adds/apps` and `.adds/nm` before you write to the Kobo.
2. Copy the `hello` folder to `/mnt/onboard/.adds/apps/hello/`. Do not leave your shell's working directory on the Kobo volume.
3. Eject the Kobo.
4. Tap the refresh item in the menu. Your action appears as "Hello · Say hello".
5. Tap it. If it fails, read `data/log.txt` in the app folder and `.adds/koboapps/data/errors.txt`.

## 9. Package

Zip the app folder so the folder itself is at the top level:

```sh
zip -r hello-1.0.0.zip hello -x 'hello/data/*'
```

Do not ship `data/`. It holds the user's tokens and logs. If you need a default config, ship `data/config.example.json` as ghsync does.

Do not include Kobo binaries or firmware files in a package.

## Coming later: native UI

Phase 3 adds a native drawer and an optional `ui.json` file that describes screens (lists, forms, buttons). Your v1 manifest stays valid: the drawer will run the same actions. `ui.json` will be optional.

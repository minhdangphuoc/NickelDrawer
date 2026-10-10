# Examples

Worked examples for the [developer guide](../docs/guide/developer-guide.md) and the [design guidelines](../docs/APPS.md). Each is a complete app folder.

| Example | Teaches |
|---|---|
| [`../apps/todo/`](../apps/todo/README.md) | The shipped To-do app is the reference for a small native screen: `field`, `list` with a row menu, `button` with a confirm and a `$var` that disables it, declared toasts, a tiny Go helper that writes `data/todo.json` atomically, English + Vietnamese. One source of truth: it lives under `apps/`, not here. |
| [`../apps/uikit/`](../apps/uikit/README.md) | The UI Kit app (hidden by default): every component with sample content. |
| [`imageview/`](imageview/) | The background pattern (start the job, return, follow it through `state`, ask the host for a rescan with a `requests` entry), reusing Kobo's native reader instead of drawing, and a small Go helper (`mkcbz`) cross-compiled for ARMv7 when BusyBox lacks a tool. |

## Try one

Validate (from the repository root):

```sh
go run ./cmd/appsctl validate apps/todo examples/imageview
```

Install on a Kobo (follow the safety steps in the guide, section "Install on a Kobo", back up
first, and do not leave your shell's working directory on the Kobo volume):

```sh
cp -R examples/imageview /Volumes/KOBOeReader/.adds/apps/imageview
```

Then eject, open the Apps drawer and tap the refresh button in its header to rescan the apps. The `imageview` example needs its `bin/mkcbz` built
first; see its README.

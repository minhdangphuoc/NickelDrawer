# Examples

Worked examples for the [app developer guide](../docs/APPS.md). Each is a complete app folder.

| Example | Teaches |
|---|---|
| [`todo/`](todo/) | A pure `sh` drawer app with a native screen: `toggle`, `field`, `list` with a row menu and `button`s, declared toasts, data in `data/`, atomic file rewrites, English + Vietnamese. |
| [`imageview/`](imageview/) | The background pattern (start the job, return, follow it through `state`, ask the host for a rescan with a `requests` entry), reusing Kobo's native reader instead of drawing, and a small Go helper (`mkcbz`) cross-compiled for ARMv7 when BusyBox lacks a tool. |

## Try one

Validate (from the repository root):

```sh
go run ./cmd/appsctl validate examples/todo examples/imageview
```

Install on a Kobo (follow the safety steps in the guide, section "Test on the device", back up
first, and do not leave your shell's working directory on the Kobo volume):

```sh
cp -R examples/todo /Volumes/KOBOeReader/.adds/apps/todo
```

Then eject and open the Apps drawer; **Refresh list** (System section) rescans the apps. The `imageview` example needs its `bin/mkcbz` built
first; see its README. Copy `data/` only if you want to keep an existing list.

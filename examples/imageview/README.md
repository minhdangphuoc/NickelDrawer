# Image viewer (v1 menu app)

Shows photos by packing them into a `.cbz` comic album and letting **Kobo's own reader** open it.
It draws no pixels itself.

## Why reuse the native reader

A custom viewer would have to reinvent what Nickel's comic reader already does well on e-ink:

- **Page-turn buttons** and tap zones that match the rest of the device.
- **Pinch-zoom and pan**, tuned for the panel.
- **Refresh control**: Nickel's own full and partial refresh rules, ghost clearing, dark mode.
- **Position memory**, library covers, collections, sleep screen handling and battery behaviour.
- Zero extra code to run, so nothing to break after a firmware update.

For an app that is mostly a script, the cheapest correct design is to produce a file the native
app understands, then ask the library to rescan.

## Use

1. Copy photos over USB into `/mnt/onboard/Images/` (png, jpg, jpeg). Folders inside it become
   separate albums; loose files become the `Images` album.
2. Tap **Image viewer · Make albums**. It returns at once; the work runs in the background
   (building albums can take longer than the 10 s a menu dialog allows).
3. A minute later tap **Image viewer · Status**. It shows the result and rescans the library.
4. Open the album from your library (`Images/Albums/<folder>.cbz`).

Albums that are newer than all their images are skipped, so running it again is cheap.

## The packer: bin/mkcbz

BusyBox on Kobo ships `unzip`, `tar` and `gzip` but, as far as we know, **no `zip` applet**
(not verified on every firmware: run `busybox --list | grep -x zip` over telnet or in a test
action). So this example includes a tiny Go helper, `cmd/mkcbz`. It uses `archive/zip` with the
Store method (no compression: photos are already compressed, and the Kobo CPU stays idle),
sorts pages in natural order (`page2` before `page10`), writes to a `.tmp` file and renames it.

Build it for the Kobo (ARMv7) from the repository root:

```sh
mkdir -p examples/imageview/bin
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
  -o examples/imageview/bin/mkcbz ./examples/imageview/cmd/mkcbz
go test ./examples/imageview/...
```

The binary is gitignored (`examples/.gitignore`); `make-albums` tells you if it is missing.

## Files

```
app.json                 two actions: make-albums, status
bin/make-albums.sh       starts the worker, returns at once
bin/build-albums.sh      background worker (not an action)
bin/status.sh            result + library rescan
bin/lib.sh               shared paths and helpers
bin/mkcbz                built helper (not committed)
cmd/mkcbz/               Go source and test
locales/en.json, vi.json
```

## Running on the Mac

Paths come from `IMAGEVIEW_*` variables (listed at the top of `bin/lib.sh`):

```sh
tmp=$(mktemp -d); go build -o "$tmp/mkcbz" ./examples/imageview/cmd/mkcbz
mkdir -p "$tmp/Images/Holiday" && cp some.jpg "$tmp/Images/Holiday/"
export IMAGEVIEW_IMAGES_DIR=$tmp/Images IMAGEVIEW_STATE_DIR=$tmp/state IMAGEVIEW_MKCBZ=$tmp/mkcbz
export IMAGEVIEW_APPSCTL=$PWD/build/appsctl-host IMAGEVIEW_APPS_DIR=$PWD/examples
examples/imageview/bin/make-albums.sh; sleep 2; examples/imageview/bin/status.sh
```

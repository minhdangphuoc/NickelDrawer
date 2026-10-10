# Apps

Apps are separate from the NickelDrawer drawer. A drawer release contains only the drawer (the host tool and the Nickel mod); each app is its own download and is installed one by one, so you only put on your Kobo what you use. An app never changes the drawer, and a drawer update never touches an installed app or its `data/` folder.

Tiếng Việt: [README.vi.md](README.vi.md).

## The apps

| App | Folder | What it does | Version | Hidden |
|---|---|---|---|---|
| [GitHub](ghsync/README.md) | `apps/ghsync/` | Sync books from a private GitHub repo (add-only, never deletes) | 0.3.0 | no |
| [Library](library/README.md) | `apps/library/` | Fix the language of Vietnamese books and find duplicate books | 0.2.0 | no |
| [LocalSend](localsend/README.md) | `apps/localsend/` | Receive books and files from your phone with LocalSend | 1.2.0 | no |
| [To-do](todo/README.md) | `apps/todo/` | A small to-do list that stays on your Kobo | 1.0.1 | no |
| [UI Kit](uikit/README.md) | `apps/uikit/` | Every NickelDrawer component with sample content, for developers and testers | 2.1.0 | yes (developers only) |
| [Wallpaper](wallpaper/README.md) | `apps/wallpaper/` | Use your own pictures as the sleep screen | 0.3.0 | no |

A hidden app does not show in the drawer until you switch it on in the drawer's settings (the gear in its header). UI Kit is a demo of every component for developers and testers.

## Install an app

The drawer must be installed first (see the [main README](../README.md#install)). Then choose one way. Back up the Kobo's `.adds` folder first.

**A. From the release.** Download `app-<id>-<version>.zip` from the [release](../../../releases) (for example `app-todo-1.0.1.zip`), unzip it, and merge the `.adds` folder inside onto the root of the Kobo drive. Hidden folders: Cmd+Shift+. in macOS Finder, "show hidden files" on Windows and Linux. **macOS Finder: if it asks whether to Replace a folder that already exists, do not.** Hold Option and choose **Merge**. Each zip holds only its own app under `.adds/apps/<id>/`, with no `data/` folder. Eject the Kobo; the app appears in the drawer after a rescan (the drawer's refresh button) or the next boot.

**B. From a checkout** (macOS or Linux, Kobo connected):

```sh
make check build dist                      # builds dist/apps/app-<id>-<version>.zip for every app
tools/install-to-kobo.sh --dry-run --app todo   # show what would happen
tools/install-to-kobo.sh --app todo        # install one app; repeat --app for more
tools/install-to-kobo.sh --apps all        # install every app
```

The installer unpacks the zips from `dist/apps/`, backs up the apps it replaces to `backups/<date-time>/` and keeps each app's `data/`. Without `--app`/`--apps` it installs the drawer only and leaves the apps on the Kobo alone. You can also copy a folder from `dist/apps` by hand after unzipping it; plain `apps/<id>/` in the source tree lacks the built binaries and is not installable.

## Update an app

Install the new zip the same way (A or B). The app's files are replaced; its `data/` (settings, tokens, lists) is kept. You can also update from the drawer: **⚙** → **Updates** → **Apps** (see [docs/UPDATES.md](../docs/UPDATES.md)). It downloads the app's zip from its own release (`<id>-v<version>`), checks the signed catalog, size and sha256, and swaps the folder; `data/` is moved, never copied. The same screen installs apps that are not on your Kobo yet. The drawer update (**Check** in the settings) updates only the drawer.

## Remove an app

Long-press the app in the drawer and choose **Remove**, or delete its folder `.adds/apps/<id>` on the Kobo drive (and `data/` inside it if you also want its settings gone). Your books are never touched.

## Adding an app to this repository

Every app lives in `apps/<id>/` with a `README.md` (install, update and remove sections included) and a row in the table above and in [README.vi.md](README.vi.md). `make dist` packs every folder that has an `app.json` into its own zip; `appsctl validate --apps-dir apps` checks them. See [docs/APPS.md](../docs/APPS.md).

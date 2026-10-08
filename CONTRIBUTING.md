# Contributing to NickelDrawer

Thank you for helping. This file tells you how to set up, what we expect, and how to send changes.

## Setup

You need:

- Go 1.24 or newer
- `make`, `zip`, `shellcheck`
- Python 3 with `fonttools` and `markdown` (only for the helper tools)

Common commands:

```sh
make check          # gofmt check, go vet, go test
make build          # ARMv7 binaries for the Kobo, plus host tools
make dist           # dist/ tree and dist/koboapps-<version>.zip
tools/test-install.sh   # runs the installer against a fake Kobo
```

Run `make check build dist` before you open a pull request. You do not need a Kobo for most work.

## Code rules (Clean Code)

- Names say what a thing is. Booleans read as questions (`isRunning`, `hasWifi`).
- Functions are small and do one job.
- One level of abstraction per function.
- No duplicated logic. Shared shell code goes in `host/lib/common.sh`.
- Check every return value. Error messages say what broke and how to fix it. Never swallow an error.
- Comments explain why, not what. Delete commented-out code.
- Match the surrounding style. Format with `gofmt`, `clang-format`, `shfmt`, `black`.
- Leave touched code cleaner, but do not refactor unrelated code in the same change.

AI coding agents: read [CONTRIBUTING.md](CONTRIBUTING.md). It has the full rules and the Kobo safety rules.

## Tests

- Every bug fix comes with a test or a validation case.
- Every new feature comes with tests.
- Shell scripts must pass `shellcheck` and `sh -n`.
- Never make a test depend on a real Kobo.

## Kobo safety

- Back up any device file before you change it.
- Do not leave your shell's working directory on the Kobo volume. It blocks ejecting.
- Run a new native mod build on a device only after the failsafe and uninstall drill.
- Native mods use C and Qt only, check every libnickel symbol with `dlsym`, and disable themselves if a symbol is missing.

## Never commit

- Kobo binaries: `libnickel.so`, firmware zips, `KoboRoot.tgz` from Kobo. Symbol names and your own notes are fine.
- Personal device data: `backups/`, `KoboReader.sqlite`, `data/` folders, logs, tokens, serial numbers.
- Secrets of any kind. Signing keys never go in the repo or CI.

## Translations

English is the source. English and Vietnamese are maintained by the project. Other languages come from the community.

To add a language:

1. Copy `locales/en.json` of the app (or `host/locales/en.json` for host strings) to `<lang>.json`. Use the Kobo locale code: `de`, `pt_BR`, `fr`.
2. Translate the values. Keep the keys, the `{placeholders}` and the `\n` breaks.
3. Keep labels within the length limits and without `:`.
4. Run `appsctl validate`. It checks placeholders.
5. Open a pull request with the "App or translation" template.

Right-to-left languages (Arabic, Hebrew) are not supported yet. See [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md).

## Commits and pull requests

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(ghsync): add retry on network error
fix(appsctl): reject labels that contain a colon
docs: explain the background script pattern
```

Types: `feat`, `fix`, `docs`, `test`, `refactor`, `build`, `ci`, `chore`.

Keep a pull request to one topic. Describe what changed and how you tested it. Link the issue if there is one.

## Writing an app

See [docs/APPS.md](docs/APPS.md).

## License

By contributing you agree that your work is released under the [MIT License](LICENSE).

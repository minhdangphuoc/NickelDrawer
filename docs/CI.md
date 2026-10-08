# CI

Workflow: `.github/workflows/ci.yml`. No secrets are needed and nothing is signed in CI; release signing stays on the maintainer's Mac (`docs/UPDATES.md`).

## What runs

| Job | Trigger | Checks |
|---|---|---|
| `build` | push, PR | Go version from `go.mod`; `make check build dist` (gofmt, vet, tests, ARM cross-build, dist tree); `tools/test-install.sh` (installer against a fake Kobo tree); `tools/test-symcheck.sh` |
| `symcheck` | push, PR | Downloads the 5 Libra 2 firmware zips listed in `re/firmware.lock` from Kobo's CDN into the runner temp dir, verifies the zip and libnickel sha256, extracts only `libnickel.so.1.0.0`, dumps its exports with `llvm-nm`, runs `tools/symcheck.py`, then deletes every binary and dump |
| `firmware-watch` | weekly (Mon 06:17 UTC), manual dispatch | Runs `tools/fw-watch.sh`; exit 10 (new firmware) becomes a warning annotation, not a failure |

Kobo files are never uploaded as artifacts, cached, or committed. Only symbol names and our own notes live in the repo.

## symcheck locally

```sh
python3 -I tools/symcheck.py                         # re/notes/symbols-required-draft.yaml x the builds it lists
python3 -I tools/symcheck.py --fw-dir DIR            # DIR/<version>/exports-mangled.txt
python3 -I tools/symcheck.py --symbols FILE --builds 4.38.23828,4.38.23697 --quiet
bash tools/test-symcheck.sh                          # fixtures in tools/testdata/symcheck/
```

Prints a matrix (symbol x build). Exit 0: every `required-for-drawer` symbol exists in every build (optional and fallback-only misses are reported only). Exit 1: a required symbol is missing. Exit 2: bad input (missing file or dump).

The default `--fw-dir` is `~/.cache/nickeldrawer/symbols` (`NICKELDRAWER_CACHE` overrides the cache root). A missing dump there is regenerated with `re/scripts/fetch-fw.sh` (needs Xcode `llvm-nm`); pass `--no-regenerate` to forbid that. The repo's `re/symbols/<fw>/exports.txt` files are demangled, so symcheck needs the mangled dumps from the cache.

## fw-watch locally

```sh
tools/fw-watch.sh            # query the upgrade API with a placeholder serial
tools/fw-watch.sh --fetch    # also fetch + dump the new build into ~/.cache/nickeldrawer
```

Exit 0: nothing newer than the newest version in `re/firmware.lock`. Exit 10: new firmware; the URL and the fetch and symcheck commands are printed. A 5.x URL (Qt6 upgrade) is flagged as unsupported. Next steps are in `docs/WORKFLOW.md` (firmware-watch).

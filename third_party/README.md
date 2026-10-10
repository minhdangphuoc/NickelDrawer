# Third-party sources

| Path | Upstream | Commit | License |
|---|---|---|---|
| `localgo/` | https://github.com/bethropolis/localgo | `f10bf9eafc9ac74ba5a8e3d9a9787edc3a7fe6af` | MIT |
| `NickelHook/` | https://github.com/pgaskin/NickelHook | `bcd90ba94d90d22562fb64ef2920ed3323c1dae5` | MIT (compatible with our MIT distribution; keep its `LICENSE`) |

## Tabler Icons

| Path | Upstream | Version | License |
|---|---|---|---|
| `tabler-icons/` | https://github.com/tabler/tabler-icons (npm `@tabler/icons`) | `3.49.0` (tag `v3.49.0`; npm tarball sha512 `l+GnSkFE...c6lw==` verified) | MIT, see `tabler-icons/LICENSE` |

Only the outline SVGs we use are vendored in `tabler-icons/svg/`: `adjustments-horizontal`, `apps`, `arrow-left`, `books`, `brand-github`, `check`, `chevron-left`, `chevron-right`, `cloud-download`, `download`, `layout-dashboard`, `layout-grid`, `layout-list`, `list-check`, `photo`, `photo-off`, `plus`, `refresh`, `send`, `settings`, `toggle-left`, `toggle-right`, `x`. `cmd/iconsgen` rasterizes them to the PNGs under `apps/*/icon.png` and `host/icons/`.

## Go modules (build tools only, not shipped to the device)

| Module | Upstream | License | Used by |
|---|---|---|---|
| `github.com/srwiley/oksvg` | https://github.com/srwiley/oksvg | BSD-3-Clause | `cmd/iconsgen` (SVG parsing) |
| `github.com/srwiley/rasterx` | https://github.com/srwiley/rasterx | BSD-3-Clause | `cmd/iconsgen` (rasterizing) |

## Go modules compiled into the binaries we ship

| Module | License | Shipped in |
|---|---|---|
| `golang.org/x/text` | BSD-3-Clause | `appsctl`, `library` |
| `golang.org/x/crypto/x509roots/fallback` (Mozilla CA roots, so TLS works without a system bundle) | BSD-3-Clause; the root list is MPL-2.0 data | `appsctl`, `ghsync` |
| Dependencies of `localgo/` (see `localgo/go.mod`: cobra, uuid, yaml.v3, charmbracelet and others) | MIT, BSD, Apache-2.0 (each module's own license) | `localgo` binary of the LocalSend app |

Exact versions are in `go.mod` / `go.sum` and `third_party/localgo/go.sum`.

## Build toolchain (not vendored, not shipped)

| Tool | Upstream | License | Used for |
|---|---|---|---|
| NickelTC image `ghcr.io/pgaskin/nickeltc` (pinned by digest in `mod/drawer/Makefile`) | https://github.com/pgaskin/NickelTC | MIT | Cross-compiling the native drawer |

The NickelTC sysroot contains Kobo's Qt build; it stays inside the container and is never copied into the repository or a release.

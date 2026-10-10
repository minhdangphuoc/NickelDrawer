# Third-party sources

| Path | Upstream | Commit | License |
|---|---|---|---|
| `localgo/` | https://github.com/bethropolis/localgo | `f10bf9eafc9ac74ba5a8e3d9a9787edc3a7fe6af` | MIT |
| `NickelHook/` | https://github.com/pgaskin/NickelHook | `bcd90ba94d90d22562fb64ef2920ed3323c1dae5` | MIT (compatible with our MIT distribution; keep its `LICENSE`) |

## Tabler Icons

| Path | Upstream | Version | License |
|---|---|---|---|
| `tabler-icons/` | https://github.com/tabler/tabler-icons (npm `@tabler/icons`) | `3.49.0` (tag `v3.49.0`; npm tarball sha512 `l+GnSkFE...c6lw==` verified) | MIT, see `tabler-icons/LICENSE` |

Only the outline SVGs we use are vendored in `tabler-icons/svg/`: `adjustments-horizontal`, `apps`, `arrow-left`, `books`, `brand-github`, `check`, `cloud-download`, `download`, `layout-grid`, `layout-list`, `photo`, `photo-off`, `plus`, `refresh`, `send`, `settings`, `x`. `cmd/iconsgen` rasterizes them to the PNGs under `apps/*/icon.png` and `host/icons/`.

## Go modules (build tools only, not shipped to the device)

| Module | Upstream | License | Used by |
|---|---|---|---|
| `github.com/srwiley/oksvg` | https://github.com/srwiley/oksvg | BSD-3-Clause | `cmd/iconsgen` (SVG parsing) |
| `github.com/srwiley/rasterx` | https://github.com/srwiley/rasterx | BSD-3-Clause | `cmd/iconsgen` (rasterizing) |

# Changelog

All notable changes to CopperUI are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.2.0]

### Added

- Pure components in `copperui::v1`, built as a static library by the NickelTC build (`copperui.mk`): `pack.h` (paging with keep-with-next), `pager.h`, `button.h`, `list_row.h`, `divider.h`, `geometry.h`, with host tests and shared paging vectors.
- Tokens `tabBar.height` (126), `view.height` (1554), `page.height` (1426), `pager.*` geometry, `icon.chevron`, `listRow.secondaryMax`.

### Changed

- `row.detailRight` and `row.tapTextInset` are 368 so the chevron icon has room.

### Removed

- Legacy tokens for the 52 px row icon, the 160 px empty icon, the 1000 px dialog, the 32 px dialog text and 2 px dialog button border and the text chevron: replaced by the real tokens.

## [0.1.0]

### Added

- Design tokens in `tokens/copperui.tokens.json`: page metrics, type scale, spacing, greys on the 16-level grid, strokes, icon sizes, row and button metrics, component metrics used by the drawer.
- Generated outputs: `include/copperui/tokens.h` (C++11, namespace `copperui::v1`, no standard library), `internal/copperui/tokens.go` in the NickelDrawer repository, and `docs/copperui-tokens.md`.
- The text-fit rule (wrap, button label, elide) with shared test vectors in `tests/text-vectors.json`.

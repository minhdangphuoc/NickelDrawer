# Changelog

All notable changes to CopperUI are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0]

### Added

- Design tokens in `tokens/copperui.tokens.json`: page metrics, type scale, spacing, greys on the 16-level grid, strokes, icon sizes, row and button metrics, component metrics used by the drawer.
- Generated outputs: `include/copperui/tokens.h` (C++11, namespace `copperui::v1`, no standard library), `internal/copperui/tokens.go` in the NickelDrawer repository, and `docs/copperui-tokens.md`.
- The text-fit rule (wrap, button label, elide) with shared test vectors in `tests/text-vectors.json`.

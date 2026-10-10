# CopperUI

The design system of NickelDrawer and of software built on it: design tokens, the text-fit rule and, as they move here from `mod/drawer`, the Qt 5.2 Widgets components that draw Nickel-native pages on Kobo e-readers. The rules behind the numbers are in [`docs/design-system.md`](../../docs/design-system.md).

Status: 0.2.0. Tokens plus the first pure components. The canvas and widget code (layout blocks, text fit, painters) still live in `mod/drawer` and `mod/common` and move here before 1.0, when this folder becomes its own repository.

## Layout

| Path | What |
|---|---|
| `tokens/copperui.tokens.json` | The source of truth for every size, grey and type setting. |
| `include/copperui/tokens.h` | Generated. C++11 constants in `copperui::v1` and string macros for style sheets. |
| `include/copperui/pack.h`, `src/pack.cc` | Paging with keep-with-next, body frame. Vectors: `tests/pack-vectors.json`. |
| `include/copperui/pager.h`, `src/pager.cc` | Pager visibility and geometry (label, chevron icons, touch halves). |
| `include/copperui/button.h`, `src/button.cc` | Button presentation: disabled, pending, async. |
| `include/copperui/list_row.h`, `src/list_row.cc` | List row geometry; the `✓` check mark. |
| `include/copperui/divider.h`, `src/divider.cc` | Row divider and section rule. |
| `include/copperui/geometry.h` | `Rect`. |
| `copperui.mk`, `Makefile` | `copperui.mk` builds `libcopperui.a` inside the NickelTC build (included by `mod/drawer/Makefile`, linked into the mod); `make -C lib/copperui docker-test` runs the host tests. |
| `tests/text-vectors.json` | Wrap and elide cases that every implementation of the text-fit rule must reproduce. |
| `VERSION`, `CHANGELOG.md` | Semantic version of the tokens and library. |

What moved and what remains: the sources above are the library; `mod/drawer/screen_logic.{h,cc}` (text fit, component sizes, gallery math) and `screen_layout.cc` / `screen_canvas.cc` (blocks to paint items, painting) stay in the mod for now because they are tied to Qt strings and `ScreenModel`. `screen_logic.h` re-exports the library's packing under `ScreenLogic::`.

In the NickelDrawer repository the same token file also generates `internal/copperui/tokens.go` and `docs/copperui-tokens.md`. Run `make tokens` after editing the token file or the vectors; `make check` fails when a generated file is stale.

## Rules for code in this library

- Targets NickelHook mods on firmware 4.x: Qt 5.2 Widgets, **no C++ standard library, no libm**, built with `-frtti`. The headers here use plain `static const` values, no templates over `std::` types.
- Every number comes from a token. A literal size or grey in a widget is a bug.
- Text is measured with the real font, never estimated; layout code receives a width-measuring function so it stays free of Qt and testable.
- No telemetry, no Kobo binaries or Kobo-copyrighted assets.

## Using the tokens

```cpp
#include "copperui/tokens.h"

int gutter = copperui::v1::GUTTER;                       // 72
const char *style =
    "QLabel { font-size: " COPPERUI_PX_TYPE_CAPTION_SIZE "; color: " COPPERUI_GREY_INK_MUTED "; }";
```

Names are the token ids in upper snake case (`type.caption.size` becomes `TYPE_CAPTION_SIZE`).

## Text fit

One algorithm, implemented in `mod/drawer/screen_logic.cc` and `cmd/ndpreview/textfit.go`, and pinned by `tests/text-vectors.json`:

- Paragraphs split at line breaks; words split at Unicode spaces and are joined by one space; lines fill greedily.
- A word wider than the line is cut into the longest pieces that fit, at least one code point each.
- Buttons keep at most two centred lines; the rest is joined into the second line and elided at the end.
- Fixed controls (rows, header, viewer buttons) keep one line: the longest prefix that fits with "…" (the middle for file names), found by binary search.

## License

PolyForm Noncommercial 1.0.0, see `LICENSE`.

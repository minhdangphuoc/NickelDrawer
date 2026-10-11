#ifndef MODS_PAGING_H
#define MODS_PAGING_H

// Page arithmetic and page keys, shared by the mods (each compiles its own copy). No Qt, no C++ standard library, so the host tests build it with the host compiler.

namespace Paging {

// At least 1, so an empty list still shows "1 / 1".
int pageCount(int itemCount, int perPage);

int clampPage(int page, int itemCount, int perPage);
int firstIndexOnPage(int page, int perPage);
int itemCountOnPage(int page, int itemCount, int perPage);

enum PageKey { PAGE_KEY_NONE, PAGE_KEY_PREVIOUS, PAGE_KEY_NEXT };

// Qt key code to paging direction. Nickel's TactileButtonFilter already turned
// the page buttons into arrows with the swap setting applied:
// Left/Up = previous, Right/Down = next (re/notes/G2-findings.md Q3).
PageKey pageKeyFor(int qtKey);

// Target page for a key, or the current page when at an edge.
int pageAfterKey(int currentPage, PageKey key, int itemCount, int perPage);

} // namespace Paging

#endif

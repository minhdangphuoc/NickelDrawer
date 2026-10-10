#ifndef COPPERUI_PACK_H
#define COPPERUI_PACK_H

#include "copperui/tokens.h"

// Paging of a screen body (docs/design-system.md, "Pages, not scrolling"): blocks never split
// across pages, and blocks that belong together are kept on one page. Pure arithmetic: no Qt, no
// C++ standard library. Pinned by lib/copperui/tests/pack-vectors.json.

namespace copperui {
namespace v1 {

// A vertical slice of a screen.
struct Block {
    int height;
    int gap;              // space after the block when another follows on the page
    bool keepWithNext;    // stays on the page of the next block (section label and its first row,
                          // field caption and box, pair title and QR, radio title and options)
};

// Fills pageOfBlock[i] and returns the page count (at least 1).
//
// A run of blocks whose all but last member keepWithNext is one group. A group goes whole onto the
// current page when it fits, else onto a fresh page, so a section label is never alone at the
// bottom of a page. A group taller than a page cannot be kept together: it is packed block by
// block (the only case where a keepWithNext block can end a page). keepWithNext on the last block
// of the body is ignored.
int packPages(const Block *blocks, int count, int available, int *pageOfBlock);

// Height of blocks [first, first + count): the gap of the last one is not counted.
int blocksHeight(const Block *blocks, int first, int count);

// Body area of a screen whose visible height is viewHeight (the page area below the status-bar
// inset: tokens PAGE_HEIGHT on the Libra 2).
struct BodyFrame {
    int top;
    int bottom;        // first pixel below the body (above the footer)
    int footerTop;     // y of the first footer block
    int pageBarTop;    // y of the pager line; only drawn when pageCount > 1
    int pageCount;
};

// Pages the body blocks; when they need more than one page the pager takes room from every page.
BodyFrame layoutBody(const Block *body, int bodyCount, int footerHeight, int viewHeight, int *pageOfBlock);

} // namespace v1
} // namespace copperui

#endif

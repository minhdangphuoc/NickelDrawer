#include "copperui/pack.h"

namespace copperui {
namespace v1 {

int blocksHeight(const Block *blocks, int first, int count)
{
    int total = 0;
    for (int i = first; i < first + count; i++) {
        total += blocks[i].height + (i < first + count - 1 ? blocks[i].gap : 0);
    }
    return total;
}

// Index of the last block of the group that starts at first.
static int groupEnd(const Block *blocks, int count, int first)
{
    int last = first;
    while (last < count - 1 && blocks[last].keepWithNext) {
        last++;
    }
    return last;
}

int packPages(const Block *blocks, int count, int available, int *pageOfBlock)
{
    int page = 0;
    int used = 0;
    int onPage = 0;
    int index = 0;
    while (index < count) {
        int last = groupEnd(blocks, count, index);
        int gapBefore = onPage > 0 ? blocks[index - 1].gap : 0;
        int groupHeight = blocksHeight(blocks, index, last - index + 1);
        if (onPage > 0 && used + gapBefore + groupHeight > available) {
            page++;
            used = 0;
            onPage = 0;
        }
        bool isTooTall = groupHeight > available;
        // A group that fits is placed whole. One that cannot fit any page is packed block by
        // block, each block moving to a new page when it does not fit.
        for (int block = index; block <= last; block++) {
            int needed = blocks[block].height + (onPage > 0 ? blocks[block - 1].gap : 0);
            if (isTooTall && onPage > 0 && used + needed > available) {
                page++;
                used = 0;
                onPage = 0;
                needed = blocks[block].height;
            }
            pageOfBlock[block] = page;
            used += needed;
            onPage++;
        }
        index = last + 1;
    }
    return page + 1;
}

BodyFrame layoutBody(const Block *body, int bodyCount, int footerHeight, int viewHeight, int *pageOfBlock)
{
    BodyFrame frame;
    frame.top = HEADER_HEIGHT + RHYTHM;
    frame.footerTop = viewHeight - MARGIN_BOTTOM - footerHeight;
    frame.bottom = viewHeight - MARGIN_BOTTOM;
    if (footerHeight > 0) {
        frame.bottom -= footerHeight + RHYTHM;
    }
    frame.pageCount = packPages(body, bodyCount, frame.bottom - frame.top, pageOfBlock);
    if (frame.pageCount > 1) {
        frame.pageCount = packPages(body, bodyCount, frame.bottom - frame.top - PAGER_HEIGHT - RHYTHM, pageOfBlock);
    }
    frame.pageBarTop = frame.bottom - PAGER_HEIGHT;
    return frame;
}

} // namespace v1
} // namespace copperui

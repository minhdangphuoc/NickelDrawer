#include "paging.h"

namespace Paging {

// Qt::Key values (Qt 5.2 qnamespace.h); repeated here so this file stays Qt-free.
static const int QT_KEY_LEFT = 0x01000012;
static const int QT_KEY_UP = 0x01000013;
static const int QT_KEY_RIGHT = 0x01000014;
static const int QT_KEY_DOWN = 0x01000015;

int pageCount(int itemCount, int perPage)
{
    if (itemCount <= 0 || perPage <= 0) {
        return 1;
    }
    return (itemCount + perPage - 1) / perPage;
}

int clampPage(int page, int itemCount, int perPage)
{
    int lastPage = pageCount(itemCount, perPage) - 1;
    if (page < 0) {
        return 0;
    }
    return page > lastPage ? lastPage : page;
}

int firstIndexOnPage(int page, int perPage)
{
    return page * perPage;
}

int itemCountOnPage(int page, int itemCount, int perPage)
{
    int remaining = itemCount - firstIndexOnPage(page, perPage);
    if (remaining <= 0) {
        return 0;
    }
    return remaining < perPage ? remaining : perPage;
}

PageKey pageKeyFor(int qtKey)
{
    if (qtKey == QT_KEY_LEFT || qtKey == QT_KEY_UP) {
        return PAGE_KEY_PREVIOUS;
    }
    if (qtKey == QT_KEY_RIGHT || qtKey == QT_KEY_DOWN) {
        return PAGE_KEY_NEXT;
    }
    return PAGE_KEY_NONE;
}

int pageAfterKey(int currentPage, PageKey key, int itemCount, int perPage)
{
    int step = key == PAGE_KEY_NEXT ? 1 : (key == PAGE_KEY_PREVIOUS ? -1 : 0);
    return clampPage(currentPage + step, itemCount, perPage);
}

} // namespace Paging

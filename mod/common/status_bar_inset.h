#ifndef NICKELDRAWER_STATUS_BAR_INSET_H
#define NICKELDRAWER_STATUS_BAR_INSET_H

// Space a pushed page must leave at its top for Nickel's status bar.
//
// Nickel's StatusBarView (time, Wi-Fi, battery) is not part of the window
// layout: it is an overlay child of the main window, raised above the
// QStackedWidget that holds the pages, and the pages start at y = 0. Every
// Nickel page reserves the room itself in its QSS (MoreView, WishlistView:
// "qproperty-topMargin: 128" on Storm-class devices, where the bar is 126 px).
// A page that does not is drawn under the bar (re/notes/G2-findings.md,
// "Header on device"). Plain Qt only: the bar is found by its object name.

#include "copperui/tokens.h"

class QWidget;

namespace StatusBarInset {

// Nickel's own topMargin on the Libra 2 class (used when the bar is not found).
static const int FALLBACK_TOP_INSET = copperui::v1::INSET_STATUS_BAR;

// Pixels at the top of view covered by the status bar, plus Nickel's 2 px gap.
// 0 when the bar is hidden or already above the view. Call it once the view is
// in the window (showEvent); before that the fallback is returned.
int topInsetFor(const QWidget *view);

// One debug-log line per widget: geometry, global top/bottom, visibility, and
// how many pixels of header lie under the status bar. header may be NULL.
void logGeometry(const char *stage, const QWidget *view, const QWidget *header);

} // namespace StatusBarInset

#endif

#ifndef NICKELDRAWER_PAGE_HEADER_H
#define NICKELDRAWER_PAGE_HEADER_H

#include <QPixmap>
#include <QWidget>

#include "press_area.h"

class QColor;
class QHBoxLayout;
class QLabel;
class QPainter;

// The header of a pushed page, as Nickel's More sub-pages draw it (Wishlist,
// N3Dialog): the back arrow, the title in Nickel's sans font and a thin
// divider under it (re/notes/G2-findings.md "Native header and press
// feedback"). Shared by mod/drawer-proto and mod/drawer.
//
// Preferred: Nickel's own TextHeader with our title, so its look and its
// pressed state (black circle, white arrow) are Nickel's. Each mod passes the
// entry points from its symbol table; when one is missing, or Nickel's arrow
// image is not in the resources, the header draws its own copy instead.
// Extra widgets (the drawer's list/grid toggles) sit at the right, in the same
// row; the divider is a separate full-width line under the whole row.
//
// Layout as Ui_WishlistView's "headerLayout": the TextHeader keeps its size
// hint (stretch 0) and a stretch fills the rest, so the title sits left, next
// to the arrow. Given the whole row, TextHeaderBase spreads the extra width
// over its image and title and the title drifts to the middle.

// libnickel entry points, filled by each mod (NULL when the symbol is missing).
struct NickelTextHeaderApi {
    void (*construct)(void *header, QWidget *parent);
    void (*setText)(void *header, const QString &text);
    void (*setDeselectedPixmap)(void *header, const QPixmap &pixmap);
    void (*setSelectedPixmap)(void *header, const QPixmap &pixmap);
    // Only the arrow shows the pressed state; without it Nickel inverts the
    // title too (TextHeaderBase::reverseColors), so the own header is used.
    void (*setInvertImageOnly)(void *header, bool isImageOnly);

    bool isComplete() const
    {
        return construct && setText && setDeselectedPixmap && setSelectedPixmap && setInvertImageOnly;
    }
};

// Nickel's back arrow (":/images/reading/back_arrow.png", pressed:
// "back_arrow_hit.png"), centred in area; drawn with QPainter when the images
// are not in Nickel's resources.
namespace BackArrowIcon {
QPixmap pixmap(bool isPressed); // null when the resource is missing
void paint(QPainter *painter, const QRect &area, bool isPressed);
// The drawn arrow alone (three round-capped lines), in ink, centred in box;
// also the viewer's full-view back button (mod/drawer/screen_canvas.cc).
void paintStroke(QPainter *painter, const QRect &box, const QColor &ink, int stroke);
} // namespace BackArrowIcon

// Our back arrow, for the fallback header: a 120 px tap target.
class BackArrowButton : public PressArea {
public:
    explicit BackArrowButton(QWidget *parent);

protected:
    void paintEvent(QPaintEvent *event);
};

class PageHeader : public QWidget {
    Q_OBJECT
public:
    static const int ROW_HEIGHT = 120; // the toggles' tap targets (ui-kit §1.5)
    static const int DIVIDER_HEIGHT = 1; // WishlistView.qss: border-bottom 1px solid #666666

    PageHeader(const QString &title, const NickelTextHeaderApi &nickelApi, QWidget *parent);

    // Build the header with the title when it is known: it is laid out for it
    // at once. A later title lays the row out again (FontSizeAdjustingLabel).
    void setTitle(const QString &title);
    void addTrailingWidget(QWidget *widget);
    // While a dialog or an action is pending the back arrow ignores presses.
    void setBackEnabled(bool isEnabled);
    bool usesNickelHeader() const { return nickelHeader != NULL; }

signals:
    void backTapped(); // from the event loop: the slot may pop and delete the page

protected:
    void paintEvent(QPaintEvent *event);

private slots:
    void onNickelHeaderPressed();
    void onNickelHeaderTapped();
    void onOwnBackTapped();

private:
    QWidget *createDivider();
    bool createNickelHeader(const QString &title);
    void relayoutNickelHeader();
    void createOwnHeader(const QString &title);

    NickelTextHeaderApi api;
    QHBoxLayout *rowLayout;
    QWidget *nickelHeader; // Nickel's TextHeader, or NULL
    BackArrowButton *ownBack;
    QLabel *ownTitle;
    QString shownTitle;
};

#endif

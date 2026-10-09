#ifndef NICKELDRAWER_PAGE_HEADER_H
#define NICKELDRAWER_PAGE_HEADER_H

#include <QPixmap>
#include <QWidget>

#include "press_area.h"

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
// Extra widgets (the drawer's list/grid toggles) sit at the right.

// libnickel entry points, filled by each mod (NULL when the symbol is missing).
struct NickelTextHeaderApi {
    void (*construct)(void *header, QWidget *parent);
    void (*setText)(void *header, const QString &text);
    void (*setDeselectedPixmap)(void *header, const QPixmap &pixmap);
    void (*setSelectedPixmap)(void *header, const QPixmap &pixmap);

    bool isComplete() const { return construct && setText && setDeselectedPixmap && setSelectedPixmap; }
};

// Nickel's back arrow (":/images/reading/back_arrow.png", pressed:
// "back_arrow_hit.png"), centred in area; drawn with QPainter when the images
// are not in Nickel's resources.
namespace BackArrowIcon {
QPixmap pixmap(bool isPressed); // null when the resource is missing
void paint(QPainter *painter, const QRect &area, bool isPressed);
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
    static const int HEIGHT = 120; // the toggles' tap targets (ui-kit §1.5)

    PageHeader(const QString &title, const NickelTextHeaderApi &nickelApi, QWidget *parent);

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
    bool createNickelHeader(const QString &title);
    void createOwnHeader(const QString &title);

    NickelTextHeaderApi api;
    QHBoxLayout *layout;
    QWidget *nickelHeader; // Nickel's TextHeader, or NULL
    BackArrowButton *ownBack;
    QLabel *ownTitle;
};

#endif

#ifndef NICKELDRAWER_DRAWER_VIEW_H
#define NICKELDRAWER_DRAWER_VIEW_H

#include <QList>
#include <QStringList>
#include <QWidget>

#include "drawer_logic.h"
#include "drawer_model.h"

class ItemWidget;
class PressArea;
class QKeyEvent;
class QLabel;
class QVBoxLayout;
class ViewToggleIcon;

// The drawer page (docs/spec/ui-kit.md §5.2): header with back chevron, serif
// title and list/grid toggles; one page of rows (7, with section labels) or
// tiles (3 x 2); hint line; page indicator with on-screen arrows. Page-turn
// keys arrive as arrow keys (Left/Up back, Right/Down forward).
//
// Rendering only: taps are reported as signals; the session decides.
class DrawerView : public QWidget {
    Q_OBJECT
public:
    DrawerView();

    // Same items in the same order: status lines and dots change in place
    // (partial refresh). Anything else rebuilds the page (full refresh).
    void setModel(const DrawerModel &model);
    // While a dialog or an action is pending: taps and page keys are ignored.
    void setInputBlocked(bool isBlocked);

signals:
    void backRequested();
    void itemTapped(const DrawerItem &item);
    void itemLongPressed(const DrawerItem &item);

protected:
    void keyPressEvent(QKeyEvent *event);
    void showEvent(QShowEvent *event);

private slots:
    void onListToggleTapped();
    void onGridToggleTapped();
    void onPreviousTapped();
    void onNextTapped();
    void onItemTapped();
    void onItemLongPressed();
    void flashNow();

private:
    QWidget *createHeader();
    QWidget *createFooter();
    PressArea *createArrow(const char *glyph, const char *slotOrSignal, QWidget *parent);

    void flatten(const DrawerModel &model);
    bool hasSameItems(const DrawerModel &model) const;
    void applyLiveStates();

    void setViewMode(DrawerLogic::ViewMode mode);
    void goToPage(int page);
    void renderPage();
    void clearPage();
    void renderListPage(int first, int count);
    void renderGridPage(int first, int count);
    void renderEmptyState();
    void updateFooter();
    void scheduleRefresh(DrawerLogic::RefreshKind kind);
    const DrawerItem *itemForSender() const;

    QList<DrawerItem> items;
    QList<int> sectionOfItem;
    QStringList sectionTitles;
    QString hint;

    QLabel *titleLabel;
    ViewToggleIcon *listToggle;
    ViewToggleIcon *gridToggle;
    QWidget *pageArea;
    QVBoxLayout *pageLayout;
    QList<ItemWidget *> pageWidgets;
    QLabel *hintLabel;
    QLabel *pageIndicator;
    QList<PressArea *> pressables;

    DrawerLogic::ViewMode viewMode;
    DrawerLogic::RefreshPolicy refreshPolicy;
    int currentPage;
    bool isInputBlocked;
    bool hasModel;
};

#endif

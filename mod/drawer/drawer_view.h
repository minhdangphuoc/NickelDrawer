#ifndef NICKELDRAWER_DRAWER_VIEW_H
#define NICKELDRAWER_DRAWER_VIEW_H

#include <QList>
#include <QPoint>
#include <QStringList>
#include <QWidget>

#include "drawer_logic.h"
#include "drawer_model.h"
#include "painted_marks.h"

class ItemWidget;
class PageHeader;
class PressArea;
class QKeyEvent;
class QResizeEvent;
class QLabel;
class QVBoxLayout;

// The drawer page (docs/spec/ui-kit.md §5.2): Nickel's page header (back
// arrow, sans title, divider) with our list/grid toggles; one page of rows (7, with section labels) or
// a launcher grid (4 columns x as many rows as fit, no host items: they are
// header buttons there); hint line; page indicator with on-screen arrows. Page-turn
// keys arrive as arrow keys (Left/Up back, Right/Down forward). The top margin
// leaves room for Nickel's status bar, which overlays the page
// (status_bar_inset.h). A page change, a page key, hiding or popping closes
// any open menu or dialog first (OpenPopups), so none is left orphaned.
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
    DrawerLogic::ViewMode currentViewMode() const { return viewMode; }
    int currentPageIndex() const { return currentPage; }
    // Back from an app screen: the drawer as the user left it (navigation.h).
    void restoreState(DrawerLogic::ViewMode mode, int page);

signals:
    void backRequested();
    void itemTapped(const DrawerItem &item);
    // anchor: the pressed item's centre, global (where the row menu opens).
    void itemLongPressed(const DrawerItem &item, const QPoint &anchor);

protected:
    void keyPressEvent(QKeyEvent *event);
    void showEvent(QShowEvent *event);
    void hideEvent(QHideEvent *event);
    void resizeEvent(QResizeEvent *event);

private slots:
    void onListToggleTapped();
    void onGridToggleTapped();
    void onPreviousTapped();
    void onNextTapped();
    void onRefreshTapped();
    void onSettingsTapped();
    void fitGridRows();
    void onItemTapped();
    void onItemLongPressed();
    void flashNow();
    void logLayoutGeometry();

private:
    PageHeader *createHeader();
    HeaderIconButton *createHostButton(HeaderIconButton::Icon icon, const char *slot, QWidget *parent);
    void applyStatusBarInset();
    void addQt6NoticeIfPending();
    QWidget *createFooter();
    PressArea *createArrow(const char *glyph, const char *slotOrSignal, QWidget *parent);

    void flatten(const DrawerModel &model);
    bool hasSameItems(const DrawerModel &model) const;
    void applyLiveStates();
    int perPage() const;
    void rebuildShownIndices();
    void updateHostButtons();
    const DrawerItem *hostItem(const char *id) const;
    QWidget *headerWidget(DrawerLogic::HeaderButton button) const;

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
    QList<int> shownIndices; // indices into items of what the current view mode lists
    QStringList sectionTitles;
    QString hint;

    QVBoxLayout *rootLayout;
    PageHeader *header;
    ViewToggleIcon *listToggle;
    ViewToggleIcon *gridToggle;
    QWidget *pageArea;
    QVBoxLayout *pageLayout;
    QList<ItemWidget *> pageWidgets;
    QLabel *hintLabel;
    QLabel *pageIndicator;
    QList<PressArea *> pressables;
    HeaderIconButton *refreshButton;  // stands for the "_refresh" host item
    HeaderIconButton *settingsButton; // stands for "_settings"
    UpdateDot *settingsDot;           // an update is waiting

    DrawerLogic::ViewMode viewMode;
    DrawerLogic::RefreshPolicy refreshPolicy;
    int gridRows; // rows of 4 cells that fit the page area
    int currentPage;
    bool isInputBlocked;
    bool hasModel;
    bool isUpdateAvailable; // the model's update_available
    bool hasLoggedLayout;
    bool hasQt6Notice;
};

#endif

#ifndef NICKELDRAWER_SCREEN_VIEW_H
#define NICKELDRAWER_SCREEN_VIEW_H

#include <QElapsedTimer>

#include "drawer_logic.h"
#include "press_area.h"
#include "screen_layout.h"
#include "screen_model.h"

// One app screen drawn over the drawer page (docs/spec/ui-kit.md §3, §5.3).
// It is a child of the drawer view and covers it, so Nickel's window stack is
// unchanged; page-turn keys, which Nickel sends to the drawer view, are taken
// by an event filter on it while the screen is open.
//
// Rendering only: it paints the ScreenLayout primitives and reports taps;
// ScreenSession decides what they do. E-ink: a new screen or page flashes once
// (full refresh); a changed value repaints only the changed primitives
// (partial), and every 9th partial update in a row is a full one.
class ScreenView : public PressArea {
    Q_OBJECT
public:
    explicit ScreenView(QWidget *drawerView);

    // A different screen starts at page 1; the same screen keeps its page.
    void showScreen(const ScreenModel &model);
    const ScreenModel &screen() const { return model; }
    const ScreenComponent *component(int index) const;
    // Shows a tapped toggle or radio at once; the reload confirms or reverts it.
    void showToggleState(int component, bool isOn);
    void showRadioValue(int component, const QString &value);
    // While a dialog or an action is pending: taps and page keys are ignored.
    void setInputBlocked(bool isBlocked);

signals:
    void backRequested();
    void headerActionTapped();
    void componentTapped(int component, const QString &value);

protected:
    bool eventFilter(QObject *watched, QEvent *event);
    void paintEvent(QPaintEvent *event);

private slots:
    void onTapped();
    void handleTap();
    void flashNow();

private:
    void placeBelowStatusBar();
    void relayout();
    void applyPage(int index, bool isFullRefresh);
    QRegion changedRegion(const ScreenPage &next, bool *isSameStructure) const;
    void scheduleRefresh(DrawerLogic::RefreshKind kind, const QRegion &region);
    bool handlePageKey(int key);
    void goToPage(int index);
    void paintItem(QPainter *painter, const PaintItem &item);

    QWidget *host;
    ScreenModel model;
    bool hasScreen;
    ScreenLayout layout;
    ScreenPage page;
    int pageIndex;
    DrawerLogic::RefreshPolicy refreshPolicy;
    bool isInputBlocked;
    bool isTapPending;
    QPoint pendingTapPosition;
    QElapsedTimer sinceLastTap;
};

#endif

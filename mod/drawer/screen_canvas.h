#ifndef NICKELDRAWER_SCREEN_CANVAS_H
#define NICKELDRAWER_SCREEN_CANVAS_H

#include "drawer_logic.h"
#include "press_area.h"
#include "screen_layout.h"
#include "screen_model.h"

// The body of an app screen under its header (docs/spec/ui-kit.md §3, §5.3):
// paints the ScreenLayout primitives and reports taps on them; ScreenSession
// decides what they do. It covers the whole ScreenView; the PageHeader is a
// sibling raised above it, never a child: a PressArea ancestor would take
// Nickel's tap gesture from the header's TextHeader (GestureOverride).
//
// Every target shows its pressed state (inverted) through PressArea; disabled
// controls have no target, so they show nothing and ignore taps. Page-turn
// keys, which Nickel sends to the drawer view, are taken by an event filter
// on it. E-ink: a new screen or page flashes once (full refresh); a changed
// value repaints only the changed primitives (partial), and every 9th partial
// update in a row is a full one.
class ScreenCanvas : public PressArea {
    Q_OBJECT
public:
    ScreenCanvas(QWidget *drawerView, QWidget *parent);

    // A different screen starts at page 1; the same screen keeps its page.
    void showScreen(const ScreenModel &model);
    const ScreenModel &screen() const { return model; }
    const ScreenComponent *component(int index) const;
    // While a dialog or an action is pending: taps and page keys are ignored.
    void setInputBlocked(bool isBlocked);
    int currentPage() const { return pageIndex; }
    void restorePageOnNextScreen(int page) { pendingPage = page; }

signals:
    void componentTapped(int component, const QString &value);

protected:
    bool eventFilter(QObject *watched, QEvent *event);
    void resizeEvent(QResizeEvent *event);
    void paintEvent(QPaintEvent *event);
    PressFeedback feedbackAt(const QPoint &position) const;

private slots:
    void onTapped();
    void flashNow();

private:
    void relayout();
    void applyPage(int index, bool isFullRefresh);
    QRegion changedRegion(const ScreenPage &next, bool *isSameStructure) const;
    void scheduleRefresh(DrawerLogic::RefreshKind kind, const QRegion &region);
    bool handlePageKey(int key);
    void goToPage(int index);
    const TapTarget *targetAt(const QPoint &position) const;

    QWidget *host;
    ScreenModel model;
    bool hasScreen;
    ScreenLayout layout;
    ScreenPage page;
    int pageIndex;
    int pendingPage; // -1: a new screen opens at page 1
    DrawerLogic::RefreshPolicy refreshPolicy;
    bool isInputBlocked;
};

#endif

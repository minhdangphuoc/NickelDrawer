#ifndef NICKELDRAWER_SCREEN_CANVAS_H
#define NICKELDRAWER_SCREEN_CANVAS_H

#include <QHash>
#include <QPixmap>
#include <QSize>

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
    // A copy: the layout (and its components) is rebuilt on the next showScreen.
    bool componentAt(int index, ScreenComponent *copy) const;
    // While a dialog or an action is pending: taps and page keys are ignored.
    void setInputBlocked(bool isBlocked);
    int currentPage() const { return pageIndex; }
    void restorePageOnNextScreen(int page) { pendingPage = page; }
    // The next new screen opens at the grid page holding this picture (back from the viewer).
    void restoreTileOnNextScreen(const QString &id) { pendingTileId = id; }

    // Viewer state for the header ("n / N", status).
    bool hasViewer() const { return layout.viewerComponent() >= 0; }
    bool isChromeShown() const { return layout.chromeShown(); }
    int viewerPosition() const { return layout.viewerShownIndex() + 1; }
    int viewerCount() const { return layout.viewerCount(); }
    QString viewerCurrentStatus() const { return hasViewer() ? layout.component(layout.viewerComponent()).currentStatus : QString(); }
    QString viewerPictureId() const;
    bool isViewerPictureCurrent() const;
    // Shown pictures held in memory (tests and the log).
    int cachedPictureCount() const { return pixmaps.size(); }
    // Pop and close: nothing stays decoded.
    void dropPictures();

signals:
    void componentTapped(int component, const QString &value);
    // A gallery tile was held; anchor in global coordinates.
    void tileLongPressed(int component, const QString &id, const QPoint &anchor);
    // A list row was held (row = index in the list, primary = its text); anchor in global coordinates.
    void listRowLongPressed(int component, int row, const QString &primary, const QPoint &anchor);
    // The viewer shows another picture (the user flipped).
    void viewerPictureChanged(const QString &id);
    // The picture, the chrome or the viewer itself changed: refresh the header.
    void viewerStateChanged();

protected:
    bool eventFilter(QObject *watched, QEvent *event);
    void resizeEvent(QResizeEvent *event);
    void hideEvent(QHideEvent *event);
    void showEvent(QShowEvent *event);
    void paintEvent(QPaintEvent *event);
    PressFeedback feedbackAt(const QPoint &position) const;

private slots:
    void onTapped();
    void onLongPressed();
    void flashNow();

private:
    void relayout();
    void applyPage(int index, bool isFullRefresh);
    QRegion changedRegion(const ScreenPage &next, bool *isSameStructure) const;
    void scheduleRefresh(DrawerLogic::RefreshKind kind, const QRegion &region);
    bool handlePageKey(int key);
    void goToPage(int index);
    void stepViewer(int step);
    void toggleChrome();
    void showChrome();
    void restoreViewerState(const QString &keptPicture);
    void refreshPictures();
    const QPixmap *pictureFor(const PaintItem &item) const;
    const TapTarget *targetAt(const QPoint &position) const;
    void logTargets() const;

    QWidget *host;
    ScreenModel model;
    bool hasScreen;
    ScreenLayout layout;
    ScreenPage page;
    int pageIndex;
    int pendingPage; // -1: a new screen opens at page 1
    DrawerLogic::RefreshPolicy refreshPolicy;
    bool isInputBlocked;
    QString pendingTileId;
    bool isChrome; // the viewer's chrome, kept across reloads of the same screen
    // Decoded pictures of the shown page only (thumbnails of the grid page, or
    // the one viewer picture), with the box each was decoded for.
    struct CachedPicture {
        QSize box;
        QPixmap pixmap; // null: unreadable, not retried until the page changes
    };
    QHash<QString, CachedPicture> pixmaps;
};

#endif

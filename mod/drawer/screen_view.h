#ifndef NICKELDRAWER_SCREEN_VIEW_H
#define NICKELDRAWER_SCREEN_VIEW_H

#include <QList>
#include <QStringList>
#include <QWidget>

#include "press_area.h"
#include "screen_model.h"

class HeaderIconButton;
class PageHeader;
class QLabel;
class ScreenCanvas;

// The header action at the right of an app screen header ("Settings"):
// underlined text, inverted while pressed.
class HeaderActionButton : public PressArea {
    Q_OBJECT
public:
    explicit HeaderActionButton(QWidget *parent);

    void setText(const QString &text);

private:
    QLabel *label;
};

// One app screen drawn over the drawer page (docs/spec/ui-kit.md §3.1, §5.3).
// It is a child of the drawer view and covers it below Nickel's status bar,
// so Nickel's window stack is unchanged.
//
// The header is the drawer's own PageHeader (Nickel's TextHeader: "←", title
// in Nickel's header sans, left-aligned, 1 px divider, pressed arrow only),
// with the status text and the optional header action in the same row. The
// body is a ScreenCanvas under it. Both are plain children of this QWidget:
// the header must not sit inside a PressArea (see screen_canvas.h).
class ScreenView : public QWidget {
    Q_OBJECT
public:
    // title: the app's name, shown until the screen model arrives; the header
    // is built with it (an empty TextHeader leaves no room for a later title).
    ScreenView(const QString &title, QWidget *drawerView);

    // A different screen starts at page 1; the same screen keeps its page.
    void showScreen(const ScreenModel &model);
    const ScreenModel &screen() const { return model; }
    // A copy: showScreen rebuilds the layout the component came from.
    bool componentAt(int index, ScreenComponent *copy) const;
    // The picture on screen in the viewer, "" without one.
    QString viewerPictureId() const;
    // Nothing stays decoded (pop, close).
    void dropPictures();
    // While a dialog or an action is pending: back, taps and page keys are ignored.
    void setInputBlocked(bool isBlocked);
    int currentPage() const;
    // Back to a parent screen: its next showScreen opens at page instead of page 1.
    void restorePageOnNextScreen(int page);
    // Back from the viewer: the gallery opens at the page holding this picture.
    void restoreTileOnNextScreen(const QString &id);

public slots:
    // Shows a tapped toggle or radio at once; the reload confirms or reverts
    // it. Slots, so a tap handler can queue them: rebuilding the layout inside
    // the handler would free the component the handler still reads.
    void showToggleState(int component, bool isOn);
    void showRadioValue(int component, const QString &value);
    // A quick action started from a button, row or empty-state button: its label is replaced by
    // the pending text until clearPending() (design-system.md A9). Queued by the tap handler.
    void showPending(int component);
    void clearPending();

signals:
    void backRequested();
    void headerActionTapped();
    void headerIconTapped(int index);
    void componentTapped(int component, const QString &value);
    void tileLongPressed(int component, const QString &id, const QPoint &anchor);
    void listRowLongPressed(int component, int row, const QString &primary, const QPoint &anchor);
    void viewerPictureChanged(const QString &id);

protected:
    bool event(QEvent *event);
    bool eventFilter(QObject *watched, QEvent *event);
    void resizeEvent(QResizeEvent *event);
    void showEvent(QShowEvent *event);

private slots:
    void showHeader();
    void onBackTapped();
    void onHeaderActionTapped();
    void onHeaderIconTapped();

private:
    void showHeaderIcons();
    PageHeader *createHeader(const QString &title);
    void placeBelowStatusBar();

    QWidget *host;
    ScreenModel model;
    ScreenCanvas *canvas;
    PageHeader *header;
    QLabel *statusLabel;
    HeaderActionButton *actionButton;
    QList<HeaderIconButton *> iconButtons;
    QStringList shownIconNames;
    bool isInputBlocked;
};

#endif

#ifndef NICKELDRAWER_SCREEN_VIEW_H
#define NICKELDRAWER_SCREEN_VIEW_H

#include <QWidget>

#include "press_area.h"
#include "screen_model.h"

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
    explicit ScreenView(QWidget *drawerView);

    // A different screen starts at page 1; the same screen keeps its page.
    void showScreen(const ScreenModel &model);
    const ScreenModel &screen() const { return model; }
    const ScreenComponent *component(int index) const;
    // Shows a tapped toggle or radio at once; the reload confirms or reverts it.
    void showToggleState(int component, bool isOn);
    void showRadioValue(int component, const QString &value);
    // While a dialog or an action is pending: back, taps and page keys are ignored.
    void setInputBlocked(bool isBlocked);
    int currentPage() const;
    // Back to a parent screen: its next showScreen opens at page instead of page 1.
    void restorePageOnNextScreen(int page);

signals:
    void backRequested();
    void headerActionTapped();
    void componentTapped(int component, const QString &value);

protected:
    bool eventFilter(QObject *watched, QEvent *event);
    void resizeEvent(QResizeEvent *event);

private slots:
    void onBackTapped();
    void onHeaderActionTapped();

private:
    PageHeader *createHeader();
    void showHeader();
    void placeBelowStatusBar();

    QWidget *host;
    ScreenModel model;
    ScreenCanvas *canvas;
    PageHeader *header;
    QLabel *statusLabel;
    HeaderActionButton *actionButton;
    bool isInputBlocked;
};

#endif

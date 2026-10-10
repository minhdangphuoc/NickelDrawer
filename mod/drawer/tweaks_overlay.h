#ifndef NICKELDRAWER_TWEAKS_OVERLAY_H
#define NICKELDRAWER_TWEAKS_OVERLAY_H

#include <QDialog>
#include <QPixmap>

#include "press_area.h"

class QVBoxLayout;

// Close button of the own overlay: Nickel's N3Dialog close image
// (":/images/search/global_close.png", pressed "…_hit.png"), else the Tabler
// "x" icon (system_icon.h). Without either, hasIcon() is false and the
// overlay leaves the button out (a tap outside the box still closes).
class CloseCrossButton : public PressArea {
public:
    explicit CloseCrossButton(QWidget *parent);
    static bool hasIcon();

protected:
    void paintEvent(QPaintEvent *event);
};

// Fallback chrome of the Tweaks modal when Nickel's N3Dialog cannot be used
// (re/notes/tweaks-modal.md): a full-screen frameless dialog over a snapshot of
// the page, a white box with a 1 px black border (N3Dialog.qss "#mainFrame")
// holding a small centred title, the close button top-right and the content.
// A tap outside the box or on ✕ rejects it. Deletes itself when closed.
class TweaksOverlay : public QDialog {
    Q_OBJECT
public:
    // content becomes the box's child.
    TweaksOverlay(const QString &title, QWidget *content);
    void present();

protected:
    void paintEvent(QPaintEvent *event);
    void resizeEvent(QResizeEvent *event);

private:
    QWidget *createTitleRow(const QString &title, QWidget *box);

    QWidget *outsideArea;
    QPixmap backdrop;
};

#endif

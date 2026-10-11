#ifndef EXTRAS_TWEAKS_SESSION_H
#define EXTRAS_TWEAKS_SESSION_H

#include <QObject>
#include <QPointer>

#include "tweaks_model.h"

class ConfirmQuestion;
class PageController;
class TweaksClient;
class TweaksPanel;

// One open Tweaks panel (helpers/extras/docs/tweaks.md, tweaks-modal.md):
//   open    → Nickel's N3Dialog (Settings' window) with our TweaksPanel as
//             content, pushed as a controller with no tab bar. `extras-tweaks list` fills it.
//   tap     → toggle: `extras-tweaks apply <id>`, the cell's value changes in place;
//             action with "confirm": Nickel's confirmation first.
//   reply   → Nickel's toast with "toast" (or the error); "nickel": Nickel's
//             sleep / reboot / library rescan, or a toast when unavailable.
//   close   → ✕, a tap outside the frame, or Nickel removing the page.
// One tweak at a time: while one runs, the cells ignore presses. Never a
// child of a Nickel widget; it deletes itself when the dialog goes away.
class TweaksSession : public QObject {
    Q_OBJECT
public:
    // From the More-tab row, outside Nickel's tap handler. A second call while
    // a panel is open is ignored.
    static void open();

protected:
    bool eventFilter(QObject *watched, QEvent *event);

private slots:
    void onTweaksLoaded(const TweaksModel &model);
    void onTweakTapped(const QString &tweakId);
    void onConfirmAnswer(bool isAccepted);
    void onTweakApplied(const TweakResult &result);
    // Nickel emits closeTapped() inside its tap gesture handler.
    void onCloseRequested();
    void closeModal();
    void onModalDestroyed();

private:
    TweaksSession();
    ~TweaksSession();

    bool present();
    void watchModal(QWidget *modalWidget);
    bool isOutsideNickelFrame(const QPoint &dialogPosition) const;
    void askThenApply(const TweakItem &item);
    void applyTweak(const QString &tweakId);
    void setBusy(bool isBusy);
    void showResult(const TweakResult &result);
    void runNickelAction(const TweakResult &result);

    TweaksClient *client;
    QPointer<TweaksPanel> panel;
    QPointer<QWidget> nickelDialog;
    QPointer<PageController> controller;
    QPointer<ConfirmQuestion> question;
    QString pendingTweakId;
    bool isBusy;
    bool isCloseScheduled;
    bool hasTouchBegunOutside;
};

// Runs Nickel's sleep or reboot a moment after the reply, from the event loop,
// so the toast reaches the screen first (reboot blocks the GUI thread).
// Independent of the dialog; deletes itself.
class DeferredPowerAction : public QObject {
    Q_OBJECT
public:
    static void start(int powerAction);

private slots:
    void run();

private:
    explicit DeferredPowerAction(int powerAction);

    int action; // NickelBridge::PowerAction
};

#endif

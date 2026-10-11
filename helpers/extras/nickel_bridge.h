#ifndef EXTRAS_NICKEL_BRIDGE_H
#define EXTRAS_NICKEL_BRIDGE_H

// The only part of the helper that touches libnickel: the NickelHook hook and symbol tables, the
// model and firmware check and every call into Nickel. Everything else is plain Qt 5.2 and this
// interface.
//
// Two features, each switched on only when every symbol it needs resolved:
//   button  the status-bar screenshot button (hook StatusBarView, StatusBarIcon, ScreenshotMaker);
//   tweaks  the Tweaks row in the More tab and its panel (hook Ui_MoreView::setupUi, IconLeftButton,
//           N3Dialog, controller push, tap gestures).
// Fail soft: every symbol is optional. While a feature is off its hook only calls Nickel's original
// and every wrapper below does nothing. One toast says so, a few seconds after the first hook runs.
//
// One hook per Nickel symbol: NickelHook's nh_dlhook returns the original through dlsym, not the
// previous GOT entry, so a second mod hooking a symbol silently unhooks the first. NickelDrawer
// hooks MoreView::MoreView(QWidget*); this helper hooks Ui_MoreView::setupUi(QWidget*), which that
// constructor calls, and StatusBarView::StatusBarView(QWidget*), which nothing else hooks.

#include <QObject>
#include <QPixmap>
#include <QString>

#include <NickelHook.h>

#include "more_row.h"

class AbstractController;
class QDialog;
class QGesture;
class QPoint;
class QWidget;

namespace NickelBridge {

// NickelHook tables, referenced by the NickelHook() declaration in extras.cc.
extern struct nh_hook hookTable[];
extern struct nh_dlsym symbolTable[];

// Called once from init(): reads .kobo/version and switches each feature on only on a supported
// model with a tested firmware and every symbol of the feature present. Returns true when at least
// one feature is on.
bool activate();
bool isButtonActive();
bool isTweaksActive();

// Called after the original constructor / setupUi, on the GUI thread, while the feature is on.
typedef void (*WidgetHandler)(QWidget *widget);
void setStatusBarBuiltHandler(WidgetHandler handler);
void setMoreViewSetupHandler(WidgetHandler handler);

// ---- Status-bar button ----
// A Nickel StatusBarIcon (a TouchLabel), or NULL (logged). Not shown or laid out yet.
QWidget *createStatusBarIcon(QWidget *parent);
void setIconPixmaps(QWidget *icon, const QPixmap &normal, const QPixmap &pressed);
// Nickel's ScreenshotMaker: its own instance, made on first use and kept.
// The name of the file the next save would write or just wrote ("/mnt/onboard/screen_NNN.png");
// empty when the maker is unavailable.
QString screenshotName();
// ScreenshotMaker::saveCurrentScreen(): blocks for the PNG encode. false when unavailable.
bool saveScreenshot();

// ---- More tab row ----
// The IconLeftButton calls the shared row code needs.
MoreRow::NickelApi moreRowApi();

// ---- Tap gestures (PressArea's providers) ----
int tapGestureType();
bool tapGesturePosition(QGesture *gesture, QPoint *screenPosition);

// ---- Window stack and the Settings-style modal ----
bool hasControllerPush();
void pushController(AbstractController *controller);
void popController(AbstractController *controller);
// Nickel's N3Dialog (the window of Settings' sub-pages: small centred title, close button top-right),
// with content as its child and the large title hidden. Not shown or pushed: the caller pushes it as
// a controller view. Its closeTapped() signal is the close button. NULL (logged) when missing.
bool hasSettingsDialog();
QWidget *createSettingsDialog(QWidget *content, const QString &title);

// ---- Feedback ----
// Nickel's toast; only logged when its symbols are missing.
void showToast(const QString &primary, const QString &secondary);
// Nickel's ConfirmationDialog (a QDialog: accepted()/rejected()), not yet shown; the caller connects
// and calls open(). NULL when missing.
bool hasConfirmDialog();
QDialog *createConfirmDialog(const QString &title, const QString &text, const QString &acceptLabel,
                             const QString &rejectLabel);

// ---- Power and library (N3PowerWorkflowManager, PlugWorkflowManager) ----
enum PowerAction { POWER_SLEEP, POWER_REBOOT };
bool hasPowerAction(PowerAction action);
// false (logged) when the symbols are missing.
bool runPowerAction(PowerAction action);
// PlugWorkflowManager::sync, NickelMenu's rescan_books. false (logged) when missing.
bool rescanLibrary();

// ---- E-ink ----
// One flashing full update of the widget (Kobo's Qt attribute 0x80 + repaint).
void requestFullRefresh(QWidget *widget);

} // namespace NickelBridge

// The one toast about an unsupported device or firmware; deletes itself.
class DisabledNotice : public QObject {
    Q_OBJECT
public:
    DisabledNotice();

public slots:
    void show();
};

#endif

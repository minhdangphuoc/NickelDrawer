#ifndef NICKELDRAWER_NICKEL_BRIDGE_H
#define NICKELDRAWER_NICKEL_BRIDGE_H

// The only part of the mod that touches libnickel: the NickelHook symbol and
// hook tables, the firmware check and every call into Nickel. Everything else
// uses plain Qt 5.2 and this interface.
//
// Fail soft: every symbol is optional. Each has*() tells whether a feature can
// be used; every call checks again and logs instead of crashing. While the
// bridge is inert (kill switch, unknown firmware, or init not run) every
// has*() is false and the More-tab hook only calls Nickel's original.

#include <QObject>
#include <QString>

#include <NickelHook.h>

class AbstractController;
class QDialog;
class QPixmap;
class QWidget;

namespace NickelBridge {

// NickelHook tables, referenced by the NickelHook() declaration in nickeldrawer.cc.
extern struct nh_hook hookTable[];
extern struct nh_dlsym symbolTable[];

// Called once from init(): reads the firmware version (file closed at once),
// and activates the bridge only on a tested firmware. Returns true when active.
bool activate();
bool isActive();

// The firmware string read by activate() ("unknown" on failure).
const char *firmware();

// ---- More tab ----
typedef void (*MoreViewCreatedHandler)(QWidget *moreView);
// Called after Nickel's MoreView constructor, on the GUI thread.
void setMoreViewCreatedHandler(MoreViewCreatedHandler handler);

bool hasIconLeftButton();
// A Nickel IconLeftButton (More-tab row look), or NULL (logged).
QWidget *createIconLeftButton(QWidget *parent);
void setIconLeftButtonPixmap(QWidget *button, const QPixmap &pixmap);
void setIconLeftButtonText(QWidget *button, const QString &text);

// ---- Window stack ----
bool hasControllerPush(); // route A
bool hasViewPush();       // route B
void pushController(AbstractController *controller);
void popController(AbstractController *controller);
void pushView(QWidget *view);
void popView(QWidget *view);

// ---- Feedback ----
bool hasToast();
void showToast(const QString &text);

bool hasMessageDialog();
// Nickel's OK dialog (ConfirmationDialogFactory::showOKDialog). False when missing.
bool showMessageDialog(const QString &title, const QString &text);

bool hasConfirmDialog();
// Nickel's ConfirmationDialog (a QDialog: accepted()/rejected()), not yet
// shown; the caller connects and calls open(). NULL when missing.
QDialog *createConfirmDialog(const QString &title, const QString &text, const QString &acceptLabel,
                             const QString &rejectLabel);

// ---- Wi-Fi and library ----
bool hasWifi();
bool isInternetAccessible();
// Starts Nickel's own connect flow (its UI); returns at once.
bool requestWifiConnect();

enum RescanKind { RESCAN_QUICK, RESCAN_FULL };
bool hasRescan(RescanKind kind);
bool rescanLibrary(RescanKind kind);

// ---- E-ink ----
// One flashing full update of the widget (Kobo's Qt attribute 0x80 + repaint).
void requestFullRefresh(QWidget *widget);

} // namespace NickelBridge

// Second half of a full rescan, run 1 s later from the event loop instead of
// sleeping on the GUI thread (NickelMenu's rescan_books_full sleeps 1 s).
class FullRescanFinisher : public QObject {
    Q_OBJECT
public:
    explicit FullRescanFinisher(void *plugWorkflowManager);

public slots:
    void finish();

private:
    void *manager;
};

#endif

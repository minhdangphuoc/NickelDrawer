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

#include "page_header.h"

class AbstractController;
class QDialog;
class QLineEdit;
class QMenu;
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

// The gesture type of Nickel's tap gesture (TapGestureRecognizer::_gestureType),
// read live; 0 when the symbol is missing, not yet registered, or the bridge is inert.
int tapGestureType();

// Nickel's TextHeader for page headers; every entry NULL while inert.
NickelTextHeaderApi textHeaderApi();

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
// Nickel's OK dialog (ConfirmationDialogFactory::showOKDialog), already
// shown, closing on a tap outside its frame. NULL when missing.
QDialog *showMessageDialog(const QString &title, const QString &text);

bool hasConfirmDialog();
// Nickel's ConfirmationDialog (a QDialog: accepted()/rejected()), not yet
// shown; the caller connects and calls open(). A tap outside its frame
// rejects it when rejectsOnOutsideTap. NULL when missing.
QDialog *createConfirmDialog(const QString &title, const QString &text, const QString &acceptLabel,
                             const QString &rejectLabel, bool rejectsOnOutsideTap);

// Nickel's popup menu (NickelTouchMenu, a QMenu: bordered box, rows with thin
// dividers, pressed row inverted, closes on a tap outside), as NickelMenu
// builds it. The caller adds rows, pops it up and deletes it when hidden.
bool hasTouchMenu();
QMenu *createTouchMenu();
// Adds a MenuTextItem row; receiver NULL: a row that ignores taps (title).
// Returns the row widget (the sender of tapSlot), or NULL (logged).
QWidget *addTouchMenuItem(QMenu *menu, const QString &text, bool isFirst, QObject *receiver, const char *tapSlot);

// Property set on views the drawer pushes over itself (the keyboard dialog);
// route B's stack watcher must not close the drawer when one becomes current.
static const char OVERLAY_PROPERTY[] = "nickelDrawerOverlay";

// ---- Keyboard (field editing; NickelHardcover's proven N3Dialog path) ----
bool hasKeyboardDialog();
// Shows Nickel's N3Dialog around content (it becomes the dialog's child) with
// Nickel's keyboard typing into lineEdit, pushed on the window stack. The
// keyboard's Go key emits commitRequested() and the dialog's close button
// closeTapped(); both are connected to receiver's slots. Returns the dialog,
// or NULL (logged) when anything is missing.
QWidget *openKeyboardDialog(QWidget *content, QLineEdit *lineEdit, const QString &title, const QString &goText,
                            QObject *receiver, const char *commitSlot, const char *closeSlot);
// Pops the dialog from the window stack (hides it and deletes it later).
void closeKeyboardDialog(QWidget *dialog);

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

#ifndef NDDRAWER_NICKEL_API_H
#define NDDRAWER_NICKEL_API_H

// libnickel entry points used by the prototypes. NickelHook fills them before
// init(); every one is optional, so each is NULL when the firmware lacks it and
// the prototype that needs it disables itself instead of crashing.

#include <QString>

#include "page_header.h"

class QPixmap;
class QWidget;
class AbstractController;

extern "C" {
extern void *(*MainWindowController_sharedInstance)();
extern void (*MainWindowController_toast)(void *window, QString const &primary, QString const &secondary, int milliseconds);
extern void (*MainWindowController_push)(void *window, AbstractController *controller, bool animate);
extern void (*MainWindowController_pop)(void *window, AbstractController *controller);
extern void (*MainWindowController_pushView)(void *window, QWidget *view);
extern void (*MainWindowController_popView)(void *window, QWidget *view);
extern void (*IconLeftButton_construct)(void *button, QWidget *parent);
extern void (*IconLeftButton_setPixmap)(void *button, QPixmap const &pixmap);
extern void (*IconLeftButton_setText)(void *button, QString const &text);
// Data symbol: the Qt::GestureType Nickel's TapGestureRecognizer registered
// (0 until Nickel registers it at startup, after our init).
extern int *TapGestureRecognizer_gestureType;
extern void (*TextHeader_construct)(void *header, QWidget *parent);
extern void (*TextHeader_setText)(void *header, QString const &text);
extern void (*TextHeaderBase_setDeselectedPixmap)(void *header, QPixmap const &pixmap);
extern void (*TextHeaderBase_setSelectedPixmap)(void *header, QPixmap const &pixmap);
// Original MoreView::MoreView(QWidget*), returned by the NickelHook PLT hook.
extern void (*MoreView_construct_original)(void *moreView, QWidget *parent);
}

namespace NickelApi {

bool hasToast();
bool hasIconLeftButton();
bool hasControllerPush(); // route A: push/pop an AbstractController
bool hasViewPush();       // route B: pushView/popView a bare widget

// Nickel's tap gesture type, read live; 0 when missing or not registered yet.
int tapGestureType();

// Nickel's TextHeader for the page header (entries NULL when missing).
NickelTextHeaderApi textHeaderApi();

// Nickel's main window controller, or NULL (logged) when unavailable.
void *mainWindow();

// Shows a toast; logs and does nothing when the toast API is missing.
void showToast(const QString &text);

} // namespace NickelApi

#endif

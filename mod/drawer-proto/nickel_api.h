#ifndef NDDRAWER_NICKEL_API_H
#define NDDRAWER_NICKEL_API_H

// libnickel entry points used by the prototypes. NickelHook fills them before
// init(); every one is optional, so each is NULL when the firmware lacks it and
// the prototype that needs it disables itself instead of crashing.

#include <QString>

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
// Original MoreView::MoreView(QWidget*), returned by the NickelHook PLT hook.
extern void (*MoreView_construct_original)(void *moreView, QWidget *parent);
}

namespace NickelApi {

bool hasToast();
bool hasIconLeftButton();
bool hasControllerPush(); // route A: push/pop an AbstractController
bool hasViewPush();       // route B: pushView/popView a bare widget

// Nickel's main window controller, or NULL (logged) when unavailable.
void *mainWindow();

// Shows a toast; logs and does nothing when the toast API is missing.
void showToast(const QString &text);

} // namespace NickelApi

#endif

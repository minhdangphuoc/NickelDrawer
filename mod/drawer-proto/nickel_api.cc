#include "nickel_api.h"

#include <NickelHook.h>

#include "debug_log.h"

extern "C" {
void *(*MainWindowController_sharedInstance)();
void (*MainWindowController_toast)(void *, QString const &, QString const &, int);
void (*MainWindowController_push)(void *, AbstractController *, bool);
void (*MainWindowController_pop)(void *, AbstractController *);
void (*MainWindowController_pushView)(void *, QWidget *);
void (*MainWindowController_popView)(void *, QWidget *);
void (*IconLeftButton_construct)(void *, QWidget *);
void (*IconLeftButton_setPixmap)(void *, QPixmap const &);
void (*IconLeftButton_setText)(void *, QString const &);
int *TapGestureRecognizer_gestureType;
void (*TextHeader_construct)(void *, QWidget *);
void (*TextHeader_setText)(void *, QString const &);
void (*TextHeaderBase_setDeselectedPixmap)(void *, QPixmap const &);
void (*TextHeaderBase_setSelectedPixmap)(void *, QPixmap const &);
void (*MoreView_construct_original)(void *, QWidget *);
}

static const int TOAST_DURATION_MILLISECONDS = 3000;

namespace NickelApi {

bool hasToast()
{
    return MainWindowController_sharedInstance && MainWindowController_toast;
}

bool hasIconLeftButton()
{
    return IconLeftButton_construct && IconLeftButton_setPixmap && IconLeftButton_setText;
}

bool hasControllerPush()
{
    return MainWindowController_sharedInstance && MainWindowController_push && MainWindowController_pop;
}

bool hasViewPush()
{
    return MainWindowController_sharedInstance && MainWindowController_pushView && MainWindowController_popView;
}

int tapGestureType()
{
    return TapGestureRecognizer_gestureType ? *TapGestureRecognizer_gestureType : 0;
}

NickelTextHeaderApi textHeaderApi()
{
    NickelTextHeaderApi api = {TextHeader_construct, TextHeader_setText, TextHeaderBase_setDeselectedPixmap,
                               TextHeaderBase_setSelectedPixmap};
    return api;
}

void *mainWindow()
{
    if (!MainWindowController_sharedInstance) {
        nh_log("MainWindowController::sharedInstance missing");
        return NULL;
    }
    void *window = MainWindowController_sharedInstance();
    if (!window) {
        nh_log("MainWindowController::sharedInstance returned null");
    }
    return window;
}

void showToast(const QString &text)
{
    if (!hasToast()) {
        nh_log("toast API missing, would have shown: %s", qPrintable(text));
        return;
    }
    void *window = mainWindow();
    DebugLog::write("toast: window %p, text \"%s\"", window, qPrintable(text));
    if (window) {
        // Empty (not null) secondary text: the form proven on device by ndhello.
        MainWindowController_toast(window, text, QStringLiteral(""), TOAST_DURATION_MILLISECONDS);
    }
}

} // namespace NickelApi

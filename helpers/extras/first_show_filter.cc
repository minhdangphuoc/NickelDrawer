#include "first_show_filter.h"

#include <QCoreApplication>
#include <QEvent>
#include <QThread>
#include <QVariant>
#include <QWidget>

#include <NickelHook.h>

#include "debug_log.h"

// Dynamic property set on each widget the handler has seen, so the handler runs once per instance
// even though the widget is shown again every time its tab opens. It lives and dies with the widget,
// so a new widget at a reused address is still handled.
static const char *const HANDLED_PROPERTY = "extrasFirstShowHandled";

static const char *watchedClassName = NULL;
static FirstShowFilter::Handler watchedHandler = NULL;

class ShowEventFilter : public QObject {
public:
    explicit ShowEventFilter(QObject *parent) : QObject(parent) {}

protected:
    bool eventFilter(QObject *watched, QEvent *event)
    {
        if (event->type() != QEvent::Show || !watched->isWidgetType()) {
            return false;
        }
        if (qstrcmp(watched->metaObject()->className(), watchedClassName) != 0) {
            return false;
        }
        if (watched->property(HANDLED_PROPERTY).toBool()) {
            return false;
        }
        watched->setProperty(HANDLED_PROPERTY, true);
        DebugLog::write("first show: %s %p", watchedClassName, static_cast<void *>(watched));
        watchedHandler(static_cast<QWidget *>(watched));
        return false; // never swallow the event: Nickel still sees its Show
    }
};

// A QtStartUpFunction: runs on the GUI thread once QCoreApplication exists.
static void installOnApplication()
{
    QCoreApplication *application = QCoreApplication::instance();
    if (!application) {
        nh_log("first show: no QCoreApplication, %s widgets are not watched", watchedClassName);
        return;
    }
    if (QThread::currentThread() != application->thread()) {
        nh_log("first show: not on the GUI thread, %s widgets are not watched", watchedClassName);
        return;
    }
    // A child of the application: deleted with it, never before.
    application->installEventFilter(new ShowEventFilter(application));
    DebugLog::write("first show: watching %s widgets", watchedClassName);
}

namespace FirstShowFilter {

void watch(const char *className, Handler handler)
{
    if (watchedHandler) {
        nh_log("first show: already watching %s, ignoring %s", watchedClassName, className);
        return;
    }
    if (!className || !handler) {
        nh_log("first show: watch() needs a class name and a handler");
        return;
    }
    watchedClassName = className;
    watchedHandler = handler;
    // NickelHook's init normally runs before Nickel builds its QApplication.
    if (QCoreApplication::instance()) {
        installOnApplication();
    } else {
        qAddPreRoutine(installOnApplication);
    }
}

} // namespace FirstShowFilter

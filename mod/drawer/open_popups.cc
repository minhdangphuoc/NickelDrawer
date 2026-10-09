#include "open_popups.h"

#include <QDialog>
#include <QEvent>
#include <QList>
#include <QMenu>
#include <QPointer>

#include "debug_log.h"

static QList<QPointer<PopupWatcher> > watchers;

static void forgetClosedWatchers()
{
    for (int i = watchers.size() - 1; i >= 0; i--) {
        if (watchers[i].isNull() || !watchers[i]->popup()) {
            watchers.removeAt(i);
        }
    }
}

namespace OpenPopups {

void track(QWidget *popup, const char *kind)
{
    if (!popup) {
        return;
    }
    forgetClosedWatchers();
    watchers.append(new PopupWatcher(popup, kind));
    DebugLog::write("popup: %s %p opened (%s), %d open", kind, static_cast<void *>(popup),
                    popup->metaObject()->className(), watchers.size());
}

void closeAll(const char *reason)
{
    forgetClosedWatchers();
    // Copy first: closing a popup removes its watcher from the list.
    QList<QPointer<PopupWatcher> > open = watchers;
    for (int i = 0; i < open.size(); i++) {
        if (open[i].isNull() || !open[i]->popup()) { // closed meanwhile by another popup's close
            continue;
        }
        QWidget *popup = open[i]->popup();
        DebugLog::write("popup: closing %s %p (%s)", open[i]->kind(), static_cast<void *>(popup), reason);
        QDialog *dialog = qobject_cast<QDialog *>(popup);
        if (dialog) {
            dialog->reject();
        } else {
            popup->close();
        }
    }
}

bool isAnyOpen()
{
    forgetClosedWatchers();
    return !watchers.isEmpty();
}

} // namespace OpenPopups

PopupWatcher::PopupWatcher(QWidget *popup, const char *kind) : QObject(NULL), watched(popup), kindName(kind)
{
    connect(popup, SIGNAL(destroyed()), this, SLOT(onPopupDestroyed()));
    if (qobject_cast<QDialog *>(popup)) {
        connect(popup, SIGNAL(finished(int)), this, SLOT(onDialogFinished(int)));
    }
    popup->installEventFilter(this);
}

// A menu has no finished(): its Hide event is the close.
bool PopupWatcher::eventFilter(QObject *object, QEvent *event)
{
    if (object == watched && event->type() == QEvent::Hide && qobject_cast<QMenu *>(watched)) {
        untrack("hidden");
    }
    return false;
}

void PopupWatcher::onDialogFinished(int result)
{
    untrack(result == QDialog::Accepted ? "accepted" : "rejected (cancel, outside tap or closed)");
}

void PopupWatcher::onPopupDestroyed()
{
    watched = NULL;
    untrack("destroyed");
}

void PopupWatcher::untrack(const char *how)
{
    DebugLog::write("popup: %s %p closed: %s", kindName, static_cast<void *>(watched), how);
    if (watched) {
        watched->removeEventFilter(this);
        disconnect(watched, NULL, this, NULL);
        watched = NULL;
    }
    deleteLater();
}

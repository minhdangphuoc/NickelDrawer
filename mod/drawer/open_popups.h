#ifndef NICKELDRAWER_OPEN_POPUPS_H
#define NICKELDRAWER_OPEN_POPUPS_H

#include <QObject>

class QWidget;

// Every popup the drawer opens (row menu, info/result/confirm dialogs), so
// none is left orphaned over a page that changed or went away. The drawer and
// its app screens close them all on a page change, a page key, hiding and pop.
// Opening and closing each popup is written to the debug log.
namespace OpenPopups {

// Starts tracking an open popup; kind names it in the log ("row menu", ...).
// It stops being tracked when it hides or is destroyed.
void track(QWidget *popup, const char *kind);
// Closes every tracked popup: menus hide, dialogs reject (= Cancel).
void closeAll(const char *reason);
bool isAnyOpen();

} // namespace OpenPopups

// Watches one popup for the log and the registry. Not parented to the popup
// (Nickel's widgets never own our objects); deletes itself with the popup.
class PopupWatcher : public QObject {
    Q_OBJECT
public:
    PopupWatcher(QWidget *popup, const char *kind);

    QWidget *popup() const { return watched; }
    const char *kind() const { return kindName; }

protected:
    bool eventFilter(QObject *object, QEvent *event);

private slots:
    void onPopupDestroyed();
    void onDialogFinished(int result);

private:
    void untrack(const char *how);

    QWidget *watched;
    const char *kindName;
};

#endif

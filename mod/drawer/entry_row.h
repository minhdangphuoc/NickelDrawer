#ifndef NICKELDRAWER_ENTRY_ROW_H
#define NICKELDRAWER_ENTRY_ROW_H

#include <QObject>
#include <QPointer>
#include <QWidget>

#include "drawer_model.h"
#include "painted_marks.h"

class ModelClient;

// The "Apps" row in Nickel's More tab (docs/spec/ui-kit.md §5.1): a Nickel
// IconLeftButton after "Beta Features". Tap opens the drawer. Its second line
// is a summary of running apps and updates, and a black dot shows when an
// update exists; both come from `appsctl drawer model`. If appsctl fails the
// row just reads "Apps".
//
// Deliberately NOT a child of the row: IconButtonBase::tapGesture runs
// __dynamic_cast on every direct child of the button on press and release
// (re/notes/protoA-crash.md). It is deleted when the row is.
class EntryRow : public QObject {
    Q_OBJECT
public:
    // MoreView-created handler registered with the bridge. Fails soft: logs
    // and leaves the More tab untouched when anything is missing.
    static void addTo(QWidget *moreView);
    // The row's object name: the Tweaks row goes right after it.
    static const char *rowObjectName();

protected:
    bool eventFilter(QObject *watched, QEvent *event);

private slots:
    void onTapped();
    // Runs from the event loop after Nickel's tap gesture handler returned.
    void handleTap();
    void refreshSummary();
    void onModelLoaded(const DrawerModel &model);

private:
    explicit EntryRow(QWidget *row);

    void showSummary(const QString &summary, bool hasUpdate);
    void placeDot();

    QPointer<QWidget> row;
    QPointer<UpdateDot> updateDot; // child of the row, so it can go first
    ModelClient *client;
    bool isTapPending;
};

// Second line of the row: "{name} {status}" for running apps with a status,
// joined with " · ", then "1 update" / "{n} updates". Empty when nothing to say.
QString entrySummary(const DrawerModel &model, int *updateCount);

#endif

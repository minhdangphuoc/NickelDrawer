#ifndef NICKELDRAWER_SCREEN_SESSION_H
#define NICKELDRAWER_SCREEN_SESSION_H

#include <QObject>
#include <QPoint>
#include <QStringList>

#include "event_queue.h"

#include "drawer_model.h"
#include "screen_model.h"

class ActionRunner;
class ModelClient;
class QTimer;
class ScreenView;

// One open app screen (Phase 4, opt out with .adds/nickeldrawer/no-screens): loads it
// with `appsctl drawer screen`, shows it in a ScreenView over the drawer page
// and turns taps into `drawer event` runs through the drawer's ActionRunner
// (same Wi-Fi, dialog and rescan rules as a drawer tap).
//   button / row / empty → its action (button confirm asked first)
//   toggle / radio       → shown at once, then the action named like the
//                          variable with NICKELDRAWER_VALUE
//   field                → Nickel's keyboard, else an "edit on your phone" notice
//   list row (tap)       → the list's tap action with the row's primary text
//   header action        → push another screen, or run an action
// Gallery (docs/spec/wallpaper-gallery.md): a tile tap pushes the viewer (or
// runs an action) with the tile id as the screen value, a long press opens
// the tile menu (tile_menu.h); the viewer's primary / secondary buttons send
// the picture on screen as the event value. Radios and toggles show their new
// value at once (deferred, never inside the tap handler) and queue their
// event; the queue sends one at a time (event_queue.h).
// Back pops pushed screens (navigation.h: the parent comes back reloaded at
// its page), then closes. A live screen (state action) is
// re-read every 5 s while idle. Child of its view: it ends with the view.
class ScreenSession : public QObject {
    Q_OBJECT
public:
    ScreenSession(const DrawerItem &app, QWidget *drawerView, ActionRunner *runner);

    void start();

signals:
    // The screen view is gone; the drawer underneath is visible again.
    void closed();

private slots:
    void reload();
    void onScreenLoaded(bool isLoaded, const ScreenModel &screen, const QString &failure);
    void onBackRequested();
    void onHeaderActionTapped();
    void onHeaderIconTapped(int index);
    void onComponentTapped(int component, const QString &value);
    void onTileLongPressed(int component, const QString &id, const QPoint &anchor);
    void onTileMenuChoice(int index);
    void onListRowLongPressed(int component, int row, const QString &primary, const QPoint &anchor);
    void onListMenuChoice(int index);
    void onTileMenuClosed();
    void onViewerPictureChanged(const QString &id);
    void pumpEventQueue();
    void retryLoading();
    void onConfirmAnswer(bool isAccepted);
    void onFieldCommitted(const QString &value);
    void onFieldFinished();
    void onBusyChanged(bool isBusy);
    void onLiveTimeout();

private:
    void load(const QString &screenId, const QString &value);
    void showLoadError(const QString &failure);
    void runAction(const QString &actionId);
    void runWithValue(const QString &actionId, const QString &value);
    void runEvent(const ScreenEvent &event);
    void tapButton(const ScreenComponent &button);
    void tapToggle(int index, const ScreenComponent &toggle);
    void tapRadio(int index, const ScreenComponent &radio, const QString &option);
    void tapField(const ScreenComponent &field);
    void tapGallery(const ScreenComponent &gallery, const QString &tileId);
    void tapViewerButton(const ScreenComponent &viewer, const QString &button);
    void askToConfirm(const DeclaredDialog &confirm, const QString &action, const QString &value, const QString &nextValue,
                      bool hasValue);
    void startConfirmedAction(const QString &action, const QString &value, const QString &nextValue, bool hasNextValue);
    void pushSubScreen(const QString &screenId, const QString &value);
    bool leaveSubScreen();
    void queueEvent(const QString &key, const QString &action, const QString &value);
    bool keepsPreviousScreen(bool isReload, bool isLoaded, const ScreenModel &screen) const;
    bool isViewerEmpty(const ScreenModel &screen) const;
    QString currentScreenValue() const;
    void close();
    QString currentScreenId() const;

    DrawerItem app;
    ActionRunner *runner;
    ModelClient *client;
    ScreenView *view;
    QTimer *liveTimer;
    bool isLoading;
    bool isEditingField;
    bool isConfirmOpen;
    bool isMenuOpen;
    bool isReloading;
    bool isQueuedEventRunning;
    bool needsReloadAfterQueue;
    EventQueue::Queue eventQueue;
    // Copies taken when the tile menu opened: the screen may reload under it.
    PictureTile menuTile;
    QString menuListBind; // list row menu: the list, the row and its primary text
    int menuRow;
    QString menuRowPrimary;
    QList<TileMenuItem> menuItems;
    QString pendingConfirmAction;
    QString pendingConfirmValue;
    bool hasPendingConfirmValue;
    QString pendingNextValue;
    bool hasPendingNextValue;
    QString editedFieldAction;
};

#endif

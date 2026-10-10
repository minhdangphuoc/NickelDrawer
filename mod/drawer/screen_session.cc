#include "screen_session.h"

#include <QMetaObject>
#include <QTimer>

#include <NickelHook.h>

#include "action_runner.h"
#include "debug_log.h"
#include "dialogs.h"
#include "fallback_strings.h"
#include "field_editor.h"
#include "model_client.h"
#include "navigation.h"
#include "nickel_bridge.h"
#include "screen_logic.h"
#include "screen_view.h"
#include "tile_menu.h"

// Host actions of the synthetic error screen; '#' cannot start an app action id.
static const char *const HOST_BACK_ACTION = "#back";
static const char *const HOST_RETRY_ACTION = "#retry";
static const char *const VALUE_TRUE = "true";
static const char *const VALUE_FALSE = "false";
static const int RUN_FROM_EVENT_LOOP = 0;
static const char *const VIEWER_PRIMARY = "primary";
static const char *const VIEWER_SECONDARY = "secondary";

static ScreenComponent textComponent(const QString &text, const char *style)
{
    ScreenComponent component;
    component.type = QStringLiteral("text");
    component.text = text;
    component.style = QString::fromLatin1(style);
    return component;
}

static ScreenComponent buttonComponent(const QString &label, const char *action, const char *style)
{
    ScreenComponent component;
    component.type = QStringLiteral("button");
    component.text = label;
    component.action = QString::fromLatin1(action);
    component.style = QString::fromLatin1(style);
    return component;
}

// Shown until the first reply arrives; it has no screen id, so it is not flashed.
static ScreenModel loadingScreen(const DrawerItem &app)
{
    ScreenModel screen;
    screen.app = app.id;
    screen.title = app.name;
    screen.body << textComponent(FallbackStrings::text(FallbackStrings::LOADING), "caption");
    return screen;
}

// ui-kit §6.5 blocking error: heading, the reason, Back and Try again.
static ScreenModel errorScreen(const DrawerItem &app, const QString &failure)
{
    ScreenModel screen;
    screen.app = app.id;
    screen.screen = QStringLiteral("#error");
    screen.title = app.name;
    screen.body << textComponent(FallbackStrings::text(FallbackStrings::SCREEN_CANT_OPEN), "heading")
                << textComponent(failure, "body");
    screen.footer << buttonComponent(FallbackStrings::text(FallbackStrings::SCREEN_BACK), HOST_BACK_ACTION, "secondary")
                  << buttonComponent(FallbackStrings::text(FallbackStrings::TRY_AGAIN), HOST_RETRY_ACTION, "primary");
    return screen;
}

ScreenSession::ScreenSession(const DrawerItem &screenApp, QWidget *drawerView, ActionRunner *actionRunner)
    : QObject(NULL), app(screenApp), runner(actionRunner), client(NULL), view(new ScreenView(screenApp.name, drawerView)),
      liveTimer(new QTimer(this)), isLoading(false), isEditingField(false), isConfirmOpen(false), isMenuOpen(false),
      isReloading(false), isQueuedEventRunning(false), needsReloadAfterQueue(false), hasPendingConfirmValue(false),
      hasPendingNextValue(false)
{
    setParent(view);
    client = new ModelClient(this);
    liveTimer->setInterval(ScreenLogic::LIVE_REFRESH_MILLISECONDS);
    connect(liveTimer, SIGNAL(timeout()), this, SLOT(onLiveTimeout()));
    connect(client, SIGNAL(screenLoaded(bool, ScreenModel, QString)), this,
            SLOT(onScreenLoaded(bool, ScreenModel, QString)));
    connect(view, SIGNAL(backRequested()), this, SLOT(onBackRequested()));
    connect(view, SIGNAL(headerActionTapped()), this, SLOT(onHeaderActionTapped()));
    connect(view, SIGNAL(headerIconTapped(int)), this, SLOT(onHeaderIconTapped(int)));
    connect(view, SIGNAL(tileLongPressed(int, QString, QPoint)), this, SLOT(onTileLongPressed(int, QString, QPoint)));
    connect(view, SIGNAL(viewerPictureChanged(QString)), this, SLOT(onViewerPictureChanged(QString)));
    connect(view, SIGNAL(componentTapped(int, QString)), this, SLOT(onComponentTapped(int, QString)));
    connect(runner, SIGNAL(busyChanged(bool)), this, SLOT(onBusyChanged(bool)));
    connect(runner, SIGNAL(modelMayHaveChanged()), this, SLOT(reload()));
    view->setInputBlocked(runner->isRunning());
}

void ScreenSession::start()
{
    nh_log("screen: open %s", qPrintable(app.id));
    view->showScreen(loadingScreen(app));
    load(QString(), QString());
}

QString ScreenSession::currentScreenId() const
{
    // The start screen is requested by an empty id; a sub-screen by its own.
    const Navigation::Entry *top = Navigation::top();
    bool isSubScreen = top && top->level == Navigation::LEVEL_SUB_SCREEN;
    return isSubScreen ? QString::fromUtf8(top->id) : QString();
}

// The value the screen on top was opened with (the viewer's picture id).
QString ScreenSession::currentScreenValue() const
{
    const Navigation::Entry *top = Navigation::top();
    bool isSubScreen = top && top->level == Navigation::LEVEL_SUB_SCREEN;
    return isSubScreen ? QString::fromUtf8(top->value) : QString();
}

void ScreenSession::load(const QString &screenId, const QString &value)
{
    isLoading = true;
    client->requestScreen(app.id, screenId, value);
}

// Events still waiting would be undone by a reload that raced them; the last
// one's reply reloads once the queue is empty.
void ScreenSession::reload()
{
    if (!eventQueue.isIdle()) {
        needsReloadAfterQueue = true;
        DebugLog::write("screen: reload held until %d queued event(s) are sent", eventQueue.waiting());
        return;
    }
    needsReloadAfterQueue = false;
    if (!isLoading) {
        isReloading = true;
        load(currentScreenId(), currentScreenValue());
    }
}

// A real screen of this app is on view (not the loading placeholder or the error screen).
static bool isRealScreen(const ScreenModel &shown, const QString &app)
{
    return shown.app == app && !shown.screen.isEmpty() && shown.screen != QLatin1String("#error");
}

// A failed or stale (defaults only) reload never replaces what is on screen.
bool ScreenSession::keepsPreviousScreen(bool isReload, bool isLoaded, const ScreenModel &screen) const
{
    const ScreenModel &shown = view->screen();
    if (!isRealScreen(shown, app.id)) {
        return false;
    }
    if (!isLoaded) {
        return isReload;
    }
    return screen.isStale && screen.app == shown.app && screen.screen == shown.screen;
}

bool ScreenSession::isViewerEmpty(const ScreenModel &screen) const
{
    for (int i = 0; i < screen.body.size(); i++) {
        if (screen.body[i].type == QLatin1String("viewer") && screen.body[i].tiles.isEmpty()) {
            return true;
        }
    }
    return false;
}

void ScreenSession::onScreenLoaded(bool isLoaded, const ScreenModel &screen, const QString &failure)
{
    bool wasReload = isReloading;
    isReloading = false;
    isLoading = false;
    if (keepsPreviousScreen(wasReload, isLoaded, screen)) {
        nh_log("screen: %s reply kept the previous screen (%s)", qPrintable(app.id),
               isLoaded ? "stale: the app's state failed" : qPrintable(failure));
        return;
    }
    if (!isLoaded) {
        showLoadError(failure);
        return;
    }
    const Navigation::Entry *top = Navigation::top();
    bool isOnSubScreen = top && top->level == Navigation::LEVEL_SUB_SCREEN;
    if (isOnSubScreen && isViewerEmpty(screen) && leaveSubScreen()) {
        return; // the last picture was removed: back to the gallery
    }
    view->showScreen(screen);
    if (isOnSubScreen && !view->viewerPictureId().isEmpty()) {
        Navigation::setTopValue(view->viewerPictureId());
    }
    if (screen.isLive && !liveTimer->isActive()) {
        liveTimer->start();
    } else if (!screen.isLive) {
        liveTimer->stop();
    }
}

void ScreenSession::showLoadError(const QString &failure)
{
    nh_log("screen: %s failed to load", qPrintable(app.id));
    liveTimer->stop();
    view->showScreen(errorScreen(app, failure));
}

void ScreenSession::onLiveTimeout()
{
    bool isIdle = !isLoading && !isEditingField && !isConfirmOpen && !isMenuOpen && !runner->isRunning() &&
                  eventQueue.isIdle();
    if (isIdle && view->isVisible()) {
        reload();
    }
}

// A queued radio / toggle event runs without blocking taps (the next tap is
// queued too); buttons, header actions and back stay blocked while their
// action runs.
void ScreenSession::onBusyChanged(bool isBusy)
{
    if (!isBusy && eventQueue.isInFlight()) {
        eventQueue.finishInFlight();
        isQueuedEventRunning = false;
        DebugLog::write("queue: event answered, %d waiting", eventQueue.waiting());
    }
    view->setInputBlocked(isBusy && !isQueuedEventRunning);
    if (!isBusy) {
        QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(pumpEventQueue()));
    }
}

void ScreenSession::pumpEventQueue()
{
    if (runner->isRunning()) {
        return; // sent when the runner turns idle
    }
    EventQueue::Event next;
    if (!eventQueue.takeNext(&next)) {
        return;
    }
    ScreenEvent event;
    event.actionId = QString::fromUtf8(next.action);
    event.value = QString::fromUtf8(next.value);
    event.hasValue = true;
    event.screenId = QString::fromUtf8(next.screen);
    isQueuedEventRunning = true;
    DebugLog::write("queue: sending %s (%d still waiting)", next.action, eventQueue.waiting());
    runner->runScreenEvent(app, event);
}

// The control's bind is the key: a newer tap replaces its queued event.
void ScreenSession::queueEvent(const QString &key, const QString &action, const QString &value)
{
    if (action.isEmpty()) {
        DebugLog::write("screen: tap has no action (read-only control)");
        return;
    }
    EventQueue::Event event = EventQueue::makeEvent(qPrintable(key), qPrintable(action), qPrintable(value),
                                                    qPrintable(view->screen().screen));
    const char *outcome = "queued";
    switch (eventQueue.enqueue(event)) {
    case EventQueue::ENQUEUED:
        break;
    case EventQueue::REPLACED_QUEUED:
        outcome = "replaced the queued event of the same control";
        break;
    case EventQueue::DROPPED_IN_FLIGHT:
        outcome = "dropped: the same value is being sent";
        break;
    case EventQueue::DROPPED_FULL:
        outcome = "refused: the queue is full";
        break;
    }
    DebugLog::write("queue: %s for %s: %s (%d waiting)", qPrintable(action), qPrintable(key), outcome, eventQueue.waiting());
    QTimer::singleShot(RUN_FROM_EVENT_LOOP, this, SLOT(pumpEventQueue()));
}

// ---- Navigation ----

void ScreenSession::onBackRequested()
{
    if (leaveSubScreen()) {
        return;
    }
    close(); // DrawerSession pops to the drawer and restores it
}

// Pops one pushed screen and reloads its parent at the page it had; coming
// back from the viewer, at the grid page holding the last viewed picture.
bool ScreenSession::leaveSubScreen()
{
    const Navigation::Entry *top = Navigation::top();
    if (!top || top->level != Navigation::LEVEL_SUB_SCREEN) {
        return false;
    }
    QString leavingValue = QString::fromUtf8(top->value);
    Navigation::Entry parent;
    if (!Navigation::pop(&parent)) {
        return false;
    }
    view->dropPictures();
    view->restorePageOnNextScreen(parent.state.page); // reloaded fresh, at the page it had
    view->restoreTileOnNextScreen(leavingValue);
    load(currentScreenId(), currentScreenValue());
    return true;
}

void ScreenSession::pushSubScreen(const QString &screenId, const QString &value)
{
    if (screenId.isEmpty()) {
        return;
    }
    if (Navigation::push(Navigation::LEVEL_SUB_SCREEN, screenId, Navigation::stateOf(0, view->currentPage()), value)) {
        load(screenId, value);
    }
}

void ScreenSession::close()
{
    nh_log("screen: close %s", qPrintable(app.id));
    liveTimer->stop();
    if (!eventQueue.isIdle()) {
        DebugLog::write("screen: closing with %d queued event(s) unsent", eventQueue.waiting());
    }
    view->dropPictures();
    emit closed();
    view->hide();
    view->deleteLater(); // and this session with it
}

void ScreenSession::onHeaderActionTapped()
{
    QString push = view->screen().headerPush; // copies: the screen is replaced by the reload
    QString action = view->screen().headerAction;
    if (!push.isEmpty()) {
        pushSubScreen(push, QString());
    } else if (!action.isEmpty()) {
        runAction(action);
    }
}

void ScreenSession::onHeaderIconTapped(int index)
{
    if (index < 0 || index >= view->screen().headerActions.size()) {
        return;
    }
    HeaderIconAction icon = view->screen().headerActions[index];
    DebugLog::write("screen: header icon \"%s\" -> %s \"%s\"", qPrintable(icon.icon), icon.push.isEmpty() ? "action" : "push",
                    qPrintable(icon.push.isEmpty() ? icon.action : icon.push));
    if (!icon.push.isEmpty()) {
        pushSubScreen(icon.push, QString());
    } else {
        runAction(icon.action);
    }
}

// ---- Taps ----

// Every handler works on a copy of the component: the layout the canvas
// handed it from is rebuilt by any showScreen (a reload, an optimistic
// toggle) and its components are freed (device bug: the radio's action was
// read from freed memory and the event never left).
void ScreenSession::onComponentTapped(int index, const QString &value)
{
    ScreenComponent component;
    if (!view->componentAt(index, &component)) {
        DebugLog::write("screen: tap on component %d dropped (no such component)", index);
        return;
    }
    const QString &type = component.type;
    bool isQueueable = type == QLatin1String("toggle") || type == QLatin1String("radio");
    if (runner->isRunning() && !(isQueuedEventRunning && isQueueable)) {
        DebugLog::write("screen: tap on component %d dropped (an action is still running)", index);
        return;
    }
    DebugLog::write("screen: tap on %s %d, action \"%s\"", qPrintable(type), index, qPrintable(component.action));
    if (type == QLatin1String("button")) {
        tapButton(component);
    } else if (type == QLatin1String("row") || type == QLatin1String("empty")) {
        runAction(component.action);
    } else if (type == QLatin1String("toggle")) {
        tapToggle(index, component);
    } else if (type == QLatin1String("radio")) {
        tapRadio(index, component, value);
    } else if (type == QLatin1String("field")) {
        tapField(component);
    } else if (type == QLatin1String("list")) {
        runWithValue(component.tap, value);
    } else if (type == QLatin1String("gallery")) {
        tapGallery(component, value);
    } else if (type == QLatin1String("viewer")) {
        tapViewerButton(component, value);
    }
}

// An empty value is the empty state's button; otherwise a tile.
void ScreenSession::tapGallery(const ScreenComponent &gallery, const QString &tileId)
{
    if (tileId.isEmpty()) {
        runAction(gallery.galleryEmpty.action);
    } else if (!gallery.tapPush.isEmpty()) {
        pushSubScreen(gallery.tapPush, tileId);
    } else {
        runWithValue(gallery.tapAction, tileId);
    }
}

void ScreenSession::startConfirmedAction(const QString &action, const QString &value, const QString &nextValue,
                                         bool hasNextValue)
{
    if (hasNextValue) {
        // The neighbour is shown after the reload (the picture removed is gone).
        Navigation::setTopValue(nextValue);
    }
    runWithValue(action, value);
}

// The picture on screen is the event's value; Remove asks first and, once
// accepted, stores the neighbour as the screen value so the reload lands on it.
void ScreenSession::tapViewerButton(const ScreenComponent &viewer, const QString &button)
{
    QString shownId = view->viewerPictureId();
    if (shownId.isEmpty()) {
        return;
    }
    if (button == QLatin1String(VIEWER_PRIMARY)) {
        runWithValue(viewer.primary.action, shownId);
        return;
    }
    if (button != QLatin1String(VIEWER_SECONDARY)) {
        return;
    }
    int shownIndex = -1;
    for (int i = 0; i < viewer.tiles.size(); i++) {
        if (viewer.tiles[i].id == shownId) {
            shownIndex = i;
        }
    }
    int neighbour = ScreenLogic::neighbourAfterRemoval(shownIndex, viewer.tiles.size());
    QString neighbourId = neighbour >= 0 ? viewer.tiles[neighbour].id : QString();
    if (!viewer.secondary.hasConfirm) {
        startConfirmedAction(viewer.secondary.action, shownId, neighbourId, true);
        return;
    }
    askToConfirm(viewer.secondary.confirm, viewer.secondary.action, shownId, neighbourId, true);
}

void ScreenSession::tapButton(const ScreenComponent &button)
{
    if (button.action == QLatin1String(HOST_BACK_ACTION)) {
        onBackRequested();
        return;
    }
    if (button.action == QLatin1String(HOST_RETRY_ACTION)) {
        // Rebuilds the screen: from the event loop, not inside the tap.
        QMetaObject::invokeMethod(this, "retryLoading", Qt::QueuedConnection);
        return;
    }
    if (!button.hasConfirm) {
        runAction(button.action);
        return;
    }
    askToConfirm(button.confirm, button.action, QString(), QString(), false);
}

void ScreenSession::retryLoading()
{
    view->showScreen(loadingScreen(app));
    load(currentScreenId(), currentScreenValue());
}

// A screen action asks first because it changes something: only the dialog's
// buttons answer, a stray tap outside does not. Everything the answer needs is
// copied here; the screen may reload while the dialog is open.
void ScreenSession::askToConfirm(const DeclaredDialog &confirm, const QString &action, const QString &value,
                                 const QString &nextValue, bool hasValue)
{
    QString cancel = confirm.cancel.isEmpty() ? FallbackStrings::text(FallbackStrings::CANCEL) : confirm.cancel;
    QString title = confirm.title.isEmpty() ? view->screen().title : confirm.title;
    pendingConfirmAction = action;
    pendingConfirmValue = value;
    pendingNextValue = nextValue;
    hasPendingConfirmValue = hasValue;
    isConfirmOpen = true;
    ConfirmQuestion *question = new ConfirmQuestion(title, confirm.text, confirm.ok, cancel, ConfirmQuestion::DESTRUCTIVE);
    connect(question, SIGNAL(answered(bool)), this, SLOT(onConfirmAnswer(bool)));
    question->ask();
}

void ScreenSession::onConfirmAnswer(bool isAccepted)
{
    isConfirmOpen = false;
    if (!isAccepted) {
        return;
    }
    if (hasPendingConfirmValue) {
        startConfirmedAction(pendingConfirmAction, pendingConfirmValue, pendingNextValue, true);
    } else {
        runAction(pendingConfirmAction);
    }
}

// The new value is shown at once but from the event loop (rebuilding the
// layout inside the tap handler freed what the handler still read), then the
// event is queued.
void ScreenSession::tapToggle(int index, const ScreenComponent &toggle)
{
    bool isOn = !toggle.isOn;
    QString action = toggle.action;
    QString key = toggle.bind.isEmpty() ? action : toggle.bind;
    QMetaObject::invokeMethod(view, "showToggleState", Qt::QueuedConnection, Q_ARG(int, index), Q_ARG(bool, isOn));
    queueEvent(key, action, QString::fromLatin1(isOn ? VALUE_TRUE : VALUE_FALSE));
}

void ScreenSession::tapRadio(int index, const ScreenComponent &radio, const QString &option)
{
    if (option == radio.value) {
        return;
    }
    QString action = radio.action;
    QString key = radio.bind.isEmpty() ? action : radio.bind;
    QMetaObject::invokeMethod(view, "showRadioValue", Qt::QueuedConnection, Q_ARG(int, index), Q_ARG(QString, option));
    queueEvent(key, action, option);
}

void ScreenSession::tapField(const ScreenComponent &field)
{
    FieldEditor *editor = FieldEditor::open(field.text, field.value, field.isSecret, field.keyboard, view);
    if (!editor) {
        // ui-kit §1.7: long or secret input goes through the phone.
        QString notice = FallbackStrings::text(FallbackStrings::SCREEN_EDIT_ON_PHONE);
        if (NickelBridge::hasToast()) {
            NickelBridge::showToast(notice);
        } else {
            Dialogs::showMessage(field.text, notice);
        }
        return;
    }
    isEditingField = true;
    editedFieldAction = field.action;
    connect(editor, SIGNAL(committed(QString)), this, SLOT(onFieldCommitted(QString)));
    connect(editor, SIGNAL(finished()), this, SLOT(onFieldFinished()));
}

void ScreenSession::onFieldCommitted(const QString &value)
{
    runWithValue(editedFieldAction, value);
}

void ScreenSession::onFieldFinished()
{
    isEditingField = false;
}

// ---- Events ----

void ScreenSession::runAction(const QString &actionId)
{
    ScreenEvent event;
    event.actionId = actionId;
    runEvent(event);
}

void ScreenSession::runWithValue(const QString &actionId, const QString &value)
{
    ScreenEvent event;
    event.actionId = actionId;
    event.value = value;
    event.hasValue = true;
    runEvent(event);
}

void ScreenSession::runEvent(const ScreenEvent &event)
{
    if (event.actionId.isEmpty()) {
        DebugLog::write("screen: tap has no action (read-only control or untappable row)");
        return;
    }
    ScreenEvent withScreen = event;
    withScreen.screenId = view->screen().screen;
    runner->runScreenEvent(app, withScreen);
}

// ---- Gallery ----

void ScreenSession::onTileLongPressed(int index, const QString &tileId, const QPoint &anchor)
{
    ScreenComponent gallery;
    if (!view->componentAt(index, &gallery) || runner->isRunning() || isMenuOpen || gallery.menu.isEmpty()) {
        DebugLog::write("screen: tile menu not opened (%s)", isMenuOpen ? "one is open" : "busy or no menu");
        return;
    }
    bool isFound = false;
    for (int i = 0; i < gallery.tiles.size() && !isFound; i++) {
        if (gallery.tiles[i].id == tileId) {
            menuTile = gallery.tiles[i];
            isFound = true;
        }
    }
    if (!isFound) {
        return;
    }
    menuItems = gallery.menu;
    isMenuOpen = true;
    TileMenu *menu = new TileMenu(menuTile.label, menuItems);
    connect(menu, SIGNAL(itemChosen(int)), this, SLOT(onTileMenuChoice(int)));
    connect(menu, SIGNAL(closed()), this, SLOT(onTileMenuClosed()));
    menu->popup(anchor);
}

void ScreenSession::onTileMenuChoice(int index)
{
    if (index < 0 || index >= menuItems.size()) {
        return;
    }
    TileMenuItem item = menuItems[index]; // copies: the choice may reload the screen
    PictureTile tile = menuTile;
    if (item.isInfo) {
        Dialogs::showMessage(tile.label, tile.info.isEmpty() ? tile.label : tile.info);
    } else if (!item.push.isEmpty()) {
        pushSubScreen(item.push, tile.id);
    } else {
        runWithValue(item.action, tile.id);
    }
}

void ScreenSession::onTileMenuClosed()
{
    isMenuOpen = false;
}

// The viewer keeps the stack entry's value current, so a reload and Back
// know the picture the user is looking at.
void ScreenSession::onViewerPictureChanged(const QString &id)
{
    const Navigation::Entry *top = Navigation::top();
    if (top && top->level == Navigation::LEVEL_SUB_SCREEN && !id.isEmpty()) {
        Navigation::setTopValue(id);
    }
}

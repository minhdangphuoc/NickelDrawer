#include "screen_session.h"

#include <QTimer>

#include <NickelHook.h>

#include "action_runner.h"
#include "dialogs.h"
#include "fallback_strings.h"
#include "field_editor.h"
#include "model_client.h"
#include "nickel_bridge.h"
#include "screen_logic.h"
#include "screen_view.h"

// Host actions of the synthetic error screen; '#' cannot start an app action id.
static const char *const HOST_BACK_ACTION = "#back";
static const char *const HOST_RETRY_ACTION = "#retry";
static const char *const VALUE_TRUE = "true";
static const char *const VALUE_FALSE = "false";

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
    : QObject(NULL), app(screenApp), runner(actionRunner), client(NULL), view(new ScreenView(drawerView)),
      liveTimer(new QTimer(this)), isLoading(false), isEditingField(false), isConfirmOpen(false)
{
    setParent(view);
    client = new ModelClient(this);
    screenStack << QString();
    liveTimer->setInterval(ScreenLogic::LIVE_REFRESH_MILLISECONDS);
    connect(liveTimer, SIGNAL(timeout()), this, SLOT(onLiveTimeout()));
    connect(client, SIGNAL(screenLoaded(bool, ScreenModel, QString)), this,
            SLOT(onScreenLoaded(bool, ScreenModel, QString)));
    connect(view, SIGNAL(backRequested()), this, SLOT(onBackRequested()));
    connect(view, SIGNAL(headerActionTapped()), this, SLOT(onHeaderActionTapped()));
    connect(view, SIGNAL(componentTapped(int, QString)), this, SLOT(onComponentTapped(int, QString)));
    connect(runner, SIGNAL(busyChanged(bool)), this, SLOT(onBusyChanged(bool)));
    connect(runner, SIGNAL(modelMayHaveChanged()), this, SLOT(reload()));
    view->setInputBlocked(runner->isRunning());
}

void ScreenSession::start()
{
    nh_log("screen: open %s", qPrintable(app.id));
    view->showScreen(loadingScreen(app));
    load(QString());
}

QString ScreenSession::currentScreenId() const
{
    return screenStack.last();
}

void ScreenSession::load(const QString &screenId)
{
    isLoading = true;
    client->requestScreen(app.id, screenId);
}

void ScreenSession::reload()
{
    if (!isLoading) {
        load(currentScreenId());
    }
}

void ScreenSession::onScreenLoaded(bool isLoaded, const ScreenModel &screen, const QString &failure)
{
    isLoading = false;
    if (!isLoaded) {
        showLoadError(failure);
        return;
    }
    view->showScreen(screen);
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
    bool isIdle = !isLoading && !isEditingField && !isConfirmOpen && !runner->isRunning();
    if (isIdle && view->isVisible()) {
        reload();
    }
}

void ScreenSession::onBusyChanged(bool isBusy)
{
    view->setInputBlocked(isBusy);
}

// ---- Navigation ----

void ScreenSession::onBackRequested()
{
    if (screenStack.size() > 1) {
        screenStack.removeLast();
        load(currentScreenId());
        return;
    }
    close();
}

void ScreenSession::close()
{
    nh_log("screen: close %s", qPrintable(app.id));
    liveTimer->stop();
    emit closed();
    view->hide();
    view->deleteLater(); // and this session with it
}

void ScreenSession::onHeaderActionTapped()
{
    const ScreenModel &screen = view->screen();
    if (!screen.headerPush.isEmpty()) {
        screenStack << screen.headerPush;
        load(screen.headerPush);
    } else if (!screen.headerAction.isEmpty()) {
        runAction(screen.headerAction);
    }
}

// ---- Taps ----

void ScreenSession::onComponentTapped(int index, const QString &value)
{
    const ScreenComponent *component = view->component(index);
    if (!component || runner->isRunning()) {
        return;
    }
    const QString &type = component->type;
    if (type == QLatin1String("button")) {
        tapButton(*component);
    } else if (type == QLatin1String("row") || type == QLatin1String("empty")) {
        runAction(component->action);
    } else if (type == QLatin1String("toggle")) {
        tapToggle(index, *component);
    } else if (type == QLatin1String("radio")) {
        tapRadio(index, *component, value);
    } else if (type == QLatin1String("field")) {
        tapField(*component);
    } else if (type == QLatin1String("list")) {
        runWithValue(component->tap, value);
    }
}

void ScreenSession::tapButton(const ScreenComponent &button)
{
    if (button.action == QLatin1String(HOST_BACK_ACTION)) {
        onBackRequested();
        return;
    }
    if (button.action == QLatin1String(HOST_RETRY_ACTION)) {
        view->showScreen(loadingScreen(app));
        load(currentScreenId());
        return;
    }
    if (!button.hasConfirm) {
        runAction(button.action);
        return;
    }
    const DeclaredDialog &confirm = button.confirm;
    QString cancel = confirm.cancel.isEmpty() ? FallbackStrings::text(FallbackStrings::CANCEL) : confirm.cancel;
    QString title = confirm.title.isEmpty() ? view->screen().title : confirm.title;
    pendingConfirmAction = button.action;
    isConfirmOpen = true;
    ConfirmQuestion *question = new ConfirmQuestion(title, confirm.text, confirm.ok, cancel);
    connect(question, SIGNAL(answered(bool)), this, SLOT(onConfirmAnswer(bool)));
    question->ask();
}

void ScreenSession::onConfirmAnswer(bool isAccepted)
{
    isConfirmOpen = false;
    if (isAccepted) {
        runAction(pendingConfirmAction);
    }
}

void ScreenSession::tapToggle(int index, const ScreenComponent &toggle)
{
    bool isOn = !toggle.isOn;
    view->showToggleState(index, isOn);
    runWithValue(toggle.action, QString::fromLatin1(isOn ? VALUE_TRUE : VALUE_FALSE));
}

void ScreenSession::tapRadio(int index, const ScreenComponent &radio, const QString &option)
{
    if (option == radio.value) {
        return;
    }
    view->showRadioValue(index, option);
    runWithValue(radio.action, option);
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
        return; // a read-only control or an untappable row
    }
    ScreenEvent withScreen = event;
    withScreen.screenId = view->screen().screen;
    runner->runScreenEvent(app, withScreen);
}

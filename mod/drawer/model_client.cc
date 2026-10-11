#include "model_client.h"

#include <QStringList>

#include <NickelHook.h>

#include "debug_log.h"
#include "drawer_logic.h"
#include "fallback_strings.h"
#include "model_parser.h"
#include "tool_call.h"

static const char *const APPSCTL_PATH = "/mnt/onboard/.adds/koboapps/bin/appsctl";

static QStringList drawerArguments(const char *command)
{
    QStringList arguments;
    arguments << QStringLiteral("drawer") << QString::fromLatin1(command);
    return arguments;
}

static QString outcomeFailure(ToolCall::Outcome outcome)
{
    switch (outcome) {
    case ToolCall::OUTCOME_NOT_STARTED:
        return FallbackStrings::text(FallbackStrings::APPSCTL_MISSING);
    case ToolCall::OUTCOME_TIMED_OUT:
        return FallbackStrings::text(FallbackStrings::APPSCTL_TIMEOUT);
    case ToolCall::OUTCOME_CRASHED:
    case ToolCall::OUTCOME_EXITED:
        break;
    }
    return FallbackStrings::text(FallbackStrings::APPSCTL_FAILED);
}

static ActionResult failedResult(const QString &failure)
{
    ActionResult result;
    result.isOk = false;
    result.needsWifi = false;
    result.text = failure;
    return result;
}

ModelClient::ModelClient(QObject *parent) : QObject(parent) {}

void ModelClient::requestModel()
{
    startCall(REQUEST_MODEL, drawerArguments("model"), DrawerLogic::timeoutMillisecondsFor(DrawerLogic::COMMAND_MODEL));
}

void ModelClient::requestRun(const QString &appId, const DrawerAction &action)
{
    DrawerLogic::AppsctlCommand command =
        action.isBackground ? DrawerLogic::COMMAND_RUN_BACKGROUND : DrawerLogic::COMMAND_RUN_SHOW;
    startCall(REQUEST_RUN, drawerArguments("run") << appId << action.id, DrawerLogic::timeoutMillisecondsFor(command));
}

void ModelClient::requestInfo(const QString &appId)
{
    startCall(REQUEST_INFO, drawerArguments("info") << appId, DrawerLogic::timeoutMillisecondsFor(DrawerLogic::COMMAND_INFO));
}

void ModelClient::requestRemove(const QString &appId)
{
    startCall(REQUEST_REMOVE, drawerArguments("remove") << appId << QStringLiteral("--confirm"),
              DrawerLogic::timeoutMillisecondsFor(DrawerLogic::COMMAND_REMOVE));
}

void ModelClient::requestScreen(const QString &appId, const QString &screenId, const QString &value)
{
    QStringList arguments = drawerArguments("screen") << appId;
    if (!screenId.isEmpty()) {
        arguments << QStringLiteral("--screen") << screenId;
    }
    if (!value.isEmpty()) {
        arguments << QStringLiteral("--value") << value;
    }
    startCall(REQUEST_SCREEN, arguments, DrawerLogic::timeoutMillisecondsFor(DrawerLogic::COMMAND_SCREEN));
}

void ModelClient::requestEvent(const QString &appId, const ScreenEvent &event)
{
    QStringList arguments = drawerArguments("event") << appId << event.actionId;
    if (event.hasValue) {
        arguments << QStringLiteral("--value") << event.value;
    }
    if (event.listIndex >= 0) {
        arguments << QStringLiteral("--list") << event.listBind << QStringLiteral("--index")
                  << QString::number(event.listIndex);
    }
    if (!event.screenId.isEmpty()) {
        arguments << QStringLiteral("--screen") << event.screenId;
    }
    startCall(REQUEST_EVENT, arguments, DrawerLogic::timeoutMillisecondsFor(DrawerLogic::COMMAND_RUN_SHOW));
}

ModelClient::~ModelClient()
{
    int killed = 0;
    int detached = 0;
    QList<ToolCall *> calls = findChildren<ToolCall *>(QString(), Qt::FindDirectChildrenOnly);
    for (int i = 0; i < calls.size(); i++) {
        if (!calls[i]->isRunning()) {
            continue;
        }
        if (isReadOnly(static_cast<Request>(calls[i]->tag()))) {
            calls[i]->abort();
            killed++;
        } else {
            calls[i]->detach();
            detached++;
        }
    }
    DebugLog::write("model client: closed, %d pending read(s) killed, %d action(s) left to finish", killed, detached);
}

bool ModelClient::isReadOnly(Request request)
{
    switch (request) {
    case REQUEST_MODEL:
    case REQUEST_INFO:
    case REQUEST_SCREEN:
        return true;
    case REQUEST_RUN:
    case REQUEST_REMOVE:
    case REQUEST_EVENT:
        break;
    }
    return false;
}

void ModelClient::startCall(Request request, const QStringList &arguments, int timeoutMilliseconds)
{
    ToolCall *call = new ToolCall(QString::fromLatin1(APPSCTL_PATH), request, arguments, timeoutMilliseconds, this);
    connect(call, SIGNAL(finished(ToolCall *)), this, SLOT(onCallFinished(ToolCall *)));
    call->start();
}

void ModelClient::onCallFinished(ToolCall *call)
{
    switch (static_cast<Request>(call->tag())) {
    case REQUEST_MODEL:
        deliverModel(call);
        break;
    case REQUEST_RUN:
        deliverActionResult(call, false);
        break;
    case REQUEST_INFO:
        deliverInfo(call);
        break;
    case REQUEST_REMOVE:
        deliverActionResult(call, true);
        break;
    case REQUEST_SCREEN:
        deliverScreen(call);
        break;
    case REQUEST_EVENT:
        deliverActionResult(call, false);
        break;
    }
    call->deleteLater();
}

bool ModelClient::replyOf(ToolCall *call, QByteArray *json, QString *failure)
{
    if (call->outcome() != ToolCall::OUTCOME_EXITED) {
        *failure = outcomeFailure(call->outcome());
        return false;
    }
    switch (DrawerLogic::classifyExitCode(call->exitCode())) {
    case DrawerLogic::EXIT_REPLY:
        *json = call->output();
        return true;
    case DrawerLogic::EXIT_PROTOCOL_ERROR:
        *failure = ModelParser::errorMessage(call->output());
        if (!failure->isEmpty()) {
            return false;
        }
        break;
    case DrawerLogic::EXIT_UNEXPECTED:
        break;
    }
    nh_log("appsctl exited %d", call->exitCode()); // the code is for the log, not for the user
    *failure = FallbackStrings::text(FallbackStrings::APPSCTL_FAILED);
    return false;
}

void ModelClient::deliverModel(ToolCall *call)
{
    QByteArray json;
    QString failure;
    DrawerModel model;
    if (!replyOf(call, &json, &failure) || !ModelParser::parseModel(json, &model, &failure)) {
        model = ModelParser::errorModel(failure);
    }
    emit modelLoaded(model);
}

void ModelClient::deliverActionResult(ToolCall *call, bool isRemove)
{
    QByteArray json;
    QString failure;
    ActionResult result;
    if (!replyOf(call, &json, &failure) || !ModelParser::parseActionResult(json, &result, &failure)) {
        result = failedResult(failure);
    }
    if (isRemove) {
        emit removeFinished(result);
    } else {
        emit actionFinished(result);
    }
}

void ModelClient::deliverInfo(ToolCall *call)
{
    QByteArray json;
    QString failure;
    AppInfo info;
    bool isLoaded = replyOf(call, &json, &failure) && ModelParser::parseAppInfo(json, &info, &failure);
    emit infoLoaded(isLoaded, info, failure);
}

void ModelClient::deliverScreen(ToolCall *call)
{
    QByteArray json;
    QString failure;
    ScreenModel screen;
    bool isLoaded = replyOf(call, &json, &failure) && ModelParser::parseScreen(json, &screen, &failure);
    emit screenLoaded(isLoaded, screen, failure);
}

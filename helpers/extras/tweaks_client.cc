#include "tweaks_client.h"

#include <QStringList>

#include <NickelHook.h>

#include "debug_log.h"
#include "helper_flags.h"
#include "helper_strings.h"
#include "tool_call.h"
#include "tweaks_parser.h"

// A tweak is a file edit or a flag file: it answers at once; 3 s is generous.
static const int CALL_TIMEOUT_MILLISECONDS = 3000;
// extras-tweaks: 0 = reply, 2 = {"v":1,"ok":false,"error":"…"} (helpers/extras/docs/tweaks.md).
static const int EXIT_REPLY = 0;
static const int EXIT_FAILED_TWEAK = 2;

// The text for a call that did not even exit.
static QString outcomeFailure(ToolCall::Outcome outcome)
{
    switch (outcome) {
    case ToolCall::OUTCOME_NOT_STARTED:
        return HelperStrings::text(HelperStrings::HELPER_MISSING);
    case ToolCall::OUTCOME_TIMED_OUT:
        return HelperStrings::text(HelperStrings::HELPER_TIMEOUT);
    case ToolCall::OUTCOME_CRASHED:
    case ToolCall::OUTCOME_EXITED:
        break;
    }
    return HelperStrings::text(HelperStrings::HELPER_FAILED);
}

TweaksClient::TweaksClient(QObject *parent) : QObject(parent) {}

TweaksClient::~TweaksClient()
{
    QList<ToolCall *> calls = findChildren<ToolCall *>(QString(), Qt::FindDirectChildrenOnly);
    for (int i = 0; i < calls.size(); i++) {
        if (!calls[i]->isRunning()) {
            continue;
        }
        if (calls[i]->tag() == REQUEST_LIST) {
            calls[i]->abort();
        } else {
            calls[i]->detach();
        }
    }
}

void TweaksClient::requestList()
{
    startCall(REQUEST_LIST, QStringList() << QStringLiteral("list"));
}

void TweaksClient::requestApply(const QString &tweakId)
{
    startCall(REQUEST_APPLY, QStringList() << QStringLiteral("apply") << tweakId);
}

void TweaksClient::startCall(Request request, const QStringList &arguments)
{
    ToolCall *call = new ToolCall(QString::fromLatin1(HelperFlags::TWEAKS_BINARY_PATH), request, arguments,
                                  CALL_TIMEOUT_MILLISECONDS, this);
    connect(call, SIGNAL(finished(ToolCall *)), this, SLOT(onCallFinished(ToolCall *)));
    call->start();
}

void TweaksClient::onCallFinished(ToolCall *call)
{
    if (call->tag() == REQUEST_LIST) {
        deliverList(call);
    } else {
        deliverApply(call);
    }
    call->deleteLater();
}

void TweaksClient::deliverList(ToolCall *call)
{
    TweaksModel model;
    QString failure;
    bool isExited = call->outcome() == ToolCall::OUTCOME_EXITED;
    if (!isExited) {
        failure = outcomeFailure(call->outcome());
    } else if (call->exitCode() != EXIT_REPLY) {
        nh_log("extras-tweaks list exited %d", call->exitCode()); // for the log, not for the user
        failure = HelperStrings::text(HelperStrings::HELPER_FAILED);
    }
    if (!failure.isEmpty() || !TweaksParser::parseTweaks(call->output(), &model, &failure)) {
        model.isUsable = false;
        model.failure = failure;
    }
    emit listLoaded(model);
}

// A failed tweak exits 2 with a sentence for the user in "error"; any other exit is the helper
// itself failing.
void TweaksClient::deliverApply(ToolCall *call)
{
    TweakResult result;
    QString failure;
    bool isExited = call->outcome() == ToolCall::OUTCOME_EXITED;
    bool isReply = isExited && (call->exitCode() == EXIT_REPLY || call->exitCode() == EXIT_FAILED_TWEAK);
    if (!isExited) {
        failure = outcomeFailure(call->outcome());
    } else if (!isReply) {
        nh_log("extras-tweaks apply exited %d", call->exitCode());
        failure = HelperStrings::text(HelperStrings::HELPER_FAILED);
    }
    if (!isReply || !TweaksParser::parseTweakResult(call->output(), &result, &failure)) {
        result.isOk = false;
        result.error = failure;
    }
    DebugLog::write("tweaks client: apply %s", result.isOk ? "ok" : "failed");
    emit applied(result);
}

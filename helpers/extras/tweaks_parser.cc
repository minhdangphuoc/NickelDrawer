#include "tweaks_parser.h"

#include <NickelHook.h>

#include "helper_strings.h"
#include "tweaks_logic.h"

// TweaksLogic::List is ~40 KB: built on the heap, never on the GUI stack.
struct ListHolder {
    TweaksLogic::List list;
};

static QString failureFor(TweaksLogic::ParseStatus status)
{
    nh_log("tweaks parser: unusable reply (status %d)", static_cast<int>(status));
    bool isTooNew = status == TweaksLogic::PARSE_VERSION_TOO_NEW;
    return HelperStrings::text(isTooNew ? HelperStrings::UPDATE_NEEDED : HelperStrings::HELPER_BAD_REPLY);
}

static TweakItem tweakItemOf(const TweaksLogic::Item &parsed)
{
    TweakItem item;
    item.id = QString::fromUtf8(parsed.id);
    item.label = QString::fromUtf8(parsed.label);
    item.caption = QString::fromUtf8(parsed.caption);
    item.kind = parsed.kind;
    item.isOn = parsed.isOn;
    item.infoText = QString::fromUtf8(parsed.text);
    item.confirm = QString::fromUtf8(parsed.confirm);
    return item;
}

namespace TweaksParser {

bool parseTweaks(const QByteArray &json, TweaksModel *model, QString *failure)
{
    ListHolder *holder = new ListHolder;
    TweaksLogic::ParseStatus status = TweaksLogic::parseList(json.constData(), json.size(), &holder->list);
    if (status == TweaksLogic::PARSE_OK) {
        model->items.clear();
        for (int i = 0; i < holder->list.count; i++) {
            model->items.append(tweakItemOf(holder->list.items[i]));
        }
        model->isUsable = true;
        if (holder->list.skippedCount) {
            nh_log("tweaks parser: %d item(s) skipped (unknown kind, no id, or too many)", holder->list.skippedCount);
        }
    } else {
        *failure = failureFor(status);
    }
    delete holder;
    return status == TweaksLogic::PARSE_OK;
}

bool parseTweakResult(const QByteArray &json, TweakResult *result, QString *failure)
{
    TweaksLogic::Reply reply;
    TweaksLogic::ParseStatus status = TweaksLogic::parseReply(json.constData(), json.size(), &reply);
    bool isUsageError = status == TweaksLogic::PARSE_VERSION_MISSING && reply.error[0];
    if (status != TweaksLogic::PARSE_OK && !isUsageError) {
        *failure = failureFor(status);
        return false;
    }
    result->isOk = reply.isOk && !isUsageError;
    result->toast = QString::fromUtf8(reply.toast);
    result->error = QString::fromUtf8(reply.error);
    result->hasValue = reply.hasValue;
    result->value = reply.value;
    result->nickel = reply.nickel;
    if (!result->isOk && result->error.isEmpty()) {
        result->error = HelperStrings::text(HelperStrings::ACTION_FAILED);
    }
    return true;
}

} // namespace TweaksParser

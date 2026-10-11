#ifndef EXTRAS_TWEAKS_CLIENT_H
#define EXTRAS_TWEAKS_CLIENT_H

#include <QObject>
#include <QString>

#include "tweaks_model.h"

class ToolCall;

// Talks to the extras-tweaks binary (`list`, `apply <id>`; helpers/extras/docs/tweaks.md)
// asynchronously and turns every outcome into a parsed reply. Failures (missing binary, timeout,
// crash, bad JSON) become localized text; nothing blocks the GUI thread.
class TweaksClient : public QObject {
    Q_OBJECT
public:
    explicit TweaksClient(QObject *parent);
    // Ends with its session: a pending read is killed, a pending apply finishes in the background
    // (ToolCall::detach), because killing it could leave a setting half written.
    ~TweaksClient();

    void requestList();
    void requestApply(const QString &tweakId);

signals:
    // Always emitted; on failure isUsable is false and failure says why.
    void listLoaded(const TweaksModel &model);
    // Always emitted; on failure isOk is false and error says why.
    void applied(const TweakResult &result);

private slots:
    void onCallFinished(ToolCall *call);

private:
    enum Request { REQUEST_LIST, REQUEST_APPLY };

    void startCall(Request request, const QStringList &arguments);
    void deliverList(ToolCall *call);
    void deliverApply(ToolCall *call);
};

#endif

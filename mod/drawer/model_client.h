#ifndef NICKELDRAWER_MODEL_CLIENT_H
#define NICKELDRAWER_MODEL_CLIENT_H

#include <QObject>
#include <QString>

#include "drawer_model.h"
#include "screen_model.h"

class ToolCall;

// Talks to appsctl (`drawer model|run|info|remove|screen|event`) asynchronously and turns
// every outcome into a parsed reply. Failures (missing binary, timeout, crash,
// bad JSON, exit 2) become localized fallback text; nothing blocks.
class ModelClient : public QObject {
    Q_OBJECT
public:
    explicit ModelClient(QObject *parent);
    // Ends with its session (tab tap, Home, back): pending reads are killed,
    // pending actions finish in the background (ToolCall::detach).
    ~ModelClient();

    void requestModel();
    void requestRun(const QString &appId, const DrawerAction &action);
    void requestInfo(const QString &appId);
    // Only after the user confirmed; passes --confirm.
    void requestRemove(const QString &appId);
    // One app screen; an empty screenId asks for the start screen. A value
    // (a picture id) is passed as --value; empty: none.
    void requestScreen(const QString &appId, const QString &screenId, const QString &value = QString());
    // A tap on an app screen; the reply arrives as actionFinished().
    void requestEvent(const QString &appId, const ScreenEvent &event);

signals:
    // Always emitted; on failure the model holds one problem row.
    void modelLoaded(const DrawerModel &model);
    // On failure: isOk false and text = the reason. A failed script is a
    // normal ok:false reply with the script's own text.
    void actionFinished(const ActionResult &result);
    void infoLoaded(bool isLoaded, const AppInfo &info, const QString &failure);
    void removeFinished(const ActionResult &result);
    // On failure: isLoaded false and failure = the reason (already localized).
    void screenLoaded(bool isLoaded, const ScreenModel &screen, const QString &failure);

private slots:
    void onCallFinished(ToolCall *call);

private:
    enum Request { REQUEST_MODEL, REQUEST_RUN, REQUEST_INFO, REQUEST_REMOVE, REQUEST_SCREEN, REQUEST_EVENT };

    static bool isReadOnly(Request request);
    void startCall(Request request, const QStringList &arguments, int timeoutMilliseconds);
    // The JSON reply of a call that exited 0, or false and the failure text.
    bool replyOf(ToolCall *call, QByteArray *json, QString *failure);
    void deliverModel(ToolCall *call);
    void deliverActionResult(ToolCall *call, bool isRemove);
    void deliverInfo(ToolCall *call);
    void deliverScreen(ToolCall *call);
};

#endif

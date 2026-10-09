#ifndef NICKELDRAWER_APPSCTL_CALL_H
#define NICKELDRAWER_APPSCTL_CALL_H

#include <QByteArray>
#include <QObject>
#include <QProcess>
#include <QStringList>

class QTimer;

// One asynchronous `appsctl` run with a hard timeout. Never waits on the GUI
// thread: completion arrives through finished(). Emits finished() exactly
// once, then the owner deletes it (deleteLater).
class AppsctlCall : public QObject {
    Q_OBJECT
public:
    enum Outcome { OUTCOME_EXITED, OUTCOME_NOT_STARTED, OUTCOME_TIMED_OUT, OUTCOME_CRASHED };

    AppsctlCall(int tag, const QStringList &arguments, int timeoutMilliseconds, QObject *parent);
    void start();

    int tag() const { return callTag; }
    Outcome outcome() const { return callOutcome; }
    int exitCode() const { return processExitCode; }
    QByteArray output() const { return standardOutput; }

signals:
    void finished(AppsctlCall *call);

private slots:
    void onProcessFinished(int exitCode, QProcess::ExitStatus status);
    void onProcessError(QProcess::ProcessError error);
    void onTimeout();

private:
    void complete(Outcome outcome);

    int callTag;
    QStringList callArguments;
    QProcess *process;
    QTimer *timer;
    bool isComplete;
    Outcome callOutcome;
    int processExitCode;
    QByteArray standardOutput;
};

#endif

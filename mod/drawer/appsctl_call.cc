#include "appsctl_call.h"

#include <QTimer>

#include <NickelHook.h>

static const char *const APPSCTL_PATH = "/mnt/onboard/.adds/koboapps/bin/appsctl";
// Run from "/" so no child keeps a working directory on /mnt/onboard (it
// would block USB export).
static const char *const WORKING_DIRECTORY = "/";

AppsctlCall::AppsctlCall(int tag, const QStringList &arguments, int timeoutMilliseconds, QObject *parent)
    : QObject(parent), callTag(tag), callArguments(arguments), process(new QProcess(this)),
      timer(new QTimer(this)), isComplete(false), callOutcome(OUTCOME_NOT_STARTED), processExitCode(-1)
{
    process->setWorkingDirectory(QString::fromLatin1(WORKING_DIRECTORY));
    process->setProcessChannelMode(QProcess::SeparateChannels);
    timer->setSingleShot(true);
    timer->setInterval(timeoutMilliseconds);
    connect(process, SIGNAL(finished(int, QProcess::ExitStatus)), this,
            SLOT(onProcessFinished(int, QProcess::ExitStatus)));
    connect(process, SIGNAL(error(QProcess::ProcessError)), this, SLOT(onProcessError(QProcess::ProcessError)));
    connect(timer, SIGNAL(timeout()), this, SLOT(onTimeout()));
}

void AppsctlCall::start()
{
    nh_log("appsctl %s", qPrintable(callArguments.join(QLatin1Char(' '))));
    timer->start();
    process->start(QString::fromLatin1(APPSCTL_PATH), callArguments, QIODevice::ReadOnly);
}

void AppsctlCall::onProcessFinished(int exitCode, QProcess::ExitStatus status)
{
    processExitCode = exitCode;
    standardOutput = process->readAllStandardOutput();
    QByteArray errors = process->readAllStandardError();
    if (!errors.isEmpty()) {
        nh_log("appsctl stderr: %s", errors.left(200).constData());
    }
    complete(status == QProcess::NormalExit ? OUTCOME_EXITED : OUTCOME_CRASHED);
}

void AppsctlCall::onProcessError(QProcess::ProcessError error)
{
    // FailedToStart never emits finished(); the others are followed by it.
    if (error == QProcess::FailedToStart) {
        nh_log("appsctl failed to start: %s", qPrintable(process->errorString()));
        complete(OUTCOME_NOT_STARTED);
    }
}

void AppsctlCall::onTimeout()
{
    nh_log("appsctl timed out after %d ms, killing it", timer->interval());
    complete(OUTCOME_TIMED_OUT);
    process->kill();
}

void AppsctlCall::complete(Outcome outcome)
{
    if (isComplete) {
        return;
    }
    isComplete = true;
    timer->stop();
    callOutcome = outcome;
    emit finished(this);
}

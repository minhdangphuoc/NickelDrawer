#include "tool_call.h"

#include <QFileInfo>
#include <QTimer>

#include <NickelHook.h>

// Run from "/" so no child keeps a working directory on /mnt/onboard (it
// would block USB export).
static const char *const WORKING_DIRECTORY = "/";

// A killed child exits at once; this only bounds the wait if it does not.
static const int KILL_WAIT_MILLISECONDS = 500;

ToolCall::ToolCall(const QString &program, int tag, const QStringList &arguments, int timeoutMilliseconds,
                   QObject *parent)
    : QObject(parent), programPath(program), programName(QFileInfo(program).fileName()), callTag(tag), callArguments(arguments), process(new QProcess(this)),
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

void ToolCall::start()
{
    nh_log("%s %s", qPrintable(programName), qPrintable(callArguments.join(QLatin1Char(' '))));
    timer->start();
    process->start(programPath, callArguments, QIODevice::ReadOnly);
}

// Never block on QProcess's own destructor wait: a still-running child is
// killed (SIGKILL) and reaped within a short bound.
ToolCall::~ToolCall()
{
    abort();
}

bool ToolCall::isRunning() const
{
    return process->state() != QProcess::NotRunning;
}

void ToolCall::abort()
{
    if (!isRunning()) {
        return;
    }
    isComplete = true; // no finished() from here on
    timer->stop();
    process->disconnect(this);
    process->kill();
    bool isReaped = process->waitForFinished(KILL_WAIT_MILLISECONDS);
    nh_log("%s %s killed (owner closed)%s", qPrintable(programName), qPrintable(callArguments.value(0)),
           isReaped ? "" : ", not reaped yet");
}

void ToolCall::detach()
{
    disconnect(this, SIGNAL(finished(ToolCall *)), 0, 0);
    setParent(NULL);
    connect(this, SIGNAL(finished(ToolCall *)), this, SLOT(deleteLater()));
    nh_log("%s %s left to finish in the background (owner closed)", qPrintable(programName),
           qPrintable(callArguments.value(0)));
}

void ToolCall::onProcessFinished(int exitCode, QProcess::ExitStatus status)
{
    processExitCode = exitCode;
    standardOutput = process->readAllStandardOutput();
    QByteArray errors = process->readAllStandardError();
    if (!errors.isEmpty()) {
        nh_log("%s stderr: %s", qPrintable(programName), errors.left(200).constData());
    }
    complete(status == QProcess::NormalExit ? OUTCOME_EXITED : OUTCOME_CRASHED);
}

void ToolCall::onProcessError(QProcess::ProcessError error)
{
    // FailedToStart never emits finished(); the others are followed by it.
    if (error == QProcess::FailedToStart) {
        nh_log("%s failed to start: %s", qPrintable(programName), qPrintable(process->errorString()));
        complete(OUTCOME_NOT_STARTED);
    }
}

void ToolCall::onTimeout()
{
    nh_log("%s timed out after %d ms, killing it", qPrintable(programName), timer->interval());
    complete(OUTCOME_TIMED_OUT);
    process->kill();
}

void ToolCall::complete(Outcome outcome)
{
    if (isComplete) {
        return;
    }
    isComplete = true;
    timer->stop();
    callOutcome = outcome;
    emit finished(this);
}

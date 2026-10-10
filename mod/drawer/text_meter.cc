#include "text_meter.h"

#include <QFontMetrics>
#include <QList>
#include <QString>
#include <QVector>

namespace {

struct CachedMetrics {
    ScreenLogic::TextRun run;
    QFontMetrics metrics;
};

// A screen uses a handful of faces; building QFontMetrics for each measurement would be slow
// on the device's CPU.
QList<CachedMetrics> &cache()
{
    static QList<CachedMetrics> faces;
    return faces;
}

bool isSameRun(const ScreenLogic::TextRun &a, const ScreenLogic::TextRun &b)
{
    return a.size == b.size && a.family == b.family && a.isBold == b.isBold;
}

const QFontMetrics &metricsFor(const ScreenLogic::TextRun &run)
{
    QList<CachedMetrics> &faces = cache();
    for (int i = 0; i < faces.size(); i++) {
        if (isSameRun(faces[i].run, run)) {
            return faces[i].metrics;
        }
    }
    CachedMetrics added = {run, QFontMetrics(QtTextMeter::fontFor(run))};
    faces.append(added);
    return faces.last().metrics;
}

int measureWithQt(const void *, const char *utf8, int byteCount, const ScreenLogic::TextRun &run)
{
    return metricsFor(run).width(QString::fromUtf8(utf8, byteCount));
}

} // namespace

namespace QtTextMeter {

NickelFonts::Role roleFor(ScreenLogic::Family family)
{
    switch (family) {
    case ScreenLogic::FAMILY_SERIF:
        return NickelFonts::ROLE_TITLE;
    case ScreenLogic::FAMILY_MONO:
        return NickelFonts::ROLE_MONO;
    case ScreenLogic::FAMILY_SANS:
        break;
    }
    return NickelFonts::ROLE_BODY;
}

QFont fontFor(const ScreenLogic::TextRun &run)
{
    return NickelFonts::font(roleFor(run.family), run.size, run.isBold);
}

const ScreenLogic::TextMeter &meter()
{
    static const ScreenLogic::TextMeter qtMeter = {measureWithQt, 0};
    return qtMeter;
}

QStringList wrapLines(const QString &text, int width, const ScreenLogic::TextRun &run, int maxLines)
{
    QByteArray utf8 = text.toUtf8();
    QVector<char> out(utf8.size() * 2 + 16);
    int count = ScreenLogic::wrapText(utf8.constData(), width, run, meter(), maxLines, out.data(), out.size());
    return count == 0 ? QStringList() : QString::fromUtf8(out.constData()).split(QLatin1Char('\n'));
}

QString elideLine(const QString &text, int width, const ScreenLogic::TextRun &run, bool isMiddle)
{
    QByteArray utf8 = text.toUtf8();
    QVector<char> out(utf8.size() + 16);
    ScreenLogic::elideText(utf8.constData(), width, run, meter(), isMiddle, out.data(), out.size());
    return QString::fromUtf8(out.constData());
}

} // namespace QtTextMeter

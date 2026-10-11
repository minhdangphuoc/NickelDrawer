#include "nickel_fonts.h"

#include <QAbstractButton>
#include <QApplication>
#include <QFontDatabase>
#include <QFontMetrics>
#include <QHash>
#include <QLabel>
#include <QStringList>
#include <QWidget>

#include "debug_log.h"

// Nickel's UI families (font files in /usr/local/Trolltech/QtEmbedded-4.6.2-arm/lib/fonts
// on 4.38; kobo-tieng-viet replaces the Georgia, Avenir Next, Rakuten Serif
// and Rakuten Sans files with Vietnamese ones). Live serif fonts are told
// from sans ones by these names.
static const char *const SERIF_NAMES[] = {"Georgia", "Rakuten Serif", "Amasis", 0};
static const char *const TITLE_FALLBACKS[] = {"Georgia", "Rakuten Serif", 0};
static const char *const BODY_FALLBACKS[] = {"Avenir Next", "Rakuten Sans", "Kobo Nickel", 0};
static const char *const MONO_FALLBACKS[] = {"Courier Mono", "Courier New", "Courier", "DejaVu Sans Mono", 0};
static const char *const GENERIC_NAMES[] = {"serif", "sans-serif", "sans serif", "sans", "monospace", 0};

static const ushort CAPITAL_U_HORN_ACUTE = 0x1EE8;  // 'Ứ'
static const ushort SMALL_U_DOT_BELOW = 0x1EE5;     // 'ụ'
static const ushort RIGHTWARDS_ARROW = 0x2192;      // '→' of "1.0.1 → 1.0.2"; without it the drawer shows "->"
static const int SELF_CHECK_PIXEL_SIZE = 32;

static const int ROLE_COUNT = 3;
static QString chosenFamilies[ROLE_COUNT];
static bool isResolved = false;

static bool isListed(const QString &name, const char *const *names)
{
    for (const char *const *candidate = names; *candidate; candidate++) {
        if (!name.compare(QString::fromLatin1(*candidate), Qt::CaseInsensitive)) {
            return true;
        }
    }
    return false;
}

// Our own labels and buttons are plain QLabel / QPushButton; Nickel's are subclasses (TouchLabel, ...).
static bool isNickelTextWidget(const QWidget *widget)
{
    if (!widget->isVisible()) {
        return false; // not polished yet: no QSS font
    }
    bool isText = qobject_cast<const QLabel *>(widget) || qobject_cast<const QAbstractButton *>(widget);
    const char *className = widget->metaObject()->className();
    return isText && qstrcmp(className, "QLabel") && qstrcmp(className, "QPushButton");
}

static QString mostFrequent(const QHash<QString, int> &tally, int *count)
{
    QString best;
    *count = 0;
    for (QHash<QString, int>::const_iterator it = tally.constBegin(); it != tally.constEnd(); ++it) {
        if (it.value() > *count) {
            best = it.key();
            *count = it.value();
        }
    }
    return best;
}

// The most used serif and sans families of Nickel's visible text widgets.
static void readLiveFamilies(const QStringList &installed, QString *serif, QString *sans)
{
    QHash<QString, int> serifTally;
    QHash<QString, int> sansTally;
    int scanned = 0;
    QList<QWidget *> widgets = QApplication::allWidgets();
    for (int i = 0; i < widgets.size(); i++) {
        if (!isNickelTextWidget(widgets[i])) {
            continue;
        }
        scanned++;
        QString name = widgets[i]->font().family();
        if (name.isEmpty() || isListed(name, GENERIC_NAMES) || !installed.contains(name, Qt::CaseInsensitive)) {
            continue;
        }
        QHash<QString, int> &tally = isListed(name, SERIF_NAMES) ? serifTally : sansTally;
        tally[name]++;
    }
    int serifCount = 0;
    int sansCount = 0;
    *serif = mostFrequent(serifTally, &serifCount);
    *sans = mostFrequent(sansTally, &sansCount);
    DebugLog::write("fonts: %d live Nickel text widgets; serif \"%s\" (%d), sans \"%s\" (%d)", scanned,
                    qPrintable(*serif), serifCount, qPrintable(*sans), sansCount);
}

static QString firstInstalled(const char *const *names, const QStringList &installed)
{
    for (const char *const *candidate = names; *candidate; candidate++) {
        QString name = QString::fromLatin1(*candidate);
        if (installed.contains(name, Qt::CaseInsensitive)) {
            return name;
        }
    }
    return QString();
}

// Live family, else an installed known one, else the first known name (never a generic family).
static QString choose(const QString &live, const char *const *fallbacks, const QStringList &installed,
                      const char **origin)
{
    if (!live.isEmpty()) {
        *origin = "live Nickel widgets";
        return live;
    }
    QString installedName = firstInstalled(fallbacks, installed);
    if (!installedName.isEmpty()) {
        *origin = "known Nickel family";
        return installedName;
    }
    *origin = "not installed, first known name";
    return QString::fromLatin1(fallbacks[0]);
}

static void selfCheck(const char *roleName, const QString &name, const char *origin)
{
    QFont probe(name);
    probe.setPixelSize(SELF_CHECK_PIXEL_SIZE);
    QFontMetrics metrics(probe);
    bool hasCapital = metrics.inFont(QChar(CAPITAL_U_HORN_ACUTE));
    bool hasSmall = metrics.inFont(QChar(SMALL_U_DOT_BELOW));
    bool hasArrow = metrics.inFont(QChar(RIGHTWARDS_ARROW));
    DebugLog::write("fonts: %s \"%s\" (%s), Vietnamese U+1EE8 %s, U+1EE5 %s, arrow U+2192 %s", roleName,
                    qPrintable(name), origin, hasCapital ? "ok" : "missing", hasSmall ? "ok" : "missing",
                    hasArrow ? "ok" : "missing");
    if (!hasCapital || !hasSmall) {
        DebugLog::write("fonts: WARNING %s font \"%s\" lacks Vietnamese glyphs; install kobo-tieng-viet or check "
                        "the font files",
                        roleName, qPrintable(name));
    }
}

static void resolve()
{
    if (isResolved) {
        return;
    }
    isResolved = true;
    QStringList installed = QFontDatabase().families(); // slow on the device: once
    QString liveSerif;
    QString liveSans;
    readLiveFamilies(installed, &liveSerif, &liveSans);

    const char *titleOrigin = "";
    const char *bodyOrigin = "";
    const char *monoOrigin = "";
    chosenFamilies[NickelFonts::ROLE_TITLE] = choose(liveSerif, TITLE_FALLBACKS, installed, &titleOrigin);
    chosenFamilies[NickelFonts::ROLE_BODY] = choose(liveSans, BODY_FALLBACKS, installed, &bodyOrigin);
    chosenFamilies[NickelFonts::ROLE_MONO] = firstInstalled(MONO_FALLBACKS, installed);
    monoOrigin = "installed mono family";
    if (chosenFamilies[NickelFonts::ROLE_MONO].isEmpty()) {
        chosenFamilies[NickelFonts::ROLE_MONO] = chosenFamilies[NickelFonts::ROLE_BODY];
        monoOrigin = "no mono font, body family";
    }
    selfCheck("title", chosenFamilies[NickelFonts::ROLE_TITLE], titleOrigin);
    selfCheck("body", chosenFamilies[NickelFonts::ROLE_BODY], bodyOrigin);
    selfCheck("mono", chosenFamilies[NickelFonts::ROLE_MONO], monoOrigin);
}

static QString quoted(const QString &name)
{
    return QLatin1Char('"') + name + QLatin1Char('"');
}

namespace NickelFonts {

QString family(Role role)
{
    resolve();
    return chosenFamilies[role];
}

QFont font(Role role, int pixelSize, bool isBold)
{
    QFont result(family(role));
    result.setPixelSize(pixelSize);
    result.setBold(isBold);
    return result;
}

QString styleSheet(const char *templateText)
{
    QString result = QString::fromUtf8(templateText);
    result.replace(QStringLiteral("@TITLE_FAMILY@"), quoted(family(ROLE_TITLE)));
    result.replace(QStringLiteral("@BODY_FAMILY@"), quoted(family(ROLE_BODY)));
    result.replace(QStringLiteral("@MONO_FAMILY@"), quoted(family(ROLE_MONO)));
    return result;
}

} // namespace NickelFonts

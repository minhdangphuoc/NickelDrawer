#include "screen_fonts.h"

#include <QFontDatabase>
#include <QStringList>

#include "debug_log.h"

static const char *const SANS_FAMILIES[] = {"Avenir Next", "Avenir", "Gill Sans", "Kobo Nickel", "DejaVu Sans", 0};
static const char *const SERIF_FAMILIES[] = {"Georgia", "Amasis", "Caecilia", "Malabar", "DejaVu Serif", 0};
static const char *const MONO_FAMILIES[] = {"Courier New", "Courier", "DejaVu Sans Mono", "Liberation Mono", 0};

struct ChosenFamily {
    bool isChosen;
    QString name; // empty: Qt's default family with a style hint
};

static ChosenFamily chosenFamilies[3] = {{false, QString()}, {false, QString()}, {false, QString()}};

static const char *const *preferencesFor(ScreenLogic::Family family)
{
    switch (family) {
    case ScreenLogic::FAMILY_SERIF:
        return SERIF_FAMILIES;
    case ScreenLogic::FAMILY_MONO:
        return MONO_FAMILIES;
    case ScreenLogic::FAMILY_SANS:
        break;
    }
    return SANS_FAMILIES;
}

static QFont::StyleHint styleHintFor(ScreenLogic::Family family)
{
    switch (family) {
    case ScreenLogic::FAMILY_SERIF:
        return QFont::Serif;
    case ScreenLogic::FAMILY_MONO:
        return QFont::TypeWriter;
    case ScreenLogic::FAMILY_SANS:
        break;
    }
    return QFont::SansSerif;
}

// Looked up once per family: QFontDatabase is slow on the device.
static const QString &familyName(ScreenLogic::Family family)
{
    ChosenFamily &chosen = chosenFamilies[family];
    if (chosen.isChosen) {
        return chosen.name;
    }
    chosen.isChosen = true;
    QStringList installed = QFontDatabase().families();
    for (const char *const *candidate = preferencesFor(family); *candidate; candidate++) {
        if (installed.contains(QString::fromLatin1(*candidate), Qt::CaseInsensitive)) {
            chosen.name = QString::fromLatin1(*candidate);
            break;
        }
    }
    DebugLog::write("screen fonts: family %d uses %s", (int)family,
                    chosen.name.isEmpty() ? "Qt's default" : qPrintable(chosen.name));
    return chosen.name;
}

namespace ScreenFonts {

QFont font(ScreenLogic::Family family, int pixelSize, bool isBold, bool isUnderlined)
{
    QFont result;
    const QString &name = familyName(family);
    if (!name.isEmpty()) {
        result.setFamily(name);
    }
    result.setStyleHint(styleHintFor(family));
    result.setPixelSize(pixelSize);
    result.setBold(isBold);
    result.setUnderline(isUnderlined);
    return result;
}

} // namespace ScreenFonts

#include "system_icon.h"

#include <QImage>
#include <QPainter>
#include <QRect>
#include <QString>

#include "debug_log.h"

namespace SystemIcon {

const char *const ICON_DIRECTORY = "/mnt/onboard/.adds/koboapps/icons/ui";

namespace {

struct Spec {
    const char *name;        // for the log
    const char *nickelPath;  // NULL: Nickel has no fitting icon
    const char *nickelPressedPath;
    const char *tablerName;  // file "<tablerName>-<tablerSize>.png" in ICON_DIRECTORY
    int tablerSize;          // the installed size closest to Nickel's (56-70 px on the Libra 2 class)
};

// Indexed by Id. Where Nickel uses each image: re/notes/system-icons.md.
const Spec SPECS[ID_COUNT] = {
    {"settings", ":/images/reading/settings_gear.png", ":/images/reading/settings_gear_hit.png", "settings", 64},
    {"import", ":/images/library/library_download_lg_off.png", ":/images/library/library_download_lg_hit.png",
     "import", 64},
    {"refresh", ":/images/navigation/browser_reload.png", ":/images/navigation/browser_reload_hit.png", "refresh",
     64},
    {"add", ":/images/reading/plus.png", ":/images/reading/plus_hit.png", "add", 64},
    {"back", ":/images/reading/back_arrow.png", ":/images/reading/back_arrow_hit.png", "back", 64},
    {"close", ":/images/search/global_close.png", ":/images/search/global_close_hit.png", "close", 64},
    {"check badge", ":/images/library/checkmark_hit.png", NULL, "check", 64},
    {"view list", NULL, NULL, "list", 64},
    {"view grid", NULL, NULL, "grid", 64},
    {"picture", NULL, NULL, "picture", 96},
    {"app", NULL, NULL, "apps", 96},
};

const char *sourceName(Source source)
{
    switch (source) {
    case SOURCE_NICKEL:
        return "Nickel";
    case SOURCE_TABLER:
        return "Tabler";
    case SOURCE_NONE:
        break;
    }
    return "none";
}

bool loadNickel(const Spec &spec, Icon *loaded)
{
    if (!spec.nickelPath) {
        return false;
    }
    loaded->normal = QPixmap(QString::fromLatin1(spec.nickelPath));
    if (loaded->normal.isNull()) {
        DebugLog::write("icon %s: %s missing from Nickel's resources", spec.name, spec.nickelPath);
        return false;
    }
    if (spec.nickelPressedPath) {
        loaded->pressed = QPixmap(QString::fromLatin1(spec.nickelPressedPath));
        if (loaded->pressed.isNull()) {
            DebugLog::write("icon %s: %s missing, press inverts instead", spec.name, spec.nickelPressedPath);
        }
    }
    loaded->source = SOURCE_NICKEL;
    return true;
}

bool loadTabler(const Spec &spec, Icon *loaded)
{
    QString path = QString::fromLatin1("%1/%2-%3.png")
                       .arg(QString::fromLatin1(ICON_DIRECTORY), QString::fromLatin1(spec.tablerName))
                       .arg(spec.tablerSize);
    loaded->normal = QPixmap(path);
    if (loaded->normal.isNull()) {
        DebugLog::write("icon %s: %s missing or unreadable (reinstall koboapps)", spec.name, qPrintable(path));
        return false;
    }
    loaded->source = SOURCE_TABLER;
    return true;
}

Icon load(const Spec &spec)
{
    Icon loaded;
    loaded.source = SOURCE_NONE;
    if (!loadNickel(spec, &loaded)) {
        loadTabler(spec, &loaded);
    }
    DebugLog::write("icon %s: %s %dx%d%s", spec.name, sourceName(loaded.source), loaded.normal.width(),
                    loaded.normal.height(), loaded.pressed.isNull() ? "" : ", pressed image");
    return loaded;
}

} // namespace

const Icon &icon(Id id)
{
    static Icon *icons = NULL; // built on first use, after Nickel registered its resources
    static bool isLoaded[ID_COUNT];
    if (!icons) {
        icons = new Icon[ID_COUNT];
        for (int i = 0; i < ID_COUNT; i++) {
            icons[i].source = SOURCE_NONE;
            isLoaded[i] = false;
        }
    }
    if (!isLoaded[id]) {
        icons[id] = load(SPECS[id]);
        isLoaded[id] = true;
    }
    return icons[id];
}

void drawCentred(QPainter *painter, const QRect &area, const QPixmap &image)
{
    if (image.isNull()) {
        return;
    }
    QSize size = image.size();
    if (size.width() > area.width() || size.height() > area.height()) {
        size.scale(area.size(), Qt::KeepAspectRatio);
    }
    QRect target(QPoint(0, 0), size);
    target.moveCenter(area.center());
    if (size == image.size()) {
        painter->drawPixmap(target.topLeft(), image);
        return;
    }
    painter->drawPixmap(target.topLeft(), image.scaled(size, Qt::KeepAspectRatio, Qt::SmoothTransformation));
}

QPixmap whiteInk(const QPixmap &image)
{
    QImage inverted = image.toImage().convertToFormat(QImage::Format_ARGB32);
    inverted.invertPixels(QImage::InvertRgb);
    return QPixmap::fromImage(inverted);
}

} // namespace SystemIcon

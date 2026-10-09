#include "view_settings.h"

#include <QDir>
#include <QSettings>

#include <NickelHook.h>

static const char *const DATA_DIRECTORY = "/mnt/onboard/.adds/koboapps/data";
static const char *const SETTINGS_PATH = "/mnt/onboard/.adds/koboapps/data/drawer.ini";
static const char *const VIEW_KEY = "drawer/view";
static const char *const VIEW_LIST_VALUE = "list";
static const char *const VIEW_GRID_VALUE = "grid";

static const char *valueFor(DrawerLogic::ViewMode mode)
{
    return mode == DrawerLogic::VIEW_GRID ? VIEW_GRID_VALUE : VIEW_LIST_VALUE;
}

namespace ViewSettings {

DrawerLogic::ViewMode load()
{
    QSettings settings(QString::fromLatin1(SETTINGS_PATH), QSettings::IniFormat);
    QString value = settings.value(QString::fromLatin1(VIEW_KEY)).toString();
    return value == QLatin1String(VIEW_GRID_VALUE) ? DrawerLogic::VIEW_GRID : DrawerLogic::VIEW_LIST;
}

void save(DrawerLogic::ViewMode mode)
{
    if (load() == mode) {
        return;
    }
    if (!QDir().mkpath(QString::fromLatin1(DATA_DIRECTORY))) {
        nh_log("view settings: cannot create %s, view choice not saved", DATA_DIRECTORY);
        return;
    }
    QSettings settings(QString::fromLatin1(SETTINGS_PATH), QSettings::IniFormat);
    settings.setValue(QString::fromLatin1(VIEW_KEY), QString::fromLatin1(valueFor(mode)));
    settings.sync();
    if (settings.status() != QSettings::NoError) {
        nh_log("view settings: writing %s failed (status %d)", SETTINGS_PATH, static_cast<int>(settings.status()));
    }
}

} // namespace ViewSettings

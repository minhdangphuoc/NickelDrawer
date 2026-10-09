#ifndef NICKELDRAWER_VIEW_SETTINGS_H
#define NICKELDRAWER_VIEW_SETTINGS_H

#include "drawer_logic.h"

// The list/grid choice, kept in /mnt/onboard/.adds/koboapps/data/drawer.ini.
// Each call opens, reads or writes, and closes the file; nothing stays open.
namespace ViewSettings {

DrawerLogic::ViewMode load();
// Writes only when the mode differs from what is stored.
void save(DrawerLogic::ViewMode mode);

} // namespace ViewSettings

#endif

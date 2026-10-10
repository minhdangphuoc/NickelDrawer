#ifndef NICKELDRAWER_QT6_NOTICE_H
#define NICKELDRAWER_QT6_NOTICE_H

namespace Qt6Notice {

// True when Kobo offers firmware 5.x and the user has not declined it
// (Kobo eReader.conf, read only; rule in DrawerLogic::isQt6UpgradePending).
// Reads the file on every call: call when the drawer is built, not in init().
//
// Why the config and not IOUtil::isQt6UpgradePending: that libnickel symbol
// exists on 4.38.23828 only and needs a Device object; the two keys it
// compares are on every build and cost no libnickel call.
bool isUpgradePending();

} // namespace Qt6Notice

#endif

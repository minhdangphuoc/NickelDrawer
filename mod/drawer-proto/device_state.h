#ifndef NDDRAWER_DEVICE_STATE_H
#define NDDRAWER_DEVICE_STATE_H

#include <stddef.h>

#include <QString>

// Which prototypes run on this boot. Decided once in init() from the flag
// files and the firmware check; changing a flag needs a reboot.
struct PrototypeSwitches {
    bool isMoreRowEnabled;      // Prototype A
    bool isDrawerEnabled;       // Prototype C
    bool usesViewPushRoute;     // Prototype C opened with pushView/popView (route B)
};

extern PrototypeSwitches prototypeSwitches;

namespace DeviceState {

// Copies the firmware field of /mnt/onboard/.kobo/version into out
// ("unknown" on any failure). The file is closed before returning.
void readFirmware(char *out, size_t outSize);

// True for the five Libra 2 builds the symbols were checked against.
bool isSupportedFirmware(const char *firmware);

bool hasFlagFile(const char *path);

// "Ứng dụng" when Nickel's CurrentLocale is Vietnamese, else "Apps".
QString appsLabel();

} // namespace DeviceState

#endif

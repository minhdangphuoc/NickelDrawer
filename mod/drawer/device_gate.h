#ifndef NICKELDRAWER_DEVICE_GATE_H
#define NICKELDRAWER_DEVICE_GATE_H

// Which Kobo models the drawer may run on. The layout is fixed to the tested
// Libra 2 screen (screen_logic.h), so only models with the same 1264x1680
// panel pass. The model comes from the device id, the last field of
// .kobo/version (re/notes/device-detection.md); the live screen size is
// checked again once Nickel's QApplication exists. No Qt, no libnickel, no C++
// standard library (host-tested).

#include <stddef.h>

namespace DeviceGate {

// Kobo device ids end in three digits that name the model (FBInk, koboutils).
static const size_t DEVICE_ID_SUFFIX_LENGTH = 3;

struct SupportedModel {
    const char *idSuffix; // last three digits of the device id, e.g. "388"
    const char *codename; // koboutils codename, for logs only
};

// The model whose device id (full id or just its last three digits) is
// supported, or NULL when it is not.
const SupportedModel *findSupportedModel(const char *deviceId);

// True for the tested panel, in portrait or landscape.
bool isSupportedScreenSize(int width, int height);

// Copies the field at index (0-based, comma-separated) of a .kobo/version line
// into out, without the line break. False (out empty) when the field is
// missing, empty or does not fit.
bool copyVersionField(const char *line, int index, char *out, size_t outSize);

// Same for the last field (the device id).
bool copyLastVersionField(const char *line, char *out, size_t outSize);

} // namespace DeviceGate

#endif

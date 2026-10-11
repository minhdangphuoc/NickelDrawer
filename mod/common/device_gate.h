#ifndef NICKELDRAWER_DEVICE_GATE_H
#define NICKELDRAWER_DEVICE_GATE_H

// Which Kobo models the mods may run on (shared by the mods, each
// compiling its own copy). The layout is
// fixed to the tested Libra 2 screen (copperui tokens), so only models with the
// same 1264x1680 panel pass. The model comes from the device id, the last field of
// .kobo/version (re/notes/device-detection.md); the live screen size is
// checked again once Nickel's QApplication exists. No Qt, no libnickel, no C++
// standard library (host-tested).

#include <stddef.h>

// Index of the build ("4.38.23828") in a .kobo/version line; the device id is the last field
// (re/notes/device-detection.md).

namespace DeviceGate {

static const int FIRMWARE_FIELD_INDEX = 2;

// Kobo device ids end in three digits that name the model (FBInk, koboutils).
static const size_t DEVICE_ID_SUFFIX_LENGTH = 3;

struct SupportedModel {
    const char *idSuffix; // last three digits of the device id, e.g. "388"
    const char *codename; // koboutils codename, for logs only
};

// The model whose device id (full id or just its last three digits) is
// supported, or NULL when it is not.
const SupportedModel *findSupportedModel(const char *deviceId);

// True when firmware is one of the tested Libra 2 builds (mirror of "supported_firmware" in
// host/config/project.json; internal/project/project_test.go keeps them equal).
bool isSupportedFirmware(const char *firmware);

// True for the tested panel, in portrait or landscape.
bool isSupportedScreenSize(int width, int height);

// Copies the field at index (0-based, comma-separated) of a .kobo/version line
// into out, without the line break. False (out empty) when the field is
// missing, empty or does not fit.
bool copyVersionField(const char *line, int index, char *out, size_t outSize);

// Same for the last field (the device id).
bool copyLastVersionField(const char *line, char *out, size_t outSize);

// Reads the one-line .kobo/version file at path (closed at once, so nothing stays open on the user
// partition) into firmware (the build) and deviceId. False when the file cannot be read or is
// empty; the outputs are then untouched. A field that is missing leaves its output untouched.
bool readVersionFile(const char *path, char *firmware, size_t firmwareSize, char *deviceId, size_t deviceIdSize);

} // namespace DeviceGate

#endif

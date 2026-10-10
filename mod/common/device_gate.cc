#include "device_gate.h"

#include <stdio.h>
#include <string.h>

#include "copperui/tokens.h"

namespace DeviceGate {

// Mirror of "supported_models" in host/config/project.json
// (internal/project/project_test.go keeps them equal). Every model here must
// have the 1264x1680 panel the layout is built for.
static const SupportedModel SUPPORTED_MODELS[] = {
    {"384", "storm"}, // Libra H2O
    {"388", "io"},    // Libra 2 (tested)
    {"390", "monza"}, // Libra Colour
};

static const char *const SUPPORTED_FIRMWARE[] = {
    "4.38.23552", "4.38.23648", "4.38.23684", "4.38.23697", "4.38.23828",
};

static const char *const LINE_END_CHARACTERS = "\r\n";

static bool isDigit(char character)
{
    return character >= '0' && character <= '9';
}

const SupportedModel *findSupportedModel(const char *deviceId)
{
    if (!deviceId) {
        return NULL;
    }
    size_t length = strlen(deviceId);
    if (length < DEVICE_ID_SUFFIX_LENGTH) {
        return NULL;
    }
    const char *suffix = deviceId + length - DEVICE_ID_SUFFIX_LENGTH;
    for (size_t i = 0; i < DEVICE_ID_SUFFIX_LENGTH; i++) {
        if (!isDigit(suffix[i])) {
            return NULL;
        }
    }
    for (size_t i = 0; i < sizeof(SUPPORTED_MODELS) / sizeof(SUPPORTED_MODELS[0]); i++) {
        if (!strcmp(suffix, SUPPORTED_MODELS[i].idSuffix)) {
            return &SUPPORTED_MODELS[i];
        }
    }
    return NULL;
}

bool isSupportedFirmware(const char *firmware)
{
    if (!firmware) {
        return false;
    }
    for (size_t i = 0; i < sizeof(SUPPORTED_FIRMWARE) / sizeof(SUPPORTED_FIRMWARE[0]); i++) {
        if (!strcmp(firmware, SUPPORTED_FIRMWARE[i])) {
            return true;
        }
    }
    return false;
}

bool isSupportedScreenSize(int width, int height)
{
    bool isPortrait = width == copperui::v1::SCREEN_WIDTH && height == copperui::v1::SCREEN_HEIGHT;
    bool isLandscape = width == copperui::v1::SCREEN_HEIGHT && height == copperui::v1::SCREEN_WIDTH;
    return isPortrait || isLandscape;
}

static bool copySpan(const char *start, size_t length, char *out, size_t outSize)
{
    if (length == 0 || length >= outSize) {
        return false;
    }
    memcpy(out, start, length);
    out[length] = '\0';
    return true;
}

// Length of the field starting at start: up to a comma or the line end.
static size_t fieldLength(const char *start)
{
    return strcspn(start, ",\r\n");
}

bool copyVersionField(const char *line, int index, char *out, size_t outSize)
{
    if (!out || outSize == 0) {
        return false;
    }
    out[0] = '\0';
    if (!line || index < 0) {
        return false;
    }
    const char *cursor = line;
    for (int field = 0; field < index; field++) {
        cursor = strpbrk(cursor, ",\r\n");
        if (!cursor || *cursor != ',') {
            return false;
        }
        cursor++;
    }
    return copySpan(cursor, fieldLength(cursor), out, outSize);
}

bool copyLastVersionField(const char *line, char *out, size_t outSize)
{
    if (!out || outSize == 0) {
        return false;
    }
    out[0] = '\0';
    if (!line) {
        return false;
    }
    size_t lineLength = strcspn(line, LINE_END_CHARACTERS);
    const char *start = line;
    for (size_t i = 0; i < lineLength; i++) {
        if (line[i] == ',') {
            start = line + i + 1;
        }
    }
    return copySpan(start, line + lineLength - start, out, outSize);
}

bool readVersionFile(const char *path, char *firmware, size_t firmwareSize, char *deviceId, size_t deviceIdSize)
{
    char line[160];
    FILE *file = fopen(path, "r");
    if (!file) {
        return false;
    }
    char *got = fgets(line, sizeof(line), file);
    fclose(file); // never keep files open on /mnt/onboard
    if (!got) {
        return false;
    }
    char field[64];
    if (copyVersionField(line, FIRMWARE_FIELD_INDEX, field, sizeof(field)) && strlen(field) < firmwareSize) {
        strcpy(firmware, field);
    }
    if (copyLastVersionField(line, field, sizeof(field)) && strlen(field) < deviceIdSize) {
        strcpy(deviceId, field);
    }
    return true;
}

} // namespace DeviceGate

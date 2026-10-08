// ndhello: minimal NickelHook mod for the NickelDrawer toolchain check.
// init() only logs the firmware and schedules one toast; everything else is
// deferred so Nickel's startup is never blocked.

#include <stdio.h>
#include <string.h>

#include <NickelHook.h>

#include "toast.h"

extern "C" {
extern void *(*MainWindowController_sharedInstance)();
extern void (*MainWindowController_toast)(void *, QString const &, QString const &, int);
}

static const char *const VERSION_FILE_PATH = "/mnt/onboard/.kobo/version";
static const int TOAST_DELAY_MILLISECONDS = 8000;
static const int FIRMWARE_FIELD_INDEX = 2; // "serial,3.0.0,4.38.23828,..."

// Copies the firmware field of the Kobo version file into out; "unknown" on any failure.
static void readFirmware(char *out, size_t outSize)
{
    snprintf(out, outSize, "unknown");

    char line[128];
    FILE *file = fopen(VERSION_FILE_PATH, "r");
    if (!file) {
        return;
    }
    char *got = fgets(line, sizeof(line), file);
    fclose(file); // never keep files open on /mnt/onboard
    if (!got) {
        return;
    }

    char *cursor = line;
    for (int field = 0; field < FIRMWARE_FIELD_INDEX; field++) {
        cursor = strchr(cursor, ',');
        if (!cursor) {
            return;
        }
        cursor++;
    }
    cursor[strcspn(cursor, ",\r\n")] = '\0';
    if (*cursor) {
        snprintf(out, outSize, "%s", cursor);
    }
}

static int initialize()
{
    char firmware[32];
    readFirmware(firmware, sizeof(firmware));
    nh_log("ndhello loaded on %s", firmware);

    ToastOnce::scheduleAfterMilliseconds(TOAST_DELAY_MILLISECONDS);
    return 0;
}

static struct nh_info modInfo = {
    .name = "ndhello",
    .desc = "NickelDrawer hello-world test mod",
    .uninstall_flag = "/mnt/onboard/.adds/nickeldrawer/uninstall-hello",
    .uninstall_xflag = NULL, // g++ 4.9 rejects skipped designated fields; NULL: no delete-to-uninstall
    .failsafe_delay = 15,
};

static struct nh_dlsym modSymbols[] = {
    {.name = "_ZN20MainWindowController14sharedInstanceEv",
     .out = nh_symoutptr(MainWindowController_sharedInstance),
     .desc = "MainWindowController::sharedInstance",
     .optional = true},
    {.name = "_ZN20MainWindowController5toastERK7QStringS2_i",
     .out = nh_symoutptr(MainWindowController_toast),
     .desc = "MainWindowController::toast",
     .optional = true},
    {0},
};

NickelHook(
    .init = initialize,
    .info = &modInfo,
    .hook = NULL,
    .dlsym = modSymbols,
    .uninstall = NULL, )

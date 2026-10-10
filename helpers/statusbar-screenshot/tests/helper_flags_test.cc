// Host unit tests for helper_flags (no Qt, no C++ standard library).
// Run: make -C helpers/statusbar-screenshot docker-test

#include "../../../mod/common/test_expect.h"
#include "../helper_flags.h"

#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>

using namespace HelperFlags;

static char folder[] = "/tmp/helper_flags_test_XXXXXX";
static char offFlag[96];

static void touch(const char *path)
{
    int descriptor = open(path, O_CREAT | O_WRONLY, 0644);
    EXPECT_EQ(true, descriptor >= 0);
    close(descriptor);
}

static void testNoOffFlagInAVisibleFolderMeansOn()
{
    EXPECT_EQ(BUTTON_ON, readButtonState(offFlag, folder));
}

static void testOffFlagMeansOff()
{
    touch(offFlag);
    EXPECT_EQ(BUTTON_OFF, readButtonState(offFlag, folder));
    EXPECT_EQ(true, hasFlag(offFlag));
    unlink(offFlag);
    EXPECT_EQ(BUTTON_ON, readButtonState(offFlag, folder)); // removing the flag turns it back on
}

static void testAnInvisibleFolderIsUnknownNotOn()
{
    // USB connected: the whole partition is gone, so an absent flag proves nothing.
    EXPECT_EQ(BUTTON_UNKNOWN, readButtonState("/tmp/helper_flags_no_such_folder/off", "/tmp/helper_flags_no_such_folder"));
}

static void testTheShippedPathsLiveInOneFolder()
{
    const char *paths[] = {UNINSTALL_PATH, DISABLE_PATH, DEBUG_PATH, OFF_PATH};
    for (unsigned i = 0; i < sizeof(paths) / sizeof(paths[0]); i++) {
        EXPECT_EQ(0, strncmp(paths[i], FOLDER, strlen(FOLDER)));
        EXPECT_EQ('/', paths[i][strlen(FOLDER)]);
    }
    EXPECT_STREQ("/mnt/onboard/.adds/helpers/screenshot/off", OFF_PATH);
}

int main()
{
    if (!mkdtemp(folder)) {
        printf("FAIL: cannot make a temp folder\n");
        return 1;
    }
    snprintf(offFlag, sizeof(offFlag), "%s/off", folder);
    testNoOffFlagInAVisibleFolderMeansOn();
    testOffFlagMeansOff();
    testAnInvisibleFolderIsUnknownNotOn();
    testTheShippedPathsLiveInOneFolder();
    rmdir(folder);
    return finishTests("helper_flags");
}

// Host unit tests for capture_check (no Qt, no C++ standard library).
// Run: make -C helpers/extras docker-test

#include "../../../mod/common/test_expect.h"
#include "../capture_check.h"

#include <fcntl.h>
#include <stdlib.h>
#include <unistd.h>

using namespace CaptureCheck;

static const FileState MISSING = {false, 0};
static const FileState PRESENT = {true, 4096};
static const FileState EMPTY_FILE = {true, 0};

static void testANewNameWithContentIsASavedScreenshot()
{
    // The counter moved past files that already existed.
    EXPECT_EQ(true, didSaveNewFile(true, PRESENT, PRESENT));
    EXPECT_EQ(true, didSaveNewFile(true, MISSING, PRESENT));
}

static void testTheSameNameThatDidNotExistBeforeIsSaved()
{
    EXPECT_EQ(true, didSaveNewFile(false, MISSING, PRESENT)); // first ever capture
}

static void testTheSameNameThatAlreadyExistedMeansNothingWasWritten()
{
    EXPECT_EQ(false, didSaveNewFile(false, PRESENT, PRESENT)); // USB mode: saveCurrentScreen returned at once
}

static void testNoFileOrAnEmptyFileIsAFailure()
{
    EXPECT_EQ(false, didSaveNewFile(true, MISSING, MISSING));   // nothing appeared (USB mode, first tap)
    EXPECT_EQ(false, didSaveNewFile(false, MISSING, MISSING));
    EXPECT_EQ(false, didSaveNewFile(true, MISSING, EMPTY_FILE)); // full partition: a 0-byte file
}

static void testStatFileSeesFilesAndNothingElse()
{
    char path[] = "/tmp/capture_check_test_XXXXXX";
    int descriptor = mkstemp(path);
    EXPECT_EQ(true, descriptor >= 0);
    EXPECT_EQ(3, (long)write(descriptor, "png", 3));
    close(descriptor);
    FileState file = statFile(path);
    EXPECT_EQ(true, file.exists);
    EXPECT_EQ(3, file.size);
    unlink(path);
    EXPECT_EQ(false, statFile(path).exists);
    EXPECT_EQ(false, statFile("/tmp").exists); // a folder is not a screenshot
    EXPECT_EQ(false, statFile("").exists);
}

int main()
{
    testANewNameWithContentIsASavedScreenshot();
    testTheSameNameThatDidNotExistBeforeIsSaved();
    testTheSameNameThatAlreadyExistedMeansNothingWasWritten();
    testNoFileOrAnEmptyFileIsAFailure();
    testStatFileSeesFilesAndNothingElse();
    return finishTests("capture_check");
}

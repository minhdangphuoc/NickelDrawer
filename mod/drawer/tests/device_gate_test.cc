// Host unit tests for device_gate (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../../common/device_gate.h"
#include "../../common/test_expect.h"

#include <stdlib.h>
#include <unistd.h>

using namespace DeviceGate;

static const char LIBRA_2_LINE[] =
    "N000000000000,4.1.15,4.38.23828,4.1.15,4.1.15,00000000-0000-0000-0000-000000000388\n";

static void testModelsWithTheTestedPanelAreSupported()
{
    EXPECT_STREQ("io", findSupportedModel("00000000-0000-0000-0000-000000000388")->codename);
    EXPECT_STREQ("storm", findSupportedModel("00000000-0000-0000-0000-000000000384")->codename);
    EXPECT_STREQ("monza", findSupportedModel("00000000-0000-0000-0000-000000000390")->codename);
    EXPECT_STREQ("388", findSupportedModel("388")->idSuffix);
}

static void testOtherModelsAreRefused()
{
    EXPECT_EQ(true, findSupportedModel("00000000-0000-0000-0000-000000000387") == NULL);  // Elipsa
    EXPECT_EQ(true, findSupportedModel("00000000-0000-0000-0000-000000000383") == NULL);  // Sage
    EXPECT_EQ(true, findSupportedModel("00000000-0000-0000-0000-000000000393") == NULL);  // Clara Colour
    EXPECT_EQ(true, findSupportedModel("00000000-0000-0000-0000-0000000003880") == NULL); // not a suffix match
}

static void testUnreadableIdsAreRefused()
{
    EXPECT_EQ(true, findSupportedModel(NULL) == NULL);
    EXPECT_EQ(true, findSupportedModel("") == NULL);
    EXPECT_EQ(true, findSupportedModel("88") == NULL);
    EXPECT_EQ(true, findSupportedModel("unknown") == NULL);
    EXPECT_EQ(true, findSupportedModel("38a") == NULL);
}

static void testScreenSizeMatchesTheTestedPanelInEitherOrientation()
{
    EXPECT_EQ(true, isSupportedScreenSize(1264, 1680));
    EXPECT_EQ(true, isSupportedScreenSize(1680, 1264));
    EXPECT_EQ(false, isSupportedScreenSize(1072, 1448)); // Clara
    EXPECT_EQ(false, isSupportedScreenSize(1404, 1872)); // Elipsa
    EXPECT_EQ(false, isSupportedScreenSize(1280, 1680)); // virtual framebuffer width, not the panel
    EXPECT_EQ(false, isSupportedScreenSize(0, 0));
}

static void testVersionFieldsAreRead()
{
    char field[64];
    EXPECT_EQ(true, copyVersionField(LIBRA_2_LINE, 2, field, sizeof(field)));
    EXPECT_STREQ("4.38.23828", field);
    EXPECT_EQ(true, copyVersionField(LIBRA_2_LINE, 0, field, sizeof(field)));
    EXPECT_STREQ("N000000000000", field);
    EXPECT_EQ(true, copyLastVersionField(LIBRA_2_LINE, field, sizeof(field)));
    EXPECT_STREQ("00000000-0000-0000-0000-000000000388", field);
}

static void testCarriageReturnIsNotPartOfTheLastField()
{
    char field[64];
    EXPECT_EQ(true, copyLastVersionField("a,b,c,d,e,00000000-0000-0000-0000-000000000384\r\n", field, sizeof(field)));
    EXPECT_STREQ("00000000-0000-0000-0000-000000000384", field);
}

static void testMissingOrOversizedFieldsFail()
{
    char field[8];
    EXPECT_EQ(false, copyVersionField("a,b", 2, field, sizeof(field)));
    EXPECT_STREQ("", field);
    EXPECT_EQ(false, copyVersionField("a,b\nc,d", 2, field, sizeof(field))); // never reads past the line
    EXPECT_EQ(false, copyVersionField("a,,c", 1, field, sizeof(field)));
    EXPECT_EQ(false, copyVersionField(LIBRA_2_LINE, 2, field, sizeof(field))); // 10 chars do not fit in 8
    EXPECT_STREQ("", field);
    EXPECT_EQ(false, copyVersionField(NULL, 0, field, sizeof(field)));
    EXPECT_EQ(false, copyVersionField("a", -1, field, sizeof(field)));
    EXPECT_EQ(false, copyLastVersionField("a,b,\n", field, sizeof(field)));
    EXPECT_EQ(false, copyLastVersionField("", field, sizeof(field)));
    EXPECT_EQ(false, copyLastVersionField(NULL, field, sizeof(field)));
}

static void testOnlyTheFiveTestedBuildsAreSupported()
{
    EXPECT_EQ(true, isSupportedFirmware("4.38.23552"));
    EXPECT_EQ(true, isSupportedFirmware("4.38.23828"));
    EXPECT_EQ(false, isSupportedFirmware("4.38.23829"));
    EXPECT_EQ(false, isSupportedFirmware("5.0.0"));
    EXPECT_EQ(false, isSupportedFirmware(""));
    EXPECT_EQ(false, isSupportedFirmware(NULL));
}

static void testVersionFileIsReadAndClosed()
{
    char path[] = "/tmp/device_gate_test_XXXXXX";
    int descriptor = mkstemp(path);
    EXPECT_EQ(true, descriptor >= 0);
    EXPECT_EQ((long)strlen(LIBRA_2_LINE), (long)write(descriptor, LIBRA_2_LINE, strlen(LIBRA_2_LINE)));
    close(descriptor);
    char firmware[32] = "unknown";
    char deviceId[64] = "unknown";
    EXPECT_EQ(true, readVersionFile(path, firmware, sizeof(firmware), deviceId, sizeof(deviceId)));
    EXPECT_STREQ("4.38.23828", firmware);
    EXPECT_STREQ("00000000-0000-0000-0000-000000000388", deviceId);
    char tooSmall[4] = "x";
    EXPECT_EQ(true, readVersionFile(path, tooSmall, sizeof(tooSmall), deviceId, sizeof(deviceId)));
    EXPECT_STREQ("x", tooSmall); // does not fit: left as it was
    unlink(path);
    EXPECT_EQ(false, readVersionFile(path, firmware, sizeof(firmware), deviceId, sizeof(deviceId)));
    EXPECT_STREQ("4.38.23828", firmware);
}

int main()
{
    testModelsWithTheTestedPanelAreSupported();
    testOtherModelsAreRefused();
    testUnreadableIdsAreRefused();
    testScreenSizeMatchesTheTestedPanelInEitherOrientation();
    testVersionFieldsAreRead();
    testCarriageReturnIsNotPartOfTheLastField();
    testMissingOrOversizedFieldsFail();
    testOnlyTheFiveTestedBuildsAreSupported();
    testVersionFileIsReadAndClosed();
    return finishTests("device_gate");
}

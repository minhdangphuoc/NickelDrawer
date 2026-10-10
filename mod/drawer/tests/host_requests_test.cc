// Host unit tests for host_requests (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../host_requests.h"
#include "../../common/test_expect.h"

using namespace HostRequests;

static void testWifiAndRescanAreAdded()
{
    Plan plan;
    EXPECT_EQ(true, isEmpty(plan));
    EXPECT_EQ(REQUEST_ADDED, add(&plan, "wifi", 0));
    EXPECT_EQ(true, plan.needsWifi);
    EXPECT_EQ(false, isEmpty(plan));
    EXPECT_EQ(REQUEST_ADDED, add(&plan, "rescan", "full"));
    EXPECT_EQ(true, plan.needsFullRescan);
}

static void testRescanModeDefaultsToQuick()
{
    Plan omitted;
    add(&omitted, "rescan", 0);
    EXPECT_EQ(RESCAN_QUICK, rescanToRun(omitted));
    Plan blank;
    add(&blank, "rescan", "");
    EXPECT_EQ(RESCAN_QUICK, rescanToRun(blank));
}

static void testUnknownRequestsAreRejected()
{
    Plan plan;
    EXPECT_EQ(REQUEST_UNKNOWN, add(&plan, "reboot", 0));
    EXPECT_EQ(REQUEST_UNKNOWN, add(&plan, "", 0));
    EXPECT_EQ(REQUEST_UNKNOWN, add(&plan, 0, 0));
    EXPECT_EQ(REQUEST_BAD_MODE, add(&plan, "rescan", "deep"));
    EXPECT_EQ(true, isEmpty(plan));
}

static void testDuplicatesCollapseToOneRescan()
{
    Plan plan;
    add(&plan, "rescan", "quick");
    add(&plan, "rescan", "quick");
    add(&plan, "wifi", 0);
    add(&plan, "wifi", 0);
    EXPECT_EQ(RESCAN_QUICK, rescanToRun(plan));
    // The manifest's rescan and a request for the same thing are one rescan.
    addManifestRescan(&plan, "quick");
    EXPECT_EQ(RESCAN_QUICK, rescanToRun(plan));
}

static void testFullRescanCoversQuick()
{
    Plan plan;
    add(&plan, "rescan", "quick");
    addManifestRescan(&plan, "full");
    EXPECT_EQ(RESCAN_FULL, rescanToRun(plan));
}

static void testManifestRescanValues()
{
    Plan plan;
    EXPECT_EQ(REQUEST_ADDED, addManifestRescan(&plan, "none"));
    EXPECT_EQ(REQUEST_ADDED, addManifestRescan(&plan, ""));
    EXPECT_EQ(REQUEST_ADDED, addManifestRescan(&plan, 0));
    EXPECT_EQ(true, isEmpty(plan));
    EXPECT_EQ(REQUEST_BAD_MODE, addManifestRescan(&plan, "sometimes"));
    EXPECT_EQ(true, isEmpty(plan));
    EXPECT_EQ(RESCAN_NONE, rescanToRun(plan));
}

static void testMergeKeepsEveryRequest()
{
    Plan first;
    add(&first, "wifi", 0);
    Plan second;
    add(&second, "rescan", "full");
    merge(&first, second);
    EXPECT_EQ(true, first.needsWifi);
    EXPECT_EQ(RESCAN_FULL, rescanToRun(first));
}

int main()
{
    testWifiAndRescanAreAdded();
    testRescanModeDefaultsToQuick();
    testUnknownRequestsAreRejected();
    testDuplicatesCollapseToOneRescan();
    testFullRescanCoversQuick();
    testManifestRescanValues();
    testMergeKeepsEveryRequest();
    return finishTests("host_requests");
}

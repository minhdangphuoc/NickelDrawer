// Host unit tests for toast_limiter (no Qt, no C++ standard library).
// Run: make -C mod/drawer docker-test

#include "../toast_limiter.h"
#include "test_expect.h"

using namespace ToastLimiter;

static State state;

static void testSecondToastWithinTwoSecondsIsDropped()
{
    reset(&state);
    EXPECT_EQ(true, allow(&state, "localsend", 10000));
    EXPECT_EQ(false, allow(&state, "localsend", 10001));
    EXPECT_EQ(false, allow(&state, "localsend", 11999));
    EXPECT_EQ(true, allow(&state, "localsend", 12000));
}

static void testDroppedToastDoesNotExtendTheWindow()
{
    reset(&state);
    EXPECT_EQ(true, allow(&state, "a", 5000));
    EXPECT_EQ(false, allow(&state, "a", 6500));
    EXPECT_EQ(true, allow(&state, "a", 7000)); // 2 s after the shown one, not after the dropped one
}

static void testAppsAreLimitedIndependently()
{
    reset(&state);
    EXPECT_EQ(true, allow(&state, "a", 5000));
    EXPECT_EQ(true, allow(&state, "b", 5001));
    EXPECT_EQ(false, allow(&state, "a", 5002));
    EXPECT_EQ(false, allow(&state, "b", 5003));
}

static void testFirstToastAtClockZeroIsAllowed()
{
    reset(&state);
    EXPECT_EQ(true, allow(&state, "a", 0));
    EXPECT_EQ(false, allow(&state, "a", 100));
}

static void testClockGoingBackwardsDoesNotMuteForever()
{
    reset(&state);
    EXPECT_EQ(true, allow(&state, "a", 90000));
    EXPECT_EQ(true, allow(&state, "a", 100));
}

static void testNullKeyIsTheHostKey()
{
    reset(&state);
    EXPECT_EQ(true, allow(&state, 0, 1000));
    EXPECT_EQ(false, allow(&state, "", 1500));
}

static void testFullTableForgetsTheOldestApp()
{
    reset(&state);
    char key[8];
    for (int i = 0; i < MAX_TRACKED_APPS; i++) {
        key[0] = 'a' + i;
        key[1] = '\0';
        EXPECT_EQ(true, allow(&state, key, 1000 + i * 10));
    }
    EXPECT_EQ(true, allow(&state, "newcomer", 1500)); // evicts "a"
    EXPECT_EQ(false, allow(&state, "b", 1600));       // still tracked
    EXPECT_EQ(true, allow(&state, "a", 1700));        // forgotten: allowed again
}

static void testLongKeyIsCutNotOverflowed()
{
    reset(&state);
    char longKey[200];
    for (int i = 0; i < 199; i++) {
        longKey[i] = 'x';
    }
    longKey[199] = '\0';
    EXPECT_EQ(true, allow(&state, longKey, 100));
    EXPECT_EQ(false, allow(&state, longKey, 200));
}

int main()
{
    testSecondToastWithinTwoSecondsIsDropped();
    testDroppedToastDoesNotExtendTheWindow();
    testAppsAreLimitedIndependently();
    testFirstToastAtClockZeroIsAllowed();
    testClockGoingBackwardsDoesNotMuteForever();
    testNullKeyIsTheHostKey();
    testFullTableForgetsTheOldestApp();
    testLongKeyIsCutNotOverflowed();
    return finishTests("toast_limiter");
}

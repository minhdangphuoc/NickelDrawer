// Host unit tests for event_queue (no Qt, no C++ standard library).
// Regression: tapping the "random" radio logged the tap but never sent the
// event; and a second tap while one ran was dropped. Run: make -C mod/drawer docker-test

#include "../event_queue.h"
#include "test_expect.h"

using namespace EventQueue;

static Event radio(const char *bind, const char *value)
{
    return makeEvent(bind, bind, value, "options");
}

static void testEventsLeaveInTheOrderTheyWereTapped()
{
    Queue queue;
    EXPECT_EQ(ENQUEUED, queue.enqueue(radio("rotation", "random")));
    EXPECT_EQ(ENQUEUED, queue.enqueue(radio("mode", "fit")));
    Event event;
    EXPECT_EQ(1, queue.takeNext(&event));
    EXPECT_STREQ("rotation", event.action);
    EXPECT_STREQ("random", event.value);
    EXPECT_STREQ("options", event.screen);
    EXPECT_EQ(0, queue.takeNext(&event)); // one at a time
    queue.finishInFlight();
    EXPECT_EQ(1, queue.takeNext(&event));
    EXPECT_STREQ("mode", event.action);
    queue.finishInFlight();
    EXPECT_EQ(1, queue.isIdle());
}

static void testNewerTapOnTheSameControlReplacesTheQueuedOne()
{
    Queue queue;
    queue.enqueue(radio("rotation", "random"));
    EXPECT_EQ(REPLACED_QUEUED, queue.enqueue(radio("rotation", "fixed")));
    EXPECT_EQ(1, queue.waiting());
    Event event;
    queue.takeNext(&event);
    EXPECT_STREQ("fixed", event.value);
}

static void testSameValueInFlightIsDropped()
{
    Queue queue;
    queue.enqueue(radio("rotation", "random"));
    Event event;
    queue.takeNext(&event);
    EXPECT_EQ(DROPPED_IN_FLIGHT, queue.enqueue(radio("rotation", "random")));
    EXPECT_EQ(0, queue.waiting());
    // A different value for the same control still goes out afterwards.
    EXPECT_EQ(ENQUEUED, queue.enqueue(radio("rotation", "fixed")));
    EXPECT_EQ(1, queue.waiting());
}

static void testFullQueueRefusesTheTap()
{
    Queue queue;
    char key[8];
    for (int i = 0; i < MAX_QUEUED; i++) {
        key[0] = 'k';
        key[1] = (char)('0' + i);
        key[2] = '\0';
        EXPECT_EQ(ENQUEUED, queue.enqueue(radio(key, "on")));
    }
    EXPECT_EQ(DROPPED_FULL, queue.enqueue(radio("extra", "on")));
    EXPECT_EQ(REPLACED_QUEUED, queue.enqueue(radio("k0", "off"))); // coalescing still works when full
}

static void testLongTextsAreCutAndTerminated()
{
    char longText[200];
    for (int i = 0; i < 199; i++) {
        longText[i] = 'x';
    }
    longText[199] = '\0';
    Event event = makeEvent(longText, longText, longText, longText);
    EXPECT_EQ(MAX_TEXT_BYTES - 1, strlen(event.key));
    EXPECT_EQ(MAX_TEXT_BYTES - 1, strlen(event.value));
}

static void testClearForgetsEverything()
{
    Queue queue;
    queue.enqueue(radio("a", "1"));
    Event event;
    queue.takeNext(&event);
    queue.enqueue(radio("b", "2"));
    queue.clear();
    EXPECT_EQ(1, queue.isIdle());
}

int main()
{
    testEventsLeaveInTheOrderTheyWereTapped();
    testNewerTapOnTheSameControlReplacesTheQueuedOne();
    testSameValueInFlightIsDropped();
    testFullQueueRefusesTheTap();
    testLongTextsAreCutAndTerminated();
    testClearForgetsEverything();
    return finishTests("event_queue");
}

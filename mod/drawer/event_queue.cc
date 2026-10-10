#include "event_queue.h"

#include <string.h>

namespace EventQueue {

static void copyText(char *target, const char *source)
{
    int i = 0;
    for (; source && source[i] && i < MAX_TEXT_BYTES - 1; i++) {
        target[i] = source[i];
    }
    target[i] = '\0';
}

Event makeEvent(const char *key, const char *action, const char *value, const char *screen)
{
    Event event;
    copyText(event.key, key);
    copyText(event.action, action);
    copyText(event.value, value);
    copyText(event.screen, screen);
    return event;
}

Queue::Queue() : count(0), hasInFlight(false)
{
    inFlight = makeEvent("", "", "", "");
}

EnqueueResult Queue::enqueue(const Event &event)
{
    if (hasInFlight && strcmp(inFlight.key, event.key) == 0 && strcmp(inFlight.value, event.value) == 0) {
        return DROPPED_IN_FLIGHT;
    }
    for (int i = 0; i < count; i++) {
        if (strcmp(queued[i].key, event.key) == 0) {
            queued[i] = event;
            return REPLACED_QUEUED;
        }
    }
    if (count >= MAX_QUEUED) {
        return DROPPED_FULL;
    }
    queued[count++] = event;
    return ENQUEUED;
}

bool Queue::takeNext(Event *event)
{
    if (hasInFlight || count == 0) {
        return false;
    }
    *event = queued[0];
    for (int i = 1; i < count; i++) {
        queued[i - 1] = queued[i];
    }
    count--;
    inFlight = *event;
    hasInFlight = true;
    return true;
}

void Queue::finishInFlight()
{
    hasInFlight = false;
}

void Queue::clear()
{
    count = 0;
    hasInFlight = false;
}

} // namespace EventQueue

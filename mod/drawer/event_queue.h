#ifndef NICKELDRAWER_EVENT_QUEUE_H
#define NICKELDRAWER_EVENT_QUEUE_H

// The small FIFO of screen events behind radios and toggles. A tap shows its
// new value at once and queues the event; the next one is sent when the
// runner is idle, so tapping "rotation" and then "mode" quickly applies both.
//   - a newer tap on the same control (same key) replaces its queued event;
//   - a tap whose event is already in flight with the same value is dropped;
//   - one event is in flight at a time.
//
// Pure logic, no Qt, no C++ standard library: tests/event_queue_test.cc builds
// it with the host compiler.

namespace EventQueue {

static const int MAX_QUEUED = 8;
static const int MAX_TEXT_BYTES = 64; // each text, NUL included; longer texts are cut

struct Event {
    char key[MAX_TEXT_BYTES];    // the control (its bind), the coalescing key
    char action[MAX_TEXT_BYTES];
    char value[MAX_TEXT_BYTES];
    char screen[MAX_TEXT_BYTES];
};

// Fills an Event, cutting every text safely.
Event makeEvent(const char *key, const char *action, const char *value, const char *screen);

enum EnqueueResult {
    ENQUEUED,
    REPLACED_QUEUED,    // coalesced with an older event of the same control
    DROPPED_IN_FLIGHT,  // same control and value are being sent right now
    DROPPED_FULL,       // MAX_QUEUED waiting: the tap is refused
};

class Queue {
public:
    Queue();

    EnqueueResult enqueue(const Event &event);
    // Moves the oldest event out and marks it in flight. False when empty or
    // when one is still in flight.
    bool takeNext(Event *event);
    // The in-flight event was answered (ok or failed).
    void finishInFlight();

    int waiting() const { return count; }
    bool isInFlight() const { return hasInFlight; }
    // Nothing waiting and nothing in flight.
    bool isIdle() const { return count == 0 && !hasInFlight; }
    void clear();

private:
    Event queued[MAX_QUEUED];
    int count;
    Event inFlight;
    bool hasInFlight;
};

} // namespace EventQueue

#endif

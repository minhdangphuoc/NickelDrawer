#ifndef NICKELDRAWER_TOAST_LIMITER_H
#define NICKELDRAWER_TOAST_LIMITER_H

// At most one toast per app every 2 s (docs/spec/ui-kit.md §3.7). Extra
// toasts are dropped, not queued: a toast is never something the user must
// act on. No Qt, no libnickel, no C++ standard library (host-tested).

namespace ToastLimiter {

static const long MIN_INTERVAL_MILLISECONDS = 2000;
// Apps remembered at once; when full, the least recently toasting one is forgotten.
static const int MAX_TRACKED_APPS = 8;
static const int KEY_SIZE = 64;

struct State {
    int count;
    char keys[MAX_TRACKED_APPS][KEY_SIZE];
    long lastShownMilliseconds[MAX_TRACKED_APPS];
};

void reset(State *state);

// True when a toast for appKey may be shown at nowMilliseconds (a monotonic
// clock); it is then recorded as shown. False: drop it. A clock that went
// backwards counts as elapsed.
bool allow(State *state, const char *appKey, long nowMilliseconds);

} // namespace ToastLimiter

#endif

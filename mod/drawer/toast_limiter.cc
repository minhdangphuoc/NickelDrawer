#include "toast_limiter.h"

#include <stddef.h>
#include <string.h>

namespace ToastLimiter {

void reset(State *state)
{
    memset(state, 0, sizeof(*state));
}

static int indexOfKey(const State &state, const char *appKey)
{
    for (int i = 0; i < state.count; i++) {
        if (strncmp(state.keys[i], appKey, KEY_SIZE - 1) == 0) {
            return i;
        }
    }
    return -1;
}

static int oldestIndex(const State &state)
{
    int oldest = 0;
    for (int i = 1; i < state.count; i++) {
        if (state.lastShownMilliseconds[i] < state.lastShownMilliseconds[oldest]) {
            oldest = i;
        }
    }
    return oldest;
}

static int slotFor(State *state, const char *appKey)
{
    int index = indexOfKey(*state, appKey);
    if (index >= 0) {
        return index;
    }
    index = state->count < MAX_TRACKED_APPS ? state->count++ : oldestIndex(*state);
    strncpy(state->keys[index], appKey, KEY_SIZE - 1);
    state->keys[index][KEY_SIZE - 1] = '\0';
    state->lastShownMilliseconds[index] = 0;
    return index;
}

bool allow(State *state, const char *appKey, long nowMilliseconds)
{
    const char *key = appKey ? appKey : "";
    int known = indexOfKey(*state, key);
    if (known >= 0) {
        long elapsed = nowMilliseconds - state->lastShownMilliseconds[known];
        bool isTooSoon = elapsed >= 0 && elapsed < MIN_INTERVAL_MILLISECONDS;
        if (isTooSoon) {
            return false;
        }
    }
    state->lastShownMilliseconds[slotFor(state, key)] = nowMilliseconds;
    return true;
}

} // namespace ToastLimiter

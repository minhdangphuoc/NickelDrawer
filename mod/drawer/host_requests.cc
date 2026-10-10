#include "host_requests.h"

#include <string.h>

namespace HostRequests {

static bool isBlank(const char *text)
{
    return !text || text[0] == '\0';
}

static Verdict addRescan(Plan *plan, const char *mode)
{
    if (isBlank(mode) || strcmp(mode, "quick") == 0) {
        plan->needsQuickRescan = true;
        return REQUEST_ADDED;
    }
    if (strcmp(mode, "full") == 0) {
        plan->needsFullRescan = true;
        return REQUEST_ADDED;
    }
    return REQUEST_BAD_MODE;
}

Verdict add(Plan *plan, const char *request, const char *mode)
{
    if (!isBlank(request) && strcmp(request, "wifi") == 0) {
        plan->needsWifi = true;
        return REQUEST_ADDED;
    }
    if (!isBlank(request) && strcmp(request, "rescan") == 0) {
        return addRescan(plan, mode);
    }
    return REQUEST_UNKNOWN;
}

Verdict addManifestRescan(Plan *plan, const char *rescan)
{
    if (isBlank(rescan) || strcmp(rescan, "none") == 0) {
        return REQUEST_ADDED;
    }
    return addRescan(plan, rescan);
}

bool isEmpty(const Plan &plan)
{
    return !plan.needsWifi && !plan.needsQuickRescan && !plan.needsFullRescan;
}

void merge(Plan *plan, const Plan &other)
{
    plan->needsWifi = plan->needsWifi || other.needsWifi;
    plan->needsQuickRescan = plan->needsQuickRescan || other.needsQuickRescan;
    plan->needsFullRescan = plan->needsFullRescan || other.needsFullRescan;
}

RescanChoice rescanToRun(const Plan &plan)
{
    if (plan.needsFullRescan) {
        return RESCAN_FULL;
    }
    return plan.needsQuickRescan ? RESCAN_QUICK : RESCAN_NONE;
}

} // namespace HostRequests

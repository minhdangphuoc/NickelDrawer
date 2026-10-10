#ifndef NICKELDRAWER_HOST_REQUESTS_H
#define NICKELDRAWER_HOST_REQUESTS_H

// Pure rules for the host services an app asks for (docs/spec/drawer-protocol.md
// "Requests"): Wi-Fi first, then one library rescan, duplicates collapsed.
// No Qt, no libnickel, no C++ standard library, so tests/host_requests_test.cc
// builds it with the host compiler. model_parser.cc reads the JSON and feeds
// each entry here; host_request_runner.cc performs the plan.

namespace HostRequests {

// What one reply asked for. A plan is built per reply and never kept.
struct Plan {
    bool needsWifi = false;
    bool needsQuickRescan = false;
    bool needsFullRescan = false;
};

enum Verdict { REQUEST_ADDED, REQUEST_UNKNOWN, REQUEST_BAD_MODE };

// request: "wifi" or "rescan"; mode (rescan only): "quick", "full", NULL or ""
// (= quick). Anything else is rejected and the plan stays unchanged.
Verdict add(Plan *plan, const char *request, const char *mode);

// The manifest's `rescan` field of a reply: "none", "quick", "full" or empty.
Verdict addManifestRescan(Plan *plan, const char *rescan);

bool isEmpty(const Plan &plan);

// A second reply arrived while the first plan is still being performed.
void merge(Plan *plan, const Plan &other);

enum RescanChoice { RESCAN_NONE, RESCAN_QUICK, RESCAN_FULL };

// One rescan per plan: a full rescan covers a quick one.
RescanChoice rescanToRun(const Plan &plan);

} // namespace HostRequests

#endif

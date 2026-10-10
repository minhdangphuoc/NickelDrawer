package drawer

import "testing"

// A finished job's one-shot rescan handed over while RunEvent resolves its toast must reach
// the mod in that reply; the next screen read no longer has it.
func TestApplyScreenSignalCarriesTheScreensRequests(t *testing.T) {
	quick := Request{Name: "rescan", Mode: "quick"}
	wifi := Request{Name: "wifi"}
	result := RunResult{Text: `{"toast":"started"}`, Requests: []Request{wifi}}
	screen := ScreenReply{Toasts: map[string]string{"started": "Started"}, Requests: []Request{quick, wifi}}
	got := applyScreenSignal(result, screenSignal{Toast: "started"}, screen)
	if got.Toast != "Started" {
		t.Fatalf("toast = %q", got.Toast)
	}
	if len(got.Requests) != 2 || got.Requests[0] != wifi || got.Requests[1] != quick {
		t.Fatalf("requests = %+v, want wifi then one quick rescan", got.Requests)
	}
}

package nmconfig

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
)

var update = flag.Bool("update", false, "rewrite golden files")

var englishLabels = Labels{Refresh: "Apps · Refresh list", Errors: "Apps · Errors (2)"}

// vietnamese stands in for locales/vi.json of the app and the host.
var vietnamese = i18n.NewBundle(i18n.Catalog{
	"actions.start.label": "Bắt đầu nhận",
	"actions.stop.label":  "Dừng nhận",
})

var vietnameseLabels = Labels{Refresh: "Ứng dụng · Làm mới danh sách", Errors: "Ứng dụng · Lỗi (2)"}

func intPtr(n int) *int    { return &n }
func boolPtr(b bool) *bool { return &b }

func localSend() manifest.App {
	return manifest.App{
		ID: "localsend", Name: "LocalSend",
		Actions: []manifest.Action{
			{ID: "start", Label: "Start receiving", Run: "bin/start.sh", Output: "show", TimeoutMS: intPtr(9500), Wifi: true, Rescan: "none"},
			{ID: "stop", Label: "Stop receiving", Run: "bin/stop.sh", Output: "show", TimeoutMS: intPtr(3000), Rescan: "quick"},
		},
	}
}

func mixedApps() []manifest.App {
	return []manifest.App{
		localSend(),
		{ID: "zeta", Name: "Zeta", Actions: []manifest.Action{
			{ID: "sync", Label: "Sync", Run: "bin/sync.sh", Output: "none", Wifi: true, Rescan: "full"},
		}},
		{ID: "alpha", Name: "Alpha", Actions: []manifest.Action{
			{ID: "ping", Label: "Ping", Run: "bin/ping.sh", Output: "show"},
		}},
		{ID: "hidden", Name: "Hidden", Menu: boolPtr(false), Actions: []manifest.Action{
			{ID: "x", Label: "X", Run: "bin/x.sh", Output: "show"},
		}},
	}
}

func TestRenderMatchesGolden(t *testing.T) {
	cases := []struct {
		golden     string
		apps       []manifest.App
		errorCount int
		labels     Labels
	}{
		{"localsend.golden", []manifest.App{localSend()}, 0, englishLabels},
		{"localsend-vi.golden", []manifest.App{localSend().Localize(vietnamese)}, 2, vietnameseLabels},
		{"mixed-with-errors.golden", mixedApps(), 2, englishLabels},
		{"empty.golden", nil, 0, englishLabels},
	}
	for _, c := range cases {
		t.Run(c.golden, func(t *testing.T) {
			got := Render(c.apps, c.errorCount, DefaultPaths(), c.labels)
			goldenPath := filepath.Join("testdata", c.golden)
			if *update {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if got != string(want) {
				t.Fatalf("output differs from %s\n--- got ---\n%s--- want ---\n%s", goldenPath, got, want)
			}
		})
	}
}

func TestItemCountSkipsHiddenApps(t *testing.T) {
	if got := ItemCount(mixedApps()); got != 4 {
		t.Fatalf("ItemCount = %d, want 4", got)
	}
}

func TestRenderUsesCustomPaths(t *testing.T) {
	got := Render([]manifest.App{localSend()}, 0, Paths{AppsRoot: "/tmp/apps", KoboappsRoot: "/tmp/host"}, englishLabels)
	if want := "/tmp/apps/localsend/bin/stop.sh"; !strings.Contains(got, want) {
		t.Fatalf("missing %s in\n%s", want, got)
	}
	if want := "/tmp/host/bin/appsctl refresh"; !strings.Contains(got, want) {
		t.Fatalf("missing %s in\n%s", want, got)
	}
}

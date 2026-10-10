package drawer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/manifest"
)

func writeStatusApp(t *testing.T, appsDir, id string) {
	t.Helper()
	dir := filepath.Join(appsDir, id)
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestJSON := []byte(`{"schema":1,"id":"` + id + `","name":"` + id + `","version":"1.0.0",
 "actions":[{"id":"status","label":"Status","run":"bin/status.sh","output":"show"}]}`)
	if err := os.WriteFile(filepath.Join(dir, "app.json"), manifestJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "status.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func testConfig(t *testing.T, runner ScriptRunner) (Config, *time.Time) {
	t.Helper()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return Config{
		AppsDir: t.TempDir(), HostDir: t.TempDir(), CacheDir: t.TempDir(), Locale: "en",
		Bundle: i18n.NewBundle(), RunScript: runner, Now: func() time.Time { return now },
		StatusTimeout: 50 * time.Millisecond,
	}, &now
}

func TestParseStatus(t *testing.T) {
	cases := map[string]Status{
		"":                                   {},
		"\n  \nIdle\nsecond line":            {Line: "Idle"},
		`{"running":true,"line":"PIN 4821"}`: {Line: "PIN 4821", Running: true},
		`{"line":"Stopped"}`:                 {Line: "Stopped"},
		`{not json`:                          {Line: "{not json"},
	}
	for input, want := range cases {
		if got := parseStatus(input); got != want {
			t.Errorf("parseStatus(%q) = %+v, want %+v", input, got, want)
		}
	}
}

func TestStatusIsCachedForTheTTL(t *testing.T) {
	var calls atomic.Int32
	config, now := testConfig(t, func(context.Context, string) ScriptOutput {
		calls.Add(1)
		return ScriptOutput{Stdout: "Idle\n"}
	})
	writeStatusApp(t, config.AppsDir, "one")
	build := func() Model {
		model, err := config.BuildModel()
		if err != nil {
			t.Fatal(err)
		}
		return model
	}
	if got := build().Sections[0].Items[0].Status; got != "Idle" {
		t.Fatalf("status = %q", got)
	}
	build()
	if calls.Load() != 1 {
		t.Fatalf("second call inside the TTL must use the cache, scripts ran %d times", calls.Load())
	}
	*now = now.Add(31 * time.Second)
	build()
	if calls.Load() != 2 {
		t.Fatalf("an expired cache must re-run the script, ran %d times", calls.Load())
	}
	config.Locale = "vi"
	build()
	if calls.Load() != 3 {
		t.Fatalf("a cache from another language must be ignored, ran %d times", calls.Load())
	}
}

func TestSlowStatusIsOmittedWithinBudget(t *testing.T) {
	config, _ := testConfig(t, func(ctx context.Context, script string) ScriptOutput {
		if filepath.Base(filepath.Dir(filepath.Dir(script))) == "slow" {
			<-ctx.Done()
			return ScriptOutput{TimedOut: true, ExitCode: -1}
		}
		return ScriptOutput{Stdout: "ready"}
	})
	writeStatusApp(t, config.AppsDir, "slow")
	writeStatusApp(t, config.AppsDir, "fast")
	start := time.Now()
	model, err := config.BuildModel()
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("model took %v; the slow app must not hold it up", elapsed)
	}
	statuses := map[string]string{}
	for _, item := range model.Sections[0].Items {
		statuses[item.ID] = item.Status
	}
	if statuses["slow"] != "" || statuses["fast"] != "ready" {
		t.Errorf("statuses = %v", statuses)
	}
}

func TestFailingStatusIsOmitted(t *testing.T) {
	config, _ := testConfig(t, func(context.Context, string) ScriptOutput {
		return ScriptOutput{Stdout: "garbage", ExitCode: 1}
	})
	writeStatusApp(t, config.AppsDir, "one")
	model, _ := config.BuildModel()
	if item := model.Sections[0].Items[0]; item.Status != "" || *item.Running {
		t.Errorf("a failing status script must leave the item blank: %+v", item)
	}
}

func TestExecScriptKillsBackgroundChildrenOnTimeout(t *testing.T) {
	script := filepath.Join(t.TempDir(), "slow.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30 &\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	output := ExecScript(ctx, script)
	if !output.TimedOut || time.Since(start) > 5*time.Second {
		t.Errorf("want a prompt timeout, got %+v after %v", output, time.Since(start))
	}
}

func TestRunShowTimeoutIsReportedNotRaised(t *testing.T) {
	config, _ := testConfig(t, func(context.Context, string) ScriptOutput { return ScriptOutput{TimedOut: true} })
	writeStatusApp(t, config.AppsDir, "one")
	result, err := config.Run("one", "status")
	if err != nil || result.OK || result.Rescan != "none" {
		t.Errorf("result = %+v, err = %v", result, err)
	}
}

func TestAppProseNeverReachesTheScreen(t *testing.T) {
	tests := []struct {
		name     string
		output   ScriptOutput
		wantOK   bool
		wantText string
		wantLog  bool
	}{
		{"prose on success is dropped", ScriptOutput{Stdout: "All done\n"}, true, "", true},
		{"silent success", ScriptOutput{}, true, "", false},
		{"toast signal is kept", ScriptOutput{Stdout: "{\"toast\":\"saved\"}\n"}, true, "{\"toast\":\"saved\"}", false},
		{"failure becomes a short message", ScriptOutput{Stdout: "oops", ExitCode: 1}, false, "Status did not work.", true},
		{"silent failure too", ScriptOutput{ExitCode: 3}, false, "Status did not work.", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, _ := testConfig(t, func(context.Context, string) ScriptOutput { return test.output })
			config.Bundle = i18n.NewBundle(i18n.Catalog{"drawer.script_failed": "{label} did not work."})
			var debugLog bytes.Buffer
			config.DebugLog = &debugLog
			writeStatusApp(t, config.AppsDir, "one")
			result, err := config.run("one", "status")
			if err != nil {
				t.Fatal(err)
			}
			if result.OK != test.wantOK || result.Text != test.wantText {
				t.Errorf("result = %+v", result)
			}
			if hasLog := strings.Contains(debugLog.String(), "dropped"); hasLog != test.wantLog {
				t.Errorf("debug log = %q, want a drop entry: %v", debugLog.String(), test.wantLog)
			}
		})
	}
}

func TestRunDropsSignalsWithoutAScreen(t *testing.T) {
	config, _ := testConfig(t, func(context.Context, string) ScriptOutput {
		return ScriptOutput{Stdout: "{\"toast\":\"saved\"}\n"}
	})
	writeStatusApp(t, config.AppsDir, "one")
	result, err := config.Run("one", "status")
	if err != nil || !result.OK || result.Text != "" {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
}

func TestActionTimeoutHonoursTimeoutMSUpToTheCap(t *testing.T) {
	config := Config{ShowTimeout: 30 * time.Second}
	shortMS, longMS := 1000, 9500
	tests := []struct {
		name   string
		action manifest.Action
		want   time.Duration
	}{
		{"declared", manifest.Action{Output: manifest.OutputShow, TimeoutMS: &shortMS}, time.Second},
		{"declared long", manifest.Action{Output: manifest.OutputShow, TimeoutMS: &longMS}, 9500 * time.Millisecond},
		{"manifest default", manifest.Action{Output: manifest.OutputShow}, time.Duration(manifest.DefaultTimeout) * time.Millisecond},
	}
	for _, test := range tests {
		if got := config.actionTimeout(test.action); got != test.want {
			t.Errorf("%s: timeout = %v, want %v", test.name, got, test.want)
		}
	}
	capped := Config{ShowTimeout: 2 * time.Second}
	if got := capped.actionTimeout(manifest.Action{Output: manifest.OutputShow, TimeoutMS: &longMS}); got != 2*time.Second {
		t.Errorf("capped timeout = %v, want 2s", got)
	}
}

func TestRunForegroundPassesTheDeclaredTimeoutToTheScript(t *testing.T) {
	var seen time.Duration
	config, _ := testConfig(t, func(ctx context.Context, _ string) ScriptOutput {
		deadline, _ := ctx.Deadline()
		seen = time.Until(deadline)
		return ScriptOutput{}
	})
	timeoutMS := 1000
	action := manifest.Action{ID: "go", Label: "Go", Run: "bin/go.sh", Output: manifest.OutputShow, TimeoutMS: &timeoutMS}
	if _, err := config.RunAction(t.TempDir(), action); err != nil {
		t.Fatal(err)
	}
	if seen <= 0 || seen > time.Second {
		t.Errorf("script deadline %v away, want within the declared 1s (not the 30s cap)", seen)
	}
}

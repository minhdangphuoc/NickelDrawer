package drawer

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"koboapps/internal/manifest"
	"koboapps/internal/update"
)

func guardedApp(hasStop bool) manifest.App {
	app := manifest.App{ID: "radio", Dir: "/apps/radio", Actions: []manifest.Action{
		{ID: "status", Run: "bin/status.sh", Output: manifest.OutputShow}}}
	if hasStop {
		app.Actions = append(app.Actions, manifest.Action{ID: "stop", Run: "bin/stop.sh", Output: manifest.OutputShow})
	}
	return app
}

// runnerForApp simulates an app that runs until its stop script is called; stopWorks says
// whether that script really stops it.
func runnerForApp(isRunning *bool, stopWorks bool) ScriptRunner {
	return func(_ context.Context, script string) ScriptOutput {
		switch filepath.Base(script) {
		case "status.sh":
			if *isRunning {
				return ScriptOutput{Stdout: `{"running":true,"line":"Playing"}`}
			}
			return ScriptOutput{Stdout: `{"running":false}`}
		case "stop.sh":
			*isRunning = *isRunning && !stopWorks
		}
		return ScriptOutput{}
	}
}

func TestEnsureStoppedStopsARunningAppThroughItsStopAction(t *testing.T) {
	isRunning := true
	config := Config{RunScript: runnerForApp(&isRunning, true)}
	if err := config.EnsureStopped(guardedApp(true)); err != nil || isRunning {
		t.Fatalf("err=%v stillRunning=%v", err, isRunning)
	}
}

func TestEnsureStoppedIsBusyWithoutAStopActionOrWhenItDoesNotStop(t *testing.T) {
	isRunning := true
	if err := (Config{RunScript: runnerForApp(&isRunning, true)}).EnsureStopped(guardedApp(false)); !errors.Is(err, update.ErrAppBusy) {
		t.Errorf("no stop action: %v", err)
	}
	if err := (Config{RunScript: runnerForApp(&isRunning, false)}).EnsureStopped(guardedApp(true)); !errors.Is(err, update.ErrAppBusy) {
		t.Errorf("stop did nothing: %v", err)
	}
}

func TestEnsureStoppedLeavesIdleAndStatuslessAppsAlone(t *testing.T) {
	isRunning := false
	if err := (Config{RunScript: runnerForApp(&isRunning, true)}).EnsureStopped(guardedApp(true)); err != nil {
		t.Errorf("idle app: %v", err)
	}
	if err := (Config{}).EnsureStopped(manifest.App{ID: "plain"}); err != nil {
		t.Errorf("app without a status action: %v", err)
	}
}

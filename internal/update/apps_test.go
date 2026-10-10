package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"koboapps/internal/manifest"
	"koboapps/internal/release"
)

var (
	todo100 = appSpec{id: "todo", version: "1.0.0", name: "To-do", vi: "Việc cần làm"}
	todo101 = appSpec{id: "todo", version: "1.0.1", name: "To-do", vi: "Việc cần làm", extra: map[string]string{"locales/en.json": "{}"}}
	paper05 = appSpec{id: "paper", version: "0.5.0", name: "Paper"}
)

var todoData = map[string]string{"list.json": `["milk","eggs"]`, "nested/state.bin": "\x00\x01\x02 binary"}

func TestCheckStoresTheVerifiedCatalogAndItsSequence(t *testing.T) {
	host := newAppsHost(t)
	host.publish(4, todo101, paper05)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	result, err := device.checkApps()
	if err != nil || result.Key != "appupdate.checked" || result.Params["updates"] != "1" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	catalog, found := ReadAppsCatalog(device.layout())
	if !found || catalog.Sequence != 4 || len(catalog.Apps) != 2 {
		t.Fatalf("cached catalog = %+v", catalog)
	}
	if readAppsSequence(device.layout()) != 4 {
		t.Error("sequence not remembered")
	}
	entries := AppEntries(catalog, InstalledAppVersions(device.layout()), "vi")
	want := map[string]AppState{"todo": AppStateUpdate, "paper": AppStateNotInstalled}
	for _, entry := range entries {
		if entry.State != want[entry.ID] {
			t.Errorf("%s state = %s, want %s", entry.ID, entry.State, want[entry.ID])
		}
	}
	if entries[1].Name != "Việc cần làm" && entries[0].Name != "Việc cần làm" {
		t.Errorf("Vietnamese name missing: %+v", entries)
	}
	record, ok := ReadAppsCheck(device.layout())
	if !ok || record.Result != CheckUpToDate {
		t.Errorf("check record = %+v", record)
	}
}

func TestADrawerSignatureIsNotACatalogSignature(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101)
	device := newAppsDevice(t, host)
	// The release key signing the catalog bytes the way release.json is signed.
	host.serveCatalog(&release.Catalog{V: 1, Sequence: 2, Issued: "2026-01-01T00:00:00Z", Apps: nil},
		func(data []byte) string { return release.Sign(host.privateKey, data) })
	_, err := device.checkApps()
	if messageKey(err) != "appupdate.error.bad_signature" {
		t.Fatalf("err = %v", err)
	}
	if _, found := ReadAppsCatalog(device.layout()); found {
		t.Error("an unverified catalog was cached")
	}
	record, _ := ReadAppsCheck(device.layout())
	if record.Result != CheckFailed || record.Report == nil || record.Report.Key != "appupdate.error.bad_signature" {
		t.Errorf("record = %+v", record)
	}
}

func TestCatalogSignedByAnotherKeyIsRejected(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101)
	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	host.serveCatalog(&release.Catalog{V: 1, Sequence: 2, Issued: "2026-01-01T00:00:00Z"},
		func(data []byte) string { return release.SignCatalog(otherKey, data) })
	if _, err := newAppsDevice(t, host).checkApps(); messageKey(err) != "appupdate.error.bad_signature" {
		t.Fatalf("err = %v", err)
	}
}

func TestACatalogWithALowerSequenceIsRefusedAndAnEqualOneIsUnchanged(t *testing.T) {
	host := newAppsHost(t)
	device := newAppsDevice(t, host)
	host.publish(5, todo101)
	if _, err := device.checkApps(); err != nil {
		t.Fatal(err)
	}
	host.publish(4, todo100) // an old, validly signed catalog replayed by a mirror
	if _, err := device.checkApps(); messageKey(err) != "appupdate.error.stale_catalog" {
		t.Fatalf("lower sequence: err = %v", err)
	}
	if catalog, _ := ReadAppsCatalog(device.layout()); catalog.Sequence != 5 || catalog.Apps[0].Version != "1.0.1" {
		t.Errorf("the cache was replaced by the old catalog: %+v", catalog)
	}
	host.publish(5, todo101)
	if _, err := device.checkApps(); err != nil {
		t.Fatalf("equal sequence: %v", err)
	}
	host.publish(6, todo101)
	if _, err := device.checkApps(); err != nil || readAppsSequence(device.layout()) != 6 {
		t.Fatalf("higher sequence: err=%v seq=%d", err, readAppsSequence(device.layout()))
	}
}

func TestMissingCatalogAndUnreachableHostAreSeparateErrors(t *testing.T) {
	host := newAppsHost(t)
	device := newAppsDevice(t, host)
	if _, err := device.checkApps(); messageKey(err) != "appupdate.error.no_catalog" {
		t.Fatalf("no catalog: err = %v", err)
	}
	host.server.Close()
	if _, err := device.checkApps(); messageKey(err) != "appupdate.error.no_network" && messageKey(err) != "appupdate.error.unreachable" {
		t.Fatalf("closed host: err = %v", err)
	}
}

func TestUpdateOneAppKeepsItsDataByteForByteAndLeavesABackup(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101, paper05)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	device.install(appSpec{id: "other", version: "3.0.0", name: "Other"}, map[string]string{"keep": "me"})
	before := device.dataOf("todo")

	result, err := device.updater().ApplyApp(context.Background(), "todo")
	if err != nil || result.Key != "appupdate.updated" || result.Params["version"] != "1.0.1" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if device.installedVersion("todo") != "1.0.1" {
		t.Fatalf("version = %s", device.installedVersion("todo"))
	}
	if !reflect.DeepEqual(device.dataOf("todo"), before) {
		t.Errorf("data changed:\nbefore %v\nafter  %v", before, device.dataOf("todo"))
	}
	if got := device.dataOf("other"); got["keep"] != "me" {
		t.Error("another app was touched")
	}
	if names := device.leftovers(); len(names) != 0 {
		t.Errorf("leftover folders: %v", names)
	}
	backup := device.layout().appRollbackDir("todo")
	app, _, err := manifest.Load(backup)
	if err != nil || app.Version != "1.0.0" || holdsData(backup) {
		t.Fatalf("backup: %v %v data=%v", app.Version, err, holdsData(backup))
	}
	info, _ := os.Stat(filepath.Join(device.root, ".adds/apps/todo/bin/run.sh"))
	if info.Mode().Perm() != 0o755 {
		t.Errorf("run.sh mode = %o", info.Mode().Perm())
	}
	if !ConsumeRescanRequest(device.layout()) || ConsumeRescanRequest(device.layout()) {
		t.Error("the rescan request must be handed over exactly once")
	}
	job, _ := ReadAppsJob(device.layout())
	if job.State != StateDone || job.Report.Key != "appupdate.updated" {
		t.Errorf("job = %+v", job)
	}
}

func TestInstallANewApp(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101, paper05)
	device := newAppsDevice(t, host)
	result, err := device.updater().ApplyApp(context.Background(), "paper")
	if err != nil || result.Key != "appupdate.installed" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if device.installedVersion("paper") != "0.5.0" || len(device.leftovers()) != 0 {
		t.Errorf("version=%s leftovers=%v", device.installedVersion("paper"), device.leftovers())
	}
	if _, err := device.updater().ApplyApp(context.Background(), "ghost"); messageKey(err) != "appupdate.error.unknown_app" {
		t.Errorf("unknown app: %v", err)
	}
}

func TestNeverInstallAnOlderOrEqualVersion(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo100)
	device := newAppsDevice(t, host)
	device.install(todo101, todoData)
	result, err := device.updater().ApplyApp(context.Background(), "todo")
	if err != nil || result.Key != "appupdate.up_to_date" || device.installedVersion("todo") != "1.0.1" {
		t.Fatalf("older: result=%+v err=%v version=%s", result, err, device.installedVersion("todo"))
	}
	host.publish(2, todo101)
	if result, err := device.updater().ApplyApp(context.Background(), "todo"); err != nil || result.Key != "appupdate.up_to_date" {
		t.Fatalf("equal: result=%+v err=%v", result, err)
	}
	all, err := device.updater().ApplyAllApps(context.Background())
	if err != nil || all.Key != "appupdate.up_to_date" {
		t.Fatalf("update all with nothing newer: %+v %v", all, err)
	}
	catalog, _ := ReadAppsCatalog(device.layout())
	if entries := AppEntries(catalog, InstalledAppVersions(device.layout()), "en"); entries[0].State != AppStateInstalled {
		t.Errorf("state = %s", entries[0].State)
	}
}

func TestDownloadsThatDoNotMatchTheSignedCatalogChangeNothing(t *testing.T) {
	cases := map[string]struct {
		serve   func(t *testing.T, app release.CatalogApp) []byte
		wantKey string
	}{
		"same size, other bytes": {func(t *testing.T, app release.CatalogApp) []byte {
			return append([]byte{}, zipOf(t, "todo", map[string]string{"app.json": "tampered-same-length-aaaa"}, nil)...)
		}, "appupdate.error.hash_mismatch"},
		"shorter": {func(t *testing.T, app release.CatalogApp) []byte { return []byte("short") }, "appupdate.error.hash_mismatch"},
		"longer": {func(t *testing.T, app release.CatalogApp) []byte {
			return append(zipOf(t, "todo", todo101.files(), nil), 0)
		}, "appupdate.error.hash_mismatch"},
	}
	for name, test := range cases {
		host := newAppsHost(t)
		host.publish(1, todo101)
		device := newAppsDevice(t, host)
		device.install(todo100, todoData)
		catalog, _ := release.ParseCatalog(host.files["/o/r/releases/download/apps/apps.json"])
		app, _ := catalog.Find("todo")
		host.replaceZip(app, test.serve(t, app))
		_, err := device.updater().ApplyApp(context.Background(), "todo")
		if messageKey(err) != test.wantKey {
			t.Errorf("%s: err = %v", name, err)
		}
		if device.installedVersion("todo") != "1.0.0" || len(device.leftovers()) != 0 || !reflect.DeepEqual(device.dataOf("todo"), todoData) {
			t.Errorf("%s: the device changed", name)
		}
	}
}

func TestUnsafeZipsAreRefusedBeforeAnythingIsTouched(t *testing.T) {
	cases := map[string]struct {
		files map[string]string
		links map[string]string
	}{
		"symlink":        {todo101.files(), map[string]string{"bin/evil": "/etc/passwd"}},
		"data folder":    {map[string]string{"app.json": todo101.manifestText(), "data/token": "x"}, nil},
		"preview state":  {map[string]string{"app.json": todo101.manifestText(), "preview-state.json": "{}"}, nil},
		"no manifest":    {map[string]string{"bin/run.sh": "x"}, nil},
		"case duplicate": {map[string]string{"app.json": todo101.manifestText(), "A.txt": "1", "a.txt": "2"}, nil},
	}
	for name, test := range cases {
		host := newAppsHost(t)
		device := newAppsDevice(t, host)
		device.install(todo100, todoData)
		zipBytes := zipOf(t, "todo", test.files, test.links)
		app := entryFor(todo101, zipBytes) // a correctly signed entry for a hostile zip
		host.mu.Lock()
		host.files["/o/r/"+app.DownloadPath()] = zipBytes
		host.mu.Unlock()
		host.serveCatalog(&release.Catalog{V: 1, Sequence: 1, Issued: "2026-01-01T00:00:00Z", Apps: []release.CatalogApp{app}}, nil)
		_, err := device.updater().ApplyApp(context.Background(), "todo")
		if k := messageKey(err); k != "appupdate.error.unsafe_zip" {
			t.Errorf("%s: err key = %q (%v)", name, k, err)
		}
		if device.installedVersion("todo") != "1.0.0" || len(device.leftovers()) != 0 || !reflect.DeepEqual(device.dataOf("todo"), todoData) {
			t.Errorf("%s: the device changed (leftovers %v)", name, device.leftovers())
		}
	}
}

func TestAnAppThatFailsValidationIsRolledBackWithItsData(t *testing.T) {
	host := newAppsHost(t)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	// Passes the zip rules and names the right id and version, but is not a valid app.json.
	broken := map[string]string{"app.json": `{"id":"todo","version":"1.0.1"}`}
	zipBytes := zipOf(t, "todo", broken, nil)
	app := entryFor(todo101, zipBytes)
	host.mu.Lock()
	host.files["/o/r/"+app.DownloadPath()] = zipBytes
	host.mu.Unlock()
	host.serveCatalog(&release.Catalog{V: 1, Sequence: 1, Issued: "2026-01-01T00:00:00Z", Apps: []release.CatalogApp{app}}, nil)
	_, err := device.updater().ApplyApp(context.Background(), "todo")
	if messageKey(err) != "appupdate.error.rolled_back" {
		t.Fatalf("err = %v", err)
	}
	if device.installedVersion("todo") != "1.0.0" || len(device.leftovers()) != 0 || !reflect.DeepEqual(device.dataOf("todo"), todoData) {
		t.Errorf("not restored: version=%s leftovers=%v data=%v", device.installedVersion("todo"), device.leftovers(), device.dataOf("todo"))
	}
}

func TestNeedsANewerDrawer(t *testing.T) {
	host := newAppsHost(t)
	needy := appSpec{id: "todo", version: "1.0.1", name: "To-do", minHost: "1.0.0"} // the test drawer is 1.0.0-beta.5
	host.publish(1, needy)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	_, err := device.updater().ApplyApp(context.Background(), "todo")
	if messageKey(err) != "appupdate.error.needs_newer_drawer" || device.installedVersion("todo") != "1.0.0" {
		t.Fatalf("err = %v version=%s", err, device.installedVersion("todo"))
	}
}

type fakeGuard struct {
	err     error
	stopped []string
}

func (guard *fakeGuard) EnsureStopped(app manifest.App) error {
	guard.stopped = append(guard.stopped, app.ID)
	return guard.err
}

func TestARunningAppThatCannotBeStoppedIsBusy(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	guard := &fakeGuard{err: ErrAppBusy}
	updater := device.updater()
	updater.options.Guard = guard
	_, err := updater.ApplyApp(context.Background(), "todo")
	if messageKey(err) != "appupdate.error.app_busy" || device.installedVersion("todo") != "1.0.0" {
		t.Fatalf("err = %v version=%s", err, device.installedVersion("todo"))
	}
	guard.err = nil
	if _, err := updater.ApplyApp(context.Background(), "todo"); err != nil || !reflect.DeepEqual(guard.stopped, []string{"todo", "todo"}) {
		t.Fatalf("after stopping: err=%v stopped=%v", err, guard.stopped)
	}
}

func TestUpdateAllContinuesPastAFailureAndCountsBoth(t *testing.T) {
	host := newAppsHost(t)
	paper06 := appSpec{id: "paper", version: "0.6.0", name: "Paper"}
	host.publish(1, todo101, paper06)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	device.install(paper05, nil)
	catalog, _ := release.ParseCatalog(host.files["/o/r/releases/download/apps/apps.json"])
	app, _ := catalog.Find("paper")
	host.replaceZip(app, []byte("garbage"))
	result, err := device.updater().ApplyAllApps(context.Background())
	if err != nil || result.Key != "appupdate.updated_some" || result.Params["done"] != "1" || result.Params["failed"] != "1" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if device.installedVersion("todo") != "1.0.1" || device.installedVersion("paper") != "0.5.0" {
		t.Errorf("todo=%s paper=%s", device.installedVersion("todo"), device.installedVersion("paper"))
	}
	host.publish(2, todo101, paper06)
	result, err = device.updater().ApplyAllApps(context.Background())
	if err != nil || result.Key != "appupdate.updated_all" || result.Params["count"] != "1" {
		t.Fatalf("second run: result=%+v err=%v", result, err)
	}
}

func TestSecondJobIsRefusedWhileOneRuns(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101)
	device := newAppsDevice(t, host)
	// PID 1 is alive and is never this test process.
	if err := writeAppsJob(device.layout(), AppsJob{State: StateRunning, Job: AppsJobUpdate, PID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := device.checkApps(); messageKey(err) != "appupdate.error.already_running" {
		t.Fatalf("err = %v", err)
	}
	if err := writeAppsJob(device.layout(), AppsJob{State: StateRunning, Job: AppsJobUpdate, PID: 0x7ffffff0}); err != nil {
		t.Fatal(err)
	}
	if job, _ := ReadAppsJob(device.layout()); job.State != StateFailed || job.Report.Key != "appupdate.error.interrupted" {
		t.Errorf("dead job = %+v", job)
	}
}

func TestNotEnoughSpaceIsReportedBeforeWriting(t *testing.T) {
	err := requireSpace(t.TempDir(), 1<<60, "todo")
	if messageKey(err) != "appupdate.error.no_space" {
		t.Fatalf("err = %v", err)
	}
	if err := requireSpace(t.TempDir(), 1, "todo"); err != nil {
		t.Fatalf("a tiny need on a normal disk: %v", err)
	}
}

func TestRollbackAppRestoresThePreviousVersionKeepingData(t *testing.T) {
	host := newAppsHost(t)
	host.publish(1, todo101)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	if _, err := device.updater().ApplyApp(context.Background(), "todo"); err != nil {
		t.Fatal(err)
	}
	result, err := device.updater().RollbackApp("todo")
	if err != nil || result.Key != "appupdate.rolled_back_app" || device.installedVersion("todo") != "1.0.0" {
		t.Fatalf("result=%+v err=%v version=%s", result, err, device.installedVersion("todo"))
	}
	if !reflect.DeepEqual(device.dataOf("todo"), todoData) || len(device.leftovers()) != 0 {
		t.Errorf("data=%v leftovers=%v", device.dataOf("todo"), device.leftovers())
	}
	if _, err := device.updater().RollbackApp("todo"); messageKey(err) != "appupdate.error.no_rollback" {
		t.Errorf("a second rollback: %v", err)
	}
	if _, err := device.updater().RollbackApp("paper"); messageKey(err) != "appupdate.error.no_rollback" {
		t.Errorf("rollback of a never-updated app: %v", err)
	}
}

// TestPowerCutAtAnyStepNeverLosesData cuts the power after each step of the swap and runs the
// recovery of the next start: the data must be byte-identical in the installed app, and no
// leftover folder may hold it.
func TestPowerCutAtAnyStepNeverLosesData(t *testing.T) {
	for _, step := range []string{stepDataMoved, stepOldRenamed, stepNewRenamed, stepValidated} {
		t.Run(step, func(t *testing.T) {
			host := newAppsHost(t)
			host.publish(1, todo101)
			device := newAppsDevice(t, host)
			device.install(todo100, todoData)
			updater := device.updater()
			updater.afterSwapStep = func(reached string) error {
				if reached == step {
					return errPowerCut
				}
				return nil
			}
			if _, err := updater.ApplyApp(context.Background(), "todo"); !errors.Is(err, errPowerCut) {
				t.Fatalf("err = %v", err)
			}
			if err := RecoverApps(device.layout()); err != nil {
				t.Fatalf("recovery: %v", err)
			}
			assertAppIntact(t, device, "todo", todoData)
		})
	}
}

func TestPowerCutWhileUnpackingLeavesTheAppUntouched(t *testing.T) {
	host := newAppsHost(t)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	writeFile(t, device.root, ".adds/apps/.todo.new/app.json", "half written", 0o644)
	if err := RecoverApps(device.layout()); err != nil {
		t.Fatal(err)
	}
	assertAppIntact(t, device, "todo", todoData)
	if device.installedVersion("todo") != "1.0.0" {
		t.Errorf("version = %s", device.installedVersion("todo"))
	}
}

func TestRecoveryNeverDeletesAFolderThatHoldsData(t *testing.T) {
	host := newAppsHost(t)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	// A staged folder with data while the app has some too: both stay, and the problem is reported.
	writeFile(t, device.root, ".adds/apps/.todo.new/data/precious", "x", 0o600)
	if err := RecoverApps(device.layout()); err == nil {
		t.Error("two data folders were not reported")
	}
	if !reflect.DeepEqual(device.dataOf("todo"), todoData) {
		t.Error("the installed data changed")
	}
	if _, err := os.Stat(filepath.Join(device.root, ".adds/apps/.todo.new/data/precious")); err != nil {
		t.Error("a folder holding data was deleted")
	}
}

func TestRecoveryRestoresAnAppWhoseNewVersionIsBroken(t *testing.T) {
	host := newAppsHost(t)
	device := newAppsDevice(t, host)
	device.install(todo100, todoData)
	// The state after the new folder was renamed in place but never validated: broken app.json.
	if err := os.Rename(filepath.Join(device.root, ".adds/apps/todo"), filepath.Join(device.root, ".adds/apps/.todo.old")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, device.root, ".adds/apps/todo/app.json", `{"broken":true}`, 0o644)
	if err := os.Rename(filepath.Join(device.root, ".adds/apps/.todo.old/data"), filepath.Join(device.root, ".adds/apps/todo/data")); err != nil {
		t.Fatal(err)
	}
	if err := RecoverApps(device.layout()); err != nil {
		t.Fatal(err)
	}
	assertAppIntact(t, device, "todo", todoData)
	if device.installedVersion("todo") != "1.0.0" {
		t.Errorf("version = %s", device.installedVersion("todo"))
	}
}

// assertAppIntact checks the app loads, its data is exactly want, and nothing is left over.
func assertAppIntact(t *testing.T, device *appsDevice, id string, want map[string]string) {
	t.Helper()
	if _, _, err := manifest.Load(filepath.Join(device.root, ".adds/apps", id)); err != nil {
		t.Errorf("the app does not load: %v", err)
	}
	if got := device.dataOf(id); !reflect.DeepEqual(got, want) {
		t.Errorf("data changed:\nwant %v\ngot  %v", want, got)
	}
	if names := device.leftovers(); len(names) != 0 {
		t.Errorf("leftover folders: %v", names)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/inbox"
	"koboapps/internal/storage"
)

const (
	shippedRoutingFile = "../../host/config/routing.json"
	pdfContent         = "%PDF-1.7 test"
)

// inboxFixture is a fake Kobo with the shipped rules and host locales installed.
type inboxFixture struct {
	t    *testing.T
	root string
}

func newInboxFixture(t *testing.T) *inboxFixture {
	t.Helper()
	root := t.TempDir()
	hostDir := filepath.Join(root, ".adds", "koboapps")
	copyHostLocales(t, hostDir)
	rules, err := os.ReadFile(shippedRoutingFile)
	if err != nil {
		t.Fatal(err)
	}
	writeLocaleFile(t, filepath.Join(hostDir, "config"), "routing.json", rules)
	writeLocaleFile(t, filepath.Join(hostDir, "inbox"), ".keep", nil)
	return &inboxFixture{t: t, root: root}
}

func (f *inboxFixture) inboxDir() string { return filepath.Join(f.root, ".adds", "koboapps", "inbox") }

func (f *inboxFixture) receive(name, content string) string {
	f.t.Helper()
	writeLocaleFile(f.t, f.inboxDir(), name, []byte(content))
	return filepath.Join(f.inboxDir(), name)
}

func (f *inboxFixture) inboxCommand(locale string, args ...string) (code int, stdout, stderr string) {
	f.t.Helper()
	full := append([]string{"inbox", args[0], "--onboard-root", f.root, "--locale", locale}, args[1:]...)
	var out, errOut bytes.Buffer
	code = run(full, &out, &errOut)
	return code, out.String(), errOut.String()
}

func (f *inboxFixture) exists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(append([]string{f.root}, parts...)...))
	return err == nil
}

func TestInboxRouteMovesAFileAndSaysSoInTheDeviceLanguage(t *testing.T) {
	f := newInboxFixture(t)
	path := f.receive("Sách hay.pdf", pdfContent)
	code, stdout, stderr := f.inboxCommand("vi", "route", "--source", "usb", path)
	if code != exitOK || stderr != "" {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if !strings.Contains(stdout, "Sách hay.pdf -> Tài liệu") || !f.exists("Documents", "Sách hay.pdf") {
		t.Errorf("stdout %q, documents holds %v", stdout, f.exists("Documents", "Sách hay.pdf"))
	}
	entries, _, _ := inbox.ReadEntries(filepath.Join(f.root, ".adds", "koboapps", "data", "inbox-journal.jsonl"))
	if len(entries) != 1 || entries[0].Source != "usb" {
		t.Errorf("journal = %+v", entries)
	}
}

func TestInboxRouteReportsRejectionsWithAReason(t *testing.T) {
	f := newInboxFixture(t)
	path := f.receive("tool.exe", "MZ")
	code, stdout, _ := f.inboxCommand("en", "route", path)
	if code != exitOK || !strings.Contains(stdout, "Rejected tool.exe: type not allowed") || !f.exists(".adds", "koboapps", "inbox", "rejected", "tool.exe") {
		t.Errorf("code %d stdout %q", code, stdout)
	}
}

func TestInboxRouteFromEnvReadsOnlyLocalgoFile(t *testing.T) {
	f := newInboxFixture(t)
	wanted := f.receive("wanted.pdf", pdfContent)
	decoy := f.receive("decoy.pdf", pdfContent)
	t.Setenv("LOCALGO_FILE", wanted)
	t.Setenv("LOCALGO_NAME", "decoy.pdf")
	t.Setenv("LOCALGO_ALIAS", "$(touch pwned)")

	code, _, stderr := f.inboxCommand("en", "route", "--from-env", "--source", "localsend")
	if code != exitOK {
		t.Fatalf("code %d stderr %q", code, stderr)
	}
	if !f.exists("Documents", "wanted.pdf") {
		t.Error("the file named by LOCALGO_FILE must be routed")
	}
	if _, err := os.Stat(decoy); err != nil {
		t.Error("no other file may be touched")
	}

	code, _, stderr = f.inboxCommand("en", "route", "--from-env", decoy)
	if code == exitOK || !strings.Contains(stderr, "LOCALGO_FILE") {
		t.Errorf("--from-env plus a path argument must fail, got %d %q", code, stderr)
	}
	t.Setenv("LOCALGO_FILE", "")
	if code, _, stderr = f.inboxCommand("en", "route", "--from-env"); code == exitOK || !strings.Contains(stderr, "LOCALGO_FILE") {
		t.Errorf("an empty LOCALGO_FILE must fail, got %d %q", code, stderr)
	}
	if code, _, _ = f.inboxCommand("en", "route"); code == exitOK {
		t.Error("route without a path must fail")
	}
}

func TestInboxRouteRefusesBadInput(t *testing.T) {
	f := newInboxFixture(t)
	outside := filepath.Join(t.TempDir(), "x.pdf")
	if err := os.WriteFile(outside, []byte(pdfContent), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := f.inboxCommand("en", "route", outside)
	if code == exitOK || !strings.Contains(stderr, "not in the inbox") {
		t.Errorf("outside path: %d %q", code, stderr)
	}
	if code, _, _ = f.inboxCommand("en", "route", "--source", "Bad Name;rm", f.receive("a.pdf", pdfContent)); code == exitOK {
		t.Error("a source name with spaces or shell characters must be refused")
	}
	if code, _, _ = f.inboxCommand("en", "route", "<clipboard>"); code == exitOK {
		t.Error("the clipboard marker is not a file")
	}
}

func TestInboxRouteWithoutShippedRulesFailsClosed(t *testing.T) {
	f := newInboxFixture(t)
	path := f.receive("a.pdf", pdfContent)
	if err := os.Remove(filepath.Join(f.root, ".adds", "koboapps", "config", "routing.json")); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := f.inboxCommand("en", "route", path)
	if code == exitOK || !strings.Contains(stderr, "Routing rules unusable") || f.exists("Documents", "a.pdf") {
		t.Errorf("code %d stderr %q", code, stderr)
	}
}

func TestInboxBrokenOverrideWarnsAndKeepsDefaults(t *testing.T) {
	f := newInboxFixture(t)
	writeLocaleFile(t, filepath.Join(f.root, ".adds", "koboapps", "data"), "routing.json", []byte(`{"destinations":{"pdf":"/etc"}}`))
	code, _, stderr := f.inboxCommand("en", "route", f.receive("a.pdf", pdfContent))
	if code != exitOK || !strings.Contains(stderr, "Using the default rules") || !f.exists("Documents", "a.pdf") {
		t.Errorf("code %d stderr %q", code, stderr)
	}
}

func TestInboxOverrideRemapsADestination(t *testing.T) {
	f := newInboxFixture(t)
	writeLocaleFile(t, filepath.Join(f.root, ".adds", "koboapps", "data"), "routing.json", []byte(`{"remove":["gif"],"destinations":{"pdf":"books"}}`))
	f.inboxCommand("en", "route", f.receive("a.pdf", pdfContent))
	_, stdout, _ := f.inboxCommand("en", "route", f.receive("b.gif", "GIF89a.."))
	if !f.exists("Books", "a.pdf") || !strings.Contains(stdout, "type not allowed") {
		t.Errorf("remap or removal not applied: %q", stdout)
	}
}

func TestInboxSortMigratesTheLegacyFolderAndNothingElse(t *testing.T) {
	f := newInboxFixture(t)
	writeLocaleFile(t, filepath.Join(f.root, "LocalSend"), "old.pdf", []byte(pdfContent))
	writeLocaleFile(t, filepath.Join(f.root, "LocalSend"), "old.exe", []byte("MZ"))
	writeLocaleFile(t, filepath.Join(f.root, "Books"), "mine.pdf", []byte(pdfContent))
	other := filepath.Join(t.TempDir(), "x")
	writeLocaleFile(t, other, "z.pdf", []byte(pdfContent))

	code, stdout, stderr := f.inboxCommand("en", "sort", "--source-dir", filepath.Join(f.root, "LocalSend"), "--source", "localsend")
	if code != exitOK || !strings.Contains(stdout, "Sorted 2 files, 1 rejected.") {
		t.Fatalf("code %d stdout %q stderr %q", code, stdout, stderr)
	}
	if !f.exists("Documents", "old.pdf") || !f.exists(".adds", "koboapps", "inbox", "rejected", "old.exe") {
		t.Error("legacy files were not routed")
	}
	for _, forbidden := range []string{filepath.Join(f.root, "Books"), other, "/"} {
		if code, _, _ = f.inboxCommand("en", "sort", "--source-dir", forbidden); code == exitOK {
			t.Errorf("sorting %s must be refused", forbidden)
		}
	}
	if _, err := os.Stat(filepath.Join(other, "z.pdf")); err != nil {
		t.Error("a folder that is not an allowed source must be untouched")
	}
}

func TestInboxSortDefaultsToTheInbox(t *testing.T) {
	f := newInboxFixture(t)
	f.receive("a.pdf", pdfContent)
	f.receive("b.png", "\x89PNG\r\n\x1a\nxx")
	code, stdout, _ := f.inboxCommand("vi", "sort", "--source", "localsend")
	if code != exitOK || !strings.Contains(stdout, "Đã sắp xếp 2 tệp, từ chối 0.") || !f.exists("Pictures", "b.png") {
		t.Errorf("code %d stdout %q", code, stdout)
	}
}

func TestInboxStatusSummarizesTheLastSessionInVietnamese(t *testing.T) {
	f := newInboxFixture(t)
	if _, stdout, _ := f.inboxCommand("vi", "status"); strings.TrimSpace(stdout) != "Chưa nhận tệp nào." {
		t.Errorf("empty status = %q", stdout)
	}
	for _, name := range []string{"a.pdf", "b.pdf", "c.png"} {
		content := pdfContent
		if strings.HasSuffix(name, ".png") {
			content = "\x89PNG\r\n\x1a\nxx"
		}
		f.inboxCommand("vi", "route", f.receive(name, content))
	}
	for _, name := range []string{"x.exe", "y.exe", "z.sh", "big.txt"} {
		content := "x"
		if name == "big.txt" {
			content = "bad\x00text"
		}
		f.inboxCommand("vi", "route", f.receive(name, content))
	}
	code, stdout, _ := f.inboxCommand("vi", "status")
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	want := []string{"Tài liệu: 2", "Ảnh: 1", "Bị từ chối: 4", "  loại tệp không được phép: 3", "  nội dung không khớp tên tệp: 1"}
	if code != exitOK || strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("status lines = %q", lines)
	}
	if len(lines) > 8 {
		t.Errorf("status must stay within 8 lines, got %d", len(lines))
	}
	_, english, _ := f.inboxCommand("en", "status")
	if !strings.Contains(english, "Documents: 2") || !strings.Contains(english, "type not allowed: 3") {
		t.Errorf("english status = %q", english)
	}
}

func TestInboxStatusNeverExceedsEightLines(t *testing.T) {
	session := inboxSession{}
	session.bundle = i18n.NewBundle(i18n.Catalog{})
	summary := inbox.Summary{Rejected: 9, Routed: map[storage.Folder]int{"books": 1, "documents": 1, "comics": 1, "pictures": 1}}
	for _, reason := range inbox.AllReasons {
		summary.Reasons = append(summary.Reasons, inbox.ReasonCount{Reason: reason, Count: 1})
	}
	if lines := strings.Count(session.formatSummary(summary), "\n"); lines > 8 {
		t.Errorf("%d lines", lines)
	}
}

func TestInboxPurgeDeletesOnlyOldRejectedFiles(t *testing.T) {
	f := newInboxFixture(t)
	rejected := filepath.Join(f.inboxDir(), "rejected")
	old := filepath.Join(rejected, "old.exe")
	fresh := filepath.Join(rejected, "fresh.exe")
	staged := f.receive("old-but-staged.pdf", pdfContent)
	writeLocaleFile(t, rejected, "old.exe", []byte("x"))
	writeLocaleFile(t, rejected, "fresh.exe", []byte("x"))
	longAgo := time.Now().Add(-8 * 24 * time.Hour)
	for _, path := range []string{old, staged} {
		if err := os.Chtimes(path, longAgo, longAgo); err != nil {
			t.Fatal(err)
		}
	}
	code, stdout, _ := f.inboxCommand("en", "purge")
	if code != exitOK || !strings.Contains(stdout, "Deleted 1 old rejected files.") {
		t.Fatalf("code %d stdout %q", code, stdout)
	}
	for path, wantGone := range map[string]bool{old: true, fresh: false, staged: false} {
		if _, err := os.Stat(path); os.IsNotExist(err) != wantGone {
			t.Errorf("%s: gone=%v, want %v", path, os.IsNotExist(err), wantGone)
		}
	}
}

func TestInboxUnknownSubcommandShowsUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"inbox", "frobnicate"}, &out, &errOut); code == exitOK || !strings.Contains(errOut.String(), "usage:") {
		t.Errorf("code %d %q", code, errOut.String())
	}
	if code := run([]string{"inbox"}, &out, &errOut); code == exitOK {
		t.Error("inbox alone must show usage")
	}
}

// Every message the inbox commands can print exists in both shipped languages.
func TestInboxMessagesExistInEnglishAndVietnamese(t *testing.T) {
	keys := []string{"inbox.routed_file", "inbox.rejected_file", "inbox.already_handled", "inbox.status_empty",
		"inbox.sorted", "inbox.purged", "inbox.warn_override", "inbox.err_outside", "inbox.err_folder",
		"inbox.err_rules", "inbox.err_failed", "inbox.reason_line", "inbox.count.rejected", "inbox.folder.rejected"}
	for _, folder := range summaryFolders {
		keys = append(keys, "inbox.folder."+string(folder), "inbox.count."+string(folder))
	}
	for _, reason := range inbox.AllReasons {
		keys = append(keys, "inbox.reason."+string(reason))
	}
	for _, language := range []string{"en", "vi"} {
		catalog, err := i18n.LoadCatalog(filepath.Join("..", "..", "host", "locales", language+".json"))
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			if catalog[key] == "" {
				t.Errorf("%s.json lacks %s", language, key)
			}
		}
	}
}

// localgoPlaceholder matches the substitutions localgo makes in a hook: %f %n %s %a %i.
var localgoPlaceholder = regexp.MustCompile(`%[fnsai]`)

var hookPattern = regexp.MustCompile(`(?m)^ROUTE_HOOK="([^"]*)"$`)

// The LocalSend hook is a shell string into which localgo pastes %-placeholders, and file names
// and sender aliases are chosen by the sender. It must not contain one.
func TestLocalSendHookHasNoPlaceholdersAndRoutesFromTheEnvironment(t *testing.T) {
	env, err := os.ReadFile("../../apps/localsend/bin/env.sh")
	if err != nil {
		t.Fatal(err)
	}
	match := hookPattern.FindSubmatch(env)
	if match == nil {
		t.Fatal("env.sh must define ROUTE_HOOK=\"...\"")
	}
	// APPSCTL is the one variable allowed: it is set from KOBOAPPS_ROOT, never from the sender.
	hook := strings.Replace(string(match[1]), "$APPSCTL", "/mnt/onboard/.adds/koboapps/bin/appsctl", 1)
	if strings.Contains(hook, "%") {
		t.Fatalf("hook %q contains a %%-placeholder", hook)
	}
	for _, metacharacter := range []string{"$", "`", ";", "&", "|", ">", "<", "\""} {
		if strings.Contains(hook, metacharacter) {
			t.Errorf("hook %q contains shell metacharacter %q", hook, metacharacter)
		}
	}
	const wantHook = "/mnt/onboard/.adds/koboapps/bin/appsctl inbox route --from-env --source localsend"
	if hook != wantHook {
		t.Errorf("hook = %q, want %q", hook, wantHook)
	}
	start, err := os.ReadFile("../../apps/localsend/bin/start.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(start), `--exec "$ROUTE_HOOK"`) || localgoPlaceholder.Match(start) {
		t.Error("start.sh must pass --exec \"$ROUTE_HOOK\" and use no localgo placeholder")
	}
	if !strings.Contains(string(start), `--dir "$INBOX"`) || !strings.Contains(string(env), "INBOX=/mnt/onboard/.adds/koboapps/inbox") {
		t.Error("localgo must receive into the inbox folder")
	}
}

// The real hook: arguments from the hook string, the path only from the environment.
func TestLocalSendHookRunsEndToEnd(t *testing.T) {
	f := newInboxFixture(t)
	path := f.receive("hook.pdf", pdfContent)
	t.Setenv("LOCALGO_FILE", path)
	args := strings.Fields(strings.TrimPrefix(
		"/mnt/onboard/.adds/koboapps/bin/appsctl inbox route --from-env --source localsend", "/mnt/onboard/.adds/koboapps/bin/appsctl "))
	args = append(args, "--onboard-root", f.root, "--locale", "en")
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != exitOK || !f.exists("Documents", "hook.pdf") {
		t.Errorf("code %d %q %q", code, out.String(), errOut.String())
	}
}

// buildAppsctl compiles the real binary so concurrent runs are separate OS processes.
func buildAppsctl(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "appsctl")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	return binary
}

func TestTwentyParallelRouteProcessesLoseNoFileAndKeepTheJournalIntact(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	f := newInboxFixture(t)
	binary := buildAppsctl(t)
	const count = 20
	var paths []string
	for index := 0; index < count; index++ {
		paths = append(paths, f.receive(fmt.Sprintf("distinct-%d.pdf", index), fmt.Sprintf("%s distinct %d", pdfContent, index)))
		// All of these sanitize to "same.pdf": the router must hand out (2), (3), ... without a clash.
		paths = append(paths, f.receive(strings.Repeat(" ", index)+"same.pdf", fmt.Sprintf("%s same %d", pdfContent, index)))
	}
	var group sync.WaitGroup
	results := make(chan string, len(paths))
	for _, path := range paths {
		group.Add(1)
		go func() {
			defer group.Done()
			command := exec.Command(binary, "inbox", "route", "--from-env", "--source", "localsend",
				"--onboard-root", f.root, "--locale", "en")
			command.Env = append(os.Environ(), "LOCALGO_FILE="+path)
			if output, err := command.CombinedOutput(); err != nil {
				results <- fmt.Sprintf("%s: %v %s", path, err, output)
			}
		}()
	}
	group.Wait()
	close(results)
	for failure := range results {
		t.Error(failure)
	}

	documents, err := os.ReadDir(filepath.Join(f.root, "Documents"))
	if err != nil || len(documents) != 2*count {
		t.Fatalf("documents hold %d files (want %d), err %v", len(documents), 2*count, err)
	}
	contents := map[string]bool{}
	for _, entry := range documents {
		data, _ := os.ReadFile(filepath.Join(f.root, "Documents", entry.Name()))
		contents[string(data)] = true
	}
	if len(contents) != 2*count {
		t.Errorf("%d distinct contents survived, want %d", len(contents), 2*count)
	}
	entries, skipped, err := inbox.ReadEntries(filepath.Join(f.root, ".adds", "koboapps", "data", "inbox-journal.jsonl"))
	if err != nil || skipped != 0 || len(entries) != 2*count {
		t.Errorf("journal has %d entries, %d corrupt lines, err %v", len(entries), skipped, err)
	}
}

func (f *inboxFixture) stateJSON(locale string) (summary string, received [][2]string) {
	f.t.Helper()
	code, stdout, stderr := f.inboxCommand(locale, "state")
	if code != exitOK || strings.Count(strings.TrimSpace(stdout), "\n") != 0 {
		f.t.Fatalf("state exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
	var parsed struct {
		Summary  string      `json:"summary"`
		Received [][2]string `json:"received"`
	}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		f.t.Fatalf("state is not one JSON object: %v\n%s", err, stdout)
	}
	return parsed.Summary, parsed.Received
}

func TestInboxStateWithEmptyJournalPrintsTheEmptyScreen(t *testing.T) {
	f := newInboxFixture(t)
	_, stdout, _ := f.inboxCommand("en", "state")
	if strings.TrimSpace(stdout) != `{"summary":"","received":[]}` {
		t.Errorf("state = %q", stdout)
	}
}

func TestInboxStateListsTheMixedSessionInVietnamese(t *testing.T) {
	f := newInboxFixture(t)
	f.inboxCommand("vi", "route", f.receive("a.pdf", pdfContent))
	f.inboxCommand("vi", "route", f.receive("b.pdf", pdfContent))
	summary, received := f.stateJSON("vi")
	if summary != "Tài liệu 2" || len(received) != 2 || received[0][0] != "b.pdf" ||
		!strings.HasPrefix(received[0][1], "Tài liệu · ") {
		t.Errorf("summary %q, received %v", summary, received)
	}
}

func TestInboxStateCountsRejectedFilesButDoesNotListThem(t *testing.T) {
	f := newInboxFixture(t)
	f.inboxCommand("en", "route", f.receive("a.pdf", pdfContent))
	f.inboxCommand("en", "route", f.receive("x.exe", "x"))
	summary, received := f.stateJSON("en")
	if summary != "Documents 1 · Rejected 1" || len(received) != 1 || received[0][0] != "a.pdf" {
		t.Errorf("summary %q, received %v", summary, received)
	}
}

func TestInboxStateListsAtMostTenRows(t *testing.T) {
	f := newInboxFixture(t)
	for index := 0; index < 13; index++ {
		f.inboxCommand("en", "route", f.receive(fmt.Sprintf("d%02d.pdf", index), pdfContent))
	}
	summary, received := f.stateJSON("en")
	if summary != "Documents 13" || len(received) != 10 {
		t.Errorf("summary %q, %d rows", summary, len(received))
	}
}

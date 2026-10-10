package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"koboapps/internal/i18n"
)

func lastCheckOf(t *testing.T, f *fixture) CheckRecord {
	t.Helper()
	record, ok := ReadLastCheck(Layout{OnboardRoot: f.root})
	if !ok {
		t.Fatal("the check must record its outcome in last-check.json")
	}
	if _, err := time.Parse(time.RFC3339, record.Time); err != nil {
		t.Fatalf("time %q is not RFC 3339", record.Time)
	}
	return record
}

func TestCheckRecordsUpToDateAndAvailable(t *testing.T) {
	f := newFixture(t, "1.0.0")
	f.publishVersion("1.1.0", "1.1.0")
	if _, err := f.updater("1.1.0").Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if record := lastCheckOf(t, f); record.Result != CheckUpToDate || record.Version != "1.1.0" {
		t.Fatalf("record = %+v", record)
	}
	if _, err := f.updater("1.0.0").Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if record := lastCheckOf(t, f); record.Result != CheckAvailable || record.Version != "1.1.0" {
		t.Fatalf("record = %+v", record)
	}
}

func TestCheckRecordsEachFailureKind(t *testing.T) {
	rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer rateLimited.Close()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer broken.Close()
	gone := httptest.NewServer(http.NotFoundHandler())
	goneURL := gone.URL
	gone.Close() // connection refused: nothing listens any more

	cases := map[string]struct {
		apiBase string
		prepare func(*fixture)
		want    string
	}{
		"no network":     {apiBase: goneURL, want: ErrorNoNetwork},
		"unreachable":    {apiBase: broken.URL, want: ErrorUnreachable},
		"rate limited":   {apiBase: rateLimited.URL, want: ErrorRateLimited},
		"no release":     {prepare: func(*fixture) {}, want: ErrorNoRelease},
		"bad signature":  {prepare: publishSignedByAnotherKey, want: ErrorBadSignature},
		"something else": {prepare: removeUpdateConfig, want: ErrorOther},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "1.0.0")
			if tc.prepare != nil {
				tc.prepare(f)
			}
			updater := f.updater("1.0.0")
			if tc.apiBase != "" {
				updater.github = newGithubClient(nil, tc.apiBase)
			}
			if _, err := updater.Check(context.Background()); err == nil {
				t.Fatal("the check must fail")
			}
			record := lastCheckOf(t, f)
			if record.Result != CheckFailed || record.ErrorKey != tc.want {
				t.Fatalf("record = %+v, want error_key %s", record, tc.want)
			}
			if (record.Report != nil) != (tc.want == ErrorOther) {
				t.Fatalf("only %q keeps its own report: %+v", ErrorOther, record)
			}
		})
	}
}

func publishSignedByAnotherKey(f *fixture) {
	_, otherKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		f.t.Fatal(err)
	}
	dist := f.t.TempDir()
	writeVersionTree(f.t, dist, "1.1.0", "1.1.0")
	f.github.publishStable("1.1.0", buildRelease(f.t, dist, "1.1.0", otherKey, ""))
}

func removeUpdateConfig(f *fixture) {
	if err := os.Remove(filepath.Join(f.root, ".adds", "koboapps", "data", "update.json")); err != nil {
		f.t.Fatal(err)
	}
}

func TestCheckKeepsTheEarlierInstallStatus(t *testing.T) {
	f := newFixture(t, "1.0.0")
	layout := Layout{OnboardRoot: f.root}
	if err := writeStatus(layout, Status{State: StateDone, Report: report("update.applied_reboot", "version", "1.1.0")}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.updater("1.0.0").Check(context.Background()); err == nil {
		t.Fatal("no release is published, so the check fails")
	}
	status, _, _ := ReadStatus(layout)
	if status.State != StateDone || status.Report.Key != "update.applied_reboot" {
		t.Fatalf("a failed check must not replace the install result: %+v", status)
	}
}

func TestStatusOfARunningCheckIsTaggedAsCheck(t *testing.T) {
	f := newFixture(t, "1.0.0")
	layout := Layout{OnboardRoot: f.root}
	updater := f.updater("1.0.0")
	if _, _, err := updater.beginCheck(); err != nil {
		t.Fatal(err)
	}
	status, _, _ := ReadStatus(layout)
	if status.State != StateRunning || status.Job != JobCheck || status.Report.Key != "update.checking" {
		t.Fatalf("status = %+v", status)
	}
}

func TestDescribeRendersEveryOutcomeInBothLanguages(t *testing.T) {
	when := time.Date(2026, 10, 10, 10, 2, 0, 0, time.UTC)
	clock := when.Local().Format("15:04")
	cases := []struct {
		record  CheckRecord
		current string
		want    map[string]string
	}{
		{CheckRecord{Result: CheckUpToDate, Version: "1.0.0"}, "1.0.0", map[string]string{
			"en": "Checked at " + clock + " · You are on the latest version (1.0.0)",
			"vi": "Đã kiểm tra lúc " + clock + " · Bạn đang dùng bản mới nhất (1.0.0)"}},
		{CheckRecord{Result: CheckAvailable, Version: "1.0.1"}, "1.0.0", map[string]string{
			"en": "Checked at " + clock + " · New version 1.0.1 available",
			"vi": "Đã kiểm tra lúc " + clock + " · Có bản mới 1.0.1"}},
		{CheckRecord{Result: CheckAvailable, Version: "1.0.0"}, "1.0.0", map[string]string{"en": "", "vi": ""}},
		{CheckRecord{Result: CheckFailed, ErrorKey: ErrorNoNetwork}, "1.0.0", map[string]string{
			"en": "Could not check at " + clock + ": no Wi-Fi",
			"vi": "Không kiểm tra được lúc " + clock + ": chưa có Wi-Fi"}},
		{CheckRecord{Result: CheckFailed, ErrorKey: ErrorUnreachable}, "1.0.0", map[string]string{
			"en": "Could not check at " + clock + ": cannot reach GitHub",
			"vi": "Không kiểm tra được lúc " + clock + ": không kết nối được GitHub"}},
		{CheckRecord{Result: CheckFailed, ErrorKey: ErrorRateLimited}, "1.0.0", map[string]string{
			"en": "Could not check at " + clock + ": GitHub is limiting requests, try later",
			"vi": "Không kiểm tra được lúc " + clock + ": GitHub đang giới hạn, thử lại sau"}},
		{CheckRecord{Result: CheckFailed, ErrorKey: ErrorBadSignature}, "1.0.0", map[string]string{
			"en": "Could not check at " + clock + ": bad signature, nothing changed",
			"vi": "Không kiểm tra được lúc " + clock + ": chữ ký sai, không có gì thay đổi"}},
		{CheckRecord{Result: CheckFailed, ErrorKey: ErrorNoRelease}, "1.0.0", map[string]string{
			"en": "Could not check at " + clock + ": no release on this channel",
			"vi": "Không kiểm tra được lúc " + clock + ": kênh này chưa có bản phát hành"}},
		{CheckRecord{Result: CheckFailed, ErrorKey: ErrorOther, Report: &Report{Key: "update.no_repo", Params: map[string]string{"file": "update.json"}}},
			"1.0.0", map[string]string{
				"en": "Could not check at " + clock + ": No update repo set. Put owner/name as \"repo\" in update.json",
				"vi": "Không kiểm tra được lúc " + clock + ": Chưa đặt kho cập nhật. Ghi chủ/tên vào \"repo\" trong update.json"}},
	}
	for _, tc := range cases {
		tc.record.Time = when.Format(time.RFC3339)
		for locale, want := range tc.want {
			bundle, err := i18n.LoadBundle(filepath.Join("..", "..", "host", "locales"), locale)
			if err != nil {
				t.Fatal(err)
			}
			if got := tc.record.Describe(bundle, tc.current); got != want {
				t.Errorf("%s %+v:\n got %q\nwant %q", locale, tc.record, got, want)
			}
		}
	}
}

func TestOtherReasonIsShortAndOnOneLine(t *testing.T) {
	bundle, err := i18n.LoadBundle(filepath.Join("..", "..", "host", "locales"), "en")
	if err != nil {
		t.Fatal(err)
	}
	long := &Report{Key: "update.failed", Params: map[string]string{"cause": strings.Repeat("word\nword ", 40)}}
	record := CheckRecord{Time: time.Now().UTC().Format(time.RFC3339), Result: CheckFailed, ErrorKey: ErrorOther, Report: long}
	reason := record.reason(bundle)
	if strings.Contains(reason, "\n") || len([]rune(reason)) > maxOtherReasonSize {
		t.Fatalf("reason = %q", reason)
	}
}

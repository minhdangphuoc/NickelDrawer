package i18n

import (
	"testing"
	"time"
)

func TestFormatPatternWritesTheLocalesOrder(t *testing.T) {
	moment := time.Date(2026, time.October, 9, 8, 5, 0, 0, time.Local)
	cases := map[string]string{
		"dd/MM/yyyy HH:mm": "09/10/2026 08:05",
		"yyyy-MM-dd HH:mm": "2026-10-09 08:05",
		"MM/dd/yyyy":       "10/09/2026",
		"HH:mm":            "08:05",
		"dd.MM. um HH.mm":  "09.10. um 08.05",
		"no tokens":        "no tokens",
	}
	for pattern, want := range cases {
		if got := FormatPattern(pattern, moment); got != want {
			t.Errorf("FormatPattern(%q) = %q, want %q", pattern, got, want)
		}
	}
}

func TestBundleDateTimeUsesTheLocaleKeyOrTheDayFirstDefault(t *testing.T) {
	moment := time.Date(2026, time.October, 9, 20, 15, 0, 0, time.Local)
	plain := NewBundle(Catalog{})
	if got := plain.DateTime(moment); got != "09/10/2026 20:15" {
		t.Errorf("default date time = %q", got)
	}
	if got := plain.Clock(moment); got != "20:15" {
		t.Errorf("default clock = %q", got)
	}
	custom := NewBundle(Catalog{DateTimeKey: "yyyy-MM-dd HH:mm", TimeKey: "HH.mm"})
	if got := custom.DateTime(moment); got != "2026-10-09 20:15" {
		t.Errorf("custom date time = %q", got)
	}
	if got := custom.Clock(moment); got != "20.15" {
		t.Errorf("custom clock = %q", got)
	}
}

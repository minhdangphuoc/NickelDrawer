package i18n

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentLocale(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "Kobo eReader.conf")
	cases := []struct{ name, content, want string }{
		{"in section", "[Other]\nCurrentLocale=de\n[ApplicationPreferences]\nFoo=1\nCurrentLocale=pt_BR\n", "pt_BR"},
		{"wrong section only", "[Other]\nCurrentLocale=de\n", "en"},
		{"no key", "[ApplicationPreferences]\nFoo=1\n", "en"},
		{"hyphen and spaces", "[ApplicationPreferences]\nCurrentLocale = vi-VN \r\n", "vi_VN"},
		{"path traversal rejected", "[ApplicationPreferences]\nCurrentLocale=../../etc\n", "en"},
		{"empty value", "[ApplicationPreferences]\nCurrentLocale=\n", "en"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeFile(t, conf, c.content)
			if got := CurrentLocale(conf); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
	if got := CurrentLocale(filepath.Join(dir, "missing.conf")); got != "en" {
		t.Fatalf("missing conf gave %q", got)
	}
}

func TestChain(t *testing.T) {
	cases := map[string][]string{
		"pt_BR": {"pt_BR", "pt", "en"},
		"vi":    {"vi", "en"},
		"en":    {"en"},
		"en_US": {"en_US", "en"},
		"":      {"en"},
	}
	for locale, want := range cases {
		if got := Chain(locale); !reflect.DeepEqual(got, want) {
			t.Errorf("Chain(%q) = %v, want %v", locale, got, want)
		}
	}
}

func TestBundleFallsBackKeyByKey(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "en.json"), `{"a":"A en","b":"B en {x}","c":"C en"}`)
	writeFile(t, filepath.Join(dir, "pt.json"), `{"a":"A pt","b":"B pt {x}"}`)
	writeFile(t, filepath.Join(dir, "pt_BR.json"), `{"a":"A pt_BR"}`)
	bundle, err := LoadBundle(dir, "pt_BR")
	if err != nil {
		t.Fatal(err)
	}
	params := map[string]string{"x": "1"}
	want := map[string]string{"a": "A pt_BR", "b": "B pt 1", "c": "C en", "zzz": "zzz"}
	for key, text := range want {
		if got := bundle.Message(key, params); got != text {
			t.Errorf("%s = %q, want %q", key, got, text)
		}
	}
}

func TestLoadBundleReportsBrokenFileButStaysUsable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "en.json"), `{"a":"A en"}`)
	writeFile(t, filepath.Join(dir, "fr.json"), `{not json`)
	bundle, err := LoadBundle(dir, "fr")
	if err == nil {
		t.Fatal("want an error naming fr.json")
	}
	if got := bundle.Message("a", nil); got != "A en" {
		t.Fatalf("got %q", got)
	}
	if _, err := LoadBundle(filepath.Join(dir, "none"), "en"); err != nil {
		t.Fatalf("missing folder is not an error: %v", err)
	}
}

func TestSubstituteAndNewlines(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "en.json"), `{"k":"IP: {ip}\nPIN: {pin} {unknown}"}`)
	bundle, _ := LoadBundle(dir, "en")
	got := bundle.Message("k", Params("ip", "10.0.0.2", "pin", "1234"))
	if want := "IP: 10.0.0.2\nPIN: 1234 {unknown}"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if (*Bundle)(nil).Message("k", nil) != "k" {
		t.Fatal("nil bundle must return the key")
	}
}

func TestCheckPlaceholders(t *testing.T) {
	if problem := CheckPlaceholders("{a} and {b} {a}", "{b} va {a}"); problem != "" {
		t.Fatalf("same set reported: %s", problem)
	}
	for _, translated := range []string{"{a}", "{a} {b} {c}", "{a} {x}", "none"} {
		if CheckPlaceholders("{a} and {b}", translated) == "" {
			t.Errorf("%q should differ from English", translated)
		}
	}
}

func TestDescribeMessageError(t *testing.T) {
	bundle := NewBundle(Catalog{"err.read": "Cannot read {path}: {cause}", "err.inner": "disk full"})
	err := NewError("err.read", "path", "/x").WithCause(NewError("err.inner"))
	if got := bundle.Describe(err); got != "Cannot read /x: disk full" {
		t.Fatalf("got %q", got)
	}
	wrapped := errors.Join(errors.New("other"), err)
	if got := bundle.Describe(wrapped); got != "Cannot read /x: disk full" {
		t.Fatalf("wrapped: %q", got)
	}
	if got := err.Error(); got != "err.read path=/x: err.inner" {
		t.Fatalf("Error() = %q", got)
	}
	if got := bundle.Describe(errors.New("plain")); got != "plain" {
		t.Fatalf("plain: %q", got)
	}
}

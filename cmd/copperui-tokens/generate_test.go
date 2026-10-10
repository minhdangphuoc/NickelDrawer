package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const repoRoot = "../.."

func TestGeneratedFilesAreCurrent(t *testing.T) {
	generated, err := generateAll(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if stale := staleFiles(repoRoot, generated); len(stale) > 0 {
		t.Fatalf("stale generated files %v: run `make tokens` after editing the token file or the text vectors", stale)
	}
}

func tokenValues(t *testing.T) map[string]int {
	t.Helper()
	spec, err := loadSpec(repoRoot + "/" + tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]int{}
	for _, token := range spec.tokens() {
		if number, err := intValue(token); err == nil {
			values[token.ID] = number
		}
	}
	return values
}

func TestDerivedTokensAgreeWithTheirParts(t *testing.T) {
	v := tokenValues(t)
	checks := []struct {
		name      string
		got, want int
	}{
		{"content.width", v["content.width"], v["screen.width"] - 2*v["gutter"]},
		{"header.height", v["header.height"], v["header.row"] + v["header.divider"]},
		{"body.top", v["body.top"], v["inset.statusBar"] + v["header.height"] + v["rhythm"]},
		{"margin.bottom", v["margin.bottom"], v["gutter"]},
		{"radio.dotInset", v["radio.dotInset"], (v["radio.ring"] - v["radio.dot"]) / 2},
		{"row.labelHeight", v["row.labelHeight"], v["type.row.line"]},
		{"notice.padding", v["notice.padding"], v["rhythm"]},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s = %d, but its parts give %d", check.name, check.got, check.want)
		}
	}
}

// The big code is capitals and digits only, so stacked diacritics cannot clip: A2 exempts it.
func TestEveryTextTokenHasLineHeightOfAtLeast14TimesTheSize(t *testing.T) {
	v := tokenValues(t)
	for id, size := range v {
		if !strings.HasSuffix(id, ".size") || !strings.HasPrefix(id, "type.") || id == "type.bigCode.size" {
			continue
		}
		line, hasLine := v[strings.TrimSuffix(id, ".size")+".line"]
		if hasLine && line*10 < size*14 {
			t.Errorf("%s: line height %d is under 1.4 x size %d (A2)", id, line, size)
		}
	}
}

func TestNamesAreDerivedFromIds(t *testing.T) {
	if got := cppName("type.headerTitle.size"); got != "TYPE_HEADER_TITLE_SIZE" {
		t.Errorf("cppName = %s", got)
	}
	if got := goName("type.headerTitle.size"); got != "TypeHeaderTitleSize" {
		t.Errorf("goName = %s", got)
	}
}

func TestInvalidTokenFilesAreRejectedWithAReason(t *testing.T) {
	cases := map[string]string{
		"grey off the grid": `{"version":"1","groups":[{"name":"g","tokens":[{"id":"ink.x","value":64,"unit":"grey","doc":""}]}]}`,
		"duplicate id":      `{"version":"1","groups":[{"name":"g","tokens":[{"id":"a.b","value":1,"unit":"px","doc":""},{"id":"a.b","value":2,"unit":"px","doc":""}]}]}`,
		"unknown unit":      `{"version":"1","groups":[{"name":"g","tokens":[{"id":"a.b","value":1,"unit":"em","doc":""}]}]}`,
		"text as number":    `{"version":"1","groups":[{"name":"g","tokens":[{"id":"a.b","value":"x","unit":"px","doc":""}]}]}`,
		"bad id":            `{"version":"1","groups":[{"name":"g","tokens":[{"id":"A_b","value":1,"unit":"px","doc":""}]}]}`,
		"unknown field":     `{"version":"1","groups":[],"colour":"red"}`,
	}
	for name, source := range cases {
		if _, err := parseSpec([]byte(source)); err == nil {
			t.Errorf("%s: the token file was accepted", name)
		}
	}
}

func TestCheckModeReportsStaleFiles(t *testing.T) {
	generated, err := generateAll(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	generated[docPath] = append(generated[docPath], "changed\n"...)
	var stderr bytes.Buffer
	if code := reportStale(repoRoot, generated, &stderr); code != 1 || !strings.Contains(stderr.String(), docPath) {
		t.Errorf("reportStale = %d, %q; want 1 naming %s", code, stderr.String(), docPath)
	}
}

func TestCStringEscapesNonAscii(t *testing.T) {
	if got := cString("a\"b…\n"); got != `"a\"b\342\200\246\n"` {
		t.Errorf("cString = %s", got)
	}
}

func TestLibraryVersionFileMatchesTheTokenFile(t *testing.T) {
	spec, err := loadSpec(repoRoot + "/" + tokensPath)
	if err != nil {
		t.Fatal(err)
	}
	version, err := os.ReadFile(repoRoot + "/lib/copperui/VERSION")
	if err != nil {
		t.Fatalf("lib/copperui/VERSION is missing: %v", err)
	}
	if strings.TrimSpace(string(version)) != spec.Version {
		t.Errorf("lib/copperui/VERSION is %q but the token file says %q", strings.TrimSpace(string(version)), spec.Version)
	}
}

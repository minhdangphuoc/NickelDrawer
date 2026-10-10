package inbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuleContentMatchesChecksTheMagicBytes(t *testing.T) {
	folder := t.TempDir()
	pdf := filepath.Join(folder, "a.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.7 body"), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(folder, "b.pdf")
	if err := os.WriteFile(fake, []byte("<html>not a pdf</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	rule := Rule{Type: "pdf", Check: "pdf"}
	if matches, err := rule.ContentMatches(pdf); err != nil || !matches {
		t.Errorf("real pdf: matches=%v err=%v", matches, err)
	}
	if matches, err := rule.ContentMatches(fake); err != nil || matches {
		t.Errorf("fake pdf: matches=%v err=%v", matches, err)
	}
	if _, err := rule.ContentMatches(filepath.Join(folder, "missing.pdf")); err == nil {
		t.Error("a missing file must be an error, not a mismatch")
	}
}

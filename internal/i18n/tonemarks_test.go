package i18n

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

// A5: Vietnamese texts use the tone-mark placement "Xóa, Hủy, Tùy" (the mark on the main vowel)
// in every locale file of the repository.

var letterRun = regexp.MustCompile(`\p{L}+`)

const (
	oldStyleAfterO = "àáảãạèéẻẽẹ" // oà oá ... oè oé ...: the mark moved to the second vowel
	oldStyleAfterU = "ỳýỷỹỵ"      // uỳ uý ...
)

// oldStyleMark reports a word that ends in an open "oa", "oe" or "uy" with the mark on the last
// vowel (xoá, khoẻ, tuỳ); closed syllables (toán, hoàn) and "qu" + vowel (quý) are the same in both styles.
func oldStyleMark(word string) bool {
	runes := []rune(strings.ToLower(word))
	if len(runes) < 2 {
		return false
	}
	last, before := string(runes[len(runes)-1]), runes[len(runes)-2]
	switch {
	case before == 'o':
		return strings.Contains(oldStyleAfterO, last)
	case before == 'u' && len(runes) > 2 && runes[len(runes)-3] != 'q':
		return strings.Contains(oldStyleAfterU, last)
	}
	return false
}

func TestVietnameseLocalesUseTheNewToneMarkStyle(t *testing.T) {
	patterns := []string{
		"../../apps/*/locales/vi.json",
		"../../host/locales/vi.json",
		"../../helpers/extras/locales/vi.json",
		"../../host/screens/*/locales/vi.json",
		"../../examples/*/locales/vi.json",
		"../../mod/drawer/fallback_strings.cc",
	}
	checked := 0
	for _, pattern := range patterns {
		paths, err := filepath.Glob(pattern)
		if err != nil || len(paths) == 0 {
			t.Fatalf("no files match %s (%v)", pattern, err)
		}
		for _, path := range paths {
			checked++
			content := readTestFile(t, path)
			if !norm.NFC.IsNormalString(content) {
				t.Errorf("%s is not in Unicode normalization form C", path)
			}
			for _, word := range letterRun.FindAllString(content, -1) {
				if oldStyleMark(word) {
					t.Errorf("%s: %q uses the old tone-mark style; write it with the mark on the main vowel (xóa, hủy, tùy)", path, word)
				}
			}
		}
	}
	if checked < 10 {
		t.Fatalf("only %d files checked", checked)
	}
}

func TestOldStyleMarkDetection(t *testing.T) {
	for _, word := range []string{"Xoá", "Huỷ", "tuỳ", "khoẻ", "hoà"} {
		if !oldStyleMark(word) {
			t.Errorf("%s should be flagged", word)
		}
	}
	for _, word := range []string{"Xóa", "Hủy", "tùy", "toán", "hoàn", "quý", "quỳnh", "háy"} {
		if oldStyleMark(word) {
			t.Errorf("%s should pass", word)
		}
	}
}

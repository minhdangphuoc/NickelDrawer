package uikit

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"koboapps/internal/copperui"
)

// Content-style helpers for the warnings of docs/design-system.md section 5: no typed ALL CAPS,
// a header status that fits its slot, and no full stop on a one-line caption. The rules only
// warn: a style slip must never stop an app from loading on the device.

// StatusMaxWidth is the room of the header status text (header.sideWidth) and StatusTextSize the
// size it is drawn at (type.headerStatus), both CopperUI tokens.
const (
	StatusMaxWidth = copperui.HeaderSideWidth
	StatusTextSize = copperui.TypeHeaderStatusSize
)

// acronyms may stay in capitals inside a sentence (they are read letter by letter).
var acronyms = map[string]bool{
	"API": true, "CBZ": true, "CPU": true, "DNS": true, "EPUB": true, "GPS": true, "HTTP": true, "HTTPS": true,
	"JPEG": true, "JPG": true, "JSON": true, "LAN": true, "NFC": true, "NFD": true, "PDF": true, "PIN": true,
	"PNG": true, "QR": true, "RTL": true, "SSID": true, "TLS": true, "UTF": true, "URL": true, "USB": true,
	"WPA": true, "ZIP": true, "OK": true, "IP": true, "ID": true, "UI": true, "SD": true, "GB": true, "MB": true,
	"KB": true, "SMS": true, "MIT": true, "BSD": true, "GPL": true, "LGPL": true, "AGPL": true, "MPL": true, "ISC": true, "SPDX": true, "README": true,
}

// UppercaseWords returns the words of text that are typed in capitals (three letters or more,
// no lower case) and are not a known acronym; "SỰ CỐ" gives both words. Section labels are
// written in sentence case: the renderer uppercases them.
func UppercaseWords(text string) []string {
	var words []string
	letters, hasLower := 0, false
	var capital []string
	for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if acronyms[word] {
			continue
		}
		wordLetters := utf8.RuneCountInString(word)
		letters += wordLetters
		isUpper := true
		for _, r := range word {
			if unicode.IsLower(r) {
				isUpper, hasLower = false, true
			}
		}
		if isUpper {
			capital = append(capital, word)
			if wordLetters >= 3 {
				words = append(words, word)
			}
		}
	}
	if len(words) == 0 && letters >= 3 && !hasLower {
		words = capital // "SỰ CỐ": short words that are capitals all through
	}
	return words
}

// EndsWithFullStop reports whether text is a single sentence that ends with a full stop: a
// caption of one line has none (body sentences keep theirs).
func EndsWithFullStop(text string) bool {
	trimmed := strings.TrimSpace(text)
	if !strings.HasSuffix(trimmed, ".") || strings.HasSuffix(trimmed, "..") || strings.HasSuffix(trimmed, "…") {
		return false
	}
	inner := strings.TrimSuffix(trimmed, ".")
	return !strings.Contains(inner, ". ") && !strings.Contains(inner, "? ") && !strings.Contains(inner, "! ")
}

var placeholderPattern = regexp.MustCompile(`\{[a-z_]+\}`)

// widthSample stands in for a {placeholder} when a text is measured: a count or a short word.
func widthSample(placeholder string) string {
	switch placeholder {
	case "{count}", "{n}", "{done}", "{total}", "{page}", "{pages}":
		return "99"
	}
	return "abcdef"
}

// statusKeyPattern names the locale keys apps use for the header status: ui.status,
// ui.status_receiving, msg.status_shown, msg.state_status_ready, ui.pictures_status.
var statusKeyPattern = regexp.MustCompile(`^(?:msg|ui)\.(?:[a-z]+_)?status(?:_[a-z_]+)?$`)

// IsStatusKey reports whether a locale key holds a header status text.
func IsStatusKey(key string) bool { return statusKeyPattern.MatchString(key) }

// StatusWidth is the estimated width in px of a header status at its type size, with each
// {placeholder} replaced by a sample value.
func StatusWidth(text string) int {
	filled := placeholderPattern.ReplaceAllStringFunc(text, widthSample)
	return EstimateWidth(filled, StatusTextSize, false)
}

// advance is the width of a character in 1/1000 em, in Avenir Next / Rakuten Sans class
// proportions. TextWidth is an estimate for warnings; layout itself is always measured with the
// real font (design-system.md 2.8). cmd/ndpreview's tests keep this table within a few percent
// of the preview font so it cannot drift unnoticed.
func advance(r rune) int {
	base := r
	if r > 0x7f {
		if decomposed := []rune(norm.NFD.String(string(r))); len(decomposed) > 0 {
			base = decomposed[0]
		}
	}
	switch {
	case base == ' ':
		return 280
	case strings.ContainsRune("il|!'", base):
		return 240
	case strings.ContainsRune("jI.,:;", base):
		return 280
	case strings.ContainsRune("ft/\\()[]{}", base):
		return 320
	case base == '-':
		return 340
	case base == 'r':
		return 360
	case strings.ContainsRune("sczJ", base):
		return 470
	case base == 'm':
		return 830
	case base == 'w':
		return 740
	case strings.ContainsRune("MW", base):
		return 900
	case unicode.IsDigit(base):
		return 560
	case base == 'Đ' || base == 'đ':
		if base == 'Đ' {
			return 720
		}
		return 560
	case unicode.IsUpper(base):
		return 660
	}
	return 540
}

// EstimateWidth is the width in px of text drawn on one line at size px.
func EstimateWidth(text string, size int, isBold bool) int {
	total := 0
	for _, r := range text {
		total += advance(r)
	}
	if isBold {
		total = total * 106 / 100
	}
	return (total*size + 999) / 1000
}

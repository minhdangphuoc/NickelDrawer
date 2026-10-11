package main

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Text fit: one wrap/elide algorithm shared with the mod (mod/drawer/screen_logic.cc). Widths
// are measured, never estimated; the vectors in lib/copperui/tests/text-vectors.json pin both
// implementations to the same results (see textfit_test.go and tests/screen_logic_test.cc).
//
//   - Paragraphs split at '\n'; words split at Unicode spaces and are joined by one space.
//   - A line is filled greedily; a word wider than the line is cut into the longest pieces that
//     fit (at least one code point each) and its last piece stays open for the next word.
//   - wrapLimited keeps at most maxLines lines; the rest is joined by spaces into the last one
//     and elided at the end (buttons: two centred lines, then "…").
//   - elide gives one line: the longest prefix (or prefix + suffix in the middle) that fits
//     together with "…", found by binary search.

const ellipsis = "…"

// textFace is what decides the width of a run of text.
type textFace struct {
	size   int
	family string
	isBold bool
}

func regular(size int, family string) textFace { return textFace{size: size, family: family} }

func bold(size int, family string) textFace {
	return textFace{size: size, family: family, isBold: true}
}

// textMeasurer returns the width in pixels of text drawn in face on one line.
type textMeasurer interface {
	width(text string, face textFace) int
}

// textMeter measures every layout decision; main sets it to the real fonts (useFonts).
var textMeter textMeasurer = estimatedMeter{}

// glyphChecker is a textMeasurer that knows which code points its font draws itself.
type glyphChecker interface {
	hasGlyph(r rune, face textFace) bool
}

const (
	rightwardsArrow = "→" // "1.0.1 → 1.0.2" on the App updates screen
	asciiArrow      = "->"
)

// drawableText is text as the font can draw it: "→" becomes "->" when the font has no arrow, so
// it is never measured or drawn as a missing glyph (mirrored by the mod's drawableUtf8 and
// ScreenLogic::replaceArrows).
func drawableText(text string, face textFace) string {
	if !strings.Contains(text, rightwardsArrow) {
		return text
	}
	checker, canCheck := textMeter.(glyphChecker)
	if !canCheck || checker.hasGlyph([]rune(rightwardsArrow)[0], face) {
		return text
	}
	return strings.ReplaceAll(text, rightwardsArrow, asciiArrow)
}

func fits(text string, limit int, face textFace) bool {
	return textMeter.width(text, face) <= limit
}

func wrapText(text string, limit int, face textFace) []string {
	var lines []string
	for _, paragraph := range strings.Split(drawableText(text, face), "\n") {
		lines = append(lines, wrapParagraph(paragraph, limit, face)...)
	}
	return lines
}

func wrapParagraph(paragraph string, limit int, face textFace) []string {
	var lines []string
	current := ""
	for _, word := range strings.Fields(paragraph) {
		if !fits(word, limit, face) && utf8.RuneCountInString(word) > 1 {
			lines = appendLine(lines, current)
			var pieces []string
			pieces, current = cutLongWord(word, limit, face)
			lines = append(lines, pieces...)
			continue
		}
		switch {
		case current == "":
			current = word
		case fits(current+" "+word, limit, face):
			current += " " + word
		default:
			lines = append(lines, current)
			current = word
		}
	}
	return appendLine(lines, current)
}

// cutLongWord returns the full lines of a word wider than the limit and the open last piece.
func cutLongWord(word string, limit int, face textFace) (full []string, rest string) {
	runes := []rune(word)
	for len(runes) > 1 && !fits(string(runes), limit, face) {
		count := longestFittingPrefix(runes, limit, face)
		full = append(full, string(runes[:count]))
		runes = runes[count:]
	}
	return full, string(runes)
}

// longestFittingPrefix is the largest count in [1, len-1] whose prefix fits; 1 when none does.
func longestFittingPrefix(runes []rune, limit int, face textFace) int {
	return largestFitting(1, len(runes)-1, func(count int) bool { return fits(string(runes[:count]), limit, face) })
}

// largestFitting is the binary search both implementations use: the largest count in
// [low, high] for which isFitting is true, or low when none is.
func largestFitting(low, high int, isFitting func(count int) bool) int {
	for low < high {
		middle := (low + high + 1) / 2
		if isFitting(middle) {
			low = middle
		} else {
			high = middle - 1
		}
	}
	return low
}

func appendLine(lines []string, line string) []string {
	if line == "" {
		return lines
	}
	return append(lines, line)
}

// wrapLimited wraps text into at most maxLines lines; the last one is elided when text is longer.
func wrapLimited(text string, limit int, face textFace, maxLines int) []string {
	lines := wrapText(text, limit, face)
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	rest := strings.Join(lines[maxLines-1:], " ")
	return append(lines[:maxLines-1:maxLines-1], elide(rest, limit, face, false))
}

// elide shortens text to one line: in the middle for file names, at the end otherwise.
func elide(text string, limit int, face textFace, isMiddle bool) string {
	text = drawableText(text, face)
	if fits(text, limit, face) {
		return text
	}
	runes := []rune(text)
	candidate := func(kept int) string { return elidedCandidate(runes, kept, isMiddle) }
	kept := largestFitting(0, max(len(runes)-1, 0), func(count int) bool { return fits(candidate(count), limit, face) })
	return candidate(kept)
}

// elidedCandidate keeps kept code points: a prefix, or half of them from each end.
func elidedCandidate(runes []rune, kept int, isMiddle bool) string {
	if !isMiddle {
		return strings.TrimRightFunc(string(runes[:kept]), unicode.IsSpace) + ellipsis
	}
	head := kept / 2
	tail := kept - head
	return string(runes[:head]) + ellipsis + string(runes[len(runes)-tail:])
}

// estimatedMeter is the fallback when no font file is found: an average advance per code
// point. Only used so layout still works without fonts; previews then warn.
type estimatedMeter struct{}

var advanceByFamily = map[string]float64{familySans: 0.56, familySerif: 0.52, familyMono: 0.6}

func (estimatedMeter) width(text string, face textFace) int {
	advance := advanceByFamily[face.family] * float64(face.size)
	if face.isBold {
		advance *= 1.1
	}
	return int(advance*float64(utf8.RuneCountInString(text)) + 0.5)
}

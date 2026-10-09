package library

import (
	"html"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	// minLetters is how much text is needed before a language guess means anything.
	minLetters = 100
	// vietnameseRatio is the share of Vietnamese-marked letters above which text counts as
	// Vietnamese. Vietnamese prose is about 0.30-0.40; French and Portuguese stay under 0.10.
	vietnameseRatio = 0.20
	// vietnameseOnlyRatio guards against accents shared with French or Spanish: letters that only
	// Vietnamese uses (ă đ ơ ư and the tone-marked vowels) must also be common.
	vietnameseOnlyRatio = 0.08

	vietnameseOnlyLetters = "ăđơưĂĐƠƯ"
	sharedMarkedLetters   = "âêôÂÊÔàáãèéìíòóõùúýĩũÀÁÃÈÉÌÍÒÓÕÙÚÝĨŨ"
	toneBlockFirst        = 0x1EA0 // Latin Extended Additional: ạ ả ấ ầ ... ỹ
	toneBlockLast         = 0x1EF9
)

// VietnameseShare returns the share of Vietnamese-marked letters among all letters, and the
// letter count. Text is normalized first so decomposed tone marks count like composed ones.
func VietnameseShare(text string) (share float64, letters int) {
	marked, only := 0, 0
	for _, character := range norm.NFC.String(strings.ToValidUTF8(text, "")) {
		if !unicode.IsLetter(character) {
			continue
		}
		letters++
		switch {
		case isVietnameseOnly(character):
			only++
			marked++
		case strings.ContainsRune(sharedMarkedLetters, character):
			marked++
		}
	}
	if letters == 0 || float64(only)/float64(letters) < vietnameseOnlyRatio {
		return 0, letters
	}
	return float64(marked) / float64(letters), letters
}

func isVietnameseOnly(character rune) bool {
	return strings.ContainsRune(vietnameseOnlyLetters, character) ||
		(character >= toneBlockFirst && character <= toneBlockLast)
}

// LooksVietnamese applies the thresholds to a text sample.
func LooksVietnamese(text string) (share float64, isVietnamese bool) {
	share, letters := VietnameseShare(text)
	return share, letters >= minLetters && share >= vietnameseRatio
}

// languageBase lower-cases a BCP 47 tag and keeps the primary subtag: "vi-VN" -> "vi".
func languageBase(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	base, _, _ := strings.Cut(strings.ReplaceAll(tag, "_", "-"), "-")
	return base
}

var blockTags = map[string]bool{
	"p": true, "div": true, "br": true, "li": true, "tr": true, "td": true, "th": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"blockquote": true, "section": true, "article": true, "hr": true,
}

var skippedElements = []string{"script", "style", "head"}

// StripTags returns the visible text of an (X)HTML fragment: tags removed, script, style and
// head dropped, entities decoded. Inline tags join without a space so split words stay whole.
// A cut-off input (a sampled prefix of a file) is fine.
func StripTags(markup []byte) string {
	source := string(markup)
	var out strings.Builder
	for position := 0; position < len(source); {
		open := strings.IndexByte(source[position:], '<')
		if open < 0 {
			out.WriteString(source[position:])
			break
		}
		out.WriteString(source[position : position+open])
		position += open
		position = skipTag(source, position, &out)
	}
	return html.UnescapeString(out.String())
}

// skipTag consumes the tag starting at position and returns the position after it
// (after the whole element for script, style and head).
func skipTag(source string, position int, out *strings.Builder) int {
	if strings.HasPrefix(source[position:], "<!--") {
		end := strings.Index(source[position:], "-->")
		if end < 0 {
			return len(source)
		}
		return position + end + len("-->")
	}
	close := strings.IndexByte(source[position:], '>')
	if close < 0 {
		return len(source)
	}
	inside := source[position+1 : position+close]
	name := tagName(inside)
	next := position + close + 1
	isOpening := !strings.HasPrefix(inside, "/") && !strings.HasSuffix(inside, "/")
	for _, skipped := range skippedElements {
		if isOpening && name == skipped {
			end := strings.Index(strings.ToLower(source[next:]), "</"+skipped)
			if end < 0 {
				return len(source)
			}
			return next + end
		}
	}
	if blockTags[name] {
		out.WriteByte(' ')
	}
	return next
}

// tagName returns the lower-case element name of the inside of a tag ("/p", "br /", "a href=..").
func tagName(inside string) string {
	inside = strings.TrimPrefix(inside, "/")
	end := strings.IndexFunc(inside, func(character rune) bool {
		return unicode.IsSpace(character) || character == '/' || character == '>'
	})
	if end >= 0 {
		inside = inside[:end]
	}
	return strings.ToLower(inside)
}

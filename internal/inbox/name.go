package inbox

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	// maxNameBytes leaves room for a " (999)" suffix inside the 255 bytes vfat allows.
	maxNameBytes = 180
	fallbackStem = "file"
	// longestPlainExtension stops a long dotted tail being mistaken for an extension.
	longestPlainExtension = 16
	// fatForbidden are characters vfat (/mnt/onboard) cannot store in names.
	fatForbidden = `<>:"|?*`
)

// SanitizeName makes a received file name safe to store on the Kobo's vfat storage:
// NFC-normalized (so Vietnamese letters compose), no path separators or control characters,
// no leading dot or space, no trailing dot or space, at most 180 bytes with the extension kept.
//
// extension is the suffix to keep (".epub"); pass "" to keep whatever follows the last dot.
func SanitizeName(name, extension string) string {
	cleaned := cleanCharacters(name)
	extension = keptExtension(cleaned, extension)
	stem := strings.TrimSuffix(cleaned, cleaned[len(cleaned)-len(extension):])
	stem = strings.Trim(stem, ". ")
	if stem == "" {
		stem = fallbackStem
	}
	return cutToBytes(stem, maxNameBytes-len(extension)) + extension
}

func cleanCharacters(name string) string {
	name = norm.NFC.String(strings.ToValidUTF8(name, "_"))
	var builder strings.Builder
	for _, character := range name {
		switch {
		case character == '/' || character == '\\' || unicode.IsControl(character):
			continue
		case strings.ContainsRune(fatForbidden, character):
			builder.WriteRune('_')
		default:
			builder.WriteRune(character)
		}
	}
	return strings.TrimRight(builder.String(), ". ")
}

// keptExtension returns the lower-case extension to preserve, or "" when none applies.
func keptExtension(cleaned, extension string) string {
	if extension != "" {
		if strings.HasSuffix(strings.ToLower(cleaned), extension) {
			return extension
		}
		return ""
	}
	plain := filepath.Ext(cleaned)
	if len(plain) < 2 || len(plain) > longestPlainExtension || strings.ContainsAny(plain, " ") {
		return ""
	}
	return strings.ToLower(plain)
}

// cutToBytes shortens text to at most limit bytes without splitting a character.
func cutToBytes(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.TrimRight(text[:cut], ". ")
}

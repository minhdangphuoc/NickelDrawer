package inbox

import (
	"bytes"
	"errors"
	"io"
	"os"
	"unicode/utf8"
)

const (
	headLength        = 4096
	textChunkLength   = 64 * 1024
	htmlLookahead     = 512
	mobiMagicOffset   = 60
	zipLocalHeaderLen = 30
)

var (
	zipMagic  = []byte("PK\x03\x04")
	rar4Magic = []byte("Rar!\x1a\x07\x00")
	rar5Magic = []byte("Rar!\x1a\x07\x01\x00")
	pngMagic  = []byte("\x89PNG\r\n\x1a\n")
	jpegMagic = []byte{0xFF, 0xD8, 0xFF}
	utf8BOM   = []byte{0xEF, 0xBB, 0xBF}
)

// headChecks decide from the first bytes alone.
var headChecks = map[string]func(head []byte) bool{
	"epub": isEpubHead,
	"mobi": func(head []byte) bool {
		return len(head) >= mobiMagicOffset+8 && string(head[mobiMagicOffset:mobiMagicOffset+8]) == "BOOKMOBI"
	},
	"pdf":  func(head []byte) bool { return bytes.HasPrefix(head, []byte("%PDF-")) },
	"html": isHTMLHead,
	"rtf":  func(head []byte) bool { return bytes.HasPrefix(head, []byte(`{\rtf`)) },
	"zip":  func(head []byte) bool { return bytes.HasPrefix(head, zipMagic) },
	"rar":  func(head []byte) bool { return bytes.HasPrefix(head, rar4Magic) || bytes.HasPrefix(head, rar5Magic) },
	"png":  func(head []byte) bool { return bytes.HasPrefix(head, pngMagic) },
	"jpeg": func(head []byte) bool { return bytes.HasPrefix(head, jpegMagic) },
	"gif": func(head []byte) bool {
		return bytes.HasPrefix(head, []byte("GIF87a")) || bytes.HasPrefix(head, []byte("GIF89a"))
	},
}

const textCheck = "utf8text"

func isKnownCheck(name string) bool {
	_, found := headChecks[name]
	return found || name == textCheck
}

// contentMatches reports whether the file content fits the named check. It reads at most the
// head of the file (the whole file only for the text check) and never unpacks anything.
func contentMatches(check string, file *os.File) (bool, error) {
	if check == textCheck {
		return isCleanUTF8(file)
	}
	head := make([]byte, headLength)
	count, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return false, err
	}
	return headChecks[check](head[:count]), nil
}

// isEpubHead checks the EPUB container rule: the first ZIP entry is a stored file named
// "mimetype" holding exactly application/epub+zip.
func isEpubHead(head []byte) bool {
	if len(head) < zipLocalHeaderLen || !bytes.HasPrefix(head, zipMagic) {
		return false
	}
	const storedMethod = 0
	method := int(head[8]) | int(head[9])<<8
	nameLength := int(head[26]) | int(head[27])<<8
	extraLength := int(head[28]) | int(head[29])<<8
	dataStart := zipLocalHeaderLen + nameLength + extraLength
	mimetype := []byte("application/epub+zip")
	if method != storedMethod || dataStart+len(mimetype) > len(head) {
		return false
	}
	return string(head[zipLocalHeaderLen:zipLocalHeaderLen+nameLength]) == "mimetype" &&
		bytes.HasPrefix(head[dataStart:], mimetype)
}

// isHTMLHead wants a '<' as the first visible character (after an optional BOM and spaces).
func isHTMLHead(head []byte) bool {
	head = bytes.TrimPrefix(head, utf8BOM)
	if len(head) > htmlLookahead {
		head = head[:htmlLookahead]
	}
	return bytes.HasPrefix(bytes.TrimLeft(head, " \t\r\n"), []byte("<"))
}

// isCleanUTF8 streams the file: valid UTF-8 and no NUL byte. Memory stays at one chunk.
func isCleanUTF8(file io.Reader) (bool, error) {
	chunk := make([]byte, textChunkLength)
	var carry []byte
	for {
		count, err := file.Read(chunk)
		pending := append(carry, chunk[:count]...)
		complete := len(pending)
		if err == nil {
			complete = len(pending) - incompleteTailLength(pending)
		}
		if bytes.IndexByte(pending[:complete], 0) >= 0 || !utf8.Valid(pending[:complete]) {
			return false, nil
		}
		carry = append([]byte(nil), pending[complete:]...)
		if errors.Is(err, io.EOF) {
			return len(carry) == 0, nil
		}
		if err != nil {
			return false, err
		}
	}
}

// incompleteTailLength counts trailing bytes that start a UTF-8 character the chunk cut in half.
func incompleteTailLength(buffer []byte) int {
	for back := 1; back < utf8.UTFMax && back <= len(buffer); back++ {
		tail := buffer[len(buffer)-back:]
		if utf8.RuneStart(tail[0]) {
			if utf8.FullRune(tail) {
				return 0
			}
			return back
		}
	}
	return 0
}

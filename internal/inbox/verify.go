package inbox

import "os"

// ContentMatches reports whether the file at path holds what the rule's content check expects
// (magic bytes for a book, clean UTF-8 for text). Other writers into the library, such as the
// GitHub sync, use it to hold to the same allowlist as the router without moving the file.
func (rule Rule) ContentMatches(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return contentMatches(rule.Check, file)
}

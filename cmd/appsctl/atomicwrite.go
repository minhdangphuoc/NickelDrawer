package main

import "koboapps/internal/update"

// writeFileAtomically writes via a temp file in the same folder and renames it,
// so NickelMenu never reads a half-written config.
func writeFileAtomically(target string, content []byte) error {
	return update.WriteFileAtomically(target, content, 0o644)
}

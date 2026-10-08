// Package storage defines where user files live on the Kobo's visible storage and offers
// read-only helpers that never leave the folder they are asked to look at.
//
// Layout (logical name, device path):
//
//	books      /mnt/onboard/Books
//	pictures   /mnt/onboard/Pictures
//	documents  /mnt/onboard/Documents
//	comics     /mnt/onboard/Comics
//	inbox      /mnt/onboard/.adds/koboapps/inbox            staging for received files
//	rejected   /mnt/onboard/.adds/koboapps/inbox/rejected   files the inbox router refused
//	screensaver /mnt/onboard/.kobo/screensaver              Nickel's sleep screen pictures
//
// The names are a hardcoded allowlist: callers pick a Folder constant, never a path from
// configuration or user input.
package storage

import (
	"path/filepath"
	"sort"
)

// Folder is a logical name in the layout.
type Folder string

const (
	Books     Folder = "books"
	Pictures  Folder = "pictures"
	Documents Folder = "documents"
	Comics    Folder = "comics"
	Inbox     Folder = "inbox"
	Rejected  Folder = "rejected"

	// Screensaver is Nickel's own folder; only the wallpaper app writes there.
	Screensaver Folder = "screensaver"

	// DefaultRoot is where Nickel mounts the user partition.
	DefaultRoot = "/mnt/onboard"
)

// relative maps each folder to its path below the root.
var relative = map[Folder]string{
	Books:     "Books",
	Pictures:  "Pictures",
	Documents: "Documents",
	Comics:    "Comics",

	Inbox:       ".adds/koboapps/inbox",
	Rejected:    ".adds/koboapps/inbox/rejected",
	Screensaver: ".kobo/screensaver",
}

// legacyLocalSendRelative is where older LocalSend versions received files.
const legacyLocalSendRelative = "LocalSend"

// koboappsRelative is the host folder (config/, data/, bin/ ...).
const koboappsRelative = ".adds/koboapps"

var root = DefaultRoot

// SetRoot moves the whole layout (tests, Mac smoke runs) and returns a function that restores it.
// Not safe for concurrent use; call it before starting work.
func SetRoot(newRoot string) (restore func()) {
	previous := root
	root = newRoot
	return func() { root = previous }
}

// Root returns the current root of the layout.
func Root() string { return root }

// Path returns the absolute path of a folder, or "" for a name that is not in the layout.
func Path(folder Folder) string {
	sub, found := relative[folder]
	if !found {
		return ""
	}
	return filepath.Join(root, filepath.FromSlash(sub))
}

// Known lists every logical name, sorted, for validation messages.
func Known() []Folder {
	names := make([]Folder, 0, len(relative))
	for folder := range relative {
		names = append(names, folder)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// IsKnown reports whether name is a logical name in the layout.
func IsKnown(name string) bool {
	_, found := relative[Folder(name)]
	return found
}

// KoboappsDir is the host folder; its config/ ships with the release and its data/ is ours to write.
func KoboappsDir() string { return filepath.Join(root, filepath.FromSlash(koboappsRelative)) }

// LegacyLocalSendDir is the old receive folder that "Sort old files" migrates.
func LegacyLocalSendDir() string { return filepath.Join(root, legacyLocalSendRelative) }

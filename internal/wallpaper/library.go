package wallpaper

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"sort"
	"strconv"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

func itoa(number int) string { return strconv.Itoa(number) }

func pictureID(id int) string { return fmt.Sprintf("%04d", id) }

func imageFileName(id int) string { return pictureID(id) + ".png" }

// Entry is one converted picture in data/images.
type Entry struct {
	ID   int    `json:"id"`
	Hash string `json:"hash"` // sha256 of the original; identifies it across folders and runs
	Mode Mode   `json:"mode"` // how the stored file was fitted
	Name string `json:"name"` // original file name, for the log, the gallery label and Info

	Width  int    `json:"width,omitempty"`  // original pixels; older entries omit it
	Height int    `json:"height,omitempty"` // original pixels
	Added  string `json:"added,omitempty"`  // RFC 3339, UTC
}

// Library is data/images: converted PNGs plus index.json.
type Library struct {
	Paths   Paths
	Entries []Entry

	// Removed holds the hashes of pictures the user removed; Import skips them.
	Removed map[string]bool
	// OnProgress is told after each source picture was handled (nil = nobody listens).
	OnProgress func(done, total int, name string)
}

// LoadLibrary reads index.json; a missing index is an empty library.
func LoadLibrary(paths Paths) (*Library, error) {
	library := &Library{Paths: paths}
	data, err := os.ReadFile(paths.Index())
	if errors.Is(err, os.ErrNotExist) {
		return library, nil
	}
	if err != nil {
		return nil, i18n.NewError("msg.err_index_read", "path", paths.Index()).WithCause(err)
	}
	if err := json.Unmarshal(data, &library.Entries); err != nil {
		return nil, i18n.NewError("msg.err_index_read", "path", paths.Index()).WithCause(err)
	}
	sort.Slice(library.Entries, func(i, j int) bool { return library.Entries[i].ID < library.Entries[j].ID })
	return library, library.loadRemoved()
}

func (library *Library) save() error {
	encoded, err := json.MarshalIndent(library.Entries, "", "  ")
	if err != nil {
		return i18n.NewError("msg.err_write", "path", library.Paths.Index()).WithCause(err)
	}
	if err := os.MkdirAll(library.Paths.Images(), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", library.Paths.Images()).WithCause(err)
	}
	return writeFileAtomic(library.Paths.Index(), ".index-", append(encoded, '\n'))
}

func (library *Library) find(id int) (Entry, bool) {
	for _, entry := range library.Entries {
		if entry.ID == id {
			return entry, true
		}
	}
	return Entry{}, false
}

func (library *Library) indexOfHash(hash string) int {
	for position, entry := range library.Entries {
		if entry.Hash == hash {
			return position
		}
	}
	return -1
}

func (library *Library) nextID() int {
	highest := 0
	for _, entry := range library.Entries {
		highest = max(highest, entry.ID)
	}
	return highest + 1
}

// ImportReport counts what one import did.
type ImportReport struct {
	Added       int
	Reconverted int
	Duplicates  int
	Skipped     int // pictures the user removed earlier
	Failed      int
	NewestID    int // id of the last picture added, 0 if none
}

// plus adds a follow-up pass (a second re-fit after a mode change). Its Duplicates, Skipped and
// Failed repeat the first pass, so only the changes count.
func (report ImportReport) plus(other ImportReport) ImportReport {
	report.Added += other.Added
	report.Reconverted += other.Reconverted
	if other.NewestID != 0 {
		report.NewestID = other.NewestID
	}
	return report
}

// Import converts every new picture from the sources and re-fits known ones whose stored mode
// differs from the wanted mode (the originals are still in the source folders). Originals are
// only read, never changed. One broken picture is logged and skipped, not fatal.
func (library *Library) Import(sources []string, size Size, mode Mode, log *Logger) (ImportReport, error) {
	var report ImportReport
	seenThisRun := map[string]bool{}
	candidates := listSources(sources, log)
	library.reportProgress(0, len(candidates), nameAt(candidates, 0))
	for position, candidate := range candidates {
		if err := library.importCandidate(candidate, seenThisRun, size, mode, &report, log); err != nil {
			return report, err
		}
		library.reportProgress(position+1, len(candidates), candidate.Name)
	}
	return report, nil
}

func nameAt(files []storage.File, position int) string {
	if position < len(files) {
		return files[position].Name
	}
	return ""
}

func (library *Library) reportProgress(done, total int, name string) {
	if library.OnProgress != nil {
		library.OnProgress(done, total, name)
	}
}

// importCandidate handles one source picture: unreadable, repeated or removed ones are counted and skipped.
func (library *Library) importCandidate(candidate storage.File, seenThisRun map[string]bool, size Size, mode Mode, report *ImportReport, log *Logger) error {
	content, err := storage.ReadLimited(candidate.Path, maxSourceBytes)
	if err != nil {
		log.Printf("skip %s: cannot read: %v", candidate.Name, err)
		report.Failed++
		return nil
	}
	hash := hashOf(content)
	switch {
	case seenThisRun[hash]:
		report.Duplicates++
		return nil
	case library.Removed[hash]:
		seenThisRun[hash] = true
		report.Skipped++
		return nil
	}
	seenThisRun[hash] = true
	return library.take(candidate.Name, hash, content, size, mode, report, log)
}

// take stores or re-fits one picture. A conversion failure is counted, a disk failure aborts.
func (library *Library) take(name, hash string, content []byte, size Size, mode Mode, report *ImportReport, log *Logger) error {
	position := library.indexOfHash(hash)
	if position >= 0 && library.Entries[position].Mode == mode {
		report.Duplicates++
		return nil
	}
	converted, err := Convert(content, size, mode)
	if err != nil {
		log.Printf("skip %s: %s", name, err)
		report.Failed++
		return nil
	}
	if err := os.MkdirAll(library.Paths.Images(), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", library.Paths.Images()).WithCause(err)
	}
	width, height := originalSize(content)
	if position >= 0 {
		return library.replace(position, converted, mode, width, height, report, log)
	}
	return library.add(name, hash, converted, mode, width, height, report, log)
}

// originalSize is 0, 0 when the header cannot be read again (Convert already accepted it).
func originalSize(content []byte) (width, height int) {
	config, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}

// replace stores the re-fitted picture; its thumbnail is dropped and comes back lazily.
func (library *Library) replace(position int, converted []byte, mode Mode, width, height int, report *ImportReport, log *Logger) error {
	entry := library.Entries[position]
	if err := writeFileAtomic(library.Paths.Image(entry.ID), ".img-", converted); err != nil {
		return err
	}
	library.removeThumb(entry.ID, log)
	library.Entries[position].Mode = mode
	if entry.Width == 0 {
		library.Entries[position].Width, library.Entries[position].Height = width, height
	}
	report.Reconverted++
	return library.save()
}

func (library *Library) add(name, hash string, converted []byte, mode Mode, width, height int, report *ImportReport, log *Logger) error {
	entry := Entry{ID: library.nextID(), Hash: hash, Mode: mode, Name: name, Width: width, Height: height, Added: time.Now().UTC().Format(time.RFC3339)}
	if err := writeFileAtomic(library.Paths.Image(entry.ID), ".img-", converted); err != nil {
		return err
	}
	library.writeThumb(entry.ID, converted, log)
	library.Entries = append(library.Entries, entry)
	report.Added++
	report.NewestID = entry.ID
	log.Printf("imported %s as %s", name, imageFileName(entry.ID))
	return library.save()
}

func hashOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// NextID returns the id after current in library order, wrapping around; 0 for an empty library.
func (library *Library) NextID(current int) int {
	for _, entry := range library.Entries {
		if entry.ID > current {
			return entry.ID
		}
	}
	if len(library.Entries) == 0 {
		return 0
	}
	return library.Entries[0].ID
}

// listSources lists the pictures of every source folder, in source order.
func listSources(sources []string, log *Logger) []storage.File {
	var files []storage.File
	for _, dir := range sources {
		files = append(files, storage.ListImages(dir, maxSourceBytes, log.Printf)...)
	}
	return files
}

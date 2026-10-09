package inbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"koboapps/internal/storage"
)

// maxReceivedListed is how many files the app screen's "Received" list shows.
const maxReceivedListed = 10

// ReceivedFile is one routed file for the screen's list.
type ReceivedFile struct {
	Name   string
	Folder storage.Folder
	Size   int64 // 0 when the file is gone or cannot be read
}

// ScreenState is the last receiving session as the app screen binds it.
type ScreenState struct {
	Books     int
	Documents int
	Comics    int
	Pictures  int
	Rejected  int
	Received  []ReceivedFile // newest first, at most maxReceivedListed
}

// BuildScreenState counts the newest session and lists its last routed files. Sizes are read
// with one stat per listed file; a missing file only leaves the size at 0.
func BuildScreenState(entries []Entry) ScreenState {
	summary := SummarizeLastSession(entries)
	state := ScreenState{
		Books:     summary.Routed[storage.Books],
		Documents: summary.Routed[storage.Documents],
		Comics:    summary.Routed[storage.Comics],
		Pictures:  summary.Routed[storage.Pictures],
		Rejected:  summary.Rejected,
	}
	routed := routedEntries(lastSession(entries))
	for index := len(routed) - 1; index >= 0 && len(state.Received) < maxReceivedListed; index-- {
		state.Received = append(state.Received, receivedFileOf(routed[index]))
	}
	return state
}

func routedEntries(entries []Entry) []Entry {
	routed := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Destination != string(storage.Rejected) {
			routed = append(routed, entry)
		}
	}
	sort.SliceStable(routed, func(i, j int) bool { return routed[i].Time.Before(routed[j].Time) })
	return routed
}

func receivedFileOf(entry Entry) ReceivedFile {
	folder := storage.Folder(entry.Destination)
	file := ReceivedFile{Name: entry.Name, Folder: folder}
	if entry.StoredAs == "" {
		return file
	}
	if info, err := os.Stat(filepath.Join(storage.Path(folder), entry.StoredAs)); err == nil {
		file.Size = info.Size()
	}
	return file
}

// ListItems turns the received files into [primary, secondary] pairs for a `list` component.
// folderLabels maps a folder id to its localized name; an unknown id shows as is.
func (state ScreenState) ListItems(folderLabels map[string]string) [][2]string {
	items := make([][2]string, 0, len(state.Received))
	for _, file := range state.Received {
		secondary := labelOrID(folderLabels, string(file.Folder))
		if file.Size > 0 {
			secondary += " · " + HumanSize(file.Size)
		}
		items = append(items, [2]string{file.Name, secondary})
	}
	return items
}

// HumanSize prints a byte count with one decimal ("1.2 MB").
func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value, suffix := float64(bytes), ""
	for _, next := range []string{"KB", "MB", "GB"} {
		value /= unit
		suffix = next
		if value < unit {
			break
		}
	}
	return fmt.Sprintf("%.1f %s", value, suffix)
}

// Summary is one line such as "Books 2 · Rejected 1" with the non-zero counters, or "" when
// nothing arrived. labels maps a folder id (and "rejected") to its localized name.
func (state ScreenState) Summary(labels map[string]string) string {
	counters := []struct {
		id    string
		count int
	}{{"books", state.Books}, {"documents", state.Documents}, {"comics", state.Comics},
		{"pictures", state.Pictures}, {"rejected", state.Rejected}}
	var parts []string
	for _, counter := range counters {
		if counter.count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", labelOrID(labels, counter.id), counter.count))
		}
	}
	return strings.Join(parts, " · ")
}

func labelOrID(labels map[string]string, id string) string {
	if label := labels[id]; label != "" {
		return label
	}
	return id
}

// JSON is the object the LocalSend screen merges into its variables: the one-line `summary`
// and the `received` list. It is one line so a shell script can splice more fields into it.
func (state ScreenState) JSON(labels map[string]string) ([]byte, error) {
	return json.Marshal(struct {
		Summary  string      `json:"summary"`
		Received [][2]string `json:"received"`
	}{state.Summary(labels), state.ListItems(labels)})
}

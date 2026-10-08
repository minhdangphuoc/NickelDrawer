package inbox

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"koboapps/internal/storage"
)

const (
	maxJournalLineBytes = 1 << 20
	// sessionGap separates two receiving sessions: files that arrive further apart than this
	// belong to different sessions.
	sessionGap = 30 * time.Minute
)

// Entry is one journal line: what arrived, from whom, and where it went or why it was refused.
type Entry struct {
	Time        time.Time `json:"time"`
	Source      string    `json:"source"`
	Name        string    `json:"name"`
	Type        string    `json:"type,omitempty"`
	Destination string    `json:"destination"`
	StoredAs    string    `json:"stored_as,omitempty"`
	Reason      Reason    `json:"reason,omitempty"`
}

// JournalPath is the append-only log of routed files.
func JournalPath() string {
	return filepath.Join(storage.KoboappsDir(), "data", "inbox-journal.jsonl")
}

// AppendEntry writes one line. Several router processes may call it at once (the LocalSend hook
// runs one per file), so each takes an exclusive flock around a single O_APPEND write, and the
// file is closed again straight away so nothing stays open on /mnt/onboard.
func AppendEntry(path string, entry Entry) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("cannot encode the journal entry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create the journal folder: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open the journal %s: %w", path, err)
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("cannot lock the journal %s: %w", path, err)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("cannot write the journal %s: %w", path, err)
	}
	return nil
}

// ReadEntries returns every well-formed line, oldest first. A missing journal is empty.
// Lines that do not parse (a cut write after a power loss) are skipped and counted.
func ReadEntries(path string) (entries []Entry, skipped int, err error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("cannot open the journal %s: %w", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxJournalLineBytes)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var entry Entry
		if json.Unmarshal(scanner.Bytes(), &entry) != nil || entry.Time.IsZero() {
			skipped++
			continue
		}
		entries = append(entries, entry)
	}
	return entries, skipped, scanner.Err()
}

// ReasonCount is how many files were refused for one reason.
type ReasonCount struct {
	Reason Reason
	Count  int
}

// Summary counts what the last receiving session routed.
type Summary struct {
	Routed   map[storage.Folder]int
	Rejected int
	Reasons  []ReasonCount // most frequent first
}

// IsEmpty is true when nothing was routed or rejected.
func (summary Summary) IsEmpty() bool {
	return summary.Rejected == 0 && len(summary.Routed) == 0
}

// SummarizeLastSession counts the newest run of entries not separated by more than sessionGap.
func SummarizeLastSession(entries []Entry) Summary {
	summary := Summary{Routed: map[storage.Folder]int{}}
	reasons := map[Reason]int{}
	for _, entry := range lastSession(entries) {
		if entry.Destination == string(storage.Rejected) {
			summary.Rejected++
			reasons[entry.Reason]++
			continue
		}
		summary.Routed[storage.Folder(entry.Destination)]++
	}
	for reason, count := range reasons {
		summary.Reasons = append(summary.Reasons, ReasonCount{reason, count})
	}
	sort.Slice(summary.Reasons, func(i, j int) bool {
		if summary.Reasons[i].Count != summary.Reasons[j].Count {
			return summary.Reasons[i].Count > summary.Reasons[j].Count
		}
		return summary.Reasons[i].Reason < summary.Reasons[j].Reason
	})
	return summary
}

func lastSession(entries []Entry) []Entry {
	ordered := append([]Entry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Time.Before(ordered[j].Time) })
	start := len(ordered)
	for start > 0 && (start == len(ordered) || ordered[start].Time.Sub(ordered[start-1].Time) <= sessionGap) {
		start--
	}
	return ordered[start:]
}

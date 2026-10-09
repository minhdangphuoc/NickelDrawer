package library

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"koboapps/internal/i18n"
)

const reportVersion = 1

// Counts are the numbers the summary and the screen show.
type Counts struct {
	Files           int `json:"files"`
	Epubs           int `json:"epubs"`
	WrongLanguage   int `json:"wrong_language"`
	NotNFC          int `json:"not_nfc"`
	Unreadable      int `json:"unreadable"`
	DuplicateGroups int `json:"duplicate_groups"`
	Skipped         int `json:"skipped"`
}

// Report is data/report.json; Summary (data/summary.json) is the same without Books so the
// state action stays fast on a large library.
type Report struct {
	Version    int              `json:"version"`
	ScannedAt  time.Time        `json:"scanned_at"`
	Counts     Counts           `json:"counts"`
	Duplicates []DuplicateGroup `json:"duplicates"`
	Books      []Book           `json:"books,omitempty"`
}

// BuildReport computes the counts and duplicate groups for the scanned books.
func BuildReport(books []Book, skipped int, scannedAt time.Time) Report {
	report := Report{Version: reportVersion, ScannedAt: scannedAt, Books: books, Duplicates: FindDuplicates(books)}
	report.Counts.Skipped = skipped
	report.refreshCounts()
	return report
}

func (report *Report) refreshCounts() {
	skipped := report.Counts.Skipped
	report.Counts = Counts{Files: len(report.Books), DuplicateGroups: len(report.Duplicates), Skipped: skipped}
	for _, book := range report.Books {
		switch {
		case book.Error != "":
			report.Counts.Unreadable++
		case book.IsEpub():
			report.Counts.Epubs++
		}
		if book.NeedsLanguageFix() {
			report.Counts.WrongLanguage++
		}
		if book.NotNFC {
			report.Counts.NotNFC++
		}
	}
}

// SaveReport writes report.json and summary.json atomically.
func SaveReport(paths Paths, report Report) error {
	full, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		return i18n.NewError("msg.err_write", "path", paths.Report()).WithCause(err)
	}
	summary := report
	summary.Books = nil
	short, err := json.Marshal(summary)
	if err != nil {
		return i18n.NewError("msg.err_write", "path", paths.Summary()).WithCause(err)
	}
	if err := writeFileAtomic(paths.Report(), full); err != nil {
		return err
	}
	return writeFileAtomic(paths.Summary(), short)
}

// LoadReport reads report.json; ok is false when no scan has run yet.
func LoadReport(paths Paths) (report Report, ok bool, err error) {
	return loadJSON[Report](paths.Report())
}

// LoadSummary reads summary.json (no Books).
func LoadSummary(paths Paths) (summary Report, ok bool, err error) {
	return loadJSON[Report](paths.Summary())
}

func loadJSON[T any](path string) (value T, ok bool, err error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return value, false, nil
	}
	if err != nil {
		return value, false, i18n.NewError("msg.err_read", "path", path).WithCause(err)
	}
	if err := json.Unmarshal(content, &value); err != nil {
		return value, false, i18n.NewError("msg.err_report_broken", "path", path).WithCause(err)
	}
	return value, true, nil
}

// Progress is data/progress.json while a run is working.
type Progress struct {
	Phase  string `json:"phase"` // "scan" or "fix"
	Done   int    `json:"done"`
	Total  int    `json:"total"`
	Detail string `json:"detail,omitempty"`
}

// progressWriter limits writes to the Kobo's storage: at most one per second plus the first
// and the last.
type progressWriter struct {
	path     string
	phase    string
	lastSave time.Time
}

func (writer *progressWriter) update(done, total int, detail string) {
	force := done == 0 || done >= total
	if !force && time.Since(writer.lastSave) < time.Second {
		return
	}
	writer.lastSave = time.Now()
	content, err := json.Marshal(Progress{Phase: writer.phase, Done: done, Total: total, Detail: detail})
	if err == nil {
		_ = writeFileAtomic(writer.path, content)
	}
}

func (writer *progressWriter) clear() { _ = os.Remove(writer.path) }

// ReadProgress returns the running progress, ok false when none is recorded.
func ReadProgress(paths Paths) (Progress, bool) {
	progress, ok, err := loadJSON[Progress](paths.Progress())
	return progress, ok && err == nil
}

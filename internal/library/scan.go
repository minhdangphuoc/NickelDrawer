package library

import (
	"path/filepath"
	"time"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

const phaseScan = "scan"

// Service runs the library operations against one app folder.
type Service struct {
	Paths  Paths
	Logger *Logger
	Now    func() time.Time
}

// NewService opens the log under the app's data/ folder.
func NewService(paths Paths) *Service {
	return &Service{Paths: paths, Logger: NewLogger(paths.Log()), Now: time.Now}
}

func absoluteOf(relative string) string {
	return filepath.Join(storage.Root(), filepath.FromSlash(relative))
}

// Scan analyzes every book, finds duplicates and writes report.json. It purges old trash first.
// The caller holds the lock.
func (service *Service) Scan() (Report, error) {
	service.PurgeTrash()
	progress := &progressWriter{path: service.Paths.Progress(), phase: phaseScan}
	defer progress.clear()
	discovery := Discover(service.Logger)
	books := make([]Book, 0, len(discovery.Files))
	for index, file := range discovery.Files {
		if service.Cancelled() {
			return Report{}, ErrCancelled
		}
		progress.update(index, len(discovery.Files), file.Relative)
		books = append(books, Analyze(file))
	}
	progress.update(len(discovery.Files), len(discovery.Files), "")
	if !HashShared(books, absoluteOf, service.Logger, service.Cancelled) {
		return Report{}, ErrCancelled
	}
	report := BuildReport(books, discovery.Skipped, service.Now())
	if err := SaveReport(service.Paths, report); err != nil {
		return Report{}, err
	}
	service.Logger.Printf("scan: %d files, %d wrong language, %d not NFC, %d duplicate groups",
		report.Counts.Files, report.Counts.WrongLanguage, report.Counts.NotNFC, report.Counts.DuplicateGroups)
	return report, nil
}

// requireReport loads the last report or explains that a scan must come first.
func (service *Service) requireReport() (Report, error) {
	report, ok, err := LoadReport(service.Paths)
	if err != nil {
		return Report{}, err
	}
	if !ok {
		return Report{}, i18n.NewError("msg.err_no_report")
	}
	return report, nil
}

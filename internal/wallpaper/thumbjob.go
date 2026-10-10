package wallpaper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

// ThumbnailJobFiles are the /tmp files of the background thumbnail job. They live outside
// /mnt/onboard, so nothing stays open on the user's storage, and a reboot clears them.
type ThumbnailJobFiles struct {
	PidFile    string // exists while a thumbnail job runs (the lock)
	FailedFile string // ids the last job could not make, so the state does not restart it for them
}

// ThumbnailReport counts what one thumbnail job did.
type ThumbnailReport struct {
	Made      int
	Failed    []int
	IsStopped bool // an import or re-fit started; it writes its own thumbnails
}

// MakeMissingThumbnails is the background thumbnail job: it makes every missing thumbnail, one
// picture at a time, and stops early when shouldStop says another job took over the library.
// Thumbnails are never made in the state read: decoding one picture takes seconds on the Kobo.
func (service *Service) MakeMissingThumbnails(shouldStop func() bool) (ThumbnailReport, error) {
	library, err := LoadLibrary(service.Paths)
	if err != nil {
		return ThumbnailReport{}, err
	}
	var report ThumbnailReport
	for _, id := range library.MissingThumbnails() {
		if shouldStop() {
			report.IsStopped = true
			break
		}
		if library.makeThumbnailFromImage(id, service.Log) {
			report.Made++
		} else {
			report.Failed = append(report.Failed, id)
		}
	}
	service.Log.Printf("thumbnails: %d made, %d failed%s", report.Made, len(report.Failed), stoppedNote(report.IsStopped))
	return report, nil
}

func stoppedNote(isStopped bool) string {
	if isStopped {
		return ", stopped for an import"
	}
	return ""
}

// makeThumbnailFromImage reads the stored picture and writes its thumbnail. A picture removed or
// re-fitted meanwhile (an atomic rename makes it another file) is skipped: no orphan thumbnail,
// and none of the old fit over the one the re-fit wrote.
func (library *Library) makeThumbnailFromImage(id int, log *Logger) bool {
	before, statErr := os.Stat(library.Paths.Image(id))
	content, err := os.ReadFile(library.Paths.Image(id))
	if err != nil || statErr != nil {
		log.Printf("thumbnail %s: cannot read the picture: %v", imageFileName(id), errors.Join(statErr, err))
		return false
	}
	thumb, err := MakeThumbnail(content)
	if err != nil {
		log.Printf("thumbnail %s not made: %v", imageFileName(id), err)
		return false
	}
	if after, err := os.Stat(library.Paths.Image(id)); err != nil || !os.SameFile(before, after) {
		log.Printf("thumbnail %s skipped: the picture was removed or replaced meanwhile", imageFileName(id))
		return false
	}
	if err := os.MkdirAll(library.Paths.Thumbs(), 0o755); err != nil {
		log.Printf("thumbnail %s not written: cannot create %s: %v", imageFileName(id), library.Paths.Thumbs(), err)
		return false
	}
	if err := writeFileAtomic(library.Paths.Thumb(id), ".thumb-", thumb); err != nil {
		log.Printf("thumbnail %s not written: %v", imageFileName(id), err)
		return false
	}
	return true
}

// IsJobRunning tells whether the process named in pidFile is alive.
func IsJobRunning(pidFile string) bool { return isProcessAlive(pidFile) }

// AcquireJobLock creates pidFile holding this process id. It fails (false, no error) while
// another live process holds it; a pid file left by a dead process is taken over.
func AcquireJobLock(pidFile string) (release func(), isAcquired bool, err error) {
	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(pidFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			return writeLockPid(file, pidFile)
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, false, fmt.Errorf("cannot create the lock %s: %w", pidFile, err)
		}
		if isProcessAlive(pidFile) {
			return nil, false, nil
		}
		if err := os.Remove(pidFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("cannot remove the stale lock %s: %w", pidFile, err)
		}
	}
	return nil, false, nil // another process took the stale lock over first
}

func writeLockPid(file *os.File, pidFile string) (func(), bool, error) {
	pid := strconv.Itoa(os.Getpid())
	_, writeErr := file.WriteString(pid + "\n")
	if err := errors.Join(writeErr, file.Close()); err != nil {
		os.Remove(pidFile)
		return nil, false, fmt.Errorf("cannot write the lock %s: %w", pidFile, err)
	}
	release := func() {
		if content, err := os.ReadFile(pidFile); err == nil && strings.TrimSpace(string(content)) == pid {
			os.Remove(pidFile)
		}
	}
	return release, true, nil
}

// SaveFailedThumbnails remembers the ids the job could not make ("" path: nowhere); an empty list
// removes the file.
func SaveFailedThumbnails(path string, ids []int) error {
	if path == "" {
		return nil
	}
	if len(ids) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	content, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, ".thumbs-failed-", content)
}

// NeedsThumbnailJob is true when a picture lacks a thumbnail that the last job did not already
// fail on, so a broken picture does not restart the job on every state read.
func NeedsThumbnailJob(missing []int, failedFile string) bool {
	failed := loadFailedThumbnails(failedFile)
	for _, id := range missing {
		if !slices.Contains(failed, id) {
			return true
		}
	}
	return false
}

func loadFailedThumbnails(path string) []int {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ids []int
	if json.Unmarshal(content, &ids) != nil {
		return nil
	}
	return ids
}

package wallpaper

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestJobLockAllowsOneHolderAndReleasesOnlyItsOwnFile(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "thumbs.pid")
	release, isAcquired, err := AcquireJobLock(pidFile)
	if err != nil || !isAcquired {
		t.Fatalf("first acquire: %v, %v", isAcquired, err)
	}
	if !IsJobRunning(pidFile) {
		t.Error("the holder (this process) must count as running")
	}
	if _, isAcquired, err := AcquireJobLock(pidFile); err != nil || isAcquired {
		t.Fatalf("second acquire while held: %v, %v", isAcquired, err)
	}
	release()
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Errorf("release must remove the pid file: %v", err)
	}
}

func TestJobLockTakesOverAStaleFile(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "thumbs.pid")
	put(t, pidFile, []byte("999999999\n")) // no such process
	release, isAcquired, err := AcquireJobLock(pidFile)
	if err != nil || !isAcquired {
		t.Fatalf("stale lock not taken over: %v, %v", isAcquired, err)
	}
	defer release()
	content, _ := os.ReadFile(pidFile)
	if string(content) != strconv.Itoa(os.Getpid())+"\n" {
		t.Errorf("pid file holds %q", content)
	}
}

func TestReleaseLeavesALockSomeoneElseTookOver(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "thumbs.pid")
	release, _, err := AcquireJobLock(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	put(t, pidFile, []byte("1\n"))
	release()
	if _, err := os.Stat(pidFile); err != nil {
		t.Errorf("another holder's pid file must stay: %v", err)
	}
}

func TestFailedThumbnailsDoNotRestartTheJob(t *testing.T) {
	failedFile := filepath.Join(t.TempDir(), "thumbs.failed")
	if !NeedsThumbnailJob([]int{2}, failedFile) {
		t.Error("a missing thumbnail without a failed record needs the job")
	}
	if err := SaveFailedThumbnails(failedFile, []int{2}); err != nil {
		t.Fatal(err)
	}
	if NeedsThumbnailJob([]int{2}, failedFile) {
		t.Error("a thumbnail the last job failed on must not restart it")
	}
	if !NeedsThumbnailJob([]int{2, 3}, failedFile) {
		t.Error("a new missing thumbnail needs the job")
	}
	if err := SaveFailedThumbnails(failedFile, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(failedFile); !os.IsNotExist(err) {
		t.Errorf("an empty failure list removes the file: %v", err)
	}
}

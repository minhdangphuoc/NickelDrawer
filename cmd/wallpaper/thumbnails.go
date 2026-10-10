package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"koboapps/internal/wallpaper"
)

const defaultThumbsPidFile = "/tmp/wallpaper-thumbs.pid"

// startDetached starts this binary with args in its own session with no open pipes, so the
// drawer neither waits for it nor kills it with the state script's process group. Tests replace it.
var startDetached = func(args []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find the wallpaper binary: %w", err)
	}
	command := exec.Command(executable, args...)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("cannot start %s: %w", executable, err)
	}
	return command.Process.Release()
}

func (current *session) thumbnailJobFiles() wallpaper.ThumbnailJobFiles {
	return wallpaper.ThumbnailJobFiles{PidFile: current.flags.thumbsPid, FailedFile: current.flags.thumbsPid + ".failed"}
}

// startThumbnailJobIfNeeded hands missing thumbnails to the background job. It does not start
// one while an import or re-fit runs (that job writes thumbnails itself) or while one is
// already working. A failure to start is logged; the gallery shows placeholders meanwhile.
func (current *session) startThumbnailJobIfNeeded(state wallpaper.ScreenState) {
	files := current.thumbnailJobFiles()
	if state.Busy || wallpaper.IsJobRunning(files.PidFile) || !wallpaper.NeedsThumbnailJob(state.MissingThumbnails, files.FailedFile) {
		return
	}
	args := []string{"thumbnails", "--app-dir", current.flags.appDir, "--pid-file", current.flags.pidFile,
		"--thumbs-pid-file", files.PidFile}
	if err := startDetached(args); err != nil {
		current.service.Log.Printf("thumbnails: %v", err)
		return
	}
	current.service.Log.Printf("thumbnails: background job started for %d picture(s)", len(state.MissingThumbnails))
}

// makeThumbnails is the background job: one at a time (the lock), stopping when an import starts.
func (current *session) makeThumbnails([]string) error {
	files := current.thumbnailJobFiles()
	release, isAcquired, err := wallpaper.AcquireJobLock(files.PidFile)
	if err != nil {
		current.service.Log.Printf("thumbnails: %v", err)
		return err
	}
	if !isAcquired {
		return nil // another job is making them
	}
	defer release()
	isImportRunning := func() bool { return wallpaper.IsJobRunning(current.flags.pidFile) }
	report, err := current.service.MakeMissingThumbnails(isImportRunning)
	if err != nil {
		current.service.Log.Printf("thumbnails: %s", current.messages.Describe(err))
		return err
	}
	if err := wallpaper.SaveFailedThumbnails(files.FailedFile, report.Failed); err != nil {
		current.service.Log.Printf("thumbnails: cannot write %s: %v", files.FailedFile, err)
	}
	return nil
}

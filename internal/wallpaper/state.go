package wallpaper

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Picture is one gallery item as the service knows it; the command localizes the texts.
type Picture struct {
	ID        string // "0003"
	Thumb     string // absolute path, "" while not generated
	Image     string // absolute path of the converted picture
	Name      string // original file name
	Width     int
	Height    int
	Added     time.Time // zero when unknown
	IsCurrent bool
}

// ScreenState is what the Wallpaper screen is built from.
type ScreenState struct {
	Pictures   []Picture
	CurrentID  string
	Mode       Mode
	Rotation   Rotation
	LastImport string

	Busy     bool
	Progress *Progress // the running job's progress, when there is one

	SleepScreenReady bool
}

// ScreenStateFiles are the /tmp files the action scripts leave behind; they are not part of the
// app's data, so tests point them elsewhere.
type ScreenStateFiles struct {
	LastOutput   string // text printed by the last background import
	PidFile      string // exists while a background run is working
	ProgressFile string // progress of the running import or re-fit
	ConfPath     string // Kobo eReader.conf, read only, for the sleep screen settings
}

// ScreenState reads the library, settings and the script leftovers. It only reads files the
// background job replaces atomically, so it never waits for the job. It creates at most
// maxLazyThumbnails missing thumbnails.
func (service *Service) ScreenState(files ScreenStateFiles) (ScreenState, error) {
	library, config, err := service.load()
	if err != nil {
		return ScreenState{}, err
	}
	library.ensureThumbnails(maxLazyThumbnails, service.Log)
	state := ScreenState{Mode: config.Mode, Rotation: config.Rotation, Pictures: []Picture{}}
	for _, entry := range library.Entries {
		picture := service.pictureOf(entry, entry.ID == config.Current)
		if picture.IsCurrent {
			state.CurrentID = picture.ID
		}
		state.Pictures = append(state.Pictures, picture)
	}
	state.LastImport = firstLine(files.LastOutput)
	state.Busy = isProcessAlive(files.PidFile)
	if progress, found, isFresh := ReadProgress(files.ProgressFile); found && (state.Busy || isFresh) {
		state.Progress, state.Busy = &progress, true
	}
	state.SleepScreenReady, _ = SleepScreenReady(files.ConfPath)
	return state, nil
}

func (service *Service) pictureOf(entry Entry, isCurrent bool) Picture {
	picture := Picture{
		ID: pictureID(entry.ID), Image: service.Paths.Image(entry.ID), Name: entry.Name,
		Width: entry.Width, Height: entry.Height, IsCurrent: isCurrent,
	}
	if added, err := time.Parse(time.RFC3339, entry.Added); err == nil {
		picture.Added = added
	}
	if _, err := os.Stat(service.Paths.Thumb(entry.ID)); err == nil {
		picture.Thumb = service.Paths.Thumb(entry.ID)
	}
	return picture
}

func firstLine(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(content)), "\n")
	return line
}

func isProcessAlive(pidFile string) bool {
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	return err == nil && pid > 0 && syscall.Kill(pid, 0) == nil
}

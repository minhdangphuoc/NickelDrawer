package wallpaper

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// ScreenState is what the Wallpaper screen binds to; the JSON keys are its variable names.
type ScreenState struct {
	Count      int      `json:"count"`
	Current    string   `json:"current"`
	Mode       Mode     `json:"mode"`
	Rotation   Rotation `json:"rotation"`
	LastImport string   `json:"last_import"`
	Busy       bool     `json:"busy"`
}

// ScreenStateFiles are the /tmp files the action scripts leave behind; they are not part of the
// app's data, so tests point them elsewhere.
type ScreenStateFiles struct {
	LastOutput string // text printed by the last background import
	PidFile    string // exists while a background run is working
}

// ScreenState reads the library, settings and the script leftovers. It never converts a picture.
func (service *Service) ScreenState(files ScreenStateFiles) (ScreenState, error) {
	library, config, err := service.load()
	if err != nil {
		return ScreenState{}, err
	}
	state := ScreenState{Count: len(library.Entries), Mode: config.Mode, Rotation: config.Rotation}
	if entry, found := library.find(config.Current); found {
		state.Current = entry.Name
	}
	state.LastImport = firstLine(files.LastOutput)
	state.Busy = isProcessAlive(files.PidFile)
	return state, nil
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

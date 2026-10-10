package main

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	"koboapps/internal/i18n"
	"koboapps/internal/wallpaper"
)

// pictureJSON is one gallery item; see docs/spec/wallpaper-gallery.md section 3.2.
type pictureJSON struct {
	ID    string `json:"id"`
	Thumb string `json:"thumb,omitempty"`
	Image string `json:"image"`
	Label string `json:"label"`
	Info  string `json:"info"`
	Badge string `json:"badge,omitempty"`
}

// screenJSON holds the screen's variables; the JSON keys are their names.
type screenJSON struct {
	Status           string        `json:"status"`
	Summary          string        `json:"summary"`
	Pictures         []pictureJSON `json:"pictures"`
	CurrentID        string        `json:"current_id"`
	Count            int           `json:"count"`
	Mode             string        `json:"mode"`
	Rotation         string        `json:"rotation"`
	Busy             bool          `json:"busy"`
	Pending          int           `json:"pending"`
	ImportStep       string        `json:"import_step"`
	SleepScreenReady bool          `json:"sleep_screen_ready"`
	SleepScreenHint  string        `json:"sleep_screen_hint"`
	// SleepScreenNotice is the one-line link on the main screen to the options; empty when ready.
	SleepScreenNotice string `json:"sleep_screen_notice"`
	SleepScreenLine   string `json:"sleep_screen_line"`
	LastImport        string `json:"last_import"`
}

// state prints the variables of the app screen as one JSON object. Texts are localized here;
// the last import line was written in the device language by the import run.
func (current *session) state([]string) error {
	state, err := current.service.ScreenState(current.stateFiles)
	if err != nil {
		return err
	}
	return json.NewEncoder(current.out).Encode(current.screenOf(state))
}

func (current *session) screenOf(state wallpaper.ScreenState) screenJSON {
	screen := screenJSON{
		Pictures: make([]pictureJSON, 0, len(state.Pictures)), CurrentID: state.CurrentID, Count: len(state.Pictures),
		Mode: string(state.Mode), Rotation: string(state.Rotation), Busy: state.Busy,
		SleepScreenReady: state.SleepScreenReady, LastImport: state.LastImport,
		SleepScreenHint: current.sleepScreenHint(state), Status: current.headerStatus(state),
	}
	for _, picture := range state.Pictures {
		screen.Pictures = append(screen.Pictures, current.pictureOf(picture))
	}
	screen.Summary = current.summary(state)
	screen.SleepScreenLine = screen.SleepScreenHint
	if !state.SleepScreenReady {
		screen.SleepScreenNotice = current.messages.Message("msg.sleep_screen_notice", nil)
	}
	if state.SleepScreenReady {
		screen.SleepScreenLine = current.messages.Message("msg.sleep_screen_ready", nil)
	}
	if state.Progress != nil {
		screen.Pending = max(state.Progress.Total-state.Progress.Done, 0)
		screen.ImportStep = current.progressStep(*state.Progress)
	}
	return screen
}

func (current *session) pictureOf(picture wallpaper.Picture) pictureJSON {
	item := pictureJSON{ID: picture.ID, Thumb: picture.Thumb, Image: picture.Image, Label: labelOf(picture.Name), Info: current.info(picture)}
	if picture.IsCurrent {
		item.Badge = "current"
	}
	return item
}

// labelOf is the file name without its extension.
func labelOf(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

// info is the multi-line text of the Info dialog; lines whose data is unknown are left out.
func (current *session) info(picture wallpaper.Picture) string {
	lines := []string{current.messages.Message("msg.info_name", i18n.Params("name", picture.Name))}
	if picture.Width > 0 && picture.Height > 0 {
		lines = append(lines, current.messages.Message("msg.info_size", i18n.Params("width", strconv.Itoa(picture.Width), "height", strconv.Itoa(picture.Height))))
	}
	if !picture.Added.IsZero() {
		lines = append(lines, current.messages.Message("msg.info_added", i18n.Params("date", picture.Added.Local().Format("02/01/2006"))))
	}
	return strings.Join(lines, "\n")
}

// summary reads like "12 ảnh · Đang dùng: Núi trắng · Lấp đầy"; empty when there are no pictures.
func (current *session) summary(state wallpaper.ScreenState) string {
	if len(state.Pictures) == 0 {
		return ""
	}
	params := i18n.Params("count", strconv.Itoa(len(state.Pictures)), "mode", current.messages.Message("msg.mode_label_"+string(state.Mode), nil))
	key := "msg.summary_no_current"
	for _, picture := range state.Pictures {
		if picture.IsCurrent {
			params["name"], key = labelOf(picture.Name), "msg.summary"
		}
	}
	return current.messages.Message(key, params)
}

func (current *session) headerStatus(state wallpaper.ScreenState) string {
	progress := state.Progress
	switch {
	case progress != nil && progress.Kind == wallpaper.ProgressRefit:
		return current.messages.Message("msg.state_refitting", progressParams(*progress))
	case progress != nil:
		return current.messages.Message("msg.state_importing", progressParams(*progress))
	case state.Busy:
		return current.messages.Message("msg.state_working", nil)
	}
	return "" // idle: the summary line under the header already says how many pictures there are
}

func progressParams(progress wallpaper.Progress) map[string]string {
	return i18n.Params("done", strconv.Itoa(progress.Done), "total", strconv.Itoa(progress.Total), "name", progress.Name)
}

func (current *session) progressStep(progress wallpaper.Progress) string {
	key := "msg.import_step"
	if progress.Kind == wallpaper.ProgressRefit {
		key = "msg.refit_step"
	}
	return current.messages.Message(key, progressParams(progress))
}

// sleepScreenHint is empty when ready, so the screen's notice stays hidden.
func (current *session) sleepScreenHint(state wallpaper.ScreenState) string {
	if state.SleepScreenReady {
		return ""
	}
	return current.messages.Message("msg.sleep_screen_hint", nil)
}

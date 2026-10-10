package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"koboapps/internal/i18n"
	"koboapps/internal/wallpaper"
)

// session is what one subcommand works with.
type session struct {
	service  *wallpaper.Service
	messages *i18n.Bundle
	out      io.Writer

	stateFiles wallpaper.ScreenStateFiles
	confPath   string
	flags      commonFlags
}

func (current *session) say(key string, pairs ...string) {
	say(current.out, current.messages, key, pairs...)
}

func (current *session) importPictures([]string) error {
	imported, applied, err := current.service.ImportAndApply()
	if err != nil {
		current.sayCounts(imported)
		return err
	}
	current.sayCounts(imported)
	current.say("msg.applied", "count", strconv.Itoa(applied.Shown))
	current.sayHintIfSleepScreenOff()
	return nil
}

// sayHintIfSleepScreenOff tells the user which Kobo settings still hide the pictures.
func (current *session) sayHintIfSleepScreenOff() {
	if ready, _ := wallpaper.SleepScreenReady(current.confPath); !ready {
		current.say("msg.sleep_screen_hint")
	}
}

func (current *session) sayCounts(imported wallpaper.ImportReport) {
	current.say("msg.imported", "added", strconv.Itoa(imported.Added), "duplicates", strconv.Itoa(imported.Duplicates))
	if imported.Failed > 0 {
		current.say("msg.import_failed", "count", strconv.Itoa(imported.Failed))
	}
}

// errNotDue is the quiet answer of first-import-due: not an error, only exit status 1.
var errNotDue = errors.New("first import is not due")

// firstImportDue succeeds (and records the marker) once, when the gallery is empty and Pictures has a picture.
func (current *session) firstImportDue([]string) error {
	due, err := current.service.FirstImportDue()
	if err != nil {
		current.service.Log.Printf("first-import-due failed: %s", current.messages.Describe(err))
		return err
	}
	if !due {
		return errNotDue
	}
	return nil
}

func (current *session) apply([]string) error {
	applied, err := current.service.Apply()
	if err == nil {
		current.say("msg.applied", "count", strconv.Itoa(applied.Shown))
	}
	return err
}

func (current *session) next([]string) error {
	entry, err := current.service.Next()
	if err == nil {
		current.say("msg.next", "number", strconv.Itoa(entry.ID), "name", entry.Name)
	}
	return err
}

func (current *session) setMode(arguments []string) error {
	mode, err := current.resolveMode(arguments[0])
	if err != nil {
		return err
	}
	if current.flags.saveOnly {
		current.say("msg.mode_saved_for_job")
		return current.service.SaveMode(mode)
	}
	report, applied, err := current.service.SetMode(mode)
	if err != nil {
		return err
	}
	current.say("msg.mode_"+string(mode), "count", strconv.Itoa(report.Reconverted))
	if applied.Shown == 0 {
		current.say("msg.mode_saved")
	}
	return nil
}

func (current *session) setRotation(arguments []string) error {
	rotation, err := current.resolveRotation(arguments[0])
	if err != nil {
		return err
	}
	applied, err := current.service.SetRotation(rotation)
	if err != nil {
		return err
	}
	current.say("msg.rotation_"+string(rotation), "count", strconv.Itoa(applied.Shown))
	return nil
}

func (current *session) turnOff([]string) error {
	removed, err := current.service.TurnOff()
	if err == nil {
		current.say("msg.off", "count", strconv.Itoa(removed))
	}
	return err
}

func (current *session) status([]string) error {
	status, err := current.service.Status()
	if err != nil {
		return err
	}
	current.say("msg.status_pictures", "count", strconv.Itoa(status.Pictures))
	current.say("msg.status_mode_" + string(status.Config.Mode))
	current.say("msg.status_rotation_" + string(status.Config.Rotation))
	current.say("msg.status_shown", "count", strconv.Itoa(status.Shown))
	current.sayHintIfSleepScreenOff()
	return nil
}

const toggleWord = "toggle"

// resolveMode turns "toggle" into the other mode, so a menu entry needs no state of its own.
func (current *session) resolveMode(word string) (wallpaper.Mode, error) {
	if word != toggleWord {
		return wallpaper.ParseMode(word)
	}
	status, err := current.service.Status()
	if err != nil {
		return "", err
	}
	if status.Config.Mode == wallpaper.ModeFill {
		return wallpaper.ModeFit, nil
	}
	return wallpaper.ModeFill, nil
}

func (current *session) resolveRotation(word string) (wallpaper.Rotation, error) {
	if word != toggleWord {
		return wallpaper.ParseRotation(word)
	}
	status, err := current.service.Status()
	if err != nil {
		return "", err
	}
	if status.Config.Rotation == wallpaper.RotationFixed {
		return wallpaper.RotationRandom, nil
	}
	return wallpaper.RotationFixed, nil
}

func (current *session) setCurrent(arguments []string) error {
	id, err := wallpaper.ParsePictureID(arguments[0])
	if err != nil {
		return err
	}
	entry, err := current.service.SetCurrent(id)
	if err == nil {
		current.say("msg.set_current", "id", pictureLabel(entry), "name", entry.Name)
	}
	return err
}

func pictureLabel(entry wallpaper.Entry) string { return fmt.Sprintf("%04d", entry.ID) }

// remove forgets one picture. The screens always ask first and pass --confirm.
func (current *session) remove(arguments []string) error {
	id, err := wallpaper.ParsePictureID(arguments[0])
	if err != nil {
		return err
	}
	if !current.flags.confirm {
		return i18n.NewError("msg.err_confirm_needed", "id", arguments[0])
	}
	report, err := current.service.Remove(id, current.flags.deleteOriginal)
	if err != nil {
		return err
	}
	key := "msg.removed"
	switch {
	case report.OriginalDeleted:
		key = "msg.removed_original"
	case report.OriginalNotFound:
		key = "msg.removed_original_kept"
	}
	current.say(key, "id", pictureLabel(report.Entry), "name", report.Entry.Name)
	return nil
}

package main

import (
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
	return nil
}

func (current *session) sayCounts(imported wallpaper.ImportReport) {
	current.say("msg.imported", "added", strconv.Itoa(imported.Added), "duplicates", strconv.Itoa(imported.Duplicates))
	if imported.Failed > 0 {
		current.say("msg.import_failed", "count", strconv.Itoa(imported.Failed))
	}
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

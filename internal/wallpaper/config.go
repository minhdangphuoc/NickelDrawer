package wallpaper

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"koboapps/internal/i18n"
)

// Mode says how a picture is fitted to the screen.
type Mode string

// Rotation says how the one picture in the screensaver folder changes.
type Rotation string

const (
	ModeFill Mode = "fill" // cover the screen, crop the overflow
	ModeFit  Mode = "fit"  // whole picture, white bars

	RotationFixed  Rotation = "fixed"  // exactly the current picture
	RotationRandom Rotation = "random" // `wallpaper next` picks a new one after each wake
)

// Config is data/config.json.
type Config struct {
	Mode     Mode     `json:"mode"`
	Rotation Rotation `json:"rotation"`
	Current  int      `json:"current"`
}

// DefaultConfig is used until the user changes something.
func DefaultConfig() Config { return Config{Mode: ModeFill, Rotation: RotationFixed} }

// ParseMode validates user input.
func ParseMode(text string) (Mode, error) {
	switch Mode(text) {
	case ModeFill, ModeFit:
		return Mode(text), nil
	}
	return "", i18n.NewError("msg.err_mode_invalid", "value", text)
}

// ParseRotation validates user input.
func ParseRotation(text string) (Rotation, error) {
	switch Rotation(text) {
	case RotationFixed, RotationRandom:
		return Rotation(text), nil
	}
	return "", i18n.NewError("msg.err_rotation_invalid", "value", text)
}

// LoadConfig reads config.json; a missing file gives the defaults.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, i18n.NewError("msg.err_config_read", "path", path).WithCause(err)
	}
	config := DefaultConfig()
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, i18n.NewError("msg.err_config_read", "path", path).WithCause(err)
	}
	if _, err := ParseMode(string(config.Mode)); err != nil {
		return Config{}, err
	}
	if _, err := ParseRotation(string(config.Rotation)); err != nil {
		return Config{}, err
	}
	return config, nil
}

// SaveConfig writes config.json atomically.
func SaveConfig(path string, config Config) error {
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return i18n.NewError("msg.err_write", "path", path).WithCause(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", filepath.Dir(path)).WithCause(err)
	}
	return writeFileAtomic(path, ".config-", append(encoded, '\n'))
}

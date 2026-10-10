package tweaks

import (
	"os"

	"koboapps/internal/i18n"
)

func readScreensaver(c Config, _ *confFile) any { return directoryExists(c.ScreensaverDir) }

func readBoolean(setting booleanSetting) func(Config, *confFile) any {
	return func(_ Config, conf *confFile) any { return setting.read(conf) }
}

func readLegibility(_ Config, conf *confFile) any {
	value, found := conf.get(preferencesSection, textRenderingKey)
	return found && value == legibleTextRendering
}

func readIP(c Config, _ *confFile) any {
	if address := c.ipAddress(); address != "" {
		return address
	}
	return c.text("tweak.ip.none", nil)
}

// toggleScreensaver renames the folder instead of deleting it, so the pictures survive.
func (c Config) toggleScreensaver() (Result, error) {
	enabled := directoryExists(c.ScreensaverDir)
	from, to := c.ScreensaverDir, c.disabledScreensaverDir()
	if !enabled {
		from, to = to, from
	}
	if !enabled && !directoryExists(from) {
		return Result{}, c.fail("tweak.err.no_screensaver", "folder", c.ScreensaverDir)
	}
	if err := os.Rename(from, to); err != nil {
		return Result{}, c.writeFailure(err)
	}
	return c.toggled(IDScreensaver, !enabled, false), nil
}

func toggleBoolean(id string, setting booleanSetting) func(Config) (Result, error) {
	return func(c Config) (Result, error) {
		newValue, err := c.editConf(func(conf *confFile) bool {
			enabled := !setting.read(conf)
			setting.write(conf, enabled)
			return enabled
		})
		if err != nil {
			return Result{}, err
		}
		return c.toggled(id, newValue, false), nil
	}
}

func (c Config) toggleLegibility() (Result, error) {
	newValue, err := c.editConf(func(conf *confFile) bool {
		value, found := conf.get(preferencesSection, textRenderingKey)
		enabled := !(found && value == legibleTextRendering)
		if enabled {
			conf.set(preferencesSection, textRenderingKey, legibleTextRendering)
		} else {
			conf.remove(preferencesSection, textRenderingKey)
		}
		return enabled
	})
	if err != nil {
		return Result{}, err
	}
	return c.toggled(IDLegibility, newValue, true), nil
}

func (c Config) invertAndReboot() (Result, error) {
	if _, err := c.editConf(func(conf *confFile) bool {
		enabled := !invertSetting.read(conf)
		invertSetting.write(conf, enabled)
		return enabled
	}); err != nil {
		return Result{}, err
	}
	return Result{V: ProtocolVersion, OK: true, Toast: c.text("tweak.toast.reboot", nil), Nickel: NickelReboot}, nil
}

func nickelAction(nickel, toastKey string) func(Config) (Result, error) {
	return func(c Config) (Result, error) {
		return Result{V: ProtocolVersion, OK: true, Toast: c.text(toastKey, nil), Nickel: nickel}, nil
	}
}

// editConf reads the conf file, lets change edit it, and saves it atomically.
func (c Config) editConf(change func(*confFile) bool) (bool, error) {
	conf, err := openConf(c.ConfPath)
	if err != nil {
		return false, c.writeFailure(err)
	}
	result := change(conf)
	if err := conf.save(); err != nil {
		return false, c.writeFailure(err)
	}
	return result, nil
}

// toggled builds the reply of a toggle; needsRestart adds the reboot hint to the toast.
func (c Config) toggled(id string, enabled, needsRestart bool) Result {
	state := "off"
	if enabled {
		state = "on"
	}
	key := "tweak.toast." + state
	if needsRestart {
		key = "tweak.toast." + state + "_restart"
	}
	return Result{
		V: ProtocolVersion, OK: true, Value: &enabled,
		Toast: c.text(key, i18n.Params("label", c.label(id))),
	}
}

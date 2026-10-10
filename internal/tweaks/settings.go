package tweaks

// Nickel's QSettings names behind the NickelMenu nickel_setting items of the same name.
// Verify them on a device before trusting a new firmware (docs/spec/tweaks.md).
const (
	featureSection     = "FeatureSettings"
	screenshotsKey     = "Screenshots"
	invertScreenKey    = "InvertScreen"
	developerSection   = "DeveloperSettings"
	autoUsbKey         = "AutoUsbGadget"
	preferencesSection = "ApplicationPreferences"
	textRenderingKey   = "webkitTextRendering"

	// legibleTextRendering is the value that makes the reader's web view draw crisper text.
	legibleTextRendering = "geometricPrecision"

	trueText  = "true"
	falseText = "false"
)

// booleanSetting is a true/false key in Kobo eReader.conf.
type booleanSetting struct {
	section, key string
}

var (
	screenshotsSetting = booleanSetting{featureSection, screenshotsKey}
	autoUsbSetting     = booleanSetting{developerSection, autoUsbKey}
	invertSetting      = booleanSetting{featureSection, invertScreenKey}
)

func (setting booleanSetting) read(conf *confFile) bool {
	value, found := conf.get(setting.section, setting.key)
	return found && value == trueText
}

func (setting booleanSetting) write(conf *confFile, enabled bool) {
	text := falseText
	if enabled {
		text = trueText
	}
	conf.set(setting.section, setting.key, text)
}

package tweaks

// Nickel's QSettings names, checked against libnickel 4.38.23828 (helpers/extras/docs/tweaks-modal.md).
// Verify them on a device before trusting a new firmware (helpers/extras/docs/tweaks.md).
const (
	featureSection   = "FeatureSettings"
	invertScreenKey  = "InvertScreen"
	screenshotsKey   = "Screenshots"
	developerSection = "DeveloperSettings"
	autoUsbKey       = "AutoUsbGadget"
	readingSection   = "Reading"
	textRenderingKey = "webkitTextRendering"

	// legibleTextRendering goes straight into the kepub page's CSS text-rendering;
	// Nickel reads it when a book's page style is built, so reopening the book applies it.
	legibleTextRendering = "optimizeLegibility"

	trueText  = "true"
	falseText = "false"
)

// booleanSetting is a true/false key in Kobo eReader.conf.
type booleanSetting struct {
	section, key string
}

var (
	autoUsbSetting = booleanSetting{developerSection, autoUsbKey}
	invertSetting  = booleanSetting{featureSection, invertScreenKey}
	// Nickel's own power-button screenshots; on, the power button no longer puts the Kobo to sleep.
	screenshotsSetting = booleanSetting{featureSection, screenshotsKey}
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

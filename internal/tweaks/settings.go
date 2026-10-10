package tweaks

// Nickel's QSettings names, checked against libnickel 4.38.23828 (re/notes/tweaks-modal.md).
// Verify them on a device before trusting a new firmware (docs/spec/tweaks.md).
const (
	featureSection   = "FeatureSettings"
	screenshotsKey   = "Screenshots"
	invertScreenKey  = "InvertScreen"
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

// Package tweaks is the backend of the Extras helper's "Tweaks" panel: the list of device
// tweaks and the application of one (contract: helpers/extras/docs/tweaks.md). It has no UI;
// cmd/extras-tweaks prints the results as JSON. It imports nothing from the drawer or the host.
package tweaks

import (
	"net"
	"os"
	"path/filepath"

	"koboapps/internal/i18n"
	"koboapps/internal/storage"
)

// ProtocolVersion is the "v" field of every response.
const ProtocolVersion = 1

// Kinds of item the modal draws.
const (
	KindToggle = "toggle"
	KindInfo   = "info"
	KindAction = "action"
)

// Native actions the mod performs after the toast.
const (
	NickelReboot        = "reboot"
	NickelSleep         = "sleep"
	NickelRescanLibrary = "rescan-library"
)

// Tweak ids, in display order.
const (
	IDScreensaver      = "screensaver"
	IDScreenshotButton = "screenshot-button"
	IDScreenshots      = "screenshots" // the old power-button setting; listed only while it is on
	IDAutoUSB          = "auto-usb"
	IDLegibility       = "legibility"
	IDIP               = "ip"
	IDInvertReboot     = "invert-reboot"
	IDSleep            = "sleep"
	IDReboot           = "reboot"
	IDLibraryRescan    = "library-rescan"
)

const (
	confRelative            = ".kobo/Kobo/Kobo eReader.conf"
	flagsRelative           = ".adds/helpers/extras"
	screenshotOffFlagName   = "screenshot-off"
	disabledScreensaverName = "screensaver.disabled"
	wifiInterface           = "wlan0"
)

// Item is one row of the modal. Value is a bool for toggles, a string for info rows and
// absent for actions.
type Item struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Caption string `json:"caption,omitempty"`
	Kind    string `json:"kind"`
	Value   any    `json:"value,omitempty"`
	Confirm string `json:"confirm,omitempty"`
}

// Listing is the reply of `drawer tweaks`.
type Listing struct {
	V     int    `json:"v"`
	Items []Item `json:"items"`
}

// Result is the reply of `drawer tweak <id>`.
type Result struct {
	V      int    `json:"v"`
	OK     bool   `json:"ok"`
	Toast  string `json:"toast"`
	Value  *bool  `json:"value,omitempty"`
	Nickel string `json:"nickel,omitempty"`
}

// Failure is the reply when a tweak cannot be applied; the message is already localized.
type Failure struct {
	V     int    `json:"v"`
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// Error carries a localized message for the Failure document.
type Error struct{ Message string }

func (e *Error) Error() string { return e.Message }

// Config holds the files the tweaks touch; tests point them at a temp folder.
type Config struct {
	Bundle         *i18n.Bundle
	ConfPath       string        // Kobo eReader.conf
	ScreensaverDir string        // <root>/.kobo/screensaver; its disabled twin is a sibling
	FlagsDir       string        // the helper's folder; holds the screenshot-off flag file
	IPAddress      func() string // default: the first IPv4 address of wlan0
}

// DefaultConfig uses the device layout below storage.Root().
func DefaultConfig(bundle *i18n.Bundle) Config {
	return Config{
		Bundle:         bundle,
		ConfPath:       filepath.Join(storage.Root(), filepath.FromSlash(confRelative)),
		ScreensaverDir: storage.Path(storage.Screensaver),
		FlagsDir:       filepath.Join(storage.Root(), filepath.FromSlash(flagsRelative)),
	}
}

// tweak describes one item: how it reads and how it is applied.
type tweak struct {
	id, kind string
	read     func(Config, *confFile) any // nil for actions
	apply    func(Config) (Result, error)
	confirm  bool
	caption  bool // a "tweak.<id>.caption" text sits under the label
	// isListed hides the item when it returns false; nil means always listed.
	isListed func(*confFile) bool
}

var registry = []tweak{
	{id: IDScreensaver, kind: KindToggle, read: readScreensaver, apply: Config.toggleScreensaver},
	{id: IDScreenshotButton, kind: KindToggle, caption: true, read: readScreenshotButton, apply: Config.toggleScreenshotButton},
	{id: IDScreenshots, kind: KindToggle, caption: true, isListed: screenshotsSetting.read,
		read: readBoolean(screenshotsSetting), apply: toggleBoolean(IDScreenshots, screenshotsSetting)},
	{id: IDAutoUSB, kind: KindToggle, read: readBoolean(autoUsbSetting), apply: toggleBoolean(IDAutoUSB, autoUsbSetting)},
	{id: IDLegibility, kind: KindToggle, read: readLegibility, apply: Config.toggleLegibility},
	{id: IDIP, kind: KindInfo, read: readIP},
	{id: IDInvertReboot, kind: KindAction, confirm: true, apply: Config.invertAndReboot},
	{id: IDSleep, kind: KindAction, apply: nickelAction(NickelSleep, "tweak.toast.sleep")},
	{id: IDReboot, kind: KindAction, confirm: true, apply: nickelAction(NickelReboot, "tweak.toast.reboot")},
	{id: IDLibraryRescan, kind: KindAction, apply: nickelAction(NickelRescanLibrary, "tweak.toast.rescan")},
}

// List reads the current state of every tweak. A conf file that cannot be read counts as
// all-off: the modal must open even when one setting is broken.
func (c Config) List() Listing {
	conf, err := openConf(c.ConfPath)
	if err != nil {
		conf = &confFile{path: c.ConfPath}
	}
	items := make([]Item, 0, len(registry))
	for _, entry := range registry {
		if entry.isListed == nil || entry.isListed(conf) {
			items = append(items, c.item(entry, conf))
		}
	}
	return Listing{V: ProtocolVersion, Items: items}
}

func (c Config) item(entry tweak, conf *confFile) Item {
	item := Item{ID: entry.id, Kind: entry.kind, Label: c.text("tweak."+entry.id+".label", nil)}
	if entry.caption {
		item.Caption = c.text("tweak."+entry.id+".caption", nil)
	}
	if entry.read != nil {
		item.Value = entry.read(c, conf)
	}
	if entry.confirm {
		item.Confirm = c.text("tweak."+entry.id+".confirm", nil)
	}
	return item
}

// Apply runs one tweak. Info rows and unknown ids are errors.
func (c Config) Apply(id string) (Result, error) {
	for _, entry := range registry {
		if entry.id == id && entry.apply != nil {
			return entry.apply(c)
		}
	}
	return Result{}, c.fail("tweak.err.unknown", "id", id)
}

func (c Config) text(key string, params map[string]string) string {
	return c.Bundle.Message(key, params)
}

func (c Config) fail(key string, pairs ...string) *Error {
	return &Error{Message: c.text(key, i18n.Params(pairs...))}
}

func (c Config) writeFailure(err error) *Error {
	return c.fail("tweak.err.write", "detail", err.Error())
}

func (c Config) label(id string) string { return c.text("tweak."+id+".label", nil) }

func (c Config) screenshotOffFlag() string {
	return filepath.Join(c.FlagsDir, screenshotOffFlagName)
}

func (c Config) disabledScreensaverDir() string {
	return filepath.Join(filepath.Dir(c.ScreensaverDir), disabledScreensaverName)
}

func (c Config) ipAddress() string {
	if c.IPAddress != nil {
		return c.IPAddress()
	}
	return wifiAddress()
}

// wifiAddress is the first IPv4 address of the Wi-Fi interface, or "" when it is down.
func wifiAddress() string {
	wifi, err := net.InterfaceByName(wifiInterface)
	if err != nil {
		return ""
	}
	addresses, err := wifi.Addrs()
	if err != nil {
		return ""
	}
	for _, address := range addresses {
		if network, isNetwork := address.(*net.IPNet); isNetwork && network.IP.To4() != nil {
			return network.IP.String()
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IDs lists every tweak id in display order (for tests that check the locale files).
func IDs() []string {
	ids := make([]string, 0, len(registry))
	for _, entry := range registry {
		ids = append(ids, entry.id)
	}
	return ids
}

// HasCaption reports whether the tweak has a "tweak.<id>.caption" text.
func HasCaption(id string) bool {
	for _, entry := range registry {
		if entry.id == id {
			return entry.caption
		}
	}
	return false
}

// HasConfirm reports whether the tweak asks for confirmation first.
func HasConfirm(id string) bool {
	for _, entry := range registry {
		if entry.id == id {
			return entry.confirm
		}
	}
	return false
}
